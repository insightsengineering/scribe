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

func Test_getPackageDetails(t *testing.T) {
	packageInfo := make(map[string]*PackageInfo)
	biocPackageInfo := make(map[string]map[string]*PackageInfo)
	for _, biocCategory := range bioconductorCategories {
		biocPackageInfo[biocCategory] = make(map[string]*PackageInfo)
	}
	biocUrls := make(map[string]string)
	localArchiveChecksums := make(map[string]*CacheInfo)
	getBiocUrls("3.13", biocUrls)

	// package1 is downloaded neither from CRAN nor from BioConductor - therefore isn't not added to any structure
	// somePackage1 is cached
	packageInfo["somePackage1"] = &PackageInfo{"1.0.0", "aaabbbccc"}
	localArchiveChecksums["aaabbbccc"] = &CacheInfo{"/tmp/scribe/somePackage1_1.0.0.tar.gz", 1000}
	// somePackage2 should be downloaded from CRAN current (we're not adding it to cache)
	packageInfo["somePackage2"] = &PackageInfo{"2.0.0", "abcdef012"}
	// somePackage3 should be downloaded from CRAN Archive - therefore it's not added to packageInfo
	// someBiocPackage1 is cached
	localArchiveChecksums["bcdef0123"] = &CacheInfo{"/tmp/scribe/someBiocPackage_1.0.1.tar.gz", 2000}
	biocPackageInfo["data/experiment"]["someBiocPackage1"] = &PackageInfo{"1.0.1", "bcdef0123"}
	// someBiocPackage2 should be downloaded from BioConductor (we're not adding it to cache)
	biocPackageInfo["workflows"]["someBiocPackage2"] = &PackageInfo{"2.0.1", "bbbcccddd"}

	action, packageURL, outputLocation, savedBandwidth := getPackageDetails(
		"package1", "5.0.1", "https://cran.r-project.org", "SomeOtherCRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://cran.r-project.org/src/contrib/package1_5.0.1.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/downloaded_packages/package_archives/package1_5.0.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, outputLocation, savedBandwidth = getPackageDetails(
		"somePackage1", "1.0.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "cache")
	assert.Equal(t, packageURL, "https://cloud.r-project.org/src/contrib/somePackage1_1.0.0.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/somePackage1_1.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(1000))
	action, packageURL, outputLocation, savedBandwidth = getPackageDetails(
		"somePackage2", "2.0.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://cloud.r-project.org/src/contrib/somePackage2_2.0.0.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/downloaded_packages/package_archives/somePackage2_2.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, outputLocation, savedBandwidth = getPackageDetails(
		"somePackage3", "3.0.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://cloud.r-project.org/src/contrib/Archive/somePackage3/somePackage3_3.0.0.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/downloaded_packages/package_archives/somePackage3_3.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, outputLocation, savedBandwidth = getPackageDetails(
		"someBiocPackage1", "1.0.1", "https://www.bioconductor.org/packages", "Bioconductor",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "cache")
	assert.Equal(t, packageURL, "https://www.bioconductor.org/packages/3.13/data/experiment/src/contrib/someBiocPackage1_1.0.1.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/someBiocPackage_1.0.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(2000))
	action, packageURL, outputLocation, savedBandwidth = getPackageDetails(
		"someBiocPackage2", "2.0.1", "https://www.bioconductor.org/packages", "Bioconductor",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://www.bioconductor.org/packages/3.13/workflows/src/contrib/someBiocPackage2_2.0.1.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/downloaded_packages/package_archives/someBiocPackage2_2.0.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
}
