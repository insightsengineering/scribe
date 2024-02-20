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
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var checkLogPath = "/tmp/scribe/check_logs"

const errConst = "ERROR"
const warnConst = "WARNING"
const noteConst = "NOTE"

type ItemCheckInfo struct {
	CheckItemType    string // NOTE, WARNING or ERROR
	CheckItemContent string // content of NOTE, WARNING or ERROR
}

type PackageCheckInfo struct {
	PackagePath         string // path to directory where the package has been installed
	PackageName         string
	LogFilePath         string // path to the file containing log of R CMD check for the package
	MostSevereCheckItem string // OK, NOTE, WARNING or ERROR
	Info                []ItemCheckInfo
	CheckTime           int
	ShouldFail          bool // Whether a NOTE or WARNING occurred that would cause the check to fail.
}

// getNewMaximumSeverity checks if checkItemType is more severe than currently most severe (mostSevereCheckItem).
// If yes, returns new one, otherwise returns previously most severe.
func getNewMaximumSeverity(checkItemType string, mostSevereCheckItem string) string {
	newMostSevereCheckItem := mostSevereCheckItem
	switch {
	case checkItemType == noteConst && mostSevereCheckItem == "OK":
		newMostSevereCheckItem = noteConst
	case checkItemType == warnConst &&
		(mostSevereCheckItem == "OK" || mostSevereCheckItem == noteConst):
		newMostSevereCheckItem = warnConst
	case checkItemType == errConst &&
		(mostSevereCheckItem == "OK" || mostSevereCheckItem == noteConst || mostSevereCheckItem == warnConst):
		newMostSevereCheckItem = errConst
	}
	return newMostSevereCheckItem
}

func checkIfShouldFail(checkItemType string, checkItem string, shouldFail *bool, packageName string) {
	if rCmdCheckFailRegex == "" {
		return
	}
	match, err := regexp.MatchString(rCmdCheckFailRegex, checkItem)
	checkError(err)
	if match {
		log.Debug(checkItem, " matches ", rCmdCheckFailRegex)
	} else {
		log.Debug(checkItem, " doesn't match ", rCmdCheckFailRegex)
	}
	if match && (checkItemType == "WARNING" || checkItemType == "NOTE") {
		log.Warn(
			"The following ", checkItemType, " encountered while checking package ",
			packageName, " matches regex \"", rCmdCheckFailRegex, "\" and will cause the",
			" check to fail: ", checkItem,
		)
		*shouldFail = true
	}
}

// parseCheckOutput parses output of R CMD check and extracts separate NOTEs, WARNINGs, and ERRORs.
// Returns most severe of statuses found (OK, NOTE, WARNING, ERROR).
func parseCheckOutput(stringToParse string, singlePackageCheckInfo *[]ItemCheckInfo, packageName string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(stringToParse))
	var checkItem string
	var previousCheckItem string
	var checkItemType string
	var previousCheckItemType string
	continuationOnNextLine := false
	// Whether a NOTE or a WARNING occurred that would cause the R CMD check to fail.
	shouldFail := false
	mostSevereCheckItem := "OK"
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
				checkItemType = noteConst
			case strings.HasSuffix(trimmedNewLine, "... WARNING"):
				checkItemType = warnConst
			case strings.HasSuffix(trimmedNewLine, "... ERROR"):
				checkItemType = errConst
			// Exceptionally, it may happen that the line will end with "..."
			// and the continuation of check item title will span subsequent lines.
			case strings.HasSuffix(trimmedNewLine, "..."):
				continuationOnNextLine = true
			default:
				checkItemType = ""
			}
			if !continuationOnNextLine {
				mostSevereCheckItem = getNewMaximumSeverity(checkItemType, mostSevereCheckItem)
			}
			if previousCheckItemType != "" {
				checkIfShouldFail(previousCheckItemType, previousCheckItem, &shouldFail, packageName)
				*singlePackageCheckInfo = append(
					*singlePackageCheckInfo,
					ItemCheckInfo{previousCheckItemType, previousCheckItem},
				)
			}
			checkItem = ""
			checkItem += newLine + " "
		} else {
			if continuationOnNextLine {
				// If the check item title spans multiple lines, we expect it
				// to end with one of these strings at the end of the line.
				trimmedNewLine := strings.TrimSpace(newLine)
				switch {
				case strings.HasSuffix(trimmedNewLine, "NOTE"):
					checkItemType = noteConst
					continuationOnNextLine = false
				case strings.HasSuffix(trimmedNewLine, "WARNING"):
					checkItemType = warnConst
					continuationOnNextLine = false
				case strings.HasSuffix(trimmedNewLine, "ERROR"):
					checkItemType = errConst
					continuationOnNextLine = false
				case strings.HasSuffix(trimmedNewLine, "OK"):
					checkItemType = ""
					continuationOnNextLine = false
				}
			}
			// Once we find the type of check item, we compare its severity
			// with existing items.
			if !continuationOnNextLine {
				mostSevereCheckItem = getNewMaximumSeverity(checkItemType, mostSevereCheckItem)
			}
			// Append new line to the currently processed check item.
			checkItem += newLine + " "
		}
	}
	return mostSevereCheckItem, shouldFail
}

