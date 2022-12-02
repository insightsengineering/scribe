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
	"fmt"
	"os"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
)

const maxInstallRoutines = 40

const temporalLibPath = "/tmp/scribe/installed_packages" //:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

func installSinglePackage(outputLocation string) error {
	log.Debugf("Package location is %s", outputLocation)
	cmd := "R CMD INSTALL -l " + temporalLibPath + " " + outputLocation
	log.Debug(cmd)
	result, err := execCommand(cmd, true, false)
	log.Error(result)
	if err != nil {
		log.Error(err)
	}
	return err
}

func mkLibPathDir() {
	for _, libPath := range strings.Split(temporalLibPath, ":") {
		if _, err := os.Stat(libPath); os.IsNotExist(err) {
			err := os.MkdirAll(libPath, os.ModePerm)
			checkError(err)
		}
	}
}

func InstallPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo) {
	mkLibPathDir()
	packages := make([]string, 0, len(renvLock.Packages))
	packagesSet := make(map[string]bool)
	for _, p := range renvLock.Packages {
		packages = append(packages, p.Package)
		packagesSet[p.Package] = true
	}

	packagesLocation := make(map[string]struct{ PackageType, Location string })
	for _, v := range *allDownloadInfo {
		packagesLocation[v.PackageName] = struct{ PackageType, Location string }{v.DownloadedPackageType, v.OutputLocation}
	}

	deps := getPackageDepsFromCrandbWithChunk(packages)
	depsBioc := getPackageDepsFromBioconductor(packagesSet, renvLock.Bioconductor.Version)
	for k, v := range depsBioc {
		deps[k] = v
	}
	var reposUrls []string
	for _, v := range renvLock.R.Repositories {
		reposUrls = append(reposUrls, v.URL)
	}
	depsRepos := getPackageDepsFromRepositoryURLs(reposUrls, packagesSet)
	for k, v := range depsRepos {
		deps[k] = v
	}

	for pName, pInfo := range packagesLocation {
		if pInfo.PackageType == "git" {
			if _, err := os.Stat(pInfo.Location); !os.IsNotExist(err) {
				packageDeps := getPackageDepsFromSinglePackageLocation(pInfo.Location, true)
				deps[pName] = packageDeps
			} else {
				log.Errorf("Directory %s for package %s does not exist", pInfo.PackageType, pInfo.Location)
			}
		}
	}

	packagesNoDeps := getMapKeyDiff(packagesSet, deps)
	for k := range packagesNoDeps {
		info := packagesLocation[k]
		if info.PackageType == "tar.gz" {
			targzDeps := getPackageDepsFromTarGz(info.Location)
			deps[k] = targzDeps
		}
	}

	depsSet := mapset.NewSet[string]()
	for k := range deps {
		depsSet.Add(k)
	}
	allSet := mapset.NewSet[string]()
	for _, v := range packages {
		allSet.Add(v)
	}
	rest1 := allSet.Difference(depsSet)

	rest2 := depsSet.Difference(allSet)
	fmt.Println("all/deps:")
	fmt.Println(rest1)
	fmt.Println("deps/all:")
	fmt.Println(rest2)

	depsOrdered := tsort(deps)
	writeJSON("depsOrdered.json", depsOrdered)
	writeJSON("deps.json", deps)

	for i := 0; i < len(depsOrdered); i++ {
		packageName := depsOrdered[i]
		fmt.Print(packageName + " ")
		if val, ok := packagesLocation[packageName]; ok {
			installSinglePackage(val.Location)
		}
	}

	log.Info("Done")
}
