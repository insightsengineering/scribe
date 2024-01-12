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
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type PackagesData struct {
	PackageName        string `json:"packageName"`
	PackageVersion     string `json:"packageVersion"`
	GitPackageShaOrRef string `json:"gitPackageShaOrRef"`
	DownloadStatusText string `json:"downloadStatusText"`
	InstallStatusText  string `json:"installStatusText"`
	CheckStatusText    string `json:"checkStatusText"`
	BuildStatusText    string `json:"buildStatusText"`
	CheckTime          string `json:"checkTime"`
	PackageRepository  string `json:"packageRepository"`
}

type ReportInfo struct {
	PackagesInformation []PackagesData `json:"packagesInformation"`
	SystemInformation   *SystemInfo    `json:"systemInformation"`
	RenvInformation     RenvInfo       `json:"renvInformation"`
	RenvInformationOld  RenvInfo       `json:"renvInformationOld"`
	TotalCheckTime      string         `json:"totalCheckTime"`
}

type RenvInfo struct {
	RenvFilename string `json:"renvFilename"`
	RenvContents string `json:"renvContents"`
}

const HTMLStatusOK = "<span class=\"badge bg-success\">OK</span>"
const HTMLLinkEnd = "</a>"

// copyFiles copies all files from sourceDirectory to destinationDirectory (not recursively).
// Adds filePrefix prefix to each copied file name.
func copyFiles(sourceDirectory string, filePrefix string, destinationDirectory string) {
	files, err := os.ReadDir(sourceDirectory)
	checkError(err)
	for _, file := range files {
		if !file.IsDir() {
			oldFileName := sourceDirectory + "/" + filepath.Base(file.Name())
			newFileName := destinationDirectory + "/" + filePrefix + filepath.Base(file.Name())
			log.Trace("Copying ", oldFileName, " to ", newFileName)
			data, err := os.ReadFile(oldFileName)
			checkError(err)
			err = os.WriteFile(newFileName, data, 0644) //#nosec
			checkError(err)
		}
	}
}

// processDownloadInfo, for each item from download info JSON, generates HTML code for badge
// in the report corresponding to the package. Returns map from package name to HTML code.
func processDownloadInfo(allDownloadInfo []DownloadInfo) map[string]string {
	downloadStatuses := make(map[string]string)
	for _, p := range allDownloadInfo {
		var downloadStatusText string
		if p.StatusCode != http.StatusOK {
			var statusDescription string
			switch p.StatusCode {
			case -1:
				statusDescription = "BioC package not found"
			case -2:
				statusDescription = "GitHub clone error"
			case -3:
				statusDescription = "GitLab clone error"
			case -4:
				statusDescription = "network error"
			case 404:
				statusDescription = "package not found"
			}
			downloadStatusText = "<span class=\"badge bg-danger\">" + statusDescription + "</span>"
		} else {
			downloadStatusText = HTMLStatusOK
		}
		downloadStatuses[p.PackageName] = downloadStatusText
	}
	return downloadStatuses
}

// processInstallInfo, for each item from installation info JSON, generates HTML code for badge
// in the report corresponding to the installation status of the package. Returns map
// from package name to HTML code.
func processInstallInfo(allInstallInfo []InstallResultInfo) map[string]string {
	installStatuses := make(map[string]string)
	for _, p := range allInstallInfo {
		var installStatusText string
		filePath := "<a href=\"./logs/install-" + filepath.Base(p.LogFilePath) + "\">"
		switch p.Status {
		case InstallResultInfoStatusSucceeded:
			installStatusText = filePath + HTMLStatusOK + HTMLLinkEnd
		case InstallResultInfoStatusFailed:
			installStatusText = filePath + "<span class=\"badge bg-danger\">failed</span></a>"
		case InstallResultInfoStatusBuildFailed:
			// If build failed, there is no link to installation logs.
			installStatusText = "<span class=\"badge bg-danger\">build failed</span>"
		}
		installStatuses[p.PackageName] = installStatusText
	}
	return installStatuses
}

// processBuildInfo, for each item from installation info JSON, generates HTML code for badge
// in the report corresponding to the build status of the package. Returns map from package
// name to HTML code.
func processBuildInfo(allInstallInfo []InstallResultInfo) map[string]string {
	buildStatuses := make(map[string]string)
	for _, p := range allInstallInfo {
		var buildStatusText string
		filePath := "<a href=\"./logs/build-" + filepath.Base(p.BuildLogFilePath) + "\">"
		switch p.BuildStatus {
		case buildStatusSucceeded:
			buildStatusText = filePath + HTMLStatusOK + HTMLLinkEnd
		case buildStatusFailed:
			buildStatusText = filePath + "<span class=\"badge bg-danger\">failed</span></a>"
		}
		buildStatuses[p.PackageName] = buildStatusText
	}
	return buildStatuses
}