// checkResultsReceiver receives data from go routines performing R CMD checks on the packages.
func checkResultsReceiver(messages chan PackageCheckInfo,
	checkWaiter chan struct{}, totalPackages int, outputFile string) {
	var allPackagesCheckInfo []PackageCheckInfo
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
					", problem content: \n", problem.CheckItemContent,
				)
			}
			allPackagesCheckInfo = append(allPackagesCheckInfo, msg)
			writeJSON(outputFile, allPackagesCheckInfo)

			if receivedResults == totalPackages {
				checkWaiter <- struct{}{}
				break results_receiver_loop
			}
		default:
			time.Sleep(time.Second)
		}
	}
}

// runCmdCheck executes R CMD check for a given package in a goroutine.
func runCmdCheck(cmdCheckChan chan string, packageFile string, logFilePath string,
	additionalOptions string) {
	log.Trace("Check logs/outputs will be saved to ", logFilePath, ".")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	checkError(err)
	defer logFile.Close()
	// Add HTML tags to highlight logs
	if _, createHTMLTagsErr := logFile.Write([]byte("<pre><code>\n")); createHTMLTagsErr != nil {
		log.Error("Error saving log to ", logFilePath, ": ", createHTMLTagsErr)
		cmdCheckChan <- ""
		return
	}
	cmd := rExecutable + " CMD check " + additionalOptions + " " + packageFile
	log.Debug("Executing command: ", cmd)
	output, err := execCommand(cmd, false,
		[]string{rLibsVarName + rLibsPaths, "LANG=en_US.UTF-8"}, logFile, true)
	checkError(err)
	// Close HTML tags
	if _, closeHTMLTagsErr := logFile.Write([]byte("\n</code></pre>\n")); closeHTMLTagsErr != nil {
		log.Error("Error saving log to ", logFilePath, ": ", closeHTMLTagsErr)
		cmdCheckChan <- ""
		return
	}
	cmdCheckChan <- output
}

// checkSinglePackage runs the goroutine with R CMD check for a single package, and waits for its completion.
func checkSinglePackage(messages chan PackageCheckInfo, guard chan struct{},
	packageFile string, additionalOptions string) {
	cmdCheckChan := make(chan string)
	packageName := strings.Split(packageFile, "_")[0]
	logFilePath := checkLogPath + "/" + packageName + htmlExtension
	go runCmdCheck(cmdCheckChan, packageFile, logFilePath, additionalOptions)
	var singlePackageCheckInfo []ItemCheckInfo
	var waitInterval = 1
	var totalWaitTime = 0
	// Wait for message from runCmdCheck (until R CMD check completes).
check_single_package_loop:
	for {
		select {
		case msg := <-cmdCheckChan:
			mostSevereCheckItem, shouldFail := parseCheckOutput(msg, &singlePackageCheckInfo, packageName)
			messages <- PackageCheckInfo{packageFile, packageName, logFilePath,
				mostSevereCheckItem, singlePackageCheckInfo, totalWaitTime, shouldFail}
			<-guard
			log.Info("R CMD check ", packageFile, " completed after ", getTimeMinutesAndSeconds(totalWaitTime))
			break check_single_package_loop
		default:
			if totalWaitTime%30 == 0 && totalWaitTime > 0 {
				log.Info("R CMD check ", packageFile, "... [", getTimeMinutesAndSeconds(totalWaitTime), " elapsed]")
			}
			time.Sleep(time.Duration(waitInterval) * time.Second)
			totalWaitTime += waitInterval
		}
	}
}

