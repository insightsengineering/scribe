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

	mapset "github.com/deckarep/golang-set/v2"
)

const maxInstallRoutines = 40

const temporalLibPath = "/tmp/scribe/installed_packages"

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

func installSinglePackage(outputLocation string) error {
	log.Debugf("Package location is %s", outputLocation)
	cmd := "R CMD INSTALL --install-tests -l" + temporalLibPath + " " + outputLocation
	log.Debug(cmd)
	result, err := execCommand(cmd, true, true)
	log.Error(result)
	if err != nil {
		log.Error(err)
	}
	return err
}

const localCranPackagesPath = localOutputDirectory + "/package_files/CRAN_PACKAGES"

func InstallPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo) {
	err := os.MkdirAll(temporalLibPath, os.ModePerm)
	checkError(err)

	packages := make([]string, 0, len(renvLock.Packages))
	for _, p := range renvLock.Packages {
		packages = append(packages, p.Package)
	}
	deps := getPackageDepsFromCrandbWithChunk(packages)
	depsBioc := getPackageDepsFromBioconductor(packages)
	for k, v := range depsBioc {
		deps[k] = v
	}

	packagesLocation := make(map[string]struct{ PackageType, Location string })
	for _, v := range *allDownloadInfo {
		packagesLocation[v.PackageName] = struct{ PackageType, Location string }{v.DownloadedPackageType, v.OutputLocation}
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

	for i := 0; i < len(depsOrdered); i++ {
		//packageName := depsOrdered[i]
		//log.Debug(packageName)
		//p := packagesLocation[packageName]
		//installSinglePackage(p.Location)

	}

	log.Info("Done")
}
