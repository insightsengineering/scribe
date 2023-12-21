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
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const packageLogPath = "/tmp/scribe/installed_logs"
const buildLogPath = "/tmp/scribe/build_logs"
const gitConst = "git"
const htmlExtension = ".html"

type InstallResultInfo struct {
	PackageName      string `json:"packageName"`
	InputLocation    string `json:"inputLocation"`
	PackageType      string `json:"packageType"`
	PackageVersion   string `json:"packageVersion"`
	Status           string `json:"status"`
	LogFilePath      string `json:"logFilePath"`
	BuildStatus      string `json:"buildStatus"`
	BuildLogFilePath string `json:"buildLogFilePath"`
}

type BuildPackageChanInfo struct {
	BuildStatus    string
	OutputLocation string
	Err            error
}

type ExecRCmdInstallChanInfo struct {
	Output string
	Err    error
}

type DownloadedPackage struct {
	PackageType       string
	PackageVersion    string
	PackageRepository string
	Location          string
}

const InstallResultInfoStatusSucceeded = "SUCCEEDED"
const InstallResultInfoStatusFailed = "FAILED"
const InstallResultInfoStatusBuildFailed = "BUILD_FAILED"

const buildStatusSucceeded = "SUCCEEDED"
const buildStatusFailed = "FAILED"
const buildStatusNotBuilt = "NOT_BUILT"

const rLibsVarName = "R_LIBS="

// Returns tar.gz file name where built package is saved.
// Searches for tar.gz file in current working directory.
// If not found, returns empty string.
func getBuiltPackageFileName(packageName string) string {
	files, err := os.ReadDir(".")
	checkError(err)
	for _, file := range files {
		if !file.IsDir() {
			fileName := file.Name()
			match1, err := regexp.MatchString("^"+packageName+`.*\.tar\.gz$`, fileName)
			checkError(err)
			// Match filename also in such a way that there's underscore immediately after package name.
			// This way e.g. scda.2022 won't be returned while looking for scda.
			match2, err := regexp.MatchString("^"+packageName+`\_`, fileName)
			checkError(err)
			if match1 && match2 {
				return fileName
			}
		}
	}
	return ""
}

