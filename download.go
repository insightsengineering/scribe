package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/src-d/go-git.v4"
)

// downloading from private GitLab repositories requires providing username and password in environment variables

const defaultCranMirrorUrl = "https://cloud.r-project.org"
const bioConductorUrl = "https://www.bioconductor.org/packages"

// within below directory:
// tar.gz packages are downloaded to packages subdirectory
// GitHub repositories are cloned into github directory
// GitLab repositories are cloned into repository with name corresponding to GitLab hostname
const localOutputDirectory = "/tmp/scribe/downloadedPackages"
var bioconductorCategories [4]string = [4]string{"bioc", "data/experiment", "data/annotation", "workflows"}
const maxDownloadRoutines = 40

type DownloadInfo struct {
	// if statusCode > 0 it is identical to HTTP status code from download
	// message field contains URL of the package
	// statusCode == -1 means that package version could not be found in any BioConductor repository
	// message field contains error message
	// statusCode == -2 means that there was an error during cloning of GitHub repository
	// message field contains error message
	// statusCode == -3 means that there was an error during cloning of GitLab repository
	// message field contains error message
	// statusCode == -4 means that a network error occured during HTTP download
	// message field contains URL of the package
	StatusCode int `json:"statusCode"`
	Message string `json:"message"`
	ContentLength int64 `json:"contentLength"`
	// file where the package is stored, or directory where git repository has been cloned
	// empty in case of errors
	OutputLocation string `json:"outputLocation"`
}

func getRepositoryUrl(renvLockRepositories []Rrepository, repository_name string) (string) {
	for _, v := range renvLockRepositories {
		if v.Name == repository_name {
			return v.URL
		}
	}
	// return default mirror if the repository is not defined in lock file
	return defaultCranMirrorUrl
}

// Returns HTTP status code for downloaded file and number of bytes in downloaded content.
func downloadFile(url string, outputFile string) (int, int64) {
	// Get the data
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	checkError(err)

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			// Create the file
			out, err := os.Create(outputFile)
			checkError(err)
			defer out.Close()
			// Write the body to file
			_, err = io.Copy(out, resp.Body)
			checkError(err)
		}

		return resp.StatusCode, resp.ContentLength
	}
	return -4, 0
}

// Function executed in parallel goroutines.
func downloadSinglePackage(packageName string, packageVersion string, repoUrl string, hash string, packageSource string,
	currentCranPackageVersions map[string]string, biocPackageVersions map[string]map[string]string,
	biocUrls map[string]string, messages chan DownloadInfo, guard chan struct{}) (error){

	var packageUrl string
	var outputLocation string

	if repoUrl == defaultCranMirrorUrl {
		// Check if package is in current CRAN repository
		versionInCran, ok := currentCranPackageVersions[packageName]
		if ok {
			log.Debug("CRAN current has package ", packageName, " version ", versionInCran, ".")
		} else {
			log.Debug("CRAN current doesn't have ", packageName, " in any version.")
		}
		if ok && versionInCran == packageVersion {
			log.Debug("Retrieving package ", packageName, " from CRAN current.")
			packageUrl = repoUrl + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
		} else {
			// If not, look for the package in Archive.
			log.Debug(
				"Attempting to retrieve ", packageName, " version ", packageVersion,
				" from CRAN Archive.",
			)
			packageUrl = (repoUrl + "/src/contrib/Archive/" + packageName +
				"/" + packageName + "_" + packageVersion + ".tar.gz")
		}
	} else if repoUrl == bioConductorUrl {
		for _, biocCategory := range(bioconductorCategories) {
			biocPackageVersion, ok := biocPackageVersions[biocCategory][packageName]
			if ok {
				log.Debug(
					"BioConductor category ", biocCategory, " has package ", packageName,
					" version ", biocPackageVersion, ".",
				)
				if biocPackageVersion == packageVersion {
					log.Debug("Package ", packageName, " will be retrieved from Bioconductor category ", biocCategory)
					packageUrl = biocUrls[biocCategory] + "/" + packageName + "_" + packageVersion + ".tar.gz"
					break
				}
			}
		}
		// Package not found in any of Bioconductor categories.
		if packageUrl == "" {
			messages <- DownloadInfo{-1, "Couldn't find " + packageName + " version " + packageVersion + " in BioConductor.", 0, ""}
			<- guard
			return nil
		}
	} else if packageSource == "GitHub" {
		// TODO this has to be modified if we plan to support other GitHub instances than https://github.com
		gitDirectory := localOutputDirectory + "/github" + strings.TrimPrefix(repoUrl, "https://github.com")
		os.MkdirAll(gitDirectory, os.ModePerm)
		log.Debug("Cloning repo to ", gitDirectory)
		_, err := git.PlainClone(
			gitDirectory, false, &git.CloneOptions{
			URL: repoUrl,
		})
		if err == nil {
			// TODO checkout Hash
			// TODO how can we know the size of repository in bytes
			messages <- DownloadInfo{200, repoUrl, 0, gitDirectory}
		} else {
			messages <- DownloadInfo{-2, "Error while cloning repo " + repoUrl + ": " + err.Error(), 0, ""}
		}
		<- guard
		return nil
	} else if packageSource == "GitLab" {
		remoteHost := strings.Join(strings.Split(repoUrl, "/")[:3], "/")
		remoteUser := strings.Split(repoUrl, "/")[3]
		remoteRepo := strings.Join(strings.Split(repoUrl, "/")[4:], "/")

		gitDirectory := localOutputDirectory + "/" + strings.Split(repoUrl, "/")[2] +
			"/" + remoteUser + "/" + remoteRepo
		os.MkdirAll(gitDirectory, os.ModePerm)
		log.Debug("Cloning repo ", remoteUser, "/", remoteRepo, " from host ", remoteHost, " to directory ", gitDirectory)
		_, err := git.PlainClone(
			gitDirectory, false, &git.CloneOptions{
			URL: repoUrl,
			// TODO document this, or change the way credentials are passed
			Auth: &githttp.BasicAuth{Username: os.Getenv("GITLAB_USER"), Password: os.Getenv("GITLAB_TOKEN")},
		})
		if err == nil {
			// TODO how can we know the size of repository in bytes
			// TODO checkout Hash
			messages <- DownloadInfo{200, repoUrl, 0, ""}
		} else {
			messages <- DownloadInfo{-3, "Error while cloning repo " + repoUrl + ": " + err.Error(), 0, gitDirectory}
		}
		<- guard
		return nil
	} else {
		// Repositories other than CRAN or BioConductor
		packageUrl = repoUrl + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
	}

	outputLocation = localOutputDirectory + "/packages/" + packageName + "_" + packageVersion + ".tar.gz"

	statusCode, contentLength := downloadFile(
		packageUrl, outputLocation,
	)
	messages <- DownloadInfo{statusCode, packageUrl, contentLength, outputLocation}
	<- guard

	return nil
}

