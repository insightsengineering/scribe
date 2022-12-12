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
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

var maxCheckRoutines = 5

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
		if strings.HasPrefix(newLine, "* DONE") {
			log.Debug("Finished processing R CMD output.")
		}
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
	for {
		select {
		case msg := <-messages:
			receivedResults++
			log.Info(msg.PackagePath, " has ", len(msg.Info), " problems.")
			// for _, problem := range msg.Info {
			// 	log.Info(problem.CheckItemType)
			// 	log.Info(problem.CheckItemContent)
			// }
			allPackagesCheckInfo = append(allPackagesCheckInfo, msg)
			if interactive {
				err := bar.Add(1)
				checkError(err)
			}
			writeJSON("allPackagesCheckInfo.json", allPackagesCheckInfo)

			if receivedResults == totalPackages {
				checkWaiter <- struct{}{}
				break
			}
		default:
			// TODO should there be any timeout in case checking some package doesn't complete?
			time.Sleep(time.Second)
		}
	}
}

func runCmdCheck(cmdCheckChan chan string, packagePath string) {
	log.Info("Checking package ", packagePath)
	out, err := exec.Command("R", "CMD", "check", packagePath).CombinedOutput()
	checkError(err)
	cmdCheckChan <- string(out)
}

func checkSinglePackage(messages chan PackageCheckInfo, guard chan struct{},
	packagePath string) {
	cmdCheckChan := make(chan string)
	go runCmdCheck(cmdCheckChan, packagePath)
	var singlePackageCheckInfo []ItemCheckInfo
	var waitInterval = 1
	var totalWaitTime = 0
	var breakOuterLoop = false
	// Wait until R CMD check completes
	for {
		select {
		case msg := <-cmdCheckChan:
			parseCheckOutput(msg, &singlePackageCheckInfo)
			messages <- PackageCheckInfo{packagePath, singlePackageCheckInfo}
			<-guard
			breakOuterLoop = true
			log.Info("R CMD check ", packagePath, " completed after ", totalWaitTime, "s")
			break
		default:
			if totalWaitTime%5 == 0 {
				log.Info("R CMD check ", packagePath, "... [", totalWaitTime, "s elapsed]")
			}
			time.Sleep(time.Duration(waitInterval) * time.Second)
			totalWaitTime += waitInterval
		}
		if breakOuterLoop {
			break
		}
	}
}

// TODO temporary function to run some R CMD checks in parallel
func checkPackages() {
	directoryPath := localOutputDirectory + "/package_archives"
	files, err := ioutil.ReadDir(directoryPath)
	checkError(err)
	// Channel to wait until all checks have completed.
	checkWaiter := make(chan struct{})
	messages := make(chan PackageCheckInfo)
	// Guard channel ensures that only a fixed number of concurrent goroutines are running.
	guard := make(chan struct{}, maxCheckRoutines)

	go checkResultsReceiver(messages, checkWaiter, len(files))
	log.Info("Number of packages to check ", len(files))
	for _, file := range files {
		if !file.IsDir() {
			guard <- struct{}{}
			go checkSinglePackage(messages, guard, directoryPath+"/"+file.Name())
		}
	}

	<-checkWaiter
	log.Info("Finished checking all packages.")
}
