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
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockedDownloadTextFile(url string, _ map[string]string) (int64, string, error) {
	switch {
	case url == "https://repository1.example.com/src/contrib/PACKAGES":
		return 0, `Package: package1
Version: 1.0.0
Imports: package2, package3 (>= 1.0.2)
Suggests: package5

Package: package2
Version: 1.0.0
Imports: package3
Depends: package4
`, nil
	case url == "https://repository2.example.com/src/contrib/PACKAGES":
		return 0, `Package: package3
Version: 1.0.0
Depends: package4

Package: package4
Version: 2.0.0
`, nil
	case url == "https://cloud.r-project.org/src/contrib/PACKAGES":
		return 0, `Package: package5
Version: 1.2.3
Imports: package1
`, nil
	}
	return 0, "", nil
}

func Test_getDepsFromPackagesFiles(t *testing.T) {
	rPackages := make(map[string]Rpackage)
	downloadedPackages := make(map[string]DownloadedPackage)
	packageDependencies := make(map[string][]string)
	rPackages["package1"] = Rpackage{"package1", "", "", "Repository1", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package2"] = Rpackage{"package2", "", "", "Repository1", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package3"] = Rpackage{"package3", "", "", "Repository2", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package4"] = Rpackage{"package4", "", "", "Repository2", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package5"] = Rpackage{"package5", "", "", "UndefinedRepository", "", "", []string{}, "", "", "", "", "", ""}
	downloadedPackages["package1"] = DownloadedPackage{"", "", "Repository1", "/tmp/scribe/downloaded_packages/package_archives/package1_1.0.0.tar.gz"}
	downloadedPackages["package2"] = DownloadedPackage{"", "", "Repository1", "/tmp/scribe/downloaded_packages/package_archives/package2_1.0.0.tar.gz"}
	downloadedPackages["package3"] = DownloadedPackage{"", "", "Repository2", "/tmp/scribe/downloaded_packages/package_archives/package3_1.0.0.tar.gz"}
	downloadedPackages["package5"] = DownloadedPackage{"", "", "UndefinedRepository", "/tmp/scribe/downloaded_packages/package_archives/package5_1.2.3.tar.gz"}
	rRepositories := []Rrepository{
		{"Repository1", "https://repository1.example.com"},
		{"Repository2", "https://repository2.example.com"},
	}
	getDepsFromPackagesFiles(rPackages, rRepositories, downloadedPackages, packageDependencies,
		mockedDownloadTextFile, []string{"UndefinedRepository"})
	assert.Equal(t, packageDependencies["package1"], []string{"package2", "package3"})
	assert.Equal(t, packageDependencies["package2"], []string{"package3"})
	assert.Equal(t, len(packageDependencies["package3"]), 0)
	assert.Equal(t, len(packageDependencies["package4"]), 0)
	assert.Equal(t, packageDependencies["package5"], []string{"package1"})
}

func Test_getDepsFromDescriptionFiles(t *testing.T) {
	rPackages := make(map[string]Rpackage)
	downloadedPackages := make(map[string]DownloadedPackage)
	packageDependencies := make(map[string][]string)
	rPackages["package1"] = Rpackage{"package1", "", "", "", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package2"] = Rpackage{"package2", "", "", "", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package3"] = Rpackage{"package3", "", "", "", "", "", []string{}, "", "", "", "", "", ""}
	rPackages["package4"] = Rpackage{"package4", "", "", "", "", "", []string{}, "", "", "", "", "", ""}
	downloadedPackages["package1"] = DownloadedPackage{"", "", "GitHub", "testdata/package1"}
	downloadedPackages["package2"] = DownloadedPackage{"", "", "GitLab", "testdata/package2"}
	downloadedPackages["package3"] = DownloadedPackage{"", "", "GitHub", "testdata/package3"}
	downloadedPackages["package4"] = DownloadedPackage{"", "", "GitLab", ""}
	getDepsFromDescriptionFiles(rPackages, downloadedPackages, packageDependencies)
	assert.Equal(t, packageDependencies["package1"], []string{"package2", "package3"})
	assert.Equal(t, packageDependencies["package2"], []string{"package3"})
	assert.Equal(t, len(packageDependencies["package3"]), 0)
	assert.Equal(t, len(packageDependencies["package4"]), 0)
}