// processCheckInfo, for each item from R CMD check info JSON, generates HTML code for badge
// in the report corresponding to the package. Returns map from package name to HTML code
// for both check statuses and check times, as well as total check time.
func processCheckInfo(allCheckInfo []PackageCheckInfo) (map[string]string, map[string]string, string) {
	checkStatuses := make(map[string]string)
	checkTimes := make(map[string]string)
	var totalCheckTime int
	for _, p := range allCheckInfo {
		var checkStatusText string
		filePath := "<a href=\"./logs/check-" + filepath.Base(p.LogFilePath) + "\">"
		switch p.MostSevereCheckItem {
		case "OK":
			checkStatusText = filePath + HTMLStatusOK + HTMLLinkEnd
		case "NOTE":
			checkStatusText = filePath +
				"<span class=\"badge bg-info text-dark\">check note(s)</span></a>"
		case "WARNING":
			checkStatusText = filePath +
				"<span class=\"badge bg-warning text-dark\">check warning(s)</span></a>"
		case "ERROR":
			checkStatusText = filePath +
				"<span class=\"badge bg-danger\">check error(s)</span></a>"
		}
		checkStatuses[p.PackageName] = checkStatusText
		checkTimes[p.PackageName] = strconv.Itoa(p.CheckTime)
		totalCheckTime += p.CheckTime
	}
	return checkStatuses, checkTimes, strconv.Itoa(totalCheckTime)
}

// processReportData returns processed download, installation and check information in a structure that
// can be consumed by Go templating engine.
func processReportData(allDownloadInfo []DownloadInfo, allInstallInfo []InstallResultInfo,
	allCheckInfo []PackageCheckInfo, systemInfo *SystemInfo, reportOutput *ReportInfo,
	renvLock Renvlock, renvLockOld Renvlock, renvLockFilenameOld string) {

	downloadStatuses := processDownloadInfo(allDownloadInfo)
	installStatuses := processInstallInfo(allInstallInfo)
	// Builiding packages is done as part of install step, so build status is stored in installation info structure.
	buildStatuses := processBuildInfo(allInstallInfo)
	checkStatuses, checkTimes, totalCheckTime := processCheckInfo(allCheckInfo)
	reportOutput.TotalCheckTime = totalCheckTime

	// Iterating through download info because it is a superset of install info and check info.
	for _, p := range allDownloadInfo {
		reportOutput.PackagesInformation = append(
			reportOutput.PackagesInformation,
			PackagesData{p.PackageName, p.PackageVersion, p.GitPackageShaOrRef, downloadStatuses[p.PackageName],
				installStatuses[p.PackageName], checkStatuses[p.PackageName], buildStatuses[p.PackageName],
				checkTimes[p.PackageName], p.PackageRepository},
		)
	}
	reportOutput.SystemInformation = systemInfo

	reportOutput.RenvInformation.RenvFilename = renvLockFilename
	indentedValue, err := json.MarshalIndent(renvLock, "", "  ")
	checkError(err)
	reportOutput.RenvInformation.RenvContents = string(indentedValue)

	if renvLockFilenameOld != "" {
		// A copy of original renv.lock has been created since renvLock structure
		// has been updated with new version of packages.
		reportOutput.RenvInformationOld.RenvFilename = renvLockFilenameOld
		indentedValueOld, err := json.MarshalIndent(renvLockOld, "", "  ")
		checkError(err)
		reportOutput.RenvInformationOld.RenvContents = string(indentedValueOld)
	}
}

func writeReport(reportData ReportInfo, outputFile string) {
	funcMap := template.FuncMap{
		// Function required for inserting HTML code into the template.
		"safe": func(s string) template.HTML {
			return template.HTML(s) // #nosec
		},
	}
	tmpl, err := template.New("report_template").Funcs(funcMap).Parse(HTMLReportTemplate)
	checkError(err)
	err = os.MkdirAll(filepath.Dir(outputFile), os.ModePerm)
	checkError(err)
	reportFile, err := os.Create(outputFile)
	checkError(err)
	defer reportFile.Close()
	err = tmpl.ExecuteTemplate(reportFile, "report_template", reportData)
	checkError(err)
}
