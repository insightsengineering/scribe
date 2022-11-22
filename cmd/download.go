package cmd

import (
	"bufio"
	"crypto/tls"
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

const defaultCranMirrorURL = "https://cloud.r-project.org"
const bioConductorURL = "https://www.bioconductor.org/packages"
const GitHub = "GitHub"

// within below directory:
// tar.gz packages are downloaded to package_archives subdirectory
// GitHub repositories are cloned into github subdirectory
// GitLab repositories are cloned into gitlab subdirectory
const localOutputDirectory = "/tmp/scribe/downloaded_packages"

var bioconductorCategories = [4]string{"bioc", "data/experiment", "data/annotation", "workflows"}

// Maximum number of concurrently running download goroutines.
const maxDownloadRoutines = 40

type DownloadInfo struct {
	// if statusCode > 0 it is identical to HTTP status code from download, or 200 in case of successful
	// git repository clone
	// message field contains URL of the package
	// statusCode == -1 means that package version could not be found in any BioConductor repository
	// message field contains error message
	// statusCode == -2 means that there was an error during cloning of GitHub repository
	// message field contains error message
	// statusCode == -3 means that there was an error during cloning of GitLab repository
	// message field contains error message
	// statusCode == -4 means that a network error occurred during HTTP download
	// message field contains URL of the package
	StatusCode    int    `json:"statusCode"`
	Message       string `json:"message"`
	ContentLength int64  `json:"contentLength"`
	// file where the package is stored, or directory where git repository has been cloned
	// empty in case of errors
	OutputLocation string `json:"outputLocation"`
}

func getRepositoryURL(renvLockRepositories []Rrepository, repositoryName string) string {
	for _, v := range renvLockRepositories {
		if v.Name == repositoryName {
			return v.URL
		}
	}
	// return default mirror if the repository is not defined in lock file
	return defaultCranMirrorURL
}

// Returns HTTP status code for downloaded file and number of bytes in downloaded content.
func downloadFile(url string, outputFile string) (int, int64) {
	// Get the data
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //#nosec
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
// It determines whether to download the package as tar.gz file, or from git repository.
// It then invokes downloadFile function or go-git library respectively.
func downloadSinglePackage(packageName string, packageVersion string, repoURL string, remoteRef string, packageSource string,
	currentCranPackageVersions map[string]string, biocPackageVersions map[string]map[string]string,
	biocUrls map[string]string, messages chan DownloadInfo, guard chan struct{}) error {

	var packageURL string
	var outputLocation string

	switch {
	case repoURL == defaultCranMirrorURL:
		// Check if package is in current CRAN repository
		versionInCran, ok := currentCranPackageVersions[packageName]
		if ok {
			log.Debug("CRAN current has package ", packageName, " version ", versionInCran, ".")
		} else {
			log.Debug("CRAN current doesn't have ", packageName, " in any version.")
		}
		if ok && versionInCran == packageVersion {
			log.Debug("Retrieving package ", packageName, " from CRAN current.")
			packageURL = repoURL + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
		} else {
			// If not, look for the package in Archive.
			log.Debug(
				"Attempting to retrieve ", packageName, " version ", packageVersion,
				" from CRAN Archive.",
			)
			packageURL = (repoURL + "/src/contrib/Archive/" + packageName +
				"/" + packageName + "_" + packageVersion + ".tar.gz")
		}
	case repoURL == bioConductorURL:
		for _, biocCategory := range bioconductorCategories {
			biocPackageVersion, ok := biocPackageVersions[biocCategory][packageName]
			if ok {
				log.Debug(
					"BioConductor category ", biocCategory, " has package ", packageName,
					" version ", biocPackageVersion, ".",
				)
				if biocPackageVersion == packageVersion {
					log.Debug("Package ", packageName, " will be retrieved from Bioconductor category ", biocCategory)
					packageURL = biocUrls[biocCategory] + "/" + packageName + "_" + packageVersion + ".tar.gz"
					break
				}
			}
		}
		// Package not found in any of Bioconductor categories.
		if packageURL == "" {
			messages <- DownloadInfo{-1, "Couldn't find " + packageName + " version " + packageVersion + " in BioConductor.", 0, ""}
			<-guard
			return nil
		}
	case packageSource == GitHub:
		// TODO this has to be modified if we plan to support other GitHub instances than https://github.com
		gitDirectory := localOutputDirectory + "/github" + strings.TrimPrefix(repoURL, "https://github.com")
		err := os.MkdirAll(gitDirectory, os.ModePerm)
		checkError(err)
		log.Debug("Cloning repo to ", gitDirectory)
		_, err = git.PlainClone(
			gitDirectory, false, &git.CloneOptions{
				URL: repoURL,
			})
		if err == nil {
			// The number of bytes downloaded is approximated by the size of repository directory.
			var gitRepoSize int64
			gitRepoSize, err = dirSize(gitDirectory)
			checkError(err)
			log.Debug("Repository size of ", repoURL, " = ", gitRepoSize, " bytes")
			messages <- DownloadInfo{200, repoURL, gitRepoSize, gitDirectory}
		} else {
			messages <- DownloadInfo{-2, "Error while cloning repo " + repoURL + ": " + err.Error(), 0, ""}
		}
		<-guard
		return nil
	case packageSource == "GitLab":
		// repoURL == https://example.com/remote-user/some/remote/repo/path
		remoteHost := strings.Join(strings.Split(repoURL, "/")[:3], "/")
		remoteUser := strings.Split(repoURL, "/")[3]
		remoteRepo := strings.Join(strings.Split(repoURL, "/")[4:], "/")

		gitDirectory := localOutputDirectory + "/gitlab/" + strings.Split(repoURL, "/")[2] +
			"/" + remoteUser + "/" + remoteRepo
		err := os.MkdirAll(gitDirectory, os.ModePerm)
		checkError(err)
		log.Debug("Cloning repo ", remoteUser, "/", remoteRepo, " from host ", remoteHost, " to directory ", gitDirectory)
		_, err = git.PlainClone(
			gitDirectory, false, &git.CloneOptions{
				URL: repoURL,
				// TODO document this, or change the way credentials are passed
				Auth: &githttp.BasicAuth{Username: os.Getenv("GITLAB_USER"), Password: os.Getenv("GITLAB_TOKEN")},
			})
		if err == nil {
			// The number of bytes downloaded is approximated by the size of repository directory.
			var gitRepoSize int64
			gitRepoSize, err = dirSize(gitDirectory)
			checkError(err)
			log.Debug("Repository size of ", repoURL, " = ", gitRepoSize, " bytes")
			messages <- DownloadInfo{200, repoURL, gitRepoSize, gitDirectory}
		} else {
			messages <- DownloadInfo{-3, "Error while cloning repo " + repoURL + ": " + err.Error(), 0, ""}
		}
		<-guard
		return nil
	default:
		// Repositories other than CRAN or BioConductor
		packageURL = repoURL + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
	}

	outputLocation = localOutputDirectory + "/package_archives/" + packageName + "_" + packageVersion + ".tar.gz"

	statusCode, contentLength := downloadFile(
		packageURL, outputLocation,
	)
	if statusCode != http.StatusOK {
		outputLocation = ""
	}
	messages <- DownloadInfo{statusCode, packageURL, contentLength, outputLocation}
	<-guard

	return nil
}

// Read PACKAGES file and save:
// * map from package names to their versions as stored in the PACKAGES file.
// * map from package names to their MD5 checksums as stored in the PACKAGES file.
func parsePackagesFile(filePath string, packageVersions map[string]string,
	packageChecksums map[string]string) {
	packages, err := os.Open(filePath)
	checkError(err)
	defer packages.Close()

	scanner := bufio.NewScanner(packages)
	var currentlyProcessedPackage string
	var packageName string
	// Iterate through lines of PACKAGES file.
	for scanner.Scan() {
		newLine := scanner.Text()
		if strings.HasPrefix(newLine, "Package:") {
			packageFields := strings.Fields(newLine)
			packageName = packageFields[1]
			currentlyProcessedPackage = packageName

			// Read next line after 'Package:' to get 'Version:'.
			scanner.Scan()
			nextLine := scanner.Text()
			versionFields := strings.Fields(nextLine)
			packageVersion := versionFields[1]
			packageVersions[packageName] = packageVersion
		}
		// Save the cheksum from PACKAGES file to compare it with locally cached
		// tar.gz checksums.
		if strings.HasPrefix(newLine, "MD5sum:") {
			checksumFields := strings.Fields(newLine)
			checksum := checksumFields[1]
			packageChecksums[currentlyProcessedPackage] = checksum
		}
	}
}

// Retrieve lists of package versions from predefined BioConductor categories.
func getBioConductorPackages(biocVersion string, biocPackageVersions map[string]map[string]string,
	biocPackageChecksums map[string]map[string]string ,biocUrls map[string]string) {
	log.Info("Retrieving PACKAGES from BioConductor version ", biocVersion, ".")
	for _, biocCategory := range bioconductorCategories {
		biocPackageVersions[biocCategory] = make(map[string]string)
		biocPackageChecksums[biocCategory] = make(map[string]string)
		biocUrls[biocCategory] = bioConductorURL + "/" + biocVersion + "/" +
			biocCategory + "/src/contrib"
		status, _ := downloadFile(
			biocUrls[biocCategory] + "/PACKAGES", localOutputDirectory +
				"/package_files/BIOC_PACKAGES_" + strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_")),
		)
		if status == http.StatusOK {
			// Get BioConductor package versions and their checksums.
			parsePackagesFile(
				localOutputDirectory + "/package_files/BIOC_PACKAGES_" +
					strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_")),
				biocPackageVersions[biocCategory],
				biocPackageChecksums[biocCategory],
			)
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
				err := bar.Add(1)
				checkError(err)
			}
			messageString := "[" +
				strconv.Itoa(int(100*float64(*successfulDownloads+*failedDownloads)/float64(totalPackages))) +
				"%] " + strconv.Itoa(msg.StatusCode) + " " + msg.Message
			if msg.StatusCode == http.StatusOK {
				log.Info(messageString)
			} else {
				log.Error(messageString)
				*downloadErrors += msg.Message + ", status = " + strconv.Itoa(msg.StatusCode) + "\n"
			}

			*allDownloadInfo = append(*allDownloadInfo, DownloadInfo{msg.StatusCode, msg.Message, msg.ContentLength, msg.OutputLocation})

			if *successfulDownloads+*failedDownloads == totalPackages {
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

// Download packages from renv.lock file, saves download result structs to allDownloadInfo.
func DownloadPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo) {

	// Clean up any previous downloaded data, except tar.gz packages.
	// We'll later calculate checksums for tar.gz files and compare them checksums in
	// in PACKAGES files, so tar.gz files don't have to be downloaded again.
	// Then, recreate these directories.
	for _, directory := range []string{"/github", "/gitlab", "/package_files"} {
		err := os.RemoveAll(localOutputDirectory + directory)
		checkError(err)
		err = os.MkdirAll(localOutputDirectory + directory, os.ModePerm)
		checkError(err)
	}

	// Ensure the directory where tar.gz packages will be downloaded exists.
	err := os.MkdirAll(localOutputDirectory + "/package_archives", os.ModePerm)
	checkError(err)

	biocPackageVersions := make(map[string]map[string]string)
	biocPackageChecksums := make(map[string]map[string]string)
	biocUrls := make(map[string]string)

	if renvLock.Bioconductor.Version != "" {
		getBioConductorPackages(
			renvLock.Bioconductor.Version, biocPackageVersions, biocPackageChecksums, biocUrls,
		)
	}

	const localCranPackagesPath = localOutputDirectory + "/package_files/CRAN_PACKAGES"

	// var repositories []string
	currentCranPackageVersions := make(map[string]string)
	currentCranPackageChecksums := make(map[string]string)
	for _, v := range renvLock.R.Repositories {
		// repositories = append(repositories, v.Name)

		// In case any packages are downloaded from CRAN, prepare a map with current versions of the packages.
		// This way, we'll know whether we should try to download the package from current repository
		// or from archive.
		// Similarly, prepare a map from package names to checksums to check if any packages have been
		// previously downloaded to local cache.
		if v.URL == defaultCranMirrorURL {
			status, _ := downloadFile(defaultCranMirrorURL + "/src/contrib/PACKAGES", localCranPackagesPath)
			if status == http.StatusOK {
				parsePackagesFile(
					localCranPackagesPath, currentCranPackageVersions, currentCranPackageChecksums,
				)
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
		allDownloadInfo,
	)

	log.Info("There are ", len(renvLock.Packages), " packages to be downloaded.")
	var repoURL string
	for _, v := range renvLock.Packages {
		if v.Package != "" && v.Version != "" {
			switch v.Source {
			case "Bioconductor":
				repoURL = bioConductorURL
			case GitHub:
				repoURL = "https://github.com/" + v.RemoteUsername + "/" + v.RemoteRepo
			case "GitLab":
				repoURL = "https://" + v.RemoteHost + "/" + v.RemoteUsername + "/" + v.RemoteRepo
			default:
				repoURL = getRepositoryURL(renvLock.R.Repositories, v.Repository)
			}

			guard <- struct{}{}
			log.Debug("Downloading package ", v.Package)
			go downloadSinglePackage(v.Package, v.Version, repoURL, v.RemoteRef, v.Source,
				currentCranPackageVersions, biocPackageVersions, biocUrls, messages, guard)
			numberOfDownloads++
		}
	}

	// Wait for downloadResultReceiver until all download statuses have been retrieved.
	<-downloadWaiter

	if downloadErrors != "" {
		fmt.Println("\n\nThe following errors were encountered during download:")
		fmt.Print(downloadErrors)
	}

	// Temporary, just to show it's possible to save information about downloaded packages to JSON.
	WriteJSON("downloadInfo.json", *allDownloadInfo)

	elapsedTime := time.Since(startTime)
	log.Info("Total download time = ", elapsedTime.Seconds(), " seconds.")
	log.Info("Downloaded ", totalDownloadedBytes, " bytes.")
	log.Info(
		"Average throughput = ",
		float64(int(8000*(float64(totalDownloadedBytes)/1000000)/(float64(elapsedTime.Milliseconds())/1000)))/1000,
		" Mbps.")
	log.Info(
		"Download succeeded for ", successfulDownloads, " packages out of ",
		numberOfDownloads, " requested packages.",
	)
	log.Info(
		"Download failed for ", failedDownloads, " packages out of ",
		numberOfDownloads, " requested packages.",
	)
}
