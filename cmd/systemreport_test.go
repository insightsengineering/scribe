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
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func randString(n int) string {
	const letterBytes string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// set some environment variables with random names and then make sure
// they are masked properly i.e. not returned by the tested function
// it doesn't matter what other environment variables are set in the
// testing environment
func Test_getEnvironmentVariables(t *testing.T) {
	envVar1 := randString(18)
	envVar2 := randString(18)
	envVar3 := randString(18)
	os.Setenv(envVar1, "whatever")
	os.Setenv(envVar2, "whatever")
	os.Setenv(envVar2, "whatever")
	envVars := getEnvironmentVariables(fmt.Sprintf("%s|%s|%s", envVar1, envVar2, envVar3))
	assert.Equal(t, strings.Contains(envVars, envVar1), false)
	assert.Equal(t, strings.Contains(envVars, envVar2), false)
	assert.Equal(t, strings.Contains(envVars, envVar2), false)
}
