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

func Test_GetRenvLock(t *testing.T) {
	var renvLock Renvlock
	GetRenvLock("testdata/renv.lock.empty.json", &renvLock)
	assert.NotNil(t, renvLock)
	assert.Equal(t, renvLock.R.Version, "4.1.1")
	assert.Equal(t, renvLock.Bioconductor.Version, "3.13")
	assert.Equal(t, renvLock.R.Repositories[0].Name, "CRAN")
	assert.Equal(t, renvLock.R.Repositories[0].URL, "https://cloud.r-project.org")
	assert.Equal(t, renvLock.Packages["SomePackage"].Package, "SomePackage")
	assert.Equal(t, renvLock.Packages["SomePackage"].Version, "1.0.0")
	assert.Equal(t, renvLock.Packages["SomePackage"].Source, "Repository")
	assert.Equal(t, renvLock.Packages["SomePackage"].Repository, "CRAN")
	assert.Equal(t, renvLock.Packages["SomeOtherPackage"].Package, "SomeOtherPackage")
	assert.Equal(t, renvLock.Packages["SomeOtherPackage"].Version, "2.0.0")
	assert.Equal(t, renvLock.Packages["SomeOtherPackage"].Source, "GitHub")
	assert.Equal(t, renvLock.Packages["SomeOtherPackage"].RemoteType, "github")
	assert.Equal(t, renvLock.Packages["SomeOtherPackage"].RemoteHost, "api.github.com")
	assert.Equal(t, renvLock.Packages["SomeOtherPackage"].RemoteUsername, "RemoteUsername")
}

func Test_ValidateRenvLock(t *testing.T) {
	var renvLock Renvlock
	GetRenvLock("testdata/renv.lock.empty.json", &renvLock)
	numberOfWarnings := ValidateRenvLock(renvLock)
	assert.Equal(t, numberOfWarnings, 0)
}
