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

// getPackageDepsFromPackagesFile retrieves the list of relevant dependencies
// of a given package from PACKAGES file structure of the repository from which
// the package has been downloaded.
func getPackageDepsFromPackagesFile(
	packageName string,
	packagesFile locksmith.PackagesFile,
	downloadedPackages map[string]DownloadedPackage,
) []string {
	var packageDependencies []string
	for _, packagesEntry := range packagesFile.Packages {
		if packagesEntry.Package == packageName {
			for _, dependency := range packagesEntry.Dependencies {
				// Check if the dependency has been successfully downloaded.
				downloadedDependency, ok := downloadedPackages[dependency.DependencyName]
				var dependencyLocation string
				if ok {
					dependencyLocation = downloadedDependency.Location
				}
				// Only add the dependency to the list of package dependencies,
				// if it's not a base R package, and it has been successfully downloaded,
				// and it hasn't been added to the list yet.
				if !locksmith.CheckIfBasePackage(dependency.DependencyName) &&
					dependencyLocation != "" &&
					!stringInSlice(dependency.DependencyName, packageDependencies) {
					packageDependencies = append(packageDependencies, dependency.DependencyName)
				}
			}
			break
		}
	}
	return packageDependencies
}

// getDepsFromPackagesFiles downloads PACKAGES files from each of rRepositories.
// It saves map entries (to packageDependencies) from package name to the list of package dependencies.
func getDepsFromPackagesFiles(
	rPackages map[string]Rpackage,
	rRepositories []Rrepository,
	downloadedPackages map[string]DownloadedPackage,
	packageDependencies map[string][]string,
) {
	for _, repository := range rRepositories {
		log.Info("repository = ", repository)
		_, _, content := locksmith.DownloadTextFile(repository.URL + "/src/contrib/PACKAGES", make(map[string]string))
		packagesFile := locksmith.ProcessPackagesFile(content)
		// Go through the list of packages, and add information to the output data structure
		// about dependencies but only those which were downloaded from this repository.
		for packageName, _ := range rPackages {
			var packageRepository string
			downloadedPackage, ok := downloadedPackages[packageName]
			if ok {
				packageRepository = downloadedPackage.PackageRepository
			}
			// Retrieve information about package dependencies from the PACKAGES file
			// downloaded from the repository from which this package has been downloaded
			// according to the renv.lock.
			// In particular, dependencies for GitHub/GitLab packages will NOT be read
			// from PACKAGES file, since the packageRepository == "GitHub"/"GitLab" for them.
			if packageRepository == repository.Name {
				packageDeps := getPackageDepsFromPackagesFile(
					packageName, packagesFile, downloadedPackages,
				)
				log.Debug(packageName, " → ", packageDeps)
				packageDependencies[packageName] = packageDeps
			}
		}
	}
}

func getDepsFromDescriptionFiles(
	rPackages map[string]Rpackage,
	downloadedPackages map[string]DownloadedPackage,
	packageDependencies map[string][]string,
) {

}

func getPackageDeps(
	rPackages map[string]Rpackage,
	bioconductorVersion string,
	rRepositories []Rrepository,
	downloadedPackages map[string]DownloadedPackage,
	includeSuggests bool,
) map[string][]string {
	log.Debug("Getting package dependencies for ", len(rPackages), " packages")
	packageDependencies := make(map[string][]string)

	getDepsFromPackagesFiles(rPackages, rRepositories, downloadedPackages, packageDependencies)
	getDepsFromDescriptionFiles(rPackages, downloadedPackages, packageDependencies)

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

	return packageDependencies
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
