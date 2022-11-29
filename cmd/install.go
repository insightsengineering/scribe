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
	"os"
)

const maxInstallRoutines = 40

const temporalLibPath = "/tmp/scribe/installed_packages"

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

func installSinglePackage(outputLocation string) error {
	log.Info("Package location is", outputLocation)
	cmd := "R CMD INSTALL " + outputLocation + " -l " + temporalLibPath
	log.Debug(cmd)
	result, err := execCommand(cmd, true, true)
	log.Error(result)
	if err != nil {
		log.Error(err)
	}
	return err
}

func InstallPackages(renvLock Renvlock, allDownloadInfo *[]DownloadInfo) {
	err := os.MkdirAll(temporalLibPath, os.ModePerm)
	checkError(err)

	packages := make([]string, 0, len(renvLock.Packages))
	for _, p := range renvLock.Packages {
		packages = append(packages, p.Package)
	}
	deps := getPackageDepsFromCrandbWithChunk(packages)

	depsOrdered := tsort(deps)

	for i := 0; i < len(depsOrdered); i++ {
		v := depsOrdered[i]
		log.Debug(v)
		//installSinglePackage(v.OutputLocation)
	}

	log.Info("Done")
}
