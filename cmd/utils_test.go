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
}

func Test_execCommand(t *testing.T) {
	res, err := execCommand("R CMD", true, false, nil)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)
}

func Test_execCommandWithEnvs(t *testing.T) {

	res, err := execCommand(`R -s --vanilla -e ".libPaths()"`, true, false, []string{"R_LIBS=/usr/lib/R/library"})
	assert.NotEmpty(t, res)
	assert.Nil(t, err)
}
