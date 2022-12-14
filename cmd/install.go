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
	"sync"

	//"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

const maxInstallRoutines = 40

const temporalLibPath = "/tmp/scribe/installed_packages" //:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"

const rLibsPaths = "/tmp/scribe/installed_packages:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"

const packageLogPath = "/tmp/scribe/installed_logs"

// for LIB_DIR sys variable
const libDirPath = "/usr/lib/x86_64-linux-gnu/pkgconfig" // /usr/lib/x86_64-linux-gnu/pkgconfig

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

type InstallationInfo struct {
	PackageName           string `json:"packageName"`
	InstallationSucceeded bool   `json:"installationSucceeded"`
}

const (
	InstallationStatusSucceeded = iota
	InstallationStatusSkipped
	InstallationStatusFailed
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

func executeInstallation(outputLocation string, packageName string) error {
	log.Infof("Executing Installation step on package %s located in %s", packageName, outputLocation)
	mkLibPathDir(packageLogPath)
	logFilePath := filepath.Join(packageLogPath, packageName+".out")

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
			"STRINGI_DISABLE_PKG_CONFIG=1",
		}, logFile)
	if err != nil {
		log.Error(cmd)
		log.Errorf("outputLocation:%s packageName:%s\nerr:%v\noutput:%s", outputLocation, packageName, err, output)
	}
	log.Infof("Executed Installation step on package %s located in %s", packageName, outputLocation)
	return err
}

func installSinglePackage(
	outputLocation string,
	packageName string,
	deps map[string][]string,
	installedPackages map[string]string,
	willUnlock map[string][]string,

	message chan InstallationInfo,
	guard chan struct{},
	wgGlobal *sync.WaitGroup,
	wgmap map[string]*sync.WaitGroup,
	mutexWillUnlock *sync.RWMutex,
	mutexInstalled *sync.RWMutex,
	waitList map[string]bool,
) {
	defer wgGlobal.Done()
	log.Tracef("Installing Single Package for package %s", packageName)

	dep := deps[packageName]
	shouldWait := false
	for _, d := range dep {
		mutexInstalled.RLock()
		_, ok := installedPackages[d]
		if !ok {
			mutexWillUnlock.Lock()
			willUnlock[d] = append(willUnlock[d], packageName)
			wg, ok := wgmap[packageName]
			if ok {
				log.Tracef("Raised by %s. Lock for package %s raise by %s. Dependencies on %v. It will unlock %v", d, packageName, d, dep, willUnlock[d])
				wg.Add(1)
				shouldWait = true
			}
			mutexWillUnlock.Unlock()
		}
		mutexInstalled.RUnlock()
	}

	wg, ok := wgmap[packageName]
	if ok && shouldWait {
		log.Debugf("Installation for package %s needs to wait", packageName)
		waitList[packageName] = true
		log.Warnf("wg.Wait() waitList: %v", waitList)
		wg.Wait()
		//waitList[packageName] = false
		delete(waitList, packageName)
	}

	log.Infof("Installing package %s", packageName)

	err := executeInstallation(outputLocation, packageName)
	if err != nil {
		mutexInstalled.Lock()
		installedPackages[packageName] = "v1"
		mutexInstalled.Unlock()
	}

	//installationSucceeded := err != nil
	//message <- InstallationInfo{packageName, installationSucceeded}
	//installedPackages[packageName] = "v1"

	log.Tracef("Package %s has been installed. Now, it will unlock next packages", packageName)
	mutexWillUnlock.RLock()
	unlock, ok := willUnlock[packageName]
	if ok {
		for _, p := range unlock {
			wg, ok := wgmap[p]
			if ok {
				log.Tracef("Unlocking for package %s", p)
				log.Warnf("wg.Done() waitList: %v", waitList)
				wg.Done()
				log.Tracef("Unlocked for package %s", p)
			}
		}
	}
	mutexWillUnlock.RUnlock()
	log.Tracef("Installed Single Package for package %s", packageName)
	<-guard
	log.Warnf("Installed Single Package waitList: %v", waitList)
}

