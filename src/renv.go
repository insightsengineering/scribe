package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
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
    RemoteType     string
    RemoteHost     string
    RemoteRepo     string
    RemoteUsername string
    RemoteRef      string
    RemoteSha      string
    Requirements   []string
}

func GetRenvLock(filename string, renv_lock *Renvlock) {
    jsonFile, err := os.Open(filename)
    if err != nil {
        fmt.Println(err)
    }
    defer jsonFile.Close()

    byteValue, err := ioutil.ReadAll(jsonFile)
    if err != nil {
        fmt.Println(err)
    }

    err = json.Unmarshal(byteValue, &renv_lock)

    if err != nil {
        fmt.Println(err)
    }
}

func ValidateRenvLock(renv_lock Renvlock) {
    var repositories []string
    for _, v := range renv_lock.R.Repositories {
        repositories = append(repositories, v.Name)
    }
    for k, v := range renv_lock.Packages {
        if v.Package == "" {
            fmt.Println("Package", k, "doesn't have the Package field set.")
        }
        if v.Version == "" {
            fmt.Println("Package", k, "doesn't have the Version field set.")
        }
        if v.Source == "" {
            fmt.Println("Package", k, "doesn't have the Source field set.")
        }
        if v.Hash == "" {
            fmt.Println("Package", k, "doesn't have the Hash field set.")
        }
        if v.Repository == "" && v.Source == "Respository" {
            fmt.Println("Package", k, "doesn't have the Repository field set.")
        } else if v.Source == "GitHub" &&
            (v.RemoteType == "" || v.RemoteHost == "" || v.RemoteRepo == "" ||
            v.RemoteUsername == "" || v.RemoteRef == "" || v.RemoteSha == "") {
            fmt.Println("Package", k, "with source", v.Source, "doesn't have the" +
            " required Remote details provided.")
        } else if !stringInSlice(v.Repository, repositories) {
            fmt.Print("Repository ", v.Repository, " has not been defined in lock" +
            " file for package ", k, ".\n")
        }
    }
}
