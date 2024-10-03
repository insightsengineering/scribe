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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_executeInstallation(t *testing.T) {
	t.Skip("skipping integration test")
	_, err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils", "test.out", "build-test.out", "tar.gz", "--no-manual", "--no-docs")
	assert.NoError(t, err)
}

func Test_executeInstallation_with_wrong_logFilePath(t *testing.T) {
	_, err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils", "", "", "tar.gz", "--no-manual", "--no-docs")
	assert.Error(t, err)
}

func Test_executeInstallation_with_wrong_path_to_package(t *testing.T) {
	_, err := executeInstallation("", "BiocBaseUtils", "test.out", "build-test.out", "tar.gz", "--no-manual", "--no-docs")
	assert.Error(t, err)
}

func Test_executeInstallationFromTargz(t *testing.T) {
	err := os.MkdirAll("testdata/targz", os.ModePerm)
	checkError(err)
	rExecutable = "R"
	temporaryLibPath = "/tmp/scribe/installed_packages"
	rLibsPaths = "/tmp/scribe/installed_packages:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"
	localOutputDirectory = defaultDownloadDirectory
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
		_, err := executeInstallation(v.targz, v.packageName, v.packageName+".out", "build-"+v.packageName+".out", "tar.gz", "--no-manual", "--no-docs")
		assert.NoError(t, err)
	}
}

func Test_getBuiltPackageFileName(t *testing.T) {
	for _, fileName := range []string{
		"tern_0.0.1.tar.gz",
		"teal_0.0.2.tar.gz",
		"teal.slice_0.0.3.tar.gz",
		"teal.modules.general_1.0.tar.gz",
		"teal.modules.clinical_1.1.tar.gz",
		"teal.reporter_1.2.tar.gz",
		"Teal.Reporter_1.2.3.tar.gz",
		"TERN_1.2.3.4.tar.gz",
		"tern_0.0.1",
		"teal_0.0.2",
		"teal.slice_0.0.3",
		"teal.modules.general_1.0",
		"teal.modules.clinical_1.1",
		"teal.reporter_1.2",
		"Teal.Reporter_1.2.3",
		"TERN_1.2.3.4",
	} {
		_, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0644)
		checkError(err)
	}
	assert.Equal(t, getBuiltPackageFileName("tern"), "tern_0.0.1.tar.gz")
	assert.Equal(t, getBuiltPackageFileName("teal"), "teal_0.0.2.tar.gz")
	assert.Equal(t, getBuiltPackageFileName("TERN"), "TERN_1.2.3.4.tar.gz")
	assert.Equal(t, getBuiltPackageFileName("teal.modules.clinical"), "teal.modules.clinical_1.1.tar.gz")
}

func Test_mapTrueLength(t *testing.T) {
	m := make(map[string]bool)
	m["test1"] = true
	m["test2"] = false
	m["test3"] = true
	m["test4"] = false
	m["test5"] = true
	assert.Equal(t, mapTrueLength(m), 3)
}

func Test_getPackageToInstall(t *testing.T) {
	packagesBeingInstalled := make(map[string]bool)
	readyPackages := make(map[string]bool)
	packagesBeingInstalled["package1"] = true
	packagesBeingInstalled["package2"] = false
	packagesBeingInstalled["package3"] = true
	readyPackages["package4"] = false
	readyPackages["package5"] = false
	readyPackages["package6"] = true
	packageName := getPackageToInstall(packagesBeingInstalled, readyPackages)
	assert.Equal(t, packageName, "package6")
	assert.True(t, packagesBeingInstalled["package6"])
	assert.False(t, readyPackages["package6"])
	packageName = getPackageToInstall(packagesBeingInstalled, readyPackages)
	assert.Equal(t, packageName, "")
}

func Test_getPackagesReadyToInstall(t *testing.T) {
	dependencies := make(map[string][]string)
	var installedPackages []string
	packagesBeingInstalled := make(map[string]bool)
	readyPackages := make(map[string]bool)
	dependencies["package1"] = []string{"package2", "package3", "package4"}
	dependencies["package2"] = []string{}
	dependencies["package3"] = []string{"package2"}
	dependencies["package4"] = []string{"package5", "package3"}
	dependencies["package5"] = []string{"package3"}
	installedPackages = append(installedPackages, "package2")
	getPackagesReadyToInstall(dependencies, installedPackages, packagesBeingInstalled, readyPackages)
	assert.True(t, readyPackages["package3"])
	_, ok := readyPackages["package4"]
	assert.False(t, ok)
	_, ok = readyPackages["package5"]
	assert.False(t, ok)
	_, ok = readyPackages["package1"]
	assert.False(t, ok)
}
