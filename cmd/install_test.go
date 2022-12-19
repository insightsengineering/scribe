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

func Test_executeInstallation(t *testing.T) {
	t.Skip("skipping integration test")
	err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils", "test.out")
	assert.NoError(t, err)
}

func Test_executeInstallationFromTargz(t *testing.T) {
	t.Skip("skipping integration test")
	cases := []struct{ targz, packageName string }{
		{"testdata/targz/OrdinalLogisticBiplot_0.4.tar.gz", "OrdinalLogisticBiplot"},
		{"testdata/targz/curl_4.3.2.tar.gz", "curl"},
		{"testdata/targz/bitops_1.0-7.tar.gz", "bitops"},
		{"testdata/targz/CompQuadForm_1.4.3.tar.gz", "CompQuadForm"},
		{"testdata/targz/dotCall64_1.0-1.tar.gz", "dotCall64"},
		{"testdata/targz/tripack_1.3-9.1.tar.gz", "tripack"},
	}
	for _, v := range cases {
		err := executeInstallation(v.targz, v.packageName, v.packageName+".out")
		assert.NoError(t, err)
	}
}

func Test_getInstalledPackagesWithVersion(t *testing.T) {
	t.Skip("skipping integration test")
	pkgVer := getInstalledPackagesWithVersion([]string{"/usr/lib/R/site-library"})
	assert.NotEmpty(t, pkgVer)
}
