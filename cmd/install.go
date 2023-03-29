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
	"encoding/json"

	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const temporalLibPath = "/tmp/scribe/installed_packages"
const rLibsPaths = "/tmp/scribe/installed_packages:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"

const packageLogPath = "/tmp/scribe/installed_logs"
const buildLogPath = "/tmp/scribe/build_logs"
const gitConst = "git"

type InstallInfo struct {
	PackageName   string `json:"packageName"`
	InputLocation string `json:"inputLocation"`
	PackageType   string `json:"packageType"`
}

type InstallResultInfo struct {
	InstallInfo
	PackageVersion   string `json:"packageVersion"`
	Status           int    `json:"status"`
	LogFilePath      string `json:"logFilePath"`
	BuildStatus      int    `json:"buildStatus"`
	BuildLogFilePath string `json:"buildLogFilePath"`
}

type BuildPackageChanInfo struct {
	BuildStatus int
	OutputLocation string
	Err error
}

type ExecRCmdInstallChanInfo struct {
	Output string
	Err error
}

const (
	InstallResultInfoStatusSucceeded = iota
	InstallResultInfoStatusSkipped
	InstallResultInfoStatusFailed
	InstallResultInfoStatusBuildFailed
)

const (
	buildStatusSucceeded = iota
	buildStatusFailed
	buildStatusNotBuilt
)

func mkLibPathDir(temporalLibPath string) {
	for _, libPath := range strings.Split(temporalLibPath, ":") {
		if _, err := os.Stat(libPath); os.IsNotExist(err) {
			err := os.MkdirAll(libPath, os.ModePerm)
			checkError(err)
			log.Tracef("Created dir %s", libPath)
		}
	}
}

func getInstalledPackagesWithVersionWithBaseRPackages(libPaths []string) map[string]string {
	installedPackages := getInstalledPackagesWithVersion(libPaths)
	basePackages := []string{"stats", "graphics", "grDevices", "utils", "datasets", "methods", "base"}
	for _, p := range basePackages {
		installedPackages[p] = ""
	}
	return installedPackages
}

func getInstalledPackagesWithVersion(libPaths []string) map[string]string {
	log.Debug("Getting installed packages")
	res := make(map[string]string)
	for _, libPathMultiple := range libPaths {
		for _, libPath := range strings.Split(libPathMultiple, ":") {
			log.Debugf("Searching for installed package under %s", libPath)
			descFilePaths := make([]string, 0)

			files, err := os.ReadDir(libPath)
			if err != nil {
				log.Errorf("libPath: %s Error: %v", libPath, err)
			}
			for _, f := range files {
				log.Tracef("Checking dir %s", f)
				if f.IsDir() {
					descFilePath := filepath.Join(libPath, f.Name(), "DESCRIPTION")
					log.Tracef("Checking file %s", descFilePath)
					if _, errStat := os.Stat(descFilePath); errStat == nil {
						descFilePaths = append(descFilePaths, descFilePath)
					}
				}
			}

			for _, descFilePath := range descFilePaths {
				descItems := parseDescriptionFile(descFilePath)
				packageName := descItems["Package"]
				packageVersion := descItems["Version"]
				if _, ok := res[packageName]; !ok {
					res[packageName] = packageVersion
				} else {
					log.Warnf("Duplicate package %s with version %s under %s libPath",
						packageName, packageVersion, libPath)
				}
			}
		}
	}
	log.Debugf("There are %d installed packages", len(res))
	return res
}

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
	cmd := "R CMD build " + additionalOptions + " " + outputLocation
	log.Trace("execCommand:" + cmd)
	buildLogFile, buildLogFileErr := os.OpenFile(buildLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if buildLogFileErr != nil {
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation, packageName,
			buildLogFileErr, buildLogFilePath)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, buildLogFileErr}
		return
	}
	defer buildLogFile.Close()
	output, err := execCommand(cmd, false,
		[]string{
			"R_LIBS=" + rLibsPaths,
			"LANG=en_US.UTF-8",
		}, buildLogFile)
	if err != nil {
		log.Errorf("Error with build: %s . Details: outputLocation:%s packageName:%s\nerr:%v\noutput:%s",
			cmd, outputLocation, packageName, err, output)
		buildPackageChan <- BuildPackageChanInfo{buildStatusFailed, outputLocation, err}
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
		"R_LIBS=" + rLibsPaths,
		"LANG=en_US.UTF-8",
	}, logFile)
	execRCmdInstallChan <- ExecRCmdInstallChanInfo{output, err}
}

