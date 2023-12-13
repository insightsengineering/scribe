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
	"sort"

	yaml "gopkg.in/yaml.v3"
	locksmith "github.com/insightsengineering/locksmith/cmd"
)

func parseDescriptionFile(descriptionFilePath string) map[string]string {
	log.Trace("Parsing ", descriptionFilePath)
	jsonFile, err := os.ReadFile(descriptionFilePath)
	checkError(err)
	// TODO after aligining with locksmith release, we'll have to add second argument = true
	cleaned := locksmith.CleanDescriptionOrPackagesEntry(string(jsonFile))
	packageMap := make(map[string]string)
	err = yaml.Unmarshal([]byte(cleaned), &packageMap)
	checkError(err)
	return packageMap
}

func getPackageContent() string {
	url := "https://cloud.r-project.org/src/contrib/PACKAGES"
	_, _, content := locksmith.DownloadTextFile(url, make(map[string]string))
	return content
}

func getDependenciesFields(includeSuggests bool) []string {
	res := []string{"Depends", "Imports", "LinkingTo"}
	if includeSuggests {
		res = append(res, "Suggests")
	}
	return res
}

func getPackageDeps(
	rPackages map[string]Rpackage,
	bioconductorVersion string,
	rRepositories []Rrepository,
	packagesLocation map[string]struct{ PackageType, Location string },
	includeSuggests bool,
) map[string][]string {
	log.Debugf("Getting package dependencies for %d packages", len(rPackages))
	packagesSet := make(map[string]bool)
	packagesWithVersion := make(map[string]string)
	deps := make(map[string][]string)
	for k, v := range rPackages {
		packagesSet[k] = true
		packagesWithVersion[k] = v.Version
	}

	// If package is stored in tar.gz, get its dependencies from a corresponding entry in PACKAGES file
	// in the repository pointed by renv.lock.
	// If the package is stored in a cloned git repository, get its dependencies from its DESCRIPTION file.

	// for pName, pInfo := range packagesLocation {
	// 	if pInfo.PackageType == gitConst {
	// 		if _, err := os.Stat(pInfo.Location); !os.IsNotExist(err) {
	// 			packageDeps := getPackageDepsFromSinglePackageLocation(pInfo.Location, true)
	// 			deps[pName] = packageDeps
	// 		} else {
	// 			log.Errorf("Directory %s for package %s does not exist", pInfo.PackageType, pInfo.Location)
	// 		}
	// 	}
	// }

	return deps
}

func sortByCounter(counter map[string]int, nodes []string) []string {
	sort.Slice(nodes, func(i, j int) bool {
		if counter[nodes[i]] == counter[nodes[j]] {
			return nodes[i] < nodes[j]
		}
		return counter[nodes[i]] < counter[nodes[j]]
	})
	return nodes
}

func isDependencyFulfilled(packageName string, dependency map[string][]string, installedPackagesWithVersion map[string]string) bool {
	// TODO: What does this mean?
	log.Tracef("Checking if package %s has fulfilled dependencies", packageName)
	deps := dependency[packageName]
	if len(deps) > 0 {
		for _, dep := range deps {
			if _, ok := installedPackagesWithVersion[dep]; !ok {
				log.Tracef("Not all dependencies are installed for package %s. Dependency not installed: %s", packageName, dep)
				return false
			}
		}
	}
	return true
}
