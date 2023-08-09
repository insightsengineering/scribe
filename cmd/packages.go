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
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

type PackagesFile struct {
	Package      string       `json:"package"`
	Version      string       `json:"version"`
	Dependencies []Dependency `json:"dependencies"`
}

type Dependency struct {
	DependencyType  string `json:"type"`
	DependencyName  string `json:"name"`
	VersionOperator string `json:"operator"`
	VersionValue    string `json:"value"`
}

func getPackagesFileFromURL(url string, allPackages *[]PackagesFile) {
	content, err := request(url)
	if err != nil {
		log.Errorf("Failed to get package content for URL %s", url)
	}
	processDescriptionFile(content, allPackages)
}

func processDescriptionFile(content string, allPackages *[]PackagesFile) {
	for _, lineGroup := range strings.Split(content, "\n\n") {
		firstLine := strings.Split(lineGroup, "\n")[0]
		packageName := strings.ReplaceAll(firstLine, "Package: ", "")
		packageMap := make(map[string]string)
		err := yaml.Unmarshal([]byte(lineGroup), &packageMap)
		if err != nil {
			log.Error("Error reading package data from PACKAGES: ", err)
		}
		var packageDependencies []Dependency
		processDependencyFields(packageMap, &packageDependencies)
		*allPackages = append(
			*allPackages,
			PackagesFile{packageName, packageMap["Version"], packageDependencies},
		)
	}
}

func processDependencyFields(packageMap map[string]string,
	packageDependencies *[]Dependency) {
	dependencyFields := []string{"Depends", "Imports", "Suggests", "Enhances", "LinkingTo"}
	re := regexp.MustCompile(`\(.*\)`)
	for _, field := range dependencyFields {
		if _, ok := packageMap[field]; ok {
			dependencyList := strings.Split(packageMap[field], ", ")
			for _, dependency := range dependencyList {
				dependencyName := strings.Split(dependency, " ")[0]
				versionConstraintOperator := ""
				versionConstraintValue := ""
				if strings.Contains(dependency, "(") && strings.Contains(dependency, ")") {
					versionConstraint := re.FindString(dependency)
					versionConstraint = versionConstraint[1 : len(versionConstraint)-1]
					versionConstraintOperator = strings.Split(versionConstraint, " ")[0]
					versionConstraintValue = strings.Split(versionConstraint, " ")[1]
				}
				*packageDependencies = append(
					*packageDependencies,
					Dependency{field, dependencyName, versionConstraintOperator, versionConstraintValue},
				)
			}
		}
	}
}
