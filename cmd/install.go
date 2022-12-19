/*
Copyright 2022 F. Hoffmann-La Roche AG

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

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const temporalLibPath = "/tmp/scribe/installed_packages"
const rLibsPaths = "/tmp/scribe/installed_packages:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"

const packageLogPath = "/tmp/scribe/installed_logs"

type InstallInfo struct {
	PackageName   string `json:"packageName"`
	InputLocation string `json:"inputLocation"`
}

type InstallResultInfo struct {
	InstallInfo
	PackageVersion string `json:"packageVersion"`
	Status         int    `json:"status"`
	LogFilePath    string `json:"logFilePath"`
}

const (
	InstallResultInfoStatusSucceeded = iota
	InstallResultInfoStatusSkipped
	InstallResultInfoStatusFailed
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

			files, err := ioutil.ReadDir(libPath)
			if err != nil {
				log.Errorf("libPath: %s Error: %v", libPath, err)
			}
			for _, f := range files {
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
					log.Tracef("Duplicated package %s with version %s under %s libPath", packageName, packageVersion, libPath)
				}
			}
		}
	}
	log.Debugf("There are %d installed packages", len(res))
	return res
}

func executeInstallation(outputLocation, packageName, logFilePath string) error {
	log.Infof("Executing Installation step on package %s located in %s", packageName, outputLocation)
	logFile, logFileErr := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	defer logFile.Close()
	if logFileErr != nil {
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\nfile:%s", outputLocation, packageName, logFileErr, logFilePath)
		return logFileErr
	}

	cmd := "R CMD INSTALL --no-lock -l " + temporalLibPath + " " + outputLocation
	log.Trace("execCommand:" + cmd)
	output, err := execCommand(cmd, false, false,
		[]string{
			"R_LIBS=" + rLibsPaths,
			"LANG=en_US.UTF-8",
		}, logFile)
	if err != nil {
		log.Error(cmd)
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\noutput:%s", outputLocation, packageName, err, output)
	}
	log.Infof("Executed Installation step on package %s located in %s", packageName, outputLocation)
	return err
}

func installSinglePackageWorker(installChan chan InstallInfo, installResultChan chan InstallResultInfo) {
	for {
		select {
		case installInfo := <-installChan:
			logFilePath := filepath.Join(packageLogPath, installInfo.PackageName+".out")
			err := executeInstallation(installInfo.InputLocation, installInfo.PackageName, logFilePath)
			packageVersion := ""
			Status := InstallResultInfoStatusFailed
			if err == nil {
				descFilePath := filepath.Join(installInfo.InputLocation, installInfo.PackageName, "DESCRIPTION")
				installedDesc := parseDescriptionFile(descFilePath)
				packageVersion = installedDesc["Version"]
				Status = InstallResultInfoStatusSucceeded
			}
			installResultChan <- InstallResultInfo{
				InstallInfo:    installInfo,
				Status:         Status,
				PackageVersion: packageVersion,
				LogFilePath:    logFilePath,
			}
		}
	}
}

func getOrderedDependencies(
	renvLock Renvlock,
	packagesLocation map[string]struct{ PackageType, Location string },
	installedDeps map[string]string,
) ([]string, map[string][]string) {
	deps := make(map[string][]string)
	var depsOrdered []string

	var reposURLs []string
	for _, v := range renvLock.R.Repositories {
		reposURLs = append(reposURLs, v.URL)
	}

	readFile := filepath.Join(temporalCacheDirectory, "deps.json")
	if _, err := os.Stat(readFile); err == nil {
		log.Info("Reading", readFile)
		jsonFile, _ := ioutil.ReadFile(readFile)
		json.Unmarshal(jsonFile, &deps)
	} else {
		depsAll := getPackageDeps(renvLock.Packages, renvLock.Bioconductor.Version, reposURLs, packagesLocation)

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
				if len(dep) > 0 {
					deps[p] = dep
				}
			}
		}
		writeJSON(readFile, deps)
	}
	defer os.Remove(readFile)

	readFile = filepath.Join(temporalCacheDirectory, "depsOrdered.json")
	if _, err := os.Stat(readFile); err == nil {
		log.Infof("Reading %s", readFile)
		jsonFile, _ := ioutil.ReadFile(readFile)
		json.Unmarshal(jsonFile, &depsOrdered)
	} else {
		log.Debugf("TSorting %d packages", len(deps))
		depsOrdered = tsort(deps)
		log.Debugf("TSorted %d packages. Ordered %d packages", len(deps), len(depsOrdered))
		writeJSON(readFile, depsOrdered)
	}
	defer os.Remove(readFile)

	depsOrderedToInstall := make([]string, 0)
	for _, packageName := range depsOrdered {
		if _, ok := packagesLocation[packageName]; ok {
			depsOrderedToInstall = append(depsOrderedToInstall, packageName)
		}
	}

	log.Tracef("There are %d packages, but after cleaning it is %d", len(depsOrdered), len(depsOrderedToInstall))
	return depsOrderedToInstall, deps
}

func InstallPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo) {
	mkLibPathDir(temporalLibPath)
	mkLibPathDir(packageLogPath)

	installedDeps := getInstalledPackagesWithVersionWithBaseRPackages([]string{temporalLibPath})
	packagesLocation := make(map[string]struct{ PackageType, Location string })
	for _, v := range *allDownloadInfo {
		packagesLocation[v.PackageName] = struct{ PackageType, Location string }{v.DownloadedPackageType, v.OutputLocation}
	}

	depsOrderedToInstall, deps := getOrderedDependencies(renvLock, packagesLocation, installedDeps)

	installChan := make(chan InstallInfo)
	defer close(installChan)
	installResultChan := make(chan InstallResultInfo)
	defer close(installResultChan)

	for i := range depsOrderedToInstall {
		log.Tracef("Starting installation worker #%d", i)
		go installSinglePackageWorker(installChan, installResultChan)
	}

	installing := make(map[string]bool)
	processed := make(map[string]bool)
	for k := range installedDeps {
		processed[k] = true
	}

	minI := 0
	maxI := 20 // max number of parallel installing workers

	// running packages which have no dependencies
	counter := minI // number of currently installing packages in queue
	for _, v := range depsOrderedToInstall {
		log.Tracef("Checking %s", v)
		_, ok := installedDeps[v]
		if !ok {
			if isDependencyFulfilled(v, deps, installedDeps) {
				counter++
				log.Tracef("Triggering %s", v)
				installing[v] = true
				installChan <- InstallInfo{v, packagesLocation[v].Location}
			}
			if counter >= maxI {
				log.Warnf("All the rest packages have dependencies. Counter:%d", counter)
				break
			}
		}
	}

	log.Tracef("running on channels")
	installResultInfos := make([]InstallResultInfo, 0)
Loop:
	for {
		select {
		case installResultInfo := <-installResultChan:
			installResultInfos = append(installResultInfos, installResultInfo)
			installing[installResultInfo.PackageName] = false
			processed[installResultInfo.PackageName] = true
			installedDeps[installResultInfo.PackageName] = ""
			for i := minI; i <= maxI && i < len(depsOrderedToInstall); i++ {
				nextPackage := depsOrderedToInstall[i]
				if !processed[nextPackage] {
					if !installing[nextPackage] {
						if isDependencyFulfilled(nextPackage, deps, installedDeps) {
							installChan <- InstallInfo{nextPackage, packagesLocation[nextPackage].Location}
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
			log.Tracef("End %s\n minI: %d\n maxI:%d\n installing: %v\n processed:%v", installResultInfo.PackageName, minI, maxI, installing, processed)
		}
	}
	installResultInfosFilePath := filepath.Join(temporalCacheDirectory, "installResultInfos.json")
	log.Tracef("Writing installation status file into %s", installResultInfosFilePath)
	writeJSON(installResultInfosFilePath, installResultInfos)
	log.Infof("Installation for %d is done", len(installResultInfos))
}
