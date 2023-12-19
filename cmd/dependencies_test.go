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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseDescriptionFile(t *testing.T) {
	cases := []struct {
		filename   string
		field      string
		fieldValue string
		extracted  []string
	}{
		{"testdata/DESCRIPTION/NominalLogisticBiplot.txt", "Depends", "R (>= 2.15.1),mirt,gmodels,MASS", []string{"R", "mirt", "gmodels", "MASS"}},
		{"testdata/DESCRIPTION/RcppNumerical.txt", "LinkingTo", "Rcpp, RcppEigen", []string{"Rcpp", "RcppEigen"}},
	}
	for _, c := range cases {
		kv := parseDescriptionFile(c.filename)
		assert.Equal(t, c.fieldValue, kv[c.field])
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
    person("NEST", , , "basel.nestcicd", role = c("aut", "cre")),
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