// getCheckedPackages returns list of package names coming from tarballs with built packages.
// The packages are filtered based on the wildcard expression from command line.
// R CMD check should be performed on the returned list of packages.
func getCheckedPackages(checkExpression string, checkAllPackages bool, builtPackagesDirectory string) []string {
	var checkPackageFiles []string
	var checkRegexp string
	switch {
	case checkExpression == "" && checkAllPackages:
		checkRegexp = `.*\.tar\.gz$`
	case checkExpression == "" && !checkAllPackages:
		// No packages are checked unless explicitly specified.
		return checkPackageFiles
	default:
		splitCheckRegexp := strings.Split(checkExpression, ",")
		var allRegExpressions []string
		// For each comma-separated wildcard expression convert "." and "*"
		// characters to regexp equivalent.
		for _, singleRegexp := range splitCheckRegexp {
			singleRegexp = strings.ReplaceAll(singleRegexp, `.`, `\.`)
			singleRegexp = strings.ReplaceAll(singleRegexp, "*", ".*")
			allRegExpressions = append(allRegExpressions, "^"+singleRegexp+`\_.*\.tar\.gz$`)
		}
		checkRegexp = strings.Join(allRegExpressions, "|")
	}
	log.Info("R CMD check will be performed on packages matching regexp ", checkRegexp)
	files, err := os.ReadDir(builtPackagesDirectory)
	checkError(err)
	for _, file := range files {
		if !file.IsDir() {
			fileName := file.Name()
			// Matching packageName_packageVersion.tar.gz
			match, err := regexp.MatchString(checkRegexp, fileName)
			checkError(err)
			if match {
				log.Trace(fileName + " matches regexp " + checkRegexp)
				checkPackageFiles = append(checkPackageFiles, fileName)
			} else {
				log.Trace(fileName + " doesn't match regexp " + checkRegexp)
			}
		}
	}
	sort.Strings(checkPackageFiles)
	return checkPackageFiles
}

func checkPackages(outputFile string, additionalOptions string) {
	err := os.MkdirAll(checkLogPath, os.ModePerm)
	checkError(err)
	// Built packages are stored in current directory.
	// Check component assumes that tar.gz packages which should be checked
	// have been previously built and saved to current working directory.
	checkPackagesFiles := getCheckedPackages(checkPackageExpression, checkAllPackages, ".")
	// Channel to wait until all checks have completed.
	checkWaiter := make(chan struct{})

	messages := make(chan PackageCheckInfo)
	// Guard channel ensures that only a fixed number of concurrent goroutines are running.
	guard := make(chan struct{}, maxCheckRoutines)

	systemMetricsWaiter := make(chan struct{})
	go systemMetricsRoutine(systemMetricsWaiter)

	log.Info("Number of packages to check: ", len(checkPackagesFiles))
	if len(checkPackagesFiles) > 0 {
		go checkResultsReceiver(messages, checkWaiter, len(checkPackagesFiles), outputFile)
		for _, packageFile := range checkPackagesFiles {
			guard <- struct{}{}
			go checkSinglePackage(messages, guard, packageFile, additionalOptions)
		}
		<-checkWaiter
	}
	systemMetricsWaiter <- struct{}{}
	// Wait until the metrics routine finishes saving the output files.
	<-systemMetricsWaiter
	log.Info("Finished checking all packages.")
}
