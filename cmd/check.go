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
	"bufio"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

var maxCheckRoutines = 5
var checkLogPath = "/tmp/scribe/check_logs"

type ItemCheckInfo struct {
	CheckItemType    string
	CheckItemContent string
}

type PackageCheckInfo struct {
	PackagePath string
	Info        []ItemCheckInfo
}

func parseCheckOutput(stringToParse string, singlePackageCheckInfo *[]ItemCheckInfo) {
	scanner := bufio.NewScanner(strings.NewReader(stringToParse))
	var checkItem string
	var previousCheckItem string
	var checkItemType string
	var previousCheckItemType string
	for scanner.Scan() {
		newLine := scanner.Text()
		// New check item.
		// Check items are delimited by a "* " string which occurs at the beginning of a line.
		if strings.HasPrefix(newLine, "* ") {
			previousCheckItem = checkItem
			previousCheckItemType = checkItemType
			trimmedNewLine := strings.TrimSpace(newLine)
			switch {
			case strings.HasSuffix(trimmedNewLine, "... NOTE"):
				checkItemType = "NOTE"
			case strings.HasSuffix(trimmedNewLine, "... WARNING"):
				checkItemType = "WARNING"
			case strings.HasSuffix(trimmedNewLine, "... ERROR"):
				checkItemType = "ERROR"
			default:
				checkItemType = ""
			}
			if previousCheckItemType != "" {
				*singlePackageCheckInfo = append(
					*singlePackageCheckInfo,
					ItemCheckInfo{previousCheckItemType, previousCheckItem},
				)
			}
			checkItem = ""
			checkItem += newLine + "\n"
		} else {
			// Append new line to the currently processed check item.
			checkItem += newLine + "\n"
		}
	}
}

func checkResultsReceiver(messages chan PackageCheckInfo,
	checkWaiter chan struct{}, totalPackages int) {
	var bar progressbar.ProgressBar
	var allPackagesCheckInfo []PackageCheckInfo
	if interactive {
		bar = *progressbar.Default(
			int64(totalPackages),
			"Checking...",
		)
	}
	var receivedResults int
results_receiver_loop:
	for {
		select {
		case msg := <-messages:
			receivedResults++
			log.Info(msg.PackagePath, " has ", len(msg.Info), " problems.")
			for _, problem := range msg.Info {
				log.Debug(
					msg.PackagePath, " has problem of type ", problem.CheckItemType,
					", problem content = ", problem.CheckItemContent,
				)
			}
			allPackagesCheckInfo = append(allPackagesCheckInfo, msg)
			if interactive {
				err := bar.Add(1)
				checkError(err)
			}
			writeJSON(temporalCacheDirectory+"/allPackagesCheckInfo.json", allPackagesCheckInfo)

			if receivedResults == totalPackages {
				checkWaiter <- struct{}{}
				break results_receiver_loop
			}
		default:
			// TODO should there be any timeout in case checking some package doesn't complete?
			time.Sleep(time.Second)
		}
	}
}

func runCmdCheck(cmdCheckChan chan string, packagePath string) {
	log.Info("Checking package ", packagePath)
	packagePathSplit := strings.Split(packagePath, "/")
	packageName := packagePathSplit[len(packagePathSplit)-1]
	logFilePath := checkLogPath + "/" + packageName
	log.Debug("Package ", packageName, " will have check output saved to ", logFilePath, ".")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	checkError(err)
	cmd := "R CMD check --no-lock -l " + temporalLibPath + " " + packagePath
	log.Debug("Executing command: ", cmd)
	output, err := execCommand(cmd, false, false,
		[]string{
			"R_LIBS=" + rLibsPaths,
			"LANG=en_US.UTF-8",
		}, logFile)
	checkError(err)
	cmdCheckChan <- output
}

func checkSinglePackage(messages chan PackageCheckInfo, guard chan struct{},
	packagePath string) {
	cmdCheckChan := make(chan string)
	go runCmdCheck(cmdCheckChan, packagePath)
	var singlePackageCheckInfo []ItemCheckInfo
	var waitInterval = 1
	var totalWaitTime = 0
	// Wait until R CMD check completes
check_single_package_loop:
	for {
		select {
		case msg := <-cmdCheckChan:
			parseCheckOutput(msg, &singlePackageCheckInfo)
			messages <- PackageCheckInfo{packagePath, singlePackageCheckInfo}
			<-guard
			log.Info("R CMD check ", packagePath, " completed after ", totalWaitTime, "s")
			break check_single_package_loop
		default:
			if totalWaitTime%5 == 0 {
				log.Info("R CMD check ", packagePath, "... [", totalWaitTime, "s elapsed]")
			}
			time.Sleep(time.Duration(waitInterval) * time.Second)
			totalWaitTime += waitInterval
		}
	}
}

// Returns list of paths to directories with installed packages, on which R CMD check should
// be performed based on the wildcard expression from command line.
func getCheckedPackages(checkExpression string, installationDirectory string) []string {
	var checkPackageDirectories []string
	var checkRegexp string
	if checkExpression == "" {
		checkRegexp = ".*"
	} else {
		splitCheckRegexp := strings.Split(checkExpression, ",")
		var allRegExpressions []string
		// For each comma-separated wildcard expression convert "." and "*"
		// characters to regexp equivalent.
		for _, singleRegexp := range splitCheckRegexp {
			singleRegexp = strings.ReplaceAll(singleRegexp, `.`, `\.`)
			singleRegexp = strings.ReplaceAll(singleRegexp, "*", ".*")
			allRegExpressions = append(allRegExpressions, "^"+singleRegexp+"$")
		}
		checkRegexp = strings.Join(allRegExpressions, "|")
	}
	log.Info("R CMD check will be performed on packages matching regexp ", checkRegexp)
	files, err := os.ReadDir(installationDirectory)
	checkError(err)
	for _, file := range files {
		if file.IsDir() {
			match, err := regexp.MatchString(checkRegexp, file.Name())
			checkError(err)
			if match {
				log.Debug(file.Name() + " matches regexp " + checkRegexp)
				checkPackageDirectories = append(
					checkPackageDirectories,
					installationDirectory+"/"+file.Name(),
				)
			} else {
				log.Debug(file.Name() + " doesn't match regexp " + checkRegexp)
			}
		}
	}
	return checkPackageDirectories
}

// func checkPackages(installResults []InstallResultInfo) {
// TODO remove
func checkPackages() {
	err := os.MkdirAll(checkLogPath, os.ModePerm)
	checkError(err)
	checkPackagesDirectories := getCheckedPackages(checkPackageExpression, temporalLibPath)
	// Channel to wait until all checks have completed.
	checkWaiter := make(chan struct{})
	messages := make(chan PackageCheckInfo)
	// Guard channel ensures that only a fixed number of concurrent goroutines are running.
	guard := make(chan struct{}, maxCheckRoutines)

	go checkResultsReceiver(messages, checkWaiter, len(checkPackagesDirectories))
	log.Info("Number of packages to check ", len(checkPackagesDirectories))
	for _, directory := range checkPackagesDirectories {
		guard <- struct{}{}
		go checkSinglePackage(messages, guard, directory)
	}

	<-checkWaiter
	log.Info("Finished checking all packages.")
}
