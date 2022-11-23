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


func Test_parsePackagesFile(t *testing.T) {
	packages := make(map[string]*PackageInfo)
	parsePackagesFile("testdata/PACKAGES", packages)
	assert.Equal(t, packages["somePackage1"].Version, "1.0.0")
	assert.Equal(t, packages["somePackage1"].Checksum, "aaa333444555666777")
	assert.Equal(t, packages["somePackage2"].Version, "2.0.0")
	assert.Equal(t, packages["somePackage2"].Checksum, "bbb222333444555666")
	assert.Equal(t, packages["somePackage3"].Version, "0.0.1")
	assert.Equal(t, packages["somePackage3"].Checksum, "ccc000111222333444")
	assert.Equal(t, packages["somePackage4"].Version, "0.2")
	assert.Equal(t, packages["somePackage4"].Checksum, "aaabbbcccdddeeefff")
}