func logError(outputLocation string, packageName string, e error, path string) {
	log.Errorf("Error details: outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation, packageName, e, path)
}

func buildPackage(buildPackageChan chan BuildPackageChanInfo, packageName string,
	outputLocation string, buildLogFilePath string, additionalOptions string) {
	log.Infof("Package %s located in %s is a source package so it has to be built first.",
		packageName, outputLocation)
	cmd := rExecutable + " CMD build " + additionalOptions + " " + outputLocation
	log.Trace("execCommand:" + cmd)
	buildLogFile, buildLogFileErr := os.OpenFile(buildLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if buildLogFileErr != nil {
		logError(outputLocation, packageName, buildLogFileErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, buildLogFileErr}
		return
	}
	defer buildLogFile.Close()
	// Add HTML tags to highlight logs
	if _, createHTMLTagsErr := buildLogFile.Write([]byte("<pre><code>\n")); createHTMLTagsErr != nil {
		logError(outputLocation, packageName, createHTMLTagsErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, createHTMLTagsErr}
		return
	}
	// Execute the command
	output, err := execCommand(cmd, false,
		[]string{rLibsVarName + rLibsPaths, "LANG=en_US.UTF-8"}, buildLogFile)
	if err != nil {
		log.Errorf("Error with build: %s . Details: outputLocation:%s packageName:%s\nerr:%v\noutput:%s",
			cmd, outputLocation, packageName, err, output)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, err}
		return
	}
	// Close HTML tags
	if _, closeHTMLTagsErr := buildLogFile.Write([]byte("\n</code></pre>\n")); closeHTMLTagsErr != nil {
		logError(outputLocation, packageName, closeHTMLTagsErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, closeHTMLTagsErr}
		return
	}
	log.Tracef("Executed build step on package %s located in %s", packageName, outputLocation)
	builtPackageName := getBuiltPackageFileName(packageName)
	if builtPackageName != "" {
		// Build succeeded.
		log.Infof("Built package is stored in %s", builtPackageName)
		// Install tar.gz file instead of directory with git source code of the package.
		buildPackageChan <- BuildPackageChanInfo{buildStatusSucceeded, builtPackageName, err}
		return
	}
	buildPackageChan <- BuildPackageChanInfo{buildStatusSucceeded, outputLocation, err}
}

func executeRCmdInstall(execRCmdInstallChan chan ExecRCmdInstallChanInfo, cmd string, logFile *os.File) {
	output, err := execCommand(cmd, false,
		[]string{rLibsVarName + rLibsPaths, "LANG=en_US.UTF-8"}, logFile)
	execRCmdInstallChan <- ExecRCmdInstallChanInfo{output, err}
}

// Returns error and build status (succeeded, failed or package not built).
func executeInstallation(outputLocation, packageName, logFilePath, buildLogFilePath, packageType string,
	additionalBuildOptions string, additionalInstallOptions string) (string, error) {
	log.Tracef("Executing installation step on package %s located in %s", packageName, outputLocation)
	logFile, logFileErr := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	buildStatus := buildStatusNotBuilt
	var err error
	if logFileErr != nil {
		logError(outputLocation, packageName, logFileErr, logFilePath)
		return buildStatus, logFileErr
	}
	defer logFile.Close()
	// Add HTML tags to highlight logs.
	if _, createHTMLTagsErr := logFile.Write([]byte("<pre><code>\n")); createHTMLTagsErr != nil {
		logError(outputLocation, packageName, createHTMLTagsErr, logFilePath)
		return buildStatus, createHTMLTagsErr
	}

	if packageType == gitConst {
		// By default previous outputLocation will be returned, except if package is successfully built.
		// In the latter case, tar.gz package name will be returned as outputLocation.
		buildPackageChan := make(chan BuildPackageChanInfo)
		go buildPackage(buildPackageChan, packageName, outputLocation, buildLogFilePath, additionalBuildOptions)
		var waitInterval = 1
		var totalWaitTime = 0
		// Wait until buildPackage() completes.
	build_package_loop:
		for {
			select {
			case msg := <-buildPackageChan:
				buildStatus = msg.BuildStatus
				outputLocation = msg.OutputLocation
				err = msg.Err
				log.Info("Building package ", packageName, " completed after ", getTimeMinutesAndSeconds(totalWaitTime))
				break build_package_loop
			default:
				if totalWaitTime%20 == 0 {
					log.Info("Building package ", packageName, "... [", getTimeMinutesAndSeconds(totalWaitTime), " elapsed]")
				}
				time.Sleep(time.Duration(waitInterval) * time.Second)
				totalWaitTime += waitInterval
			}
		}
		if err != nil {
			return buildStatus, err
		}
	}

	cmd := rExecutable + " CMD INSTALL --no-lock -l " + temporaryLibPath + " " + additionalInstallOptions + " " + outputLocation
	log.Trace("Executing command: " + cmd)
	execRCmdInstallChan := make(chan ExecRCmdInstallChanInfo)
	go executeRCmdInstall(execRCmdInstallChan, cmd, logFile)
	var waitInterval = 1
	var totalWaitTime = 0
	var output string
	// Wait until executeRCmdInstall() completes.
r_cmd_install_loop:
	for {
		select {
		case msg := <-execRCmdInstallChan:
			output = msg.Output
			err = msg.Err
			log.Info("R CMD INSTALL ", packageName, " completed after ", getTimeMinutesAndSeconds(totalWaitTime))
			break r_cmd_install_loop
		default:
			if totalWaitTime%20 == 0 {
				log.Info("R CMD INSTALL ", packageName, "... [", getTimeMinutesAndSeconds(totalWaitTime), " elapsed]")
			}
			time.Sleep(time.Duration(waitInterval) * time.Second)
			totalWaitTime += waitInterval
		}
	}
	if err != nil {
		log.Errorf("Error running: %s. Details: outputLocation:%s packageName:%s\nerr:%v\noutput:%s", cmd, outputLocation, packageName, err, output)
	}
	if _, closeHTMLTagsErr := logFile.Write([]byte("\n</code></pre>\n")); closeHTMLTagsErr != nil {
		logError(outputLocation, packageName, closeHTMLTagsErr, logFilePath)
		return buildStatus, closeHTMLTagsErr
	}
	log.Tracef("Executed installation step on package %s located in %s", packageName, outputLocation)
	return buildStatus, err
}

func installSinglePackage(installResultChan chan InstallResultInfo, packageName string, packageType string,
	inputLocation string, additionalBuildOptions string, additionalInstallOptions string) {
	logFilePath := filepath.Join(packageLogPath, packageName+htmlExtension)
	buildLogFilePath := filepath.Join(buildLogPath, packageName+htmlExtension)
	buildStatus, err := executeInstallation(inputLocation, packageName,
		logFilePath, buildLogFilePath, packageType, additionalBuildOptions, additionalInstallOptions)
	packageVersion := ""
	var status string
	switch {
	case err == nil:
		descFilePath := filepath.Join(temporaryLibPath, packageName, "DESCRIPTION")
		installedDesc := parseDescriptionFile(descFilePath)
		packageVersion = installedDesc["Version"]
		status = InstallResultInfoStatusSucceeded
	case buildStatus == buildStatusFailed:
		status = InstallResultInfoStatusBuildFailed
	default:
		status = InstallResultInfoStatusFailed
	}
	installResultChan <- InstallResultInfo{
		PackageName:      packageName,
		InputLocation:    inputLocation,
		PackageType:      packageType,
		Status:           status,
		PackageVersion:   packageVersion,
		LogFilePath:      logFilePath,
		BuildStatus:      buildStatus,
		BuildLogFilePath: buildLogFilePath,
	}
}

// getPackagesReadyToInstall iterates through all packages which should eventually be
// installed, and marks package in readyPackages as ready to install, if all
// package dependencies have been installed, the package is not currently being installed
// and has not yet been installed.
func getPackagesReadyToInstall(
	dependencies map[string][]string,
	installedPackages []string,
	packagesBeingInstalled map[string]bool,
	readyPackages map[string]bool,
) {
	for packageName, packageDeps := range dependencies {
		packageReady := true
		for _, d := range packageDeps {
			if !stringInSlice(d, installedPackages) {
				packageReady = false
			}
		}
		pkgBeingInstalled, ok := packagesBeingInstalled[packageName]
		if !ok {
			pkgBeingInstalled = false
		}
		if packageReady && !stringInSlice(packageName, installedPackages) &&
			!pkgBeingInstalled {
			readyPackages[packageName] = true
		}
	}
}

// mapToList returns a slice of map keys for those elements of the map for which the value is true.
func mapToList(m map[string]bool) []string {
	var outputList []string
	for k, v := range m {
		if v {
			outputList = append(outputList, k)
		}
	}
	return outputList
}

// getPackageToInstall gets the first available package from the ready-to-install queue.
func getPackageToInstall(
	packagesBeingInstalled map[string]bool,
	readyPackages map[string]bool,
) string {
	for k, v := range readyPackages {
		if v {
			packagesBeingInstalled[k] = true
			readyPackages[k] = false
			return k
		}
	}
	return ""
}

func installPackages(
	renvLock Renvlock,
	allDownloadInfo *[]DownloadInfo,
	allInstallInfo *[]InstallResultInfo,
	additionalBuildOptions string,
	additionalInstallOptions string,
) {
	err := os.MkdirAll(temporaryLibPath, os.ModePerm)
	checkError(err)
	err = os.MkdirAll(packageLogPath, os.ModePerm)
	checkError(err)

	downloadedPackages := make(map[string]DownloadedPackage)
	for _, v := range *allDownloadInfo {
		downloadedPackages[v.PackageName] = DownloadedPackage{
			v.DownloadedPackageType, v.PackageVersion, v.PackageRepository, v.OutputLocation,
		}
	}

	dependencies := getPackageDeps(
		renvLock.Packages,
		renvLock.R.Repositories,
		downloadedPackages,
	)

	var installedPackages []string
	readyPackages := make(map[string]bool)
	packagesBeingInstalled := make(map[string]bool)
	installationResultChan := make(chan InstallResultInfo)

	getPackagesReadyToInstall(dependencies, installedPackages, packagesBeingInstalled, readyPackages)

	packagesInstalledSuccessfully := 0
	packagesInstalledUnsuccessfully := 0

package_installation_loop:
	for {
		select {
		case msg := <-installationResultChan:
			receivedPackageName := msg.PackageName
			receivedStatus := msg.Status
			log.Info("Installation of ", receivedPackageName, " completed, status = ", receivedStatus, ".")
			*allInstallInfo = append(*allInstallInfo, msg)

			if receivedStatus == InstallResultInfoStatusSucceeded {
				packagesInstalledSuccessfully++
			} else {
				packagesInstalledUnsuccessfully++
			}

			// Mark the package as installed, and not one being installed.
			installedPackages = append(installedPackages, receivedPackageName)
			packagesBeingInstalled[receivedPackageName] = false

			// Recalculate the list of packages ready to be installed.
			getPackagesReadyToInstall(dependencies, installedPackages, packagesBeingInstalled, readyPackages)

			log.Info(
				len(mapToList(readyPackages)), " packages ready. ",
				len(mapToList(packagesBeingInstalled)), " packages being installed. ",
				len(installedPackages), "/", len(downloadedPackages), " packages processed (",
				packagesInstalledSuccessfully, " succeeded, ", packagesInstalledUnsuccessfully,
				" failed).",
			)
		default:
			if len(mapToList(readyPackages))+len(mapToList(packagesBeingInstalled)) == 0 {
				break package_installation_loop
			}
			if uint(len(mapToList(packagesBeingInstalled))) < numberOfWorkers {
				packageName := getPackageToInstall(packagesBeingInstalled, readyPackages)
				if packageName != "" {
					log.Info("Installing ", packageName, "...")
					go installSinglePackage(installationResultChan, packageName,
						downloadedPackages[packageName].PackageType,
						downloadedPackages[packageName].Location,
						additionalBuildOptions, additionalInstallOptions)
				} else {
					// No package ready to install.
					time.Sleep(2 * time.Second)
				}
			} else {
				// Maximum number of concurrent installations reached.
				time.Sleep(2 * time.Second)
			}
		}
	}

	installResultFilePath := filepath.Join(tempCacheDirectory, "installResultInfo.json")
	writeJSON(installResultFilePath, *allInstallInfo)
	log.Info("Installation of ", len(*allInstallInfo), " packages completed.")
}