// Read PACKAGES file and return map of packages and their versions as stored in the PACKAGES file.
func getPackageVersions(filePath string, packageVersions map[string]string) {
	packages, err := os.Open(filePath)
	checkError(err)
	defer packages.Close()

	scanner := bufio.NewScanner(packages)
	// Iterate through lines of PACKAGES file.
	for scanner.Scan() {
		newLine := scanner.Text()
		if strings.HasPrefix(newLine, "Package:") {
			packageFields := strings.Fields(newLine)
			packageName := packageFields[1]

			// Read next line after 'Package:' to get 'Version:'.
			scanner.Scan()
			nextLine := scanner.Text()
			versionFields := strings.Fields(nextLine)
			packageVersion := versionFields[1]
			packageVersions[packageName] = packageVersion
		}
	}
}

// Receive messages from goroutines responsible for package downloads.
func downloadResultReceiver(messages chan DownloadInfo, successfulDownloads *int,
	failedDownloads *int, totalPackages int, totalDownloadedBytes *int64, downloadWaiter chan struct{},
	downloadErrors *string, allDownloadInfo *[]DownloadInfo) {
	*successfulDownloads = 0
	*failedDownloads = 0
	idleSeconds := 0
	var bar progressbar.ProgressBar
	if Interactive {
		bar = *progressbar.Default(
			int64(totalPackages),
			"Downloading...",
		)
	}
	const maxIdleSeconds = 20
	for {
		select {
			case msg := <-messages:
				if msg.StatusCode == http.StatusOK {
					*successfulDownloads++
					*totalDownloadedBytes += msg.ContentLength
				} else {
					*failedDownloads++
				}
				idleSeconds = 0
				if Interactive {
					bar.Add(1)
				}
				messageString := "[" +
					strconv.Itoa(int(100 * float64(*successfulDownloads + *failedDownloads)/float64(totalPackages))) +
					 "%] " + strconv.Itoa(msg.StatusCode) + " " + msg.Message
				if msg.StatusCode == http.StatusOK {
					log.Info(messageString)
				} else {
					log.Error(messageString)
					*downloadErrors += msg.Message + ", status = " + strconv.Itoa(msg.StatusCode) + "\n"
				}

				*allDownloadInfo = append(*allDownloadInfo, DownloadInfo{msg.StatusCode, msg.Message, msg.ContentLength, msg.OutputLocation})

				if *successfulDownloads + *failedDownloads == totalPackages {
					// As soon as we got statuses for all packages we want to return to main routine.
					idleSeconds = maxIdleSeconds
				}
			default:
				time.Sleep(time.Second)
				idleSeconds++
		}
		// Last maxIdleWaits attempts at receiving status from package downloaders didn't yield any
		// messages. Or all packages have been downloaded. Hence, we finish waiting for any other statuses.
		if idleSeconds >= maxIdleSeconds {
			break
		}
	}
	// Signal to DownloadPackages function that all downloads have been completed.
	downloadWaiter <- struct{}{}
}