func installResultReceiver(
	message chan InstallationInfo,
	successfulInstallation *int,
	failedInstallation *int,
	totalPackages int,
	allInstallationInfo *[]InstallationInfo,
	installWaiter chan struct{},
) {

	*successfulInstallation = 0
	*failedInstallation = 0
	idleSeconds := 0
	var bar progressbar.ProgressBar
	if interactive {
		bar = *progressbar.Default(
			int64(totalPackages),
			"Installing...",
		)
	}
	const maxIdleSeconds = 20

	for {
		select {
		case msg := <-message:

			if msg.InstallationSucceeded {
				*successfulInstallation++
			} else {
				*failedInstallation++
			}
			idleSeconds = 0
			if interactive {
				err := bar.Add(1)
				checkError(err)
			}

			*allInstallationInfo = append(*allInstallationInfo, msg)

			if *successfulInstallation+*failedInstallation == totalPackages {
				// As soon as we got statuses for all packages we want to return to main routine.
				idleSeconds = maxIdleSeconds
			}
		default:
			time.Sleep(time.Second)
			idleSeconds++
		}
		// Last maxIdleWaits attempts at receiving status from package downloader didn't yield any
		// messages. Or all packages have been downloaded. Hence, we finish waiting for any other statuses.
		if idleSeconds >= maxIdleSeconds {
			break
		}
	}
	// Signal to DownloadPackages function that all downloads have been completed.
	installWaiter <- struct{}{}

}

func InstallPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo) {
	mkLibPathDir(temporalLibPath)

	var installationInfo []InstallationInfo

	packages := make([]string, 0, len(renvLock.Packages))
	for _, p := range renvLock.Packages {
		packages = append(packages, p.Package)
	}

	var reposUrls []string
	for _, v := range renvLock.R.Repositories {
		reposUrls = append(reposUrls, v.URL)
	}

	packagesLocation := make(map[string]struct{ PackageType, Location string })
	for _, v := range *allDownloadInfo {
		packagesLocation[v.PackageName] = struct{ PackageType, Location string }{v.DownloadedPackageType, v.OutputLocation}
	}
	installedDeps := getInstalledPackagesWithVersionWithBaseRPackages([]string{temporalLibPath})

	deps := make(map[string][]string)
	var depsOrdered []string

	readFile := "deps.json"
	if _, err := os.Stat(readFile); err == nil {
		log.Info("Reading", readFile)
		jsonFile, _ := ioutil.ReadFile(readFile)
		json.Unmarshal(jsonFile, &deps)
	} else {
		depsAll := getPackageDeps(packages, renvLock.Bioconductor.Version, allDownloadInfo, reposUrls, packagesLocation)
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

	readFile = "depsOrdered.json"
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

	messages := make(chan InstallationInfo)
	defer close(messages)

	maxDownloadRoutines := 10
	guard := make(chan struct{}, maxDownloadRoutines)
	defer close(guard)

	var successfulDownloads, failedDownloads int
	totalPackages := len(renvLock.Packages)
	installWaiter := make(chan struct{})
	defer close(installWaiter)

	go installResultReceiver(
		messages,
		&successfulDownloads,
		&failedDownloads,
		totalPackages,
		&installationInfo,
		installWaiter,
	)

	wgGlobal := new(sync.WaitGroup)
	wgmap := make(map[string]*sync.WaitGroup)
	depsOrderedToInstall := make([]string, 0)
	for _, packageName := range depsOrdered {
		if _, ok := packagesLocation[packageName]; ok {
			var waitgroup sync.WaitGroup
			wgmap[packageName] = &waitgroup
			depsOrderedToInstall = append(depsOrderedToInstall, packageName)
		}
	}

	var mutexWillUnlock = sync.RWMutex{}
	var mutexInstalled = sync.RWMutex{}
	willUnlock := map[string][]string{}
	waitList := map[string]bool{}

	for i := 0; i < len(depsOrderedToInstall); i++ {
		packageName := depsOrderedToInstall[i]
		log.Tracef("Processing package %s", packageName)

		if val, ok := packagesLocation[packageName]; ok {
			_, ok := installedDeps[packageName]
			if !ok {
				guard <- struct{}{}
				wgGlobal.Add(1)
				go installSinglePackage(val.Location, packageName,
					deps, installedDeps, willUnlock,
					messages, guard,
					wgGlobal, wgmap,
					&mutexWillUnlock, &mutexInstalled,
					waitList,
				)
			} else {
				log.Trace("Package " + packageName + " is already installed")
			}
		}
	}
	wgGlobal.Wait()
	<-installWaiter

	log.Info("Installation is done")
}
