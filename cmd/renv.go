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
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

type Renvlock struct {
	R            Rversion
	Bioconductor BioC
	Packages     map[string]Rpackage
}

type BioC struct {
	Version string
}

type Rversion struct {
	Version      string
	Repositories []Rrepository
}

type Rrepository struct {
	Name string
	URL  string
}

type Rpackage struct {
	Package      string
	Version      string
	Source       string
	Repository   string
	Hash         string
	RemoteType   string   `json:",omitempty"`
	Requirements []string `json:",omitempty"`
	// Below 'Remote' properties only exist in renv.lock
	// if package comes from git repository.
	RemoteHost     string `json:",omitempty"`
	RemoteRepo     string `json:",omitempty"`
	RemoteUsername string `json:",omitempty"`
	RemoteRef      string `json:",omitempty"`
	RemoteSha      string `json:",omitempty"`
	RemoteSubdir   string `json:",omitempty"`
}

func getRenvLock(filename string, renvLock *Renvlock) {
	byteValue, err := os.ReadFile(filename)
	checkError(err)

	err = json.Unmarshal(byteValue, &renvLock)
	checkError(err)
}

func getRenvRepositoryURL(renvLockRepositories []Rrepository, repositoryName string) string {
	for _, v := range renvLockRepositories {
		if v.Name == repositoryName {
			return v.URL
		}
	}
	// Return default mirror if the repository is not defined in lock file.
	return defaultCranMirrorURL
}

// appendIfNotInSlice checks whether itemToAppend already exists in slice.
// If not, it appends itemToAppend to slice.
func appendIfNotInSlice(itemToAppend string, slice *[]string) {
	if !stringInSlice(itemToAppend, *slice) {
		*slice = append(*slice, itemToAppend)
	}
}

// validatePackageFields returns the number of warnings occurring during validation of
// package fields in the renv.lock. If, according to renv.lock, the package should be downloaded
// from a repository not defined in the renv.lock header, validatePackageFields appends
// that repository name to erroneousRepositoryNames.
func validatePackageFields(packageName string, packageFields Rpackage,
	repositories []string, erroneousRepositoryNames *[]string) int {
	var numberOfWarnings int
	switch {
	case packageFields.Package == "":
		log.Warn("Package ", packageName, " doesn't have the Package field set.")
		numberOfWarnings++
	case packageFields.Version == "":
		log.Warn("Package ", packageName, " doesn't have the Version field set.")
		numberOfWarnings++
	case packageFields.Source == "":
		log.Warn("Package ", packageName, " doesn't have the Source field set.")
		numberOfWarnings++
	}
	if packageFields.Repository == "" {
		appendIfNotInSlice(packageFields.Repository, erroneousRepositoryNames)
		switch {
		case packageFields.Source == "Repository":
			log.Warn("Package ", packageName, " doesn't have the Repository field set.")
			numberOfWarnings++
		case (packageFields.Source == GitHub || packageFields.Source == GitLab) &&
			(packageFields.RemoteType == "" || packageFields.RemoteHost == "" ||
				packageFields.RemoteRepo == "" || packageFields.RemoteUsername == "" ||
				(packageFields.RemoteRef == "" && packageFields.RemoteSha == "")):
			log.Warn("Package ", packageName, " with source ", packageFields.Source,
				" doesn't have the required Remote details provided.")
			numberOfWarnings++
		}
	} else if !stringInSlice(packageFields.Repository, repositories) {
		appendIfNotInSlice(packageFields.Repository, erroneousRepositoryNames)
		log.Warn("Repository \"", packageFields.Repository, "\" has not been defined in lock"+
			" file for package ", packageName, ".\n")
		numberOfWarnings++
	}
	return numberOfWarnings
}

// validateRenvLock returns number of warnings during validation of renv.lock file.
func validateRenvLock(renvLock Renvlock, erroneousRepositoryNames *[]string) int {
	var repositories []string
	var numberOfWarnings int
	for _, v := range renvLock.R.Repositories {
		repositories = append(repositories, v.Name)
	}
	for k, v := range renvLock.Packages {
		newWarnings := validatePackageFields(k, v, repositories, erroneousRepositoryNames)
		numberOfWarnings += newWarnings
	}
	return numberOfWarnings
}

// updatePackagesRenvLock checks if any packages in renv.lock file should be updated.
// A copy of renv.lock file is created - it contains updated versions
// of selected packages as well as updated git HEAD SHAs.
// This is checked by cloning the packages' git repositories.
func updatePackagesRenvLock(renvLock *Renvlock, outputFilename string, updatedPackages string) {
	splitUpdatePackages := strings.Split(updatedPackages, ",")
	var allUpdateExpressions []string
	// For each comma-separated wildcard expression convert "." and "*"
	// characters to regexp equivalent.
	for _, singleRegexp := range splitUpdatePackages {
		singleRegexp = strings.ReplaceAll(singleRegexp, `.`, `\.`)
		singleRegexp = strings.ReplaceAll(singleRegexp, "*", ".*")
		allUpdateExpressions = append(allUpdateExpressions, "^"+singleRegexp+"$")
	}
	// Create temporary directory to clone the packages to be updated.
	updateRegexp := strings.Join(allUpdateExpressions, "|")
	err := os.RemoveAll(localOutputDirectory + "/git_updates")
	checkError(err)
	err = os.MkdirAll(localOutputDirectory+"/git_updates", os.ModePerm)
	checkError(err)
	for k, v := range renvLock.Packages {
		match, err2 := regexp.MatchString(updateRegexp, k)
		checkError(err2)
		if match && (v.Source == "GitLab" || v.Source == "GitHub") {
			log.Debug("Package ", k, " matches updated packages regexp ", allUpdateExpressions)
			var credentialsType string
			if v.Source == "GitLab" {
				credentialsType = "gitlab"
			} else if v.Source == "GitHub" {
				credentialsType = "github"
			}
			// Clone package's default branch.
			gitErr, _, newPackageSha := cloneGitRepo(
				localOutputDirectory+"/git_updates/"+k,
				getRepositoryURL(v, renvLock.R.Repositories),
				credentialsType,
				"", "",
			)
			if gitErr != "" {
				log.Error(gitErr)
			}
			// Read newest package version from DESCRIPTION.
			var remoteSubdir string
			if v.RemoteSubdir != "" {
				remoteSubdir = "/" + v.RemoteSubdir
			}
			descriptionContents := parseDescriptionFile(
				localOutputDirectory + "/git_updates/" + k + remoteSubdir + "/DESCRIPTION",
			)
			newPackageVersion := descriptionContents["Version"]
			// Update renv structure with new package version and SHA.
			if entry, ok := renvLock.Packages[k]; ok {
				log.Info("Updating package ", k, " version: ",
					entry.Version, " -> ", newPackageVersion,
					", SHA: ", entry.RemoteSha, " -> ", newPackageSha,
				)
				entry.Version = newPackageVersion
				entry.RemoteSha = newPackageSha
				// Clear hash since it's likely not valid anymore.
				entry.Hash = ""
				renvLock.Packages[k] = entry
			}
		} else {
			log.Debug(
				"Package ", k, " doesn't match updated packages regexp ",
				allUpdateExpressions, " or is not a git repository.",
			)
		}
	}
	bytes, err := json.MarshalIndent(*renvLock, "", "  ")
	checkError(err)
	output, err := os.Create(outputFilename)
	checkError(err)
	defer output.Close()
	_, err = output.WriteString(string(bytes))
	checkError(err)
}
