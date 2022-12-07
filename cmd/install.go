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
	"fmt"
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
const libDirPath = "/usr/lib/x86_64-linux-gnu/pkgconfig/"

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

func executeInstallation(outputLocation string, packageName string) error {
	log.Tracef("Package location is %s", outputLocation)
	mkLibPathDir(packageLogPath)
	logFilePath := filepath.Join(packageLogPath, packageName+".out")

	logFile, logFileErr := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	defer logFile.Close()
	if logFileErr != nil {
		log.Errorf("outputLocation:%s packageName:%s err:%v", outputLocation, packageName, logFileErr)
		return logFileErr
	}

	cmd := "R CMD INSTALL --install-tests --configure-vars='LIB_DIR=" + libDirPath + "' -l " + temporalLibPath + " " + outputLocation
	log.Debug(cmd)
	_, err := execCommand(cmd, false, false,
		[]string{
			"R_LIBS=" + rLibsPaths,
			"LANG=en_US.UTF-8",
			"LD_LIBRARY_PATH=/usr/lib/R/lib:/usr/lib/gcc/x86_64-linux-gnu/7:/usr/lib/x86_64-linux-gnu:/lib:/usr/lib/x86_64-linux-gnu:/usr/lib/jvm/default-java/lib/server",
			"R_INCLUDE_DIR=/usr/share/R/include",
			"R_LIBS_SITE=/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library",
			"R_LIBS_USER=~/R/x86_64-pc-linux-gnu-library/3.4",
		}, logFile)
	if err != nil {
		log.Errorf("outputLocation:%s packageName:%s err:%v", outputLocation, packageName, err)
	}
	return err
}

func installSinglePackage(
	outputLocation string,
	packageName string,
	message chan InstallationInfo,
	guard chan struct{},
) {

	err := executeInstallation(outputLocation, packageName)
	installationSucceeded := err != nil
	message <- InstallationInfo{packageName, installationSucceeded}
	<-guard
}

func mkLibPathDir(temporalLibPath string) {
	for _, libPath := range strings.Split(temporalLibPath, ":") {
		if _, err := os.Stat(libPath); os.IsNotExist(err) {
			err := os.MkdirAll(libPath, os.ModePerm)
			checkError(err)
			log.Tracef("Created dir %s", libPath)
		}
	}
}

type InstallationInfo struct {
	PackageName           string `json:"packageName"`
	InstallationSucceeded bool   `json:"installationSucceeded"`
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
		// Last maxIdleWaits attempts at receiving status from package downloaders didn't yield any
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

	var deps map[string][]string
	var depsOrdered []string

	readFile := "deps.json"
	if _, err := os.Stat(readFile); err == nil {
		log.Info("Reading", readFile)
		jsonFile, _ := ioutil.ReadFile(readFile)
		json.Unmarshal(jsonFile, &deps)
	} else {
		deps = getPackageDeps(packages, renvLock.Bioconductor.Version, allDownloadInfo, reposUrls, packagesLocation)
		writeJSON(readFile, deps)
	}

	readFile = "depsOrdered.json"
	if _, err := os.Stat(readFile); err == nil {
		log.Info("Reading", readFile)
		jsonFile, _ := ioutil.ReadFile(readFile)
		json.Unmarshal(jsonFile, &depsOrdered)
	} else {
		depsOrdered = tsort(deps)
		writeJSON(readFile, depsOrdered)
	}

	messages := make(chan InstallationInfo)
	maxDownloadRoutines := 10
	guard := make(chan struct{}, maxDownloadRoutines)

	var successfulDownloads, failedDownloads int
	totalPackages := len(renvLock.Packages)
	installWaiter := make(chan struct{})

	go installResultReceiver(
		messages,
		&successfulDownloads,
		&failedDownloads,
		totalPackages,
		&installationInfo,
		installWaiter,
	)

	for i := 0; i < len(depsOrdered); i++ {
		packageName := depsOrdered[i]
		fmt.Print(packageName + " ")
		if val, ok := packagesLocation[packageName]; ok {
			guard <- struct{}{}
			go installSinglePackage(val.Location, packageName, messages, guard)
		}
	}
	<-installWaiter

	log.Info("Installation is done")
}
