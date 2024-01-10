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

	locksmith "github.com/insightsengineering/locksmith/cmd"
	yaml "gopkg.in/yaml.v3"
)

// getPackageDepsFromPackagesFile retrieves the list of relevant dependencies
// of a given package from PACKAGES file structure of the repository from which
// the package has been downloaded.
func getPackageDepsFromPackagesFile(
	packageName string,
	packagesFile locksmith.PackagesFile,
	downloadedPackages map[string]DownloadedPackage,
) []string {
	var packageDependencies []string
	// Find the packageName in the PACKAGES file.
	for _, packagesEntry := range packagesFile.Packages {
		if packagesEntry.Package == packageName {
			log.Debug("Processing package ", packageName)
			// Read its dependencies.
			for _, dependency := range packagesEntry.Dependencies {
				// Check if the dependency has been successfully downloaded.
				downloadedDependency, ok := downloadedPackages[dependency.DependencyName]
				var dependencyLocation string
				if ok {
					dependencyLocation = downloadedDependency.Location
				}
				log.Debug("    Processing dependency ", dependency.DependencyName)
				log.Debug("    Location: ", dependencyLocation)
				// Only add the dependency to the list of package dependencies,
				// if it's not a base R package, and it has been successfully downloaded,
				// and it hasn't been added to the list yet.
				// Dependencies are retrieved from PACKAGES file only for packages not downloaded
				// from git repositories, and for such packages Suggested packages are not treated
				// as dependencies.
				if !locksmith.CheckIfBasePackage(dependency.DependencyName) &&
					dependencyLocation != "" &&
					!stringInSlice(dependency.DependencyName, packageDependencies) &&
					dependency.DependencyType != "Suggests" {
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
	downloadFileFunction func(string, map[string]string) (int, int64, string),
	erroneousRepositoryNames []string,
) {
	for _, repository := range rRepositories {
		log.Debug("Processing packages from repository: ", repository)
		_, _, content := downloadFileFunction(repository.URL+"/src/contrib/PACKAGES", make(map[string]string))
		packagesFile := locksmith.ProcessPackagesFile(content)
		// Go through the list of packages from renv.lock, and add information to the output data structure
		// about dependencies but only those which were downloaded from this repository.
		for packageName := range rPackages {
			// Check if package downloaded successfully.
			var packageRepository string
			downloadedPackage, ok := downloadedPackages[packageName]
			if ok {
				packageRepository = downloadedPackage.PackageRepository
			} else {
				log.Warn("Skipping package ", packageName, " because it hasn't been",
					" downloaded properly.")
				continue
			}
			// Retrieve information about package dependencies from the PACKAGES file.
			// The PACKAGES file is downloaded from the same repository as the package
			// (according to the renv.lock).
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

	// Iterate through packages which have the repository name set to one which is not defined
	// in the renv.lock header. For these packages we'll use PACKAGES file from CRAN to determine
	// their dependencies.
	_, _, cranPackagesContent := downloadFileFunction(
		"https://cloud.r-project.org/src/contrib/PACKAGES", make(map[string]string),
	)
	cranPackagesFile := locksmith.ProcessPackagesFile(cranPackagesContent)
	log.Info("Dependencies for packages with Repository renv.lock field equal to any of ",
		erroneousRepositoryNames, " will be determined based on PACKAGES file from CRAN.")
	for packageName := range rPackages {
		// Check if package downloaded successfully.
		var packageRepository string
		downloadedPackage, ok := downloadedPackages[packageName]
		if ok {
			packageRepository = downloadedPackage.PackageRepository
		} else {
			log.Warn(
				"Skipping package ", packageName, " because it hasn't been",
				" downloaded properly.",
			)
			continue
		}
		if stringInSlice(packageRepository, erroneousRepositoryNames) {
			packageDeps := getPackageDepsFromPackagesFile(
				packageName, cranPackagesFile, downloadedPackages,
			)
			log.Debug(packageName, " → ", packageDeps)
			packageDependencies[packageName] = packageDeps
		}
	}
}

// getDepsFromDescriptionFiles for each package downloaded as git repository, reads its dependencies
// from the DESCRIPTION file. It saves map entries (to packageDependencies) from package name to
// the list of package dependencies.
func getDepsFromDescriptionFiles(
	rPackages map[string]Rpackage,
	downloadedPackages map[string]DownloadedPackage,
	packageDependencies map[string][]string,
) {
	// Iterate through packages from renv.lock.
	for packageName := range rPackages {
		var packageRepository string
		var packageLocation string
		// Check if the package has been successfully downloaded and where.
		downloadedPackage, ok := downloadedPackages[packageName]
		if ok {
			packageRepository = downloadedPackage.PackageRepository
			packageLocation = downloadedPackage.Location
		}
		if packageRepository == GitLab || packageRepository == GitHub {
			if packageLocation == "" {
				log.Warn("Skipping installation of ", packageName, " as it hasn't been downloaded properly.")
				continue
			}
			log.Trace(packageName, " = ", downloadedPackage)
			// Read package dependencies from its DESCRIPTION file.
			byteValue, err := os.ReadFile(packageLocation + "/DESCRIPTION")
			checkError(err)
			cleanedDescription := locksmith.CleanDescriptionOrPackagesEntry(string(byteValue), true)
			packageMap := make(map[string]string)
			err = yaml.Unmarshal([]byte(cleanedDescription), &packageMap)
			checkError(err)
			var packageDeps []locksmith.Dependency
			locksmith.ProcessDependencyFields(packageMap, &packageDeps)

			// Filter only relevant dependencies.
			var filteredDependencies []string
			for _, dependency := range packageDeps {
				// Check if the dependency has been successfully downloaded.
				downloadedDependency, ok := downloadedPackages[dependency.DependencyName]
				var dependencyLocation string
				if ok {
					dependencyLocation = downloadedDependency.Location
				}
				// Only add the dependency to the list of package dependencies,
				// if it's not a base R package, and it has been successfully downloaded,
				// and it hasn't been added to the list yet. For packages downloaded
				// from git, the Suggested packages are treated as ordinary dependencies.
				if !locksmith.CheckIfBasePackage(dependency.DependencyName) &&
					dependencyLocation != "" &&
					!stringInSlice(dependency.DependencyName, filteredDependencies) {
					filteredDependencies = append(filteredDependencies, dependency.DependencyName)
				}
			}
			log.Debug(packageName, " → ", filteredDependencies)
			packageDependencies[packageName] = filteredDependencies
		}
	}
}

func getPackageDeps(
	rPackages map[string]Rpackage,
	rRepositories []Rrepository,
	downloadedPackages map[string]DownloadedPackage,
	erroneousRepositoryNames []string,
) map[string][]string {
	// A map with keys being renv.lock package names, and values being lists of dependencies
	// (packages that should be installed in the system before the package corresponding
	// to map key can be installed).
	packageDependencies := make(map[string][]string)

	// If package is stored in tar.gz, get its dependencies from a corresponding
	// entry in PACKAGES file in the repository pointed by renv.lock.
	getDepsFromPackagesFiles(rPackages, rRepositories, downloadedPackages, packageDependencies,
		locksmith.DownloadTextFile, erroneousRepositoryNames)

	// If the package is stored in a cloned git repository, get its dependencies
	// from its DESCRIPTION file.
	getDepsFromDescriptionFiles(rPackages, downloadedPackages, packageDependencies)

	return packageDependencies
}
