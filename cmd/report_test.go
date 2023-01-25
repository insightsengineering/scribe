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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_processDownloadInfo(t *testing.T) {
	var allDownloadInfo []DownloadInfo
	readJSON("testdata/downloadInfo.json", &allDownloadInfo)
	downloadStatuses := processDownloadInfo(allDownloadInfo)
	assert.Equal(t, downloadStatuses["httpuv"], "<span class=\"badge bg-danger\">network error</span>")
	assert.Equal(t, downloadStatuses["covr"], "<span class=\"badge bg-danger\">package not found</span>")
	assert.Equal(t, downloadStatuses["scda"], "<span class=\"badge bg-success\">OK</span>")
	assert.Equal(t, downloadStatuses["formatters"], "<span class=\"badge bg-success\">OK</span>")
	assert.Equal(t, downloadStatuses["teal.reporter"], "<span class=\"badge bg-danger\">GitHub clone error</span>")
	assert.Equal(t, downloadStatuses["teal.widgets"], "<span class=\"badge bg-danger\">GitLab clone error</span>")
	assert.Equal(t, downloadStatuses["httr"], "<span class=\"badge bg-danger\">BioC package not found</span>")
}

func Test_processInstallInfo(t *testing.T) {
	var allInstallInfo []InstallResultInfo
	readJSON("testdata/installInfo.json", &allInstallInfo)
	installStatuses := processInstallInfo(allInstallInfo)
	assert.Equal(t, installStatuses["Matrix"], "<a href=\"./logs/install-Matrix.out\"><span class=\"badge bg-success\">OK</span></a>")
	assert.Equal(t, installStatuses["package1"], "<a href=\"./logs/install-package1.out\"><span class=\"badge bg-info text-dark\">skipped</span></a>")
	assert.Equal(t, installStatuses["package2"], "<a href=\"./logs/install-package2.out\"><span class=\"badge bg-danger\">failed</span></a>")
}

func Test_processCheckInfo(t *testing.T) {
	var allCheckInfo []PackageCheckInfo
	readJSON("testdata/checkInfo.json", &allCheckInfo)
	checkStatuses := processCheckInfo(allCheckInfo)
	assert.Equal(t, checkStatuses["package1"], "<a href=\"./logs/check-package1.out\"><span class=\"badge bg-danger\">check error(s)</span></a>")
	assert.Equal(t, checkStatuses["package2"], "<a href=\"./logs/check-package2.out\"><span class=\"badge bg-warning text-dark\">check warning(s)</span></a>")
	assert.Equal(t, checkStatuses["package3"], "<a href=\"./logs/check-package3.out\"><span class=\"badge bg-success\">OK</span></a>")
	assert.Equal(t, checkStatuses["package4"], "<a href=\"./logs/check-package4.out\"><span class=\"badge bg-info text-dark\">check note(s)</span></a>")
}

func Test_processBuildInfo(t *testing.T) {
	var allInstallInfo []InstallResultInfo
	readJSON("testdata/installInfo.json", &allInstallInfo)
	buildStatuses := processBuildInfo(allInstallInfo)
	assert.Equal(t, buildStatuses["Matrix"], "<a href=\"./logs/build-Matrix.out\"><span class=\"badge bg-success\">OK</span></a>")
	assert.Equal(t, buildStatuses["package1"], "<a href=\"./logs/build-package1.out\"><span class=\"badge bg-danger\">failed</span></a>")
	assert.Equal(t, buildStatuses["package2"], "<a href=\"./logs/build-package2.out\"><span class=\"badge bg-success\">OK</span></a>")
}
