package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_parsePackagesFile(t *testing.T) {
	packages := make(map[string]*PackageInfo)
	assert.Equal(t, true, true)
}

func Test_parsePackagesFile(t *testing.T) {
	description := `
	Package: tern
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
	control.
		assert.Equal(t, true, true)
}
