/*
Copyright 2023 F. Hoffmann-La Roche AG

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"crypto/md5" // #nosec
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

const defaultCranMirrorURL = "https://cloud.r-project.org"
const bioConductorURL = "https://www.bioconductor.org/packages"
const GitHub = "GitHub"
const GitLab = "GitLab"
const cache = "cache"
const download = "download"
const github = "github"
const gitlab = "gitlab"
const targzExtensionFile = "tar.gz"

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
	// number of bytes saved thanks to caching (size of cached package file)
	SavedBandwidth int64 `json:"savedBandwidth"`
	// possible values: tar.gz, git, bioconductor or empty value in case of error
	DownloadedPackageType string `json:"downloadedPackageType"`
	PackageName           string `json:"packageName"`
	PackageVersion        string `json:"packageVersion"`
	// Contains git SHA of cloned package, or exceptionally git tag or branch, if SHA was not provided in renv.lock.
	GitPackageShaOrRef string `json:"gitPackageShaOrRef"`
}

// Struct used to store data about tar.gz packages saved in local cache.
type CacheInfo struct {
	Path   string
	Length int64
}

type PackageInfo struct {
	Version  string
	Checksum string
}

func getRepositoryURL(v Rpackage, repositories []Rrepository) string {
	var repoURL string
	switch v.Source {
	case "Bioconductor":
		repoURL = bioConductorURL
	case GitHub:
		repoURL = "https://github.com/" + v.RemoteUsername + "/" + v.RemoteRepo
	case GitLab:
		// The behavior of renv.lock is not standardized in terms of whether GitLab host address
		// starts with 'https://' or not.
		var remoteHost string
		if strings.HasPrefix(v.RemoteHost, "https://") {
			remoteHost = v.RemoteHost
		} else {
			remoteHost = "https://" + v.RemoteHost
		}
		repoURL = remoteHost + "/" + v.RemoteUsername + "/" + v.RemoteRepo
	default:
		repoURL = getRenvRepositoryURL(repositories, v.Repository)
	}
	return repoURL
}

// Returns HTTP status code for downloaded file and number of bytes in downloaded content.
func downloadFile(url string, outputFile string) (int, int64) {
	// Get the data
	tr := &http.Transport{ // #nosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec
	} // #nosec
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

// Clones git repository and returns string with error value (empty if cloning was
// successful), approximate number of downloaded bytes, and cloned version of the package (tag, branch or commit SHA).
// If commitSha or branchOrTagName is specified, the respective commit, branch or tag are checked out.
// If environmentCredentialsType is "gitlab", this function expects username and token to be set in
// GITLAB_USER and GITLAB_TOKEN environment variables.
// If environmentCredentialsType is "github", this function expects token to be set in
// GITHUB_TOKEN environment variable.
// This implementation assumes that RemoteSha renv.lock field contains commit SHA,
// and that RemoteRef renv.lock field contains branch name or tag name.
// It is assumed that RemoteRef matching regex `v\d+(\.\d+)*` (where \d is a digit) is a tag name.
// Otherwise, that it is a branch name.
// If any of these assumptions are not correct for some package, the default branch will be checked out.
func cloneGitRepo(gitDirectory string, repoURL string, environmentCredentialsType string,
	commitSha string, branchOrTagName string) (string, int64, string) {
	err := os.MkdirAll(gitDirectory, os.ModePerm)
	checkError(err)
	var gitCloneOptions *git.CloneOptions
	switch {
	case environmentCredentialsType == gitlab:
		gitCloneOptions = &git.CloneOptions{
			URL: repoURL,
			Auth: &githttp.BasicAuth{
				Username: "This can be any string.",
				Password: os.Getenv("GITLAB_TOKEN"),
			},
		}
	case environmentCredentialsType == github:
		gitCloneOptions = &git.CloneOptions{
			URL: repoURL,
			Auth: &githttp.BasicAuth{
				Username: "This can be any string.",
				Password: os.Getenv("GITHUB_TOKEN"),
			},
		}
	default:
		gitCloneOptions = &git.CloneOptions{
			URL: repoURL,
		}
	}
	repository, err := git.PlainClone(gitDirectory, false, gitCloneOptions)
	if err == nil {
		var gitPackageShaOrRef string
		w, er := repository.Worktree()
		checkError(er)
		switch {
		case commitSha != "":
			// Checkout the commit.
			log.Info("Checking out commit ", commitSha, " in ", gitDirectory)
			err = w.Checkout(&git.CheckoutOptions{
				Hash: plumbing.NewHash(commitSha),
			})
			if err != git.NoErrAlreadyUpToDate {
				checkError(err)
			}
			gitPackageShaOrRef = commitSha
		case branchOrTagName != "" && branchOrTagName != "HEAD":
			// Checkout the branch or tag.
			match, err2 := regexp.MatchString(`v\d+(\.\d+)*`, branchOrTagName)
			checkError(err2)
			var refName string
			var checkoutRefName string
			if match {
				log.Debug(branchOrTagName + " matches tag name regexp.")
				checkoutRefName = fmt.Sprintf("refs/tags/%s", branchOrTagName)
				refName = fmt.Sprintf("%s:%s", checkoutRefName, checkoutRefName)
			} else {
				log.Debug(branchOrTagName, " doesn't match tag name regexp.")
				checkoutRefName = fmt.Sprintf("refs/heads/%s", branchOrTagName)
				refName = fmt.Sprintf("%s:%s", checkoutRefName, checkoutRefName)
			}
			log.Info("Checking out branch or tag ", checkoutRefName, " in ", gitDirectory)
			refSpec := config.RefSpec(refName)
			var fetchOptions *git.FetchOptions
			switch {
			case environmentCredentialsType == gitlab:
				fetchOptions = &git.FetchOptions{
					RefSpecs: []config.RefSpec{refSpec},
					Auth: &githttp.BasicAuth{
						Username: "This can be any string.",
						Password: os.Getenv("GITLAB_TOKEN"),
					},
				}
			case environmentCredentialsType == github:
				fetchOptions = &git.FetchOptions{
					RefSpecs: []config.RefSpec{refSpec},
					Auth: &githttp.BasicAuth{
						Username: "This can be any string.",
						Password: os.Getenv("GITHUB_TOKEN"),
					},
				}
			default:
				fetchOptions = &git.FetchOptions{
					RefSpecs: []config.RefSpec{refSpec},
				}
			}
			err = repository.Fetch(fetchOptions)
			if err != git.NoErrAlreadyUpToDate {
				checkError(err)
			}
			err = w.Checkout(&git.CheckoutOptions{
				Branch: plumbing.ReferenceName(checkoutRefName),
			})
			if err != git.NoErrAlreadyUpToDate {
				checkError(err)
			}
			gitPackageShaOrRef = branchOrTagName
		default:
			// Leave HEAD checked out and return its SHA.
			// This case is used during package update phase,
			// where we check the newest package version in git repository.
			ref, err2 := repository.Head()
			checkError(err2)
			gitPackageShaOrRef = ref.Hash().String()
		}
		// The number of bytes downloaded is approximated by the size of repository directory.
		var gitRepoSize int64
		gitRepoSize, err = dirSize(gitDirectory)
		checkError(err)
		log.Debug("Repository size of ", repoURL, " = ", gitRepoSize/1024, " KiB")
		return "", gitRepoSize, gitPackageShaOrRef
	}
	return "Error while cloning repo " + repoURL + ": " + err.Error(), 0, ""
}

// Returns:
// * information how the package should be accessed:
//   - "download" means the package should be downloaded as a tar.gz file from CRAN, Bioconductor or some other repo
//   - "cache" means the package is available in local cache because it has been previously downloaded
//   - "github" means the package should be cloned as a GitHub repository
//   - "gitlab" means the package should be cloned as a GitLab repository
//   - "notfound_bioc" means the package couldn't be found in Bioconductor
//
// * URL from which the package should be downloaded or cloned (or has originally been downloaded from, if it's available in cache)
// * fallback URL - in case specific package version can't be found in CRAN, it is downloaded in the newest available CRAN version
// * location where the package will be downloaded (filepath to the tar.gz file or git repo directory)
// * fallback location - filepath to tar.gz file in case package is downloaded from the fallback URL
// * number of bytes saved due to retrieving file from cache (size of the tar.gz file in cache), if not found in cache: 0
func getPackageDetails(packageName string, packageVersion string, repoURL string,
	packageSource string, currentCranPackageInfo map[string]*PackageInfo,
	biocPackageInfo map[string]map[string]*PackageInfo, biocUrls map[string]string,
	localArchiveChecksums map[string]*CacheInfo) (string, string, string, string, string, int64) {
	var packageURL string
	switch {
	case repoURL == defaultCranMirrorURL:
		outputLocation := localOutputDirectory + "/package_archives/" + packageName +
			"_" + packageVersion + ".tar.gz"
		// Check if package is in current CRAN repository.
		var versionInCran string
		packageInfo, ok := currentCranPackageInfo[packageName]
		if ok {
			versionInCran = packageInfo.Version
		}
		if ok {
			log.Debug("CRAN current has package ", packageName, " version ", versionInCran, ".")
		} else {
			log.Debug("CRAN current doesn't have ", packageName, " in any version.")
		}
		if ok && versionInCran == packageVersion {
			// Check if the package is cached locally.
			localCachedFile, ok := localArchiveChecksums[packageInfo.Checksum]
			packageURL = repoURL + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
			if ok {
				return cache, packageURL, "", localCachedFile.Path, "", localCachedFile.Length
			}
			// Package not cached locally.
			log.Debug("Retrieving package ", packageName, " from CRAN current.")
			return download, packageURL, "", outputLocation, "", 0
		}
		// If CRAN current doesn't have the package version, look for the package in Archive.
		log.Debug(
			"Attempting to retrieve ", packageName, " version ", packageVersion,
			" from CRAN Archive.",
		)
		packageURL = repoURL + "/src/contrib/Archive/" + packageName +
			"/" + packageName + "_" + packageVersion + ".tar.gz"
		// In case the requested package version cannot be found neither in current CRAN or CRAN archive,
		// we'll try to download the version from current CRAN as fallback.
		fallbackPackageURL := repoURL + "/src/contrib/" + packageName + "_" + versionInCran + ".tar.gz"
		fallbackOutputLocation := localOutputDirectory + "/package_archives/" + packageName +
			"_" + versionInCran + ".tar.gz"
		return download, packageURL, fallbackPackageURL, outputLocation, fallbackOutputLocation, 0

	case repoURL == bioConductorURL:
		var packageChecksum string
		outputLocation := localOutputDirectory + "/package_archives/" + packageName +
			"_" + packageVersion + ".tar.gz"
		for _, biocCategory := range bioconductorCategories {
			biocPackageInfo, ok := biocPackageInfo[biocCategory][packageName]
			if ok {
				log.Debug(
					"BioConductor category ", biocCategory, " has package ", packageName,
					" version ", biocPackageInfo.Version, ".",
				)
				if biocPackageInfo.Version == packageVersion {
					log.Debug("Retrieving package ", packageName, " from BioConductor current.")
					packageURL = biocUrls[biocCategory] + "/" + packageName +
						"_" + packageVersion + ".tar.gz"
					packageChecksum = biocPackageInfo.Checksum
				} else {
					// Package not found in current Bioconductor.
					// Try to retrieve it from Bioconductor archive.
					log.Debug(
						"Attempting to retrieve ", packageName, " version ", packageVersion,
						" from Bioconductor Archive.",
					)
					packageURL = biocUrls[biocCategory] + "/Archive/" + packageName + "/" + packageName +
						"_" + packageVersion + ".tar.gz"
				}
				break
			}
		}
		if packageURL != "" {
			// Check if package is cached locally.
			localCachedFile, ok := localArchiveChecksums[packageChecksum]
			if ok {
				return cache, packageURL, "", localCachedFile.Path, "", localCachedFile.Length
			}
			// Package not cached locally.
			return download, packageURL, "", outputLocation, "", 0
		}
		// Package not found in any Bioconductor category.
		return "notfound_bioc", "", "", "", "", 0

	case packageSource == GitHub:
		// TODO this has to be modified if we plan to support other GitHub instances than https://github.com
		gitDirectory := localOutputDirectory + "/github" +
			strings.TrimPrefix(repoURL, "https://github.com")
		log.Debug("Cloning ", repoURL, " to ", gitDirectory)
		return github, repoURL, "", gitDirectory, "", 0

	case packageSource == GitLab:
		// repoURL == https://example.com/remote-user/some/remote/repo/path
		remoteHost := strings.Join(strings.Split(repoURL, "/")[:3], "/")
		remoteUser := strings.Split(repoURL, "/")[3]
		remoteRepo := strings.Join(strings.Split(repoURL, "/")[4:], "/")

		gitDirectory := localOutputDirectory + "/gitlab/" + strings.Split(repoURL, "/")[2] +
			"/" + remoteUser + "/" + remoteRepo
		log.Debug("Cloning repo ", remoteUser, "/", remoteRepo, " from host ",
			remoteHost, " to directory ", gitDirectory)
		return gitlab, repoURL, "", gitDirectory, "", 0

	default:
		// Repositories other than CRAN or BioConductor
		packageURL = repoURL + "/src/contrib/" + packageName + "_" + packageVersion + ".tar.gz"
		outputLocation := localOutputDirectory + "/package_archives/" + packageName +
			"_" + packageVersion + ".tar.gz"
		return download, packageURL, "", outputLocation, "", 0
	}
}

func getPackageOutputLocation(outputLocation, packageSubdir string) string {
	if packageSubdir != "" {
		return outputLocation + "/" + packageSubdir
	}
	return outputLocation
}

// Function executed in parallel goroutines.
// First, it determines in what way to retrieve the package.
// Then, it performs appropriate action based on what's been determined.
func downloadSinglePackage(packageName string, packageVersion string,
	repoURL string, gitCommitSha string, gitBranch string,
	packageSource string, packageSubdir string, currentCranPackageInfo map[string]*PackageInfo,
	biocPackageInfo map[string]map[string]*PackageInfo, biocUrls map[string]string,
	localArchiveChecksums map[string]*CacheInfo,
	downloadFileFunction func(string, string) (int, int64),
	gitCloneFunction func(string, string, string, string, string) (string, int64, string),
	messages chan DownloadInfo, guard chan struct{}) {

	// Determine whether to download the package as tar.gz file, or from git repository.
	action, packageURL, fallbackPackageURL, outputLocation, fallbackOutputLocation, savedBandwidth := getPackageDetails(
		packageName, packageVersion, repoURL, packageSource, currentCranPackageInfo,
		biocPackageInfo, biocUrls, localArchiveChecksums,
	)

	switch action {
	case cache:
		log.Debug("Package ", packageName, " version ", packageVersion,
			" found in cache: ", outputLocation)
		var packageType string
		if strings.HasPrefix(packageURL, bioConductorURL) {
			packageType = "bioconductor"
		} else {
			packageType = targzExtensionFile
		}
		messages <- DownloadInfo{200, "[cached] " + packageURL, 0, outputLocation, savedBandwidth,
			packageType, packageName, packageVersion, ""}
	case download:
		statusCode, contentLength := downloadFileFunction(packageURL, outputLocation)
		if statusCode != http.StatusOK {
			// Download may fail in case the requested package version cannot be found
			// neither in current CRAN nor in CRAN archive. In that case, we try
			// to download the newest package version from CRAN current.
			if fallbackPackageURL != "" && fallbackOutputLocation != "" {
				statusCode, contentLength = downloadFileFunction(fallbackPackageURL, fallbackOutputLocation)
				packageURL = fallbackPackageURL
				if statusCode == http.StatusOK {
					outputLocation = fallbackOutputLocation
					log.Warn("Package ", packageName, " downloaded from ", fallbackPackageURL,
						" because requested version ", packageVersion, " is not available.")
				} else {
					outputLocation = ""
				}
			} else {
				outputLocation = ""
			}
		}
		var packageType string
		if strings.HasPrefix(packageURL, bioConductorURL) {
			packageType = "bioconductor"
		} else {
			packageType = targzExtensionFile
		}
		messages <- DownloadInfo{statusCode, packageURL, contentLength, outputLocation, 0, packageType,
			packageName, packageVersion, ""}
	case "notfound_bioc":
		messages <- DownloadInfo{-1, "Couldn't find " + packageName + " version " +
			packageVersion + " in BioConductor.", 0, "", 0, "", packageName, "", ""}
	case github:
		message, gitRepoSize, gitPackageShaOrRef := gitCloneFunction(outputLocation, packageURL, github,
			gitCommitSha, gitBranch)
		if message == "" {
			messages <- DownloadInfo{200, repoURL, gitRepoSize,
				getPackageOutputLocation(outputLocation, packageSubdir), 0,
				"git", packageName, packageVersion, gitPackageShaOrRef}
		} else {
			messages <- DownloadInfo{-2, message, 0, "", 0, "", packageName, "", ""}
		}
	case gitlab:
		message, gitRepoSize, gitPackageShaOrRef := gitCloneFunction(outputLocation, packageURL, gitlab,
			gitCommitSha, gitBranch)
		if message == "" {
			messages <- DownloadInfo{200, repoURL, gitRepoSize,
				getPackageOutputLocation(outputLocation, packageSubdir), 0,
				"git", packageName, packageVersion, gitPackageShaOrRef}
		} else {
			messages <- DownloadInfo{-3, message, 0, "", 0, "", packageName, "", ""}
		}
	default:
		messages <- DownloadInfo{-5, "Internal error: unknown action " + action, 0, "", 0, "", "", "", ""}
	}
	<-guard
}

// Read PACKAGES file and save:
// * map from package names to their versions as stored in the PACKAGES file.
// * map from package names to their MD5 checksums as stored in the PACKAGES file.
func parsePackagesFile(filePath string, packageInfo map[string]*PackageInfo) {
	packages, err := os.Open(filePath)
	checkError(err)
	defer packages.Close()

	scanner := bufio.NewScanner(packages)
	var currentlyProcessedPackageName string
	var currentlyProcessedPackageVersion string
	var packageName string
	// Iterate through lines of PACKAGES file.
	for scanner.Scan() {
		newLine := scanner.Text()
		if strings.HasPrefix(newLine, "Package:") {
			packageFields := strings.Fields(newLine)
			packageName = packageFields[1]
			currentlyProcessedPackageName = packageName

			// Read next line after 'Package:' to get 'Version:'.
			scanner.Scan()
			nextLine := scanner.Text()
			versionFields := strings.Fields(nextLine)
			currentlyProcessedPackageVersion = versionFields[1]
		}
		// Save the cheksum from PACKAGES file to compare it with locally cached
		// tar.gz checksums.
		if strings.HasPrefix(newLine, "MD5sum:") {
			checksumFields := strings.Fields(newLine)
			checksum := checksumFields[1]
			packageInfo[currentlyProcessedPackageName] = &PackageInfo{
				currentlyProcessedPackageVersion, checksum}
		}
	}
}

func getBiocUrls(biocVersion string, biocUrls map[string]string) {
	for _, biocCategory := range bioconductorCategories {
		biocUrls[biocCategory] = bioConductorURL + "/" + biocVersion + "/" +
			biocCategory + "/src/contrib"
	}
}

// Retrieve lists of package versions from predefined BioConductor categories.
func getBioConductorPackages(biocVersion string, biocPackageInfo map[string]map[string]*PackageInfo,
	biocUrls map[string]string, downloadFileFunction func(string, string) (int, int64)) {
	log.Info("Retrieving PACKAGES from BioConductor version ", biocVersion, ".")
	for _, biocCategory := range bioconductorCategories {
		biocPackageInfo[biocCategory] = make(map[string]*PackageInfo)
		status, _ := downloadFileFunction(
			biocUrls[biocCategory]+"/PACKAGES", localOutputDirectory+
				"/package_files/BIOC_PACKAGES_"+
				strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_")),
		)
		if status == http.StatusOK {
			// Get BioConductor package versions and their checksums.
			parsePackagesFile(
				localOutputDirectory+"/package_files/BIOC_PACKAGES_"+
					strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_")),
				biocPackageInfo[biocCategory],
			)
		}
	}
}

// Iterate through files in directoryName and save the checksums of .tar.gz files found there.
// TODO parallelize this if required - takes around 3 seconds for 820 MB of data
func computeChecksums(directoryPath string, localArchiveChecksums map[string]*CacheInfo) {
	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(info.Name(), ".tar.gz") {
			filePath := directoryPath + "/" + info.Name()
			byteValue, err := os.ReadFile(filePath)
			checkError(err)
			fileLength := int64(len(byteValue))
			hash := md5.Sum(byteValue) // #nosec
			hashValue := hex.EncodeToString(hash[:])
			localArchiveChecksums[hashValue] = &CacheInfo{filePath, fileLength}
		}
		return nil
	})
	checkError(err)
}

// Receive messages from goroutines responsible for package downloads.
func downloadResultReceiver(messages chan DownloadInfo, successfulDownloads *int,
	failedDownloads *int, totalPackages int, totalDownloadedBytes *int64,
	totalSavedBandwidth *int64, downloadWaiter chan struct{},
	downloadErrors *string, allDownloadInfo *[]DownloadInfo) {
	*successfulDownloads = 0
	*failedDownloads = 0
	*totalSavedBandwidth = 0
	idleSeconds := 0
	// Such big idle timeout is (unfortunately) required for some big packages like rmint.sdtm.
	const maxIdleSeconds = 200
	for {
		select {
		case msg := <-messages:
			if msg.StatusCode == http.StatusOK {
				*successfulDownloads++
				*totalDownloadedBytes += msg.ContentLength
			} else {
				*failedDownloads++
			}
			*totalSavedBandwidth += msg.SavedBandwidth
			idleSeconds = 0
			messageString := "[" +
				strconv.Itoa(int(100*float64(*successfulDownloads+*failedDownloads)/
					float64(totalPackages))) +
				"%] " + strconv.Itoa(msg.StatusCode) + " " + msg.Message
			if msg.StatusCode == http.StatusOK {
				log.Info(messageString)
			} else {
				log.Error(messageString)
				*downloadErrors += msg.Message + ", status = " + strconv.Itoa(msg.StatusCode) + "\n"
			}

			*allDownloadInfo = append(
				*allDownloadInfo,
				DownloadInfo{msg.StatusCode, msg.Message, msg.ContentLength, msg.OutputLocation,
					msg.SavedBandwidth, msg.DownloadedPackageType, msg.PackageName,
					msg.PackageVersion, msg.GitPackageShaOrRef},
			)

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
func downloadPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo,
	downloadFileFunction func(string, string) (int, int64),
	gitCloneFunction func(string, string, string, string, string) (string, int64, string)) {

	// Clean up any previous downloaded data, except tar.gz packages.
	// We'll later calculate checksums for tar.gz files and compare them with checksums in
	// in PACKAGES files, so tar.gz files don't have to be downloaded again.
	// Then, recreate these directories.
	for _, directory := range []string{"/github", "/gitlab", "/package_files"} {
		err := os.RemoveAll(localOutputDirectory + directory)
		checkError(err)
		err = os.MkdirAll(localOutputDirectory+directory, os.ModePerm)
		checkError(err)
	}

	// Ensure the directory where tar.gz packages will be downloaded exists.
	err := os.MkdirAll(localOutputDirectory+"/package_archives", os.ModePerm)
	checkError(err)

	biocPackageInfo := make(map[string]map[string]*PackageInfo)
	biocUrls := make(map[string]string)

	if renvLock.Bioconductor.Version != "" {
		getBiocUrls(renvLock.Bioconductor.Version, biocUrls)
		getBioConductorPackages(
			renvLock.Bioconductor.Version, biocPackageInfo, biocUrls,
			downloadFileFunction,
		)
	}

	const localCranPackagesPath = localOutputDirectory + "/package_files/CRAN_PACKAGES"

	currentCranPackageInfo := make(map[string]*PackageInfo)
	// Prepare a map from package name to the current versions of the
	// packages and their checksums as read from PACKAGES file.
	// This way, we'll know whether we should try to download the package from current CRAN repository
	// or from archive, and if we should download the package at all (it may have been already
	// downloaded to local cache).
	// This file is always downloaded because even if CRAN is not specified in renv.lock,
	// it will be used as a fallback for packages that should be downloaded from a repository not
	// defined in the Repositories section of renv.lock.
	status, _ := downloadFileFunction(defaultCranMirrorURL+"/src/contrib/PACKAGES",
		localCranPackagesPath)
	if status == http.StatusOK {
		parsePackagesFile(
			localCranPackagesPath, currentCranPackageInfo,
		)
	}

	// Before downloading any packages, check which packages have already been downloaded to the cache
	// and calculate their checksums. Later on, if we see a package to be downloaded that will have a matching
	// checksum in the PACKAGES file, we'll skip the download and point to already existing file in the cache.
	localArchiveChecksums := make(map[string]*CacheInfo)
	log.Info("Calculating local cache checksums...")
	startTime := time.Now()
	computeChecksums(localOutputDirectory+"/package_archives", localArchiveChecksums)
	elapsedTime := time.Since(startTime)
	log.Info("Calculating local cache checksums took ", fmt.Sprintf("%.2f", elapsedTime.Seconds()),
		" seconds.")
	for k, v := range localArchiveChecksums {
		log.Debug(k, " = ", v)
	}

	messages := make(chan DownloadInfo)

	// Guard channel ensures that only a fixed number of concurrent goroutines are running.
	guard := make(chan struct{}, maxDownloadRoutines)
	// Channel to wait until all downloads have completed.
	downloadWaiter := make(chan struct{})
	numberOfDownloads := 0
	var successfulDownloads, failedDownloads int
	var totalDownloadedBytes int64
	var totalSavedBandwidth int64
	var downloadErrors string

	startTime = time.Now()

	go downloadResultReceiver(messages, &successfulDownloads, &failedDownloads,
		len(renvLock.Packages), &totalDownloadedBytes, &totalSavedBandwidth,
		downloadWaiter, &downloadErrors, allDownloadInfo,
	)

	log.Info("There are ", len(renvLock.Packages), " packages to be downloaded.")
	var repoURL string
	for _, v := range renvLock.Packages {
		if v.Package != "" && v.Version != "" {
			repoURL = getRepositoryURL(v, renvLock.R.Repositories)
			guard <- struct{}{}
			log.Debug("Downloading package ", v.Package)
			go downloadSinglePackage(v.Package, v.Version, repoURL, v.RemoteSha, v.RemoteRef,
				v.Source, v.RemoteSubdir, currentCranPackageInfo, biocPackageInfo, biocUrls,
				localArchiveChecksums, downloadFileFunction, gitCloneFunction, messages, guard)
			numberOfDownloads++
		}
	}

	// Wait for downloadResultReceiver until all download statuses have been retrieved.
	<-downloadWaiter

	if downloadErrors != "" {
		// Not using log because we want to always see this information.
		fmt.Println("\n\nThe following errors were encountered during download:")
		fmt.Print(downloadErrors)
	}

	elapsedTime = time.Since(startTime)
	averageThroughputMbps := float64(int(8000*(float64(totalDownloadedBytes)/
		1000000)/(float64(elapsedTime.Milliseconds())/1000))) / 1000
	averageThroughputBytesPerSecond := float64(totalDownloadedBytes) /
		(float64(elapsedTime.Milliseconds()) / 1000)
	downloadTimeSaved := float64(totalSavedBandwidth) / averageThroughputBytesPerSecond
	log.Info("Total download time = ", fmt.Sprintf("%.2f", elapsedTime.Seconds()), " seconds.")
	log.Info("Downloaded ", totalDownloadedBytes, " bytes.")
	log.Info("Saved ", totalSavedBandwidth, " bytes of bandwidth and ",
		fmt.Sprintf("%.2f", downloadTimeSaved), " seconds of download time due to caching.")
	log.Info("Average throughput = ", averageThroughputMbps, " Mbps.")
	log.Info(
		"Download succeeded for ", successfulDownloads, " packages out of ",
		numberOfDownloads, " requested packages.",
	)
	log.Info(
		"Download failed for ", failedDownloads, " packages out of ",
		numberOfDownloads, " requested packages.",
	)
}
