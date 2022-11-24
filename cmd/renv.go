/*
Copyright 2022 F. Hoffmann-La Roche AG

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
	// return default mirror if the repository is not defined in lock file
	return defaultCranMirrorURL
}

// Returns number of warnings occurring during validation
func validatePackageFields(packageName string, packageFields Rpackage, repositories []string) int {
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
		switch {
		case packageFields.Source == "Repository":
			log.Warn("Package ", packageName, " doesn't have the Repository field set.")
			numberOfWarnings++
		case packageFields.Source == GitHub &&
			(packageFields.RemoteType == "" || packageFields.RemoteHost == "" || packageFields.RemoteRepo == "" ||
				packageFields.RemoteUsername == "" || packageFields.RemoteRef == "" || packageFields.RemoteSha == ""):
			log.Warn("Package ", packageName, " with source ", packageFields.Source, " doesn't have the"+
				" required Remote details provided.")
			numberOfWarnings++
		}
	} else if !stringInSlice(packageFields.Repository, repositories) {
		log.Warn("Repository \"", packageFields.Repository, "\" has not been defined in lock"+
			" file for package ", packageName, ".\n")
		numberOfWarnings++
	}
	return numberOfWarnings
}

// Returns number of warnings during validation of renv.lock file
func validateRenvLock(renvLock Renvlock) int {
	var repositories []string
	var numberOfWarnings int
	for _, v := range renvLock.R.Repositories {
		repositories = append(repositories, v.Name)
	}
	for k, v := range renvLock.Packages {
		newWarnings := validatePackageFields(k, v, repositories)
		numberOfWarnings += newWarnings
	}
	return numberOfWarnings
}
