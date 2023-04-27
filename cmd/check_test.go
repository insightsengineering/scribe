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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseCheckOutput(t *testing.T) {
	var allCheckInfo []ItemCheckInfo
	checkOutput, err := os.ReadFile("testdata/r_cmd_check.txt")
	checkError(err)
	maximumSeverity := parseCheckOutput(string(checkOutput), &allCheckInfo)
	assert.Equal(t, maximumSeverity, "ERROR")
	assert.Equal(t, allCheckInfo[0].CheckItemType, "WARNING")
	assert.Equal(t, allCheckInfo[0].CheckItemContent,
		"* checking Rd metadata ... WARNING\nSome warning 1\n  Some warning 2\n")
	assert.Equal(t, allCheckInfo[1].CheckItemType, "ERROR")
	assert.Equal(t, allCheckInfo[1].CheckItemContent,
		"* checking Rd metadata ... ERROR\n\n\nSome error 7\n  Some error 8\n\n")
	assert.Equal(t, allCheckInfo[2].CheckItemType, "NOTE")
	assert.Equal(t, allCheckInfo[2].CheckItemContent,
		"* checking Rd contents ... NOTE\n  Some note 3\nSome note 4\n\nSome note 5\n\n")
	assert.Equal(t, allCheckInfo[3].CheckItemType, "ERROR")
	assert.Equal(t, allCheckInfo[3].CheckItemContent,
		"* checking for unstated dependencies in ‘tests’ ... ERROR\nSome error 1\n  Some error 2\n\nSome error 3\n")
	assert.Equal(t, allCheckInfo[4].CheckItemType, "WARNING")
	assert.Equal(t, allCheckInfo[4].CheckItemContent,
		"* checking for unstated dependencies in ‘tests’ ... WARNING\n    Some error 4\n  Some error 5\nSome error 6\n")
	assert.Equal(t, allCheckInfo[5].CheckItemType, "ERROR")
	assert.Equal(t, allCheckInfo[5].CheckItemContent,
		"* checking tests ...\n  Running ‘testthat.R’\n ERROR\nRunning the tests in ‘tests/testthat.R’ failed.\n")
}

func Test_getCheckedPackages(t *testing.T) {
	var testRootDir = "testdata/getcheckedpackages"
	err := os.MkdirAll(testRootDir, os.ModePerm)
	checkError(err)
	for _, fileName := range []string{
		"tern_0.0.1.tar.gz",
		"teal_0.0.2.tar.gz",
		"teal.slice_0.0.3.tar.gz",
		"teal.modules.general_1.0.tar.gz",
		"teal.modules.clinical_1.1.tar.gz",
		"teal.reporter_1.2.tar.gz",
		"Teal.Reporter_1.2.3.tar.gz",
		"TERN_1.2.3.4.tar.gz",
	} {
		_, err := os.OpenFile(filepath.Join(testRootDir, fileName), os.O_RDONLY|os.O_CREATE, 0644)
		checkError(err)
	}
	assert.Equal(t,
		getCheckedPackages("", true, testRootDir),
		// All packages returned.
		[]string{
			"TERN_1.2.3.4.tar.gz",
			"Teal.Reporter_1.2.3.tar.gz",
			"teal.modules.clinical_1.1.tar.gz",
			"teal.modules.general_1.0.tar.gz",
			"teal.reporter_1.2.tar.gz",
			"teal.slice_0.0.3.tar.gz",
			"teal_0.0.2.tar.gz",
			"tern_0.0.1.tar.gz",
		})
	assert.Equal(t,
		getCheckedPackages("teal", false, testRootDir),
		[]string{"teal_0.0.2.tar.gz"})
	assert.Equal(t,
		getCheckedPackages("te*", false, testRootDir),
		[]string{
			"teal.modules.clinical_1.1.tar.gz",
			"teal.modules.general_1.0.tar.gz",
			"teal.reporter_1.2.tar.gz",
			"teal.slice_0.0.3.tar.gz",
			"teal_0.0.2.tar.gz",
			"tern_0.0.1.tar.gz",
		})
	assert.Equal(t,
		getCheckedPackages("teal,teal.modules*,TERN", false,
			testRootDir),
		[]string{
			"TERN_1.2.3.4.tar.gz",
			"teal.modules.clinical_1.1.tar.gz",
			"teal.modules.general_1.0.tar.gz",
			"teal_0.0.2.tar.gz",
		})
}