// Returns error and build status (succeeded, failed or package not built).
func executeInstallation(outputLocation, packageName, logFilePath, buildLogFilePath, packageType string,
		additionalBuildOptions string, additionalInstallOptions string) (int, error) {
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

	if packageType == gitConst {
		// By default previous outputLocation will be returned, except if package is successfully built.
		// In the latter case, tar.gz package name will be returned as outputLocation.
		buildPackageChan := make(chan BuildPackageChanInfo)
		go buildPackage(buildPackageChan, packageName, outputLocation, buildLogFilePath, additionalBuildOptions)
		var waitInterval = 1
		var totalWaitTime = 0
build_package_loop:
		for {
			select {
			case msg := <-buildPackageChan:
				buildStatus = msg.BuildStatus
				outputLocation = msg.OutputLocation
				err = msg.Err
				log.Info("Package build for ", packageName, " completed after ", totalWaitTime, "s")
				break build_package_loop
			default:
				if totalWaitTime%5 == 0 {
					log.Info("Building package ", packageName, "... [", totalWaitTime, "s elapsed]")
				}
				time.Sleep(time.Duration(waitInterval) * time.Second)
				totalWaitTime += waitInterval
			}
		}
		if err != nil {
			return buildStatus, err
		}
	}

	cmd := "R CMD INSTALL --no-lock -l " + temporalLibPath + " " + additionalInstallOptions + " " + outputLocation
	log.Trace("Executing command:" + cmd)
	execRCmdInstallChan := make(chan ExecRCmdInstallChanInfo)
	go executeRCmdInstall(execRCmdInstallChan, cmd, logFile)
	var waitInterval = 1
	var totalWaitTime = 0
	var output string
r_cmd_install_loop:
	for {
		select {
		case msg := <-execRCmdInstallChan:
			output = msg.Output
			err = msg.Err
			log.Info("R CMD INSTALL ", packageName, " completed after ", totalWaitTime, "s")
			break r_cmd_install_loop
		default:
			if totalWaitTime%5 == 0 {
				log.Info("R CMD INSTALL ", packageName, "... [", totalWaitTime, "s elapsed]")
			}
			time.Sleep(time.Duration(waitInterval) * time.Second)
			totalWaitTime += waitInterval
		}
	}
	if err != nil {
		log.Error(cmd)
		log.Errorf("Error running: %s. Details: outputLocation:%s packageName:%s\nerr:%v\noutput:%s", cmd, outputLocation, packageName, err, output)
	}
	log.Infof("Executed installation step on package %s located in %s", packageName, outputLocation)
	return buildStatus, err
}

