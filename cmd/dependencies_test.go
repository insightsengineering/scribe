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
