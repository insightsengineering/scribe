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
	res, err := execCommand("R CMD", false, nil, nil)
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

	res, err := execCommand(`R -s --vanilla -e ".libPaths()"`, false, []string{"R_LIBS=/usr/lib/R/library"}, logFile)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)

	content, err := os.ReadFile(filePath)

	fmt.Print(content)
	assert.NotEmpty(t, content)
	assert.Nil(t, err)

}

func Test_toEmptyMapString(t *testing.T) {
	testcases := []struct {
		slice   []string
		mapping map[string]string
	}{}

	mapeq := func(map1 map[string]string, map2 map[string]string) bool {
		if map1 == nil || map2 == nil {
			return false
		}
		if len(map1) != len(map2) {
			return false
		}

		for k, v := range map1 {
			v2, ok := map2[k]
			if !ok {
				return false
			}
			if v != v2 {
				return false
			}
		}

		for k, v := range map2 {
			v1, ok := map1[k]
			if !ok {
				return false
			}
			if v != v1 {
				return false
			}
		}
		return true
	}

	for _, c := range testcases {
		actual := toEmptyMapString(c.slice)

		if !mapeq(c.mapping, actual) {
			t.Fatalf("toEmptyMapString returns wrong value (%v). It should %v", actual, c.mapping)
		}
	}
}

func Test_fillEnvFromSystem(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	envs := fillEnvFromSystem([]string{"LANG"})
	assert.Equal(t, "LANG=en_US.UTF-8", envs[0])
}