// Download packages from renv.lock file.
func DownloadPackages(renvLock Renvlock) {

	var allDownloadInfo []DownloadInfo

	// Clean up any previous downloaded data.
	os.RemoveAll(localOutputDirectory)
	os.MkdirAll(localOutputDirectory + "/packages", os.ModePerm)

	biocPackageVersions := make(map[string]map[string]string)
	biocUrls := make(map[string]string)

	// Retrieve lists of package versions from predefined BioConductor categories.
	if renvLock.Bioconductor.Version != "" {
		log.Info("Retrieving PACKAGES from BioConductor version ", renvLock.Bioconductor.Version, ".")
		for _, biocCategory := range(bioconductorCategories) {
			biocPackageVersions[biocCategory] = make(map[string]string)
			biocUrls[biocCategory] = bioConductorUrl + "/" + renvLock.Bioconductor.Version + "/" +
				biocCategory + "/src/contrib"
			status, _ := downloadFile(
				biocUrls[biocCategory] + "/PACKAGES", localOutputDirectory +
				"/BIOC_PACKAGES_" + strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_")),
			)
			if status == http.StatusOK {
				getPackageVersions(
					localOutputDirectory + "/BIOC_PACKAGES_" +
					strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_")),
					biocPackageVersions[biocCategory],
				)
			}
		}
	}

	const localCranPackagesPath = localOutputDirectory + "/CRAN_PACKAGES"

	// var repositories []string
	currentCranPackageVersions := make(map[string]string)
	for _, v := range renvLock.R.Repositories {
		// repositories = append(repositories, v.Name)

		// In case any packages are downloaded from CRAN, prepare a map with current versions of the packages.
		// This way, we'll know whether we should try to download the package from current repository
		// or from archive.
		if v.URL == defaultCranMirrorUrl {
			status, _ := downloadFile(defaultCranMirrorUrl + "/src/contrib/PACKAGES", localCranPackagesPath)
			if status == http.StatusOK {
				getPackageVersions(localCranPackagesPath, currentCranPackageVersions)
			}
		}
	}

	messages := make(chan DownloadInfo)

	// Guard channel ensures that only a fixed number of concurrent goroutines are running.
	guard := make(chan struct{}, maxDownloadRoutines)
	// Channel to wait until all downloads have completed.
	downloadWaiter := make(chan struct{})
	numberOfDownloads := 0
	var successfulDownloads, failedDownloads int
	var totalDownloadedBytes int64
	var downloadErrors string

	startTime := time.Now()

	go downloadResultReceiver(messages, &successfulDownloads, &failedDownloads,
		len(renvLock.Packages), &totalDownloadedBytes, downloadWaiter, &downloadErrors,
		&allDownloadInfo,
	)

	log.Info("There are ", len(renvLock.Packages), " packages to be downloaded.")
	var repoUrl string
	for _, v := range renvLock.Packages {
		if v.Package != "" && v.Version != "" {
			if v.Source == "Bioconductor" {
				repoUrl = bioConductorUrl
			} else if v.Source == "GitHub" &&
				(v.RemoteHost == "api.github.com" || v.RemoteHost == "https://api.github.com") {
				repoUrl = "https://github.com/" + v.RemoteUsername + "/" + v.RemoteRepo
			} else if v.Source == "GitLab" {
				repoUrl = "https://" + v.RemoteHost + "/" + v.RemoteUsername + "/" + v.RemoteRepo
			} else {
				repoUrl = getRepositoryUrl(renvLock.R.Repositories, v.Repository)
			}

			guard <- struct{}{}
			log.Debug("Downloading package ", v.Package)
			go downloadSinglePackage(v.Package, v.Version, repoUrl, v.Hash, v.Source,
				currentCranPackageVersions, biocPackageVersions, biocUrls, messages, guard)
			numberOfDownloads++
		}
	}

	// Wait for downloadResultReceiver until all download statuses have been retrieved.
	<- downloadWaiter

	if downloadErrors != "" {
		fmt.Println("\n\nThe following errors were encountered during download:")
		fmt.Print(downloadErrors)
	}

	// Temporary, just to show it's possible to save information about downloaded packages to JSON.
	s, err := json.MarshalIndent(allDownloadInfo, "", "  ")
	checkError(err)

	err = os.WriteFile("downloadInfo.json", []byte(s), 0644)
	checkError(err)

	elapsedTime := time.Since(startTime)
	log.Info("Total download time = ", elapsedTime)
	log.Info("Downloaded ", totalDownloadedBytes, " bytes")
	log.Info(
		"Average throughput = ",
		float64(int(8000 * (float64(totalDownloadedBytes) / 1000000) / (float64(elapsedTime.Milliseconds()) / 1000))) / 1000,
		" Mbps")
	log.Info(
		"Download succeeded for ", successfulDownloads, " packages out of ",
		numberOfDownloads, " requested packages.",
	)
	log.Info(
		"Download failed for ", failedDownloads, " packages out of ",
		numberOfDownloads, " requested packages.",
	)
}
