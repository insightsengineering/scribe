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

type InstallInfo struct {
	PackageName   string `json:"packageName"`
	InputLocation string `json:"inputLocation"`
	PackageType   string `json:"packageType"`
}

type InstallResultInfo struct {
	InstallInfo
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
const InstallResultInfoStatusSkipped = "SKIPPED"
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

func buildPackage(buildPackageChan chan BuildPackageChanInfo, packageName string,
	outputLocation string, buildLogFilePath string, additionalOptions string) {
	log.Infof("Package %s located in %s is a source package so it has to be built first.",
		packageName, outputLocation)
	cmd := rExecutable + " CMD build " + additionalOptions + " " + outputLocation
	log.Trace("execCommand:" + cmd)
	buildLogFile, buildLogFileErr := os.OpenFile(buildLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if buildLogFileErr != nil {
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation, packageName,
			buildLogFileErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, buildLogFileErr}
		return
	}
	defer buildLogFile.Close()
	// Add HTML tags to highlight logs
	if _, createHTMLTagsErr := buildLogFile.Write([]byte("<pre><code>\n")); createHTMLTagsErr != nil {
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation, packageName,
			createHTMLTagsErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, createHTMLTagsErr}
		return
	}
	// Execute the command
	output, err := execCommand(cmd, false,
		[]string{
			rLibsVarName + rLibsPaths,
			"LANG=en_US.UTF-8",
		}, buildLogFile)
	if err != nil {
		log.Errorf("Error with build: %s . Details: outputLocation:%s packageName:%s\nerr:%v\noutput:%s",
			cmd, outputLocation, packageName, err, output)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, err}
		return
	}
	// Close HTML tags
	if _, closeHTMLTagsErr := buildLogFile.Write([]byte("\n</code></pre>\n")); closeHTMLTagsErr != nil {
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation, packageName,
			closeHTMLTagsErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, closeHTMLTagsErr}
		return
	}
	log.Infof("Executed build step on package %s located in %s", packageName, outputLocation)
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
		[]string{
			rLibsVarName + rLibsPaths,
			"LANG=en_US.UTF-8",
		}, logFile)
	execRCmdInstallChan <- ExecRCmdInstallChanInfo{output, err}
}

// Returns error and build status (succeeded, failed or package not built).
func executeInstallation(outputLocation, packageName, logFilePath, buildLogFilePath, packageType string,
	additionalBuildOptions string, additionalInstallOptions string) (string, error) {
	log.Infof("Executing installation step on package %s located in %s", packageName, outputLocation)
	logFile, logFileErr := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	buildStatus := buildStatusNotBuilt
	var err error
	if logFileErr != nil {
		log.Errorf("Error details: outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation,
			packageName, logFileErr, logFilePath)
		return buildStatus, logFileErr
	}
	defer logFile.Close()
	// Add HTML tags to highlight logs
	if _, createHTMLTagsErr := logFile.Write([]byte("<pre><code>\n")); createHTMLTagsErr != nil {
		log.Errorf("Error details: outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation,
			packageName, createHTMLTagsErr, logFilePath)
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
	log.Trace("Executing command:" + cmd)
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
		log.Error(cmd)
		log.Errorf("Error running: %s. Details: outputLocation:%s packageName:%s\nerr:%v\noutput:%s", cmd, outputLocation, packageName, err, output)
	}
	if _, closeHTMLTagsErr := logFile.Write([]byte("\n</code></pre>\n")); closeHTMLTagsErr != nil {
		log.Errorf("Error details: outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation,
			packageName, closeHTMLTagsErr, logFilePath)
		return buildStatus, closeHTMLTagsErr
	}
	log.Infof("Executed installation step on package %s located in %s", packageName, outputLocation)
	return buildStatus, err
}

