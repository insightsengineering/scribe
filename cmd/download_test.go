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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getRepositoryURL(t *testing.T) {
	var renvLock Renvlock
	getRenvLock("testdata/renv.lock.empty.json", &renvLock)
	repoURL := getRepositoryURL(renvLock.Packages["SomePackage"], renvLock.R.Repositories)
	assert.Equal(t, repoURL, defaultCranMirrorURL)
	repoURL = getRepositoryURL(renvLock.Packages["SomeOtherPackage5"], renvLock.R.Repositories)
	// default value returned because SomeOtherPackage5 has repository set to undefined CRAN1
	assert.Equal(t, repoURL, defaultCranMirrorURL)
	repoURL = getRepositoryURL(renvLock.Packages["SomeBiocPackage"], renvLock.R.Repositories)
	assert.Equal(t, repoURL, bioConductorURL)
	repoURL = getRepositoryURL(renvLock.Packages["SomeOtherPackage"], renvLock.R.Repositories)
	assert.Equal(t, repoURL, "https://github.com/RemoteUsername/RemoteRepo")
	repoURL = getRepositoryURL(renvLock.Packages["SomeOtherPackage2"], renvLock.R.Repositories)
	assert.Equal(t, repoURL, "https://gitlab.com/RemoteUsername/RemoteRepo")
}

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
	// someBiocPackage3 doesn't exist in any BioConductor category - therefore not added to packageInfo

	action, packageURL, _, outputLocation, _, savedBandwidth := getPackageDetails(
		"package1", "5.0.1", "https://cran.r-project.org", "SomeOtherCRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://cran.r-project.org/src/contrib/package1_5.0.1.tar.gz")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/package1_5.0.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"somePackage1", "1.0.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "cache")
	assert.Equal(t, packageURL, "https://cloud.r-project.org/src/contrib/somePackage1_1.0.0.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/somePackage1_1.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(1000))

	// In this test we want somePackage2 in version 1.9.0 but CRAN only has version 2.0.0.
	// That's why we expect fallback version 2.0.0.
	var fallbackPackageURL string
	var fallbackOutputLocation string
	action, packageURL, fallbackPackageURL, outputLocation, fallbackOutputLocation, savedBandwidth = getPackageDetails(
		"somePackage2", "1.9.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://cloud.r-project.org/src/contrib/Archive/somePackage2/somePackage2_1.9.0.tar.gz")
	assert.Equal(t, fallbackPackageURL, "https://cloud.r-project.org/src/contrib/somePackage2_2.0.0.tar.gz")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/somePackage2_1.9.0.tar.gz")
	assert.Equal(t, fallbackOutputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/somePackage2_2.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))

	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"somePackage2", "2.0.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL, "https://cloud.r-project.org/src/contrib/somePackage2_2.0.0.tar.gz")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/somePackage2_2.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"somePackage3", "3.0.0", "https://cloud.r-project.org", "CRAN",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL,
		"https://cloud.r-project.org/src/contrib/Archive/somePackage3/somePackage3_3.0.0.tar.gz")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/somePackage3_3.0.0.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"someBiocPackage1", "1.0.1", "https://www.bioconductor.org/packages", "Bioconductor",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "cache")
	assert.Equal(t, packageURL,
		"https://www.bioconductor.org/packages/3.13/data/experiment/src/contrib/someBiocPackage1_1.0.1.tar.gz")
	assert.Equal(t, outputLocation, "/tmp/scribe/someBiocPackage_1.0.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(2000))
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"someBiocPackage2", "2.0.1", "https://www.bioconductor.org/packages", "Bioconductor",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL,
		"https://www.bioconductor.org/packages/3.13/workflows/src/contrib/someBiocPackage2_2.0.1.tar.gz")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/someBiocPackage2_2.0.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))

	// The package in requested version is not available in Bioconductor current but
	// it should be attempted to download it from Bioconductor Archive.
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"someBiocPackage2", "1.9.1", "https://www.bioconductor.org/packages", "Bioconductor",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "download")
	assert.Equal(t, packageURL,
		"https://www.bioconductor.org/packages/3.13/workflows/src/contrib/Archive/someBiocPackage2/someBiocPackage2_1.9.1.tar.gz")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/package_archives/someBiocPackage2_1.9.1.tar.gz")
	assert.Equal(t, savedBandwidth, int64(0))

	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"someBiocPackage3", "3.0.1", "https://www.bioconductor.org/packages", "Bioconductor",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "notfound_bioc")
	assert.Equal(t, packageURL, "")
	assert.Equal(t, outputLocation, "")
	assert.Equal(t, savedBandwidth, int64(0))

	// git packages
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"gitHubPackage", "0.0.5", "https://github.com/insightsengineering/gitHubPackage", "GitHub",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "github")
	assert.Equal(t, packageURL, "https://github.com/insightsengineering/gitHubPackage")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/github/insightsengineering/gitHubPackage")
	assert.Equal(t, savedBandwidth, int64(0))
	action, packageURL, _, outputLocation, _, savedBandwidth = getPackageDetails(
		"gitLabPackage", "0.0.6", "https://gitlab.com/example/gitLabPackage", "GitLab",
		packageInfo, biocPackageInfo, biocUrls, localArchiveChecksums,
	)
	assert.Equal(t, action, "gitlab")
	assert.Equal(t, packageURL, "https://gitlab.com/example/gitLabPackage")
	assert.Equal(t, outputLocation,
		"/tmp/scribe/downloaded_packages/gitlab/gitlab.com/example/gitLabPackage")
	assert.Equal(t, savedBandwidth, int64(0))
}

