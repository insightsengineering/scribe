package main

import (
	"encoding/json"
	"os"
)

type Renvlock struct {
	R        Rversion
	Packages map[string]Rpackage
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
	Package        string
	Version        string
	Source         string
	Repository     string
	Hash           string
	RemoteType     string `json:",omitempty"`
	Requirements   []string `json:",omitempty"`
	// Below 'Remote' properties only exist in renv.lock
	// if package comes from git repository.
	RemoteHost     string `json:",omitempty"`
	RemoteRepo     string `json:",omitempty"`
	RemoteUsername string `json:",omitempty"`
	RemoteRef      string `json:",omitempty"`
	RemoteSha      string `json:",omitempty"`
}

func GetRenvLock(filename string, renv_lock *Renvlock) {
	byteValue, err := os.ReadFile(filename)
	checkError(err)

	err = json.Unmarshal(byteValue, &renv_lock)
	checkError(err)
}

func WriteRenvLock(filename string, renv_lock Renvlock) {
	s, err := json.MarshalIndent(renv_lock, "", "  ")
	checkError(err)

	err = os.WriteFile(filename, []byte(s), 0644)
	checkError(err)
}

func ValidateRenvLock(renv_lock Renvlock) {
	var repositories []string
	for _, v := range renv_lock.R.Repositories {
		repositories = append(repositories, v.Name)
	}
	for k, v := range renv_lock.Packages {
		if v.Package == "" {
			log.Warn("Package ", k, " doesn't have the Package field set.")
		}
		if v.Version == "" {
			log.Warn("Package ", k, " doesn't have the Version field set.")
		}
		if v.Source == "" {
			log.Warn("Package ", k, " doesn't have the Source field set.")
		}
		if v.Hash == "" {
			log.Warn("Package ", k, " doesn't have the Hash field set.")
		}
		if v.Repository == "" && v.Source == "Respository" {
			log.Warn("Package ", k, " doesn't have the Repository field set.")
		} else if v.Source == "GitHub" &&
			(v.RemoteType == "" || v.RemoteHost == "" || v.RemoteRepo == "" ||
			v.RemoteUsername == "" || v.RemoteRef == "" || v.RemoteSha == "") {
			log.Warn("Package ", k, " with source ", v.Source, " doesn't have the" +
			" required Remote details provided.")
		} else if !stringInSlice(v.Repository, repositories) {
			log.Warn("Repository \"", v.Repository, "\" has not been defined in lock" +
			" file for package ", k, ".\n")
		}
	}
}