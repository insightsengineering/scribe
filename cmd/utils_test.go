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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_stringInSlice(t *testing.T) {
	var testSlice = []string{"a", "b", "c", "d"}
	assert.True(t, true, stringInSlice("a", testSlice))
}

func Test_writeJSON(t *testing.T) {
	var renvLock Renvlock
	getRenvLock("testdata/renv.lock.empty.json", &renvLock)
	numberOfBytes := writeJSON("testdata/test_output.json", renvLock)
	assert.Greater(t, numberOfBytes, 0)
	os.Remove("testdata/test_output.json")
}

func Test_getTimeMinutesAndSeconds(t *testing.T) {
	assert.Equal(t, getTimeMinutesAndSeconds(30), "30s")
	assert.Equal(t, getTimeMinutesAndSeconds(80), "1m20s")
}

func Test_execCommand(t *testing.T) {
	t.Skip("skipping integration test")
	res, err := execCommand("R CMD", false, nil, nil, false)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)
}

func Test_execCommandWithEnvs(t *testing.T) {
	t.Skip("skipping integration test")
	filePath := "Test_execCommandWithEnvs.log"
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		os.Remove(filePath)
	}

	logFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	assert.Nil(t, err)
	defer logFile.Close()

	res, err := execCommand(`R -s --vanilla -e ".libPaths()"`, false, []string{"R_LIBS=/usr/lib/R/library"}, logFile, false)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)

	content, err := os.ReadFile(filePath)

	fmt.Print(content)
	assert.NotEmpty(t, content)
	assert.Nil(t, err)

}

func Test_fillEnvFromSystem(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	envs := fillEnvFromSystem([]string{"LANG"})
	assert.Equal(t, "LANG=en_US.UTF-8", envs[0])
}

func Test_parseDescriptionFile(t *testing.T) {
	cases := []struct {
		filename   string
		field      string
		fieldValue string
		extracted  []string
	}{
		{"testdata/DESCRIPTION/NominalLogisticBiplot.txt", "Depends", "R (>= 2.15.1),mirt,gmodels,MASS", []string{"R", "mirt", "gmodels", "MASS"}},
		{"testdata/DESCRIPTION/RcppNumerical.txt", "LinkingTo", "Rcpp, RcppEigen", []string{"Rcpp", "RcppEigen"}},
	}
	for _, c := range cases {
		kv := parseDescriptionFile(c.filename)
		assert.Equal(t, c.fieldValue, kv[c.field])
	}
}
