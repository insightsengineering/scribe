package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultCranMirrorUrl = "https://cloud.r-project.org"
const localOutputDirectory = "/tmp/scribe/downloadedPackages"

type DownloadInfo struct {
	statusCode int
	packageUrl string
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

func downloadFile(url string, outputFile string)  (int) {
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

		return resp.StatusCode
	}
	return -1
}

func downloadSinglePackage(packageName string, packageVersion string, repoUrl string, currentCranPackageVersions map[string]string, messages chan DownloadInfo, guard chan struct{}) (error){

	var packageUrl string

	if repoUrl == defaultCranMirrorUrl {
		// Check if package is in current CRAN repository
		versionInCran, ok := currentCranPackageVersions[packageName]
		if ok {
			log.Debug("CRAN current has package ", packageName, " in version ", versionInCran, ".")
		} else {
			log.Debug("CRAN current doesn't have ", packageName, " in any version.")
		}
		if ok && versionInCran == packageVersion {
			log.Debug("CRAN current has package ", packageName, " in sought version ", packageVersion, ".")
			packageUrl = repoUrl + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
		} else {
			// If not, look for the package in Archive
			log.Debug("Will attempt to retrieve ", packageName, " version ", packageVersion, " from CRAN Archive.")
			packageUrl = repoUrl + "/src/contrib/Archive/" + packageName + "/" + packageName + "_" + packageVersion + ".tar.gz"
		}
	} else {
		packageUrl = repoUrl + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
	}

	statusCode := downloadFile(packageUrl, localOutputDirectory + "/" + packageName + "_" + packageVersion + ".tar.gz")

	log.Info(packageUrl, " = ", statusCode)

	messages <- DownloadInfo{statusCode, packageUrl}
	<- guard

	return nil
}

func getPackageVersions(filePath string, packageVersions map[string]string) {
	packages, err := os.Open(filePath)
	checkError(err)
	defer packages.Close()

	scanner := bufio.NewScanner(packages)
	// Iterate through lines of PACKAGES file
	for scanner.Scan() {
		newLine := scanner.Text()
		if strings.HasPrefix(newLine, "Package:") {
			packageFields := strings.Fields(newLine)
			packageName := packageFields[1]

			// Read next line after 'Package:' to get 'Version:'
			scanner.Scan()
			nextLine := scanner.Text()
			versionFields := strings.Fields(nextLine)
			packageVersion := versionFields[1]
			packageVersions[packageName] = packageVersion
		}
	}
}

func downloadResultReceiver(messages chan DownloadInfo, successfulDownloads *int, failedDownloads *int) {
	*successfulDownloads = 0
	*failedDownloads = 0
	idleSeconds := 0
	const maxIdleSeconds = 10
	for {
		select {
			case msg := <-messages:
				if msg.statusCode == http.StatusOK {
					*successfulDownloads++
				} else {
					*failedDownloads++
				}
				idleSeconds = 0
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
}

func DownloadPackages(renvLock Renvlock) {

	os.RemoveAll(localOutputDirectory)
	os.MkdirAll(localOutputDirectory, os.ModePerm)

	const localCranPackagesPath = localOutputDirectory + "/CRAN_PACKAGES"

	var repositories []string
	currentCranPackageVersions := make(map[string]string)
	for _, v := range renvLock.R.Repositories {
		repositories = append(repositories, v.Name)

		// In case any packages are downloaded from CRAN, prepare a map with current versions of the packages.
		// This way, we'll know whether we should try to download the package from current repository
		// or from archive.
		if v.URL == defaultCranMirrorUrl {
			status := downloadFile(defaultCranMirrorUrl + "/src/contrib/PACKAGES", localCranPackagesPath)
			if status == http.StatusOK {
				getPackageVersions(localCranPackagesPath, currentCranPackageVersions)
			}
		}
	}

	messages := make(chan DownloadInfo)
	guard := make(chan struct{}, 10)
	numberOfDownloads := 0
	var successfulDownloads, failedDownloads int

	go downloadResultReceiver(messages, &successfulDownloads, &failedDownloads)

	for _, v := range renvLock.Packages {
		if v.Package != "" && v.Version != "" {
			repoUrl := getRepositoryUrl(renvLock.R.Repositories, v.Repository)

			guard <- struct{}{}
			log.Info("Downloading package ", v.Package)
			go downloadSinglePackage(v.Package, v.Version, repoUrl, currentCranPackageVersions, messages, guard)
			numberOfDownloads++
		}
	}

	log.Info("Successfully downloaded ", successfulDownloads, " packages out of ", numberOfDownloads, " requested packages.")
	log.Info("Downloads failed for ", failedDownloads, " packages out of ", numberOfDownloads, " requested packages.")

}
