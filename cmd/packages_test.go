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

func Test_processDescriptionFile(t *testing.T) {
	var allPackages []PackagesFile
	byteValue, err := os.ReadFile("testdata/PACKAGES")
	checkError(err)
	processDescriptionFile(string(byteValue), &allPackages)
	assert.Equal(t, allPackages,
		[]PackagesFile{
			{
				"somePackage1",
				"1.0.0",
				[]Dependency{
					{
						"Depends",
						"R",
						">=",
						"2.15.0",
					},
				},
			},
			{
				"somePackage2",
				"2.0.0",
				[]Dependency{
					{
						"Depends",
						"R",
						">=",
						"3.6.0",
					},
					{
						"Imports",
						"magrittr",
						"",
						"",
					},
					{
						"Imports",
						"dplyr",
						"",
						"",
					},
				},
			},
			{
				"somePackage3",
				"0.0.1",
				[]Dependency{
					{
						"Depends",
						"R",
						">=",
						"3.1.0",
					},
					{
						"Imports",
						"ggplot2",
						">=",
						"3.1.0",
					},
					{
						"Imports",
						"shiny",
						">=",
						"1.3.1",
					},
					{
						"Suggests",
						"rmarkdown",
						">=",
						"1.13",
					},
					{
						"Suggests",
						"knitr",
						">=",
						"1.22",
					},
				},
			},
			{
				"somePackage4",
				"0.2",
				[]Dependency{
					{
						"Suggests",
						"testthat",
						">=",
						"3.0.0",
					},
				},
			},
		},
	)
}