func installSinglePackageWorker(installChan chan InstallInfo, installResultChan chan InstallResultInfo,
	additionalBuildOptions string, additionalInstallOptions string) {
	for installInfo := range installChan {
		logFilePath := filepath.Join(packageLogPath, installInfo.PackageName+htmlExtension)
		buildLogFilePath := filepath.Join(buildLogPath, installInfo.PackageName+htmlExtension)
		buildStatus, err := executeInstallation(installInfo.InputLocation, installInfo.PackageName,
			logFilePath, buildLogFilePath, installInfo.PackageType, additionalBuildOptions, additionalInstallOptions)
		packageVersion := ""
		var status string
		switch {
		case err == nil:
			log.Tracef("No error after installation of %s", installInfo.PackageName)
			descFilePath := filepath.Join(temporaryLibPath, installInfo.PackageName, "DESCRIPTION")
			installedDesc := parseDescriptionFile(descFilePath)
			packageVersion = installedDesc["Version"]
			status = InstallResultInfoStatusSucceeded
		case buildStatus == buildStatusFailed:
			status = InstallResultInfoStatusBuildFailed
		default:
			status = InstallResultInfoStatusFailed
		}
		log.Tracef("Sending response from %s", installInfo.PackageName)
		installResultChan <- InstallResultInfo{
			InstallInfo:      installInfo,
			Status:           status,
			PackageVersion:   packageVersion,
			LogFilePath:      logFilePath,
			BuildStatus:      buildStatus,
			BuildLogFilePath: buildLogFilePath,
		}
		log.Tracef("Installation of package %s is done", installInfo.PackageName)
	}
}

func getOrderedDependencies(
	renvLock Renvlock,
	downloadedPackages map[string]DownloadedPackage,
	includeSuggests bool,
) {
// ) ([]string, map[string][]string) {
	// deps := make(map[string][]string)
	// var depsOrdered []string

	depsAll := getPackageDeps(
		renvLock.Packages,
		renvLock.Bioconductor.Version,
		renvLock.R.Repositories,
		downloadedPackages,
		includeSuggests,
	)

	log.Info(depsAll)

	// packagesLocation -> downloadedPackages



	// for p, depAll := range depsAll {
	// 	if _, ok := packagesLocation[p]; ok {
	// 		dep := make([]string, 0)
	// 		for _, d := range depAll {
	// 			_, okInpackagesLocation := packagesLocation[d]
	// 			_, okInInstalledDeps := installedDeps[d]
	// 			if okInpackagesLocation || okInInstalledDeps {
	// 				dep = append(dep, d)
	// 			}
	// 		}
	// 		deps[p] = dep
	// 	}
	// }

	// log.Debug("Running a topological sort on ", len(deps), " packages")
	// depsOrdered = tsort(deps)
	// log.Debugf("Success running topological sort on %d packages. Ordering complete for %d packages", len(deps), len(depsOrdered))

	// depsOrderedToInstall := make([]string, 0)
	// for _, packageName := range depsOrdered {
	// 	if _, ok := packagesLocation[packageName]; ok {
	// 		depsOrderedToInstall = append(depsOrderedToInstall, packageName)
	// 	}
	// }
	// return depsOrderedToInstall, deps
}

