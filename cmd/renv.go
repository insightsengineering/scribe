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

func GetRenvLock(filename string, renvLock *Renvlock) {
	byteValue, err := os.ReadFile(filename)
	checkError(err)

	err = json.Unmarshal(byteValue, &renvLock)
	checkError(err)
}

func WriteRenvLock(filename string, renvLock Renvlock) {
	s, err := json.MarshalIndent(renvLock, "", "  ")
	checkError(err)

	err = os.WriteFile(filename, s, 0644) //#nosec
	checkError(err)
}

func ValidateRenvLock(renvLock Renvlock) {
	var repositories []string
	for _, v := range renvLock.R.Repositories {
		repositories = append(repositories, v.Name)
	}
	for k, v := range renvLock.Packages {
		switch {
		case v.Package == "":
			log.Warn("Package ", k, " doesn't have the Package field set.")
		case v.Version == "":
			log.Warn("Package ", k, " doesn't have the Version field set.")
		case v.Source == "":
			log.Warn("Package ", k, " doesn't have the Source field set.")
		case v.Hash == "":
			log.Warn("Package ", k, " doesn't have the Hash field set.")
		}
		if v.Repository == "" {
			switch {
			case v.Source == "Repository":
				log.Warn("Package ", k, " doesn't have the Repository field set.")
			case v.Source == "GitHub" &&
				(v.RemoteType == "" || v.RemoteHost == "" || v.RemoteRepo == "" ||
					v.RemoteUsername == "" || v.RemoteRef == "" || v.RemoteSha == ""):
				log.Warn("Package ", k, " with source ", v.Source, " doesn't have the"+
					" required Remote details provided.")
			}
		} else if !stringInSlice(v.Repository, repositories) {
			log.Warn("Repository \"", v.Repository, "\" has not been defined in lock"+
				" file for package ", k, ".\n")
		}
	}
}
