package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func Test_getMapKeyDiff(t *testing.T) {
	original := map[string]bool{"a": true, "b": false, "c": true, "e": false}
	mapsKeysToRemove := map[string][]string{"b": {}, "c": {}, "d": {}}

	res := getMapKeyDiff(original, mapsKeysToRemove)

	assert.NotEmpty(t, res)
	assert.Equal(t, map[string]bool{"a": true, "e": false}, res)
	assert.Equal(t, map[string]bool{"a": true, "b": false, "c": true, "e": false}, original)
	assert.Equal(t, map[string][]string{"b": {}, "c": {}, "d": {}}, mapsKeysToRemove)
}

func Test_parseDescriptionFile(t *testing.T) {
	cases := []struct {filename, field, fieldValue string, extracted []string } {
		{"testdata/DESCRIPTION/NominalLogisticBiplot.txt", "Depends", "R (>= 2.15.1),mirt,gmodels,MASS", []string{"R","mirt","gmodels","MASS"}},
		{"RcppNumerical", "LinkingTo", "Rcpp, RcppEigen", []string{"Rcpp", "RcppEigen"}},
	}
	for _, c := range cases {
		kw := parseDescriptionFile()
		assert.Equal(t, c.fieldValue, kv[c.field])
		pv := removePackageVersionConstraints(kv[c.field])
		assert.Equal(t, c.extracted ,pv)
	}
}

func Test_cleanDescription(t *testing.T) {
	cleanedDescription := cleanDescription(descriptionContent)
	assert.True(t, strings.Contains(cleanedDescription, "Imports"))
	assert.False(t, strings.Contains(cleanedDescription, "BugReports"))
	assert.False(t, strings.Contains(cleanedDescription, "Roxygen"))
	assert.Equal(t, cleanDescriptionContent, cleanedDescription)
}

func Test_parseDescription(t *testing.T) {
	kv := parseDescription(descriptionContent)
	assert.Equal(t, "methods, optimx, parallel,", kv["Imports"])
}

func Test_getPackageContent(t *testing.T) {
	content := getPackageContent()
	assert.True(t, strings.Contains(content, "Package:"))
}

func Test_removePackageVersionConstraints(t *testing.T) {
	cases := []struct{ input, expected string }{
		{"", ""},
		{"R", "R"},
		{"R (>=4.0.3)", "R"},
		{"R(>=4.0.3)", "R"},
		{" R(>=4.0.3)", "R"},
		{" R >=4.0.3", "R"},
		{" R>=4.0.3", "R"},
		{" R (  >=   4.0.3) ", "R"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, removePackageVersionConstraints(c.input))
	}
}

func Test_getPackageDepsFromTarGz(t *testing.T) {
	cases := []struct{ targz string }{
		{"testdata/targz/OrdinalLogisticBiplot_0.4.tar.gz"},
		{"testdata/targz/curl_4.3.2.tar.gz"},
	}
	for _, v := range cases {
		deps := getPackageDepsFromTarGz(v.targz)
		assert.NotEmpty(t, deps)
	}
}

func Test_getPackageDepsFromRepositoryURLs(t *testing.T) {
	deps := getPackageDepsFromRepositoryURLs([]string{"http://rkalvrexper.kau.roche.com:4242/roche-ghe@default/latest"},
		map[string]bool{"ArtifactDB": true, "gp.auth": true})
	assert.NotEmpty(t, deps)
	assert.NotEmpty(t, deps["ArtifactDB"])
	assert.NotEmpty(t, deps["gp.auth"])
}

func Test_getPackageDepsFromSinglePackageLocation(t *testing.T) {
	//repoLocation := "/tmp/scribe/downloaded_packages/github/insightsengineering/teal.code"
	repoLocation := "testdata/BiocBaseUtils"
	packDeps := getPackageDepsFromSinglePackageLocation(repoLocation, true)

	assert.NotEmpty(t, packDeps)
	assert.True(t, slices.Contains(packDeps, "R"))

}

func Test_getPackageDepsFromPackagesFile(t *testing.T) {
	packagesFilePath := "testdata/BIOC_PACKAGES_BIOC"
	packDeps := getPackageDepsFromPackagesFile(packagesFilePath, map[string]bool{"Rgraphviz": true, "S4Vectors": true})
	assert.NotNil(t, packDeps)
	assert.NotEmpty(t, packDeps["Rgraphviz"])
	assert.NotEmpty(t, packDeps["S4Vectors"])
}

func Test_getPackageDepsFromBioconductor(t *testing.T) {
	deps := getPackageDepsFromBioconductor(map[string]bool{"Rgraphviz": true, "S4Vectors": true}, "3.16")
	assert.NotEmpty(t, deps["Rgraphviz"])
	assert.NotEmpty(t, deps["S4Vectors"])
}

func Test_getPackageDepsFromCrandb(t *testing.T) {
	pkgs := []string{"ggplot2", "dplyr"}
	packDeps := getPackageDepsFromCrandb(pkgs)

	assert.NotEmpty(t, packDeps)
	assert.Contains(t, packDeps["dplyr"], "rlang")
}

func Test_getPackageDepsFromCrandbWithChunk(t *testing.T) {
	pkgs := []string{"childsds", "ini", "teal.logger", "withr", "BiocFileCache", "contrast", "spatial", "stringr"}
	packDeps := getPackageDepsFromCrandbWithChunk(toEmptyMapString(pkgs))

	assert.NotEmpty(t, packDeps)
	assert.Contains(t, packDeps["childsds"], "tidyr")
}

func Test_getCrandbUrl(t *testing.T) {
	pkgs := []string{"ggplot2"}
	url := getCrandbUrl(pkgs)
	assert.True(t, strings.Contains(url, "ggplot2"))
}

func Test_getDependenciesFields(t *testing.T) {
	cases := []struct{ included bool }{
		{true},
		{false},
	}
	var fileds []string
	for _, v := range cases {
		fileds = getDependenciesFields(v.included)
		assert.True(t, slices.Contains(fileds, "Suggests") == v.included)

	}
}

const cleanDescriptionContent = `Package: tern
Version: 0.7.6.9037
Depends:
    R (>= 3.6),
Imports:
    methods,
    optimx,
    parallel,
Suggests:
    knitr,
    testthat (>= 2.0)
`
const descriptionContent = `Package: tern
Title: Create Common TLGs used in Clinical Trials
Version: 0.7.6.9037
Date: 2022-01-27
Authors@R: c(
    person("NEST", , , "basel.nestcicd@roche.com", role = c("aut", "cre")),
  )
Description: Table, Listings, and Graphs (TLG) library for common outputs
    used in clinical trials.
License: Apache License 2.0 | file LICENSE
URL: https://github.com/insightsengineering/tern
BugReports: https://github.com/insightsengineering/tern/issues
Depends:
    R (>= 3.6),
Imports:
    methods,
    optimx,
    parallel,
Suggests:
    knitr,
    testthat (>= 2.0)
VignetteBuilder:
    knitr
Encoding: UTF-8
LazyData: true
Roxygen: list(markdown = TRUE)
RoxygenNote: 7.1.2
Collate:
    'formats.R'
`