func mockedDownloadFile(_ string, _ string) (int, int64) {
	return 200, 1
}

func mockedCloneGitRepo(_ string, _ string, _ bool, _ string, _ string) (string, int64) {
	return "", 1
}

func Test_downloadPackages(t *testing.T) {
	var renvLock Renvlock
	getRenvLock("testdata/renv.lock.empty.json", &renvLock)
	var allDownloadInfo []DownloadInfo
	downloadPackages(renvLock, &allDownloadInfo, mockedDownloadFile, mockedCloneGitRepo)
	var localFiles []string
	var messages []string
	for _, v := range allDownloadInfo {
		localFiles = append(localFiles, v.OutputLocation)
		messages = append(messages, v.Message)
	}
	sort.Strings(localFiles)
	sort.Strings(messages)
	assert.Equal(t, localFiles, []string{"",
		"/tmp/scribe/downloaded_packages/github/RemoteUsername/RemoteRepo",
		"/tmp/scribe/downloaded_packages/package_archives/SomeOtherPackage3_1.0.0.tar.gz",
		"/tmp/scribe/downloaded_packages/package_archives/SomeOtherPackage4_1.0.0.tar.gz",
		"/tmp/scribe/downloaded_packages/package_archives/SomeOtherPackage5_1.0.0.tar.gz",
		"/tmp/scribe/downloaded_packages/package_archives/SomePackage_1.0.0.tar.gz"},
	)
	assert.Equal(t, messages, []string{"Couldn't find SomeBiocPackage version 1.0.1 in BioConductor.",
		"https://cloud.r-project.org/src/contrib/Archive/SomeOtherPackage3/SomeOtherPackage3_1.0.0.tar.gz",
		"https://cloud.r-project.org/src/contrib/Archive/SomeOtherPackage4/SomeOtherPackage4_1.0.0.tar.gz",
		"https://cloud.r-project.org/src/contrib/Archive/SomeOtherPackage5/SomeOtherPackage5_1.0.0.tar.gz",
		"https://cloud.r-project.org/src/contrib/Archive/SomePackage/SomePackage_1.0.0.tar.gz",
		"https://github.com/RemoteUsername/RemoteRepo"},
	)
}