// nolint: gocyclo
func installPackages(
	renvLock Renvlock,
	allDownloadInfo *[]DownloadInfo,
	installResultInfos *[]InstallResultInfo,
	includeSuggests bool,
	additionalBuildOptions string,
	additionalInstallOptions string,
) {
	err := os.MkdirAll(temporaryLibPath, os.ModePerm)
	checkError(err)
	err = os.MkdirAll(packageLogPath, os.ModePerm)
	checkError(err)

	downloadedPackages := make(map[string]DownloadedPackage)
	for _, v := range *allDownloadInfo {
		downloadedPackages[v.PackageName] = struct{
				PackageType, PackageVersion, PackageRepository, Location string
			}{
			v.DownloadedPackageType, v.PackageVersion, v.PackageRepository, v.OutputLocation,
		}
	}

	// depsOrderedToInstall, deps := getOrderedDependencies(renvLock, downloadedPackages, includeSuggests)
	getOrderedDependencies(renvLock, downloadedPackages, includeSuggests)
	os.Exit(0)

	// installChan := make(chan InstallInfo)
	// defer close(installChan)
	// installResultChan := make(chan InstallResultInfo)
	// defer close(installResultChan)

	// for i := range depsOrderedToInstall {
	// 	log.Tracef("Starting installation worker #%d", i)
	// 	go installSinglePackageWorker(installChan, installResultChan, additionalBuildOptions, additionalInstallOptions)
	// }

	// installing := make(map[string]bool)
	// processed := make(map[string]bool)
	// for k := range installedDeps {
	// 	processed[k] = true
	// }

	// minI := 0
	// maxI := int(numberOfWorkers) // max number of parallel installing workers

	// // running packages which have no dependencies
	// counter := minI // number of currently installing packages in queue

	// for _, p := range depsOrderedToInstall {
	// 	log.Tracef("Checking %s", p)
	// 	ver, ok := installedDeps[p]
	// 	if !ok {
	// 		if isDependencyFulfilled(p, deps, installedDeps) {
	// 			counter++
	// 			log.Tracef("Triggering %s", p)
	// 			installing[p] = true
	// 			installChan <- InstallInfo{p, packagesLocation[p].Location, packagesLocation[p].PackageType}
	// 		}
	// 		if counter >= maxI {
	// 			// TODO: What does this mean?
	// 			log.Infof("All the rest packages have dependencies. Counter:%d", counter)
	// 			break
	// 		}
	// 	} else {
	// 		*installResultInfos = append(*installResultInfos, InstallResultInfo{
	// 			InstallInfo: InstallInfo{
	// 				PackageName:   p,
	// 				InputLocation: packagesLocation[p].Location,
	// 				PackageType:   packagesLocation[p].PackageType,
	// 			},
	// 			PackageVersion: ver,
	// 			LogFilePath:    "",
	// 			Status:         InstallResultInfoStatusSkipped,
	// 		})
	// 	}
	// }

	// if len(*installResultInfos) < len(depsOrderedToInstall) {
	// Loop:
	// 	for installResultInfo := range installResultChan {
	// 		*installResultInfos = append(*installResultInfos, installResultInfo)
	// 		delete(installing, installResultInfo.PackageName)
	// 		processed[installResultInfo.PackageName] = true
	// 		installedDeps[installResultInfo.PackageName] = ""
	// 		for i := minI; i <= maxI && i < len(depsOrderedToInstall); i++ {
	// 			nextPackage := depsOrderedToInstall[i]
	// 			if !processed[nextPackage] {
	// 				if !installing[nextPackage] {
	// 					if isDependencyFulfilled(nextPackage, deps, installedDeps) {
	// 						installChan <- InstallInfo{nextPackage, packagesLocation[nextPackage].Location,
	// 							packagesLocation[nextPackage].PackageType}
	// 						installing[nextPackage] = true
	// 					}
	// 				}
	// 			} else {
	// 				if i == minI {
	// 					minI++ // increment if package with index minI has been installed
	// 					maxI++
	// 				}
	// 			}
	// 		}
	// 		if minI >= len(depsOrderedToInstall) {
	// 			break Loop
	// 		}
	// 		// TODO: What does this mean?
	// 		log.Tracef("End %s\n minI: %d\n maxI:%d\n installing: %v\n processed:%v", installResultInfo.PackageName, minI, maxI, installing, processed)
	// 	}
	// }

	// installResultInfosFilePath := filepath.Join(tempCacheDirectory, "installResultInfo.json")
	// writeJSON(installResultInfosFilePath, *installResultInfos)
	// log.Info("Installation of ", len(*installResultInfos), " packages completed.")
}