func installSinglePackageWorker(installChan chan InstallInfo, installResultChan chan InstallResultInfo,
		additionalBuildOptions string, additionalInstallOptions string) {
	for installInfo := range installChan {
		logFilePath := filepath.Join(packageLogPath, installInfo.PackageName+".out")
		buildLogFilePath := filepath.Join(buildLogPath, installInfo.PackageName+".out")
		buildStatus, err := executeInstallation(installInfo.InputLocation, installInfo.PackageName,
			logFilePath, buildLogFilePath, installInfo.PackageType, additionalBuildOptions, additionalInstallOptions)
		packageVersion := ""
		var status int
		switch {
		case err == nil:
			log.Tracef("No error after installation of %s", installInfo.PackageName)
			descFilePath := filepath.Join(temporalLibPath, installInfo.PackageName, "DESCRIPTION")
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
	packagesLocation map[string]struct{ PackageType, Location string },
	installedDeps map[string]string,
	includeSuggests bool,
) ([]string, map[string][]string) {
	deps := make(map[string][]string)
	var depsOrdered []string

	var reposURLs []string
	for _, v := range renvLock.R.Repositories {
		reposURLs = append(reposURLs, v.URL)
	}

	readFile := filepath.Join(tempCacheDirectory, "deps.json")
	if _, err := os.Stat(readFile); err == nil {
		log.Info("Reading file ", readFile)
		jsonFile, errr := os.ReadFile(readFile)
		checkError(errr)
		errunmarshal := json.Unmarshal(jsonFile, &deps)
		checkError(errunmarshal)
	} else {
		depsAll := getPackageDeps(
			renvLock.Packages,
			renvLock.Bioconductor.Version,
			reposURLs,
			packagesLocation,
			includeSuggests,
		)

		for p, depAll := range depsAll {
			if _, ok := packagesLocation[p]; ok {
				dep := make([]string, 0)
				for _, d := range depAll {
					_, okInpackagesLocation := packagesLocation[d]
					_, okInInstalledDeps := installedDeps[d]
					if okInpackagesLocation || okInInstalledDeps {
						dep = append(dep, d)
					}
				}
				deps[p] = dep
			}
		}
		writeJSON(readFile, deps)
	}
	defer os.Remove(readFile)

	readFile = filepath.Join(tempCacheDirectory, "depsOrdered.json")
	if _, err := os.Stat(readFile); err == nil {
		log.Infof("Reading %s", readFile)
		jsonFile, errr := os.ReadFile(readFile)
		checkError(errr)
		errUnmarshal := json.Unmarshal(jsonFile, &depsOrdered)
		checkError(errUnmarshal)
	} else {
		log.Debugf("Running a topological sort on %d packages", len(deps))
		depsOrdered = tsort(deps)
		log.Debugf("Success running topological sort on %d packages. Ordering complete for %d packages", len(deps), len(depsOrdered))
		writeJSON(readFile, depsOrdered)
	}
	defer os.Remove(readFile)

	depsOrderedToInstall := make([]string, 0)
	for _, packageName := range depsOrdered {
		if _, ok := packagesLocation[packageName]; ok {
			depsOrderedToInstall = append(depsOrderedToInstall, packageName)
		}
	}
	// TODO: What does this mean?
	log.Tracef("There are %d packages, but after cleaning it is %d", len(depsOrdered), len(depsOrderedToInstall))
	return depsOrderedToInstall, deps
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
	mkLibPathDir(temporalLibPath)
	mkLibPathDir(packageLogPath)

	installedDeps := getInstalledPackagesWithVersionWithBaseRPackages([]string{temporalLibPath})
	log.Tracef("There are %d installed packages under %s location", len(installedDeps), temporalLibPath)
	packagesLocation := make(map[string]struct{ PackageType, Location string })
	for _, v := range *allDownloadInfo {
		packagesLocation[v.PackageName] = struct{ PackageType, Location string }{v.DownloadedPackageType, v.OutputLocation}
	}

	depsOrderedToInstall, deps := getOrderedDependencies(renvLock, packagesLocation, installedDeps, includeSuggests)

	installChan := make(chan InstallInfo)
	defer close(installChan)
	installResultChan := make(chan InstallResultInfo)
	defer close(installResultChan)

	for i := range depsOrderedToInstall {
		log.Tracef("Starting installation worker #%d", i)
		go installSinglePackageWorker(installChan, installResultChan, additionalBuildOptions, additionalInstallOptions)
	}

	installing := make(map[string]bool)
	processed := make(map[string]bool)
	for k := range installedDeps {
		processed[k] = true
	}

	minI := 0
	maxI := int(numberOfWorkers) // max number of parallel installing workers

	// running packages which have no dependencies
	counter := minI // number of currently installing packages in queue

	for _, p := range depsOrderedToInstall {
		log.Tracef("Checking %s", p)
		ver, ok := installedDeps[p]
		if !ok {
			if isDependencyFulfilled(p, deps, installedDeps) {
				counter++
				log.Tracef("Triggering %s", p)
				installing[p] = true
				installChan <- InstallInfo{p, packagesLocation[p].Location, packagesLocation[p].PackageType}
			}
			if counter >= maxI {
				// TODO: What does this mean?
				log.Infof("All the rest packages have dependencies. Counter:%d", counter)
				break
			}
		} else {
			*installResultInfos = append(*installResultInfos, InstallResultInfo{
				InstallInfo: InstallInfo{
					PackageName:   p,
					InputLocation: packagesLocation[p].Location,
					PackageType:   packagesLocation[p].PackageType,
				},
				PackageVersion: ver,
				LogFilePath:    "",
				Status:         InstallResultInfoStatusSkipped,
			})
		}
	}

	if len(*installResultInfos) < len(depsOrderedToInstall) {
		// TODO: What does this mean?
		log.Tracef("Running on channels")
	Loop:
		for installResultInfo := range installResultChan {
			*installResultInfos = append(*installResultInfos, installResultInfo)
			delete(installing, installResultInfo.PackageName)
			processed[installResultInfo.PackageName] = true
			installedDeps[installResultInfo.PackageName] = ""
			for i := minI; i <= maxI && i < len(depsOrderedToInstall); i++ {
				nextPackage := depsOrderedToInstall[i]
				if !processed[nextPackage] {
					if !installing[nextPackage] {
						if isDependencyFulfilled(nextPackage, deps, installedDeps) {
							installChan <- InstallInfo{nextPackage, packagesLocation[nextPackage].Location,
								packagesLocation[nextPackage].PackageType}
							installing[nextPackage] = true
						}
					}
				} else {
					if i == minI {
						minI++ // increment if package with index minI has been installed
						maxI++
					}
				}
			}
			if minI >= len(depsOrderedToInstall) {
				break Loop
			}
			// TODO: What does this mean?
			log.Tracef("End %s\n minI: %d\n maxI:%d\n installing: %v\n processed:%v", installResultInfo.PackageName, minI, maxI, installing, processed)
		}
	}

	installResultInfosFilePath := filepath.Join(tempCacheDirectory, "installResultInfos.json")
	log.Tracef("Writing installation status file into %s", installResultInfosFilePath)
	writeJSON(installResultInfosFilePath, *installResultInfos)
	log.Infof("Installation for %d is done", len(*installResultInfos))
}
