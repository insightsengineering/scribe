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
	"strings"
)

type CheckInfo struct {
	CheckItemType    string
	CheckItemContent string
}

func parseCheckOutput(stringToParse string, allCheckInfo *[]CheckInfo) {
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
		// new check item
		if strings.HasPrefix(newLine, "* ") {
			previousCheckItem = checkItem
			previousCheckItemType = checkItemType
			trimmedNewLine := strings.TrimSpace(newLine)
			if strings.HasSuffix(trimmedNewLine, "... NOTE") {
				checkItemType = "NOTE"
			} else if strings.HasSuffix(trimmedNewLine, "... WARNING") {
				checkItemType = "WARNING"
			} else if strings.HasSuffix(trimmedNewLine, "... ERROR") {
				checkItemType = "ERROR"
			} else {
				checkItemType = ""
			}
			if previousCheckItemType != "" {
				*allCheckInfo = append(
					*allCheckInfo,
					CheckInfo{previousCheckItemType, previousCheckItem},
				)
			}
			checkItem = ""
			checkItem += newLine + "\n"
		} else {
			checkItem += newLine + "\n"
		}
	}
}
