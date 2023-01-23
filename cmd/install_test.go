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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getInstalledPackagesWithVersionWithBaseRPackages(t *testing.T) {
	pkgs := getInstalledPackagesWithVersionWithBaseRPackages([]string{})
	basePkgs := []string{"stats", "graphics", "grDevices", "utils", "datasets", "methods", "base"}
	for i := 0; i < len(basePkgs); i++ {
		pkg := basePkgs[i]
		assert.Contains(t, pkgs, pkg)
	}
}

func Test_mkLibPathDir(t *testing.T) {
	dirs := []string{"/tmp/scribe/testdir1", "/tmp/scribe/testdir2"}
	dirsCon := ""
	for _, d := range dirs {
		os.RemoveAll(d)
		dirsCon += ":" + d
		assert.NoDirExists(t, d)
	}

	assert.NotEmpty(t, dirsCon)
	mkLibPathDir(dirsCon)
	for _, d := range dirs {
		assert.DirExists(t, d)
		os.RemoveAll(d)
	}
}

func Test_executeInstallation(t *testing.T) {
	t.Skip("skipping integration test")
	err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils", "test.out", "build-test.out", "tar.gz")
	assert.NoError(t, err)
}

func Test_executeInstallation_with_wrong_logFilePath(t *testing.T) {
	err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils", "", "", "tar.gz")
	assert.Error(t, err)
}

func Test_executeInstallation_with_wrong_path_to_package(t *testing.T) {
	err := executeInstallation("", "BiocBaseUtils", "test.out", "build-test.out", "tar.gz")
	assert.Error(t, err)
}

func Test_executeInstallationFromTargz(t *testing.T) {
	err := os.MkdirAll("testdata/targz", os.ModePerm)
	checkError(err)
	downloadFile(
		"https://cran.r-project.org/src/contrib/Archive/bitops/bitops_1.0-6.tar.gz",
		"testdata/targz/bitops_1.0-6.tar.gz",
	)
	downloadFile(
		"https://cran.r-project.org/src/contrib/Archive/CompQuadForm/CompQuadForm_1.4.2.tar.gz",
		"testdata/targz/CompQuadForm_1.4.2.tar.gz",
	)
	downloadFile(
		"https://cran.r-project.org/src/contrib/Archive/tripack/tripack_1.3-9.tar.gz",
		"testdata/targz/tripack_1.3-9.tar.gz",
	)
	cases := []struct{ targz, packageName string }{
		{"testdata/targz/bitops_1.0-6.tar.gz", "bitops"},
		{"testdata/targz/CompQuadForm_1.4.2.tar.gz", "CompQuadForm"},
		{"testdata/targz/tripack_1.3-9.tar.gz", "tripack"},
	}
	for _, v := range cases {
		err := executeInstallation(v.targz, v.packageName, v.packageName+".out", "build-"+v.packageName+".out", "tar.gz")
		assert.NoError(t, err)
	}
}

func Test_getInstalledPackagesWithVersion(t *testing.T) {
	t.Skip("skipping integration test")
	pkgVer := getInstalledPackagesWithVersion([]string{"/usr/lib/R/site-library"})
	assert.NotEmpty(t, pkgVer)
}
