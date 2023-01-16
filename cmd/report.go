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
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PackagesData struct {
	PackageName        string `json:"packageName"`
	PackageVersion     string `json:"packageVersion"`
	DownloadStatusText string `json:"downloadStatusText"`
	InstallStatusText  string `json:"installStatusText"`
	CheckStatusText    string `json:"checkStatusText"`
}

type ReportInfo struct {
	PackagesInformation []PackagesData `json:"packagesInformation"`
	SystemInformation   *SystemInfo    `json:"systemInformation"`
}

const HTMLStatusOK = "<span class=\"badge bg-success\">OK</span>"

// Copies all files from sourceDirectory to destinationDirectory.
// Adds filePrefix prefix to each copied file name.
func copyLogFiles(sourceDirectory string, filePrefix string, destinationDirectory string) {
	files, err := os.ReadDir(sourceDirectory)
	checkError(err)
	for _, file := range files {
		if !file.IsDir() {
			oldFileName := sourceDirectory + "/" + filepath.Base(file.Name())
			newFileName := destinationDirectory + "/" + filePrefix + filepath.Base(file.Name())
			log.Debugf("Copying %s to %s.", oldFileName, newFileName)
			data, err := os.ReadFile(oldFileName)
			checkError(err)
			err = os.WriteFile(newFileName, data, 0644) //#nosec
			checkError(err)
		}
	}
}

func preprocessReportData(allDownloadInfo []DownloadInfo, allInstallInfo []InstallResultInfo,
	allCheckInfo []PackageCheckInfo, systemInfo *SystemInfo, reportOutput *ReportInfo) {
	rand.Seed(time.Now().UnixNano())
	downloadStatuses := make(map[string]string)
	installStatuses := make(map[string]string)
	checkStatuses := make(map[string]string)
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
	for _, p := range allInstallInfo {
		var installStatusText string
		filePath := "<a href=\"./logs/install-" + filepath.Base(p.LogFilePath) + "\">"
		switch p.Status {
		case InstallResultInfoStatusSucceeded:
			installStatusText = filePath + HTMLStatusOK + "</a>"
		case InstallResultInfoStatusSkipped:
			installStatusText = filePath + "<span class=\"badge bg-info text-dark\">skipped</span></a>"
		case InstallResultInfoStatusFailed:
			installStatusText = filePath + "<span class=\"badge bg-danger\">failed</span></a>"
		}
		installStatuses[p.PackageName] = installStatusText
	}
	for _, p := range allCheckInfo {
		var checkStatusText string
		filePath := "<a href=\"./logs/check-" + filepath.Base(p.LogFilePath) + "\">"
		switch p.MostSevereCheckItem {
		case "OK":
			checkStatusText = filePath + HTMLStatusOK + "</a>"
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
	}
	// TODO can it happen that allDownloadInfo, allCheckInfo and allInstallInfo will have different sets of keys?
	for _, p := range allDownloadInfo {
		reportOutput.PackagesInformation = append(
			reportOutput.PackagesInformation,
			PackagesData{p.PackageName, p.PackageVersion, downloadStatuses[p.PackageName],
				installStatuses[p.PackageName], checkStatuses[p.PackageName]},
		)
	}
	reportOutput.SystemInformation = systemInfo
	reportOutput.SystemInformation.SystemPackages = strings.ReplaceAll(
		reportOutput.SystemInformation.SystemPackages,
		"\n", "<br />")
	reportOutput.SystemInformation.EnvVariables = strings.ReplaceAll(
		reportOutput.SystemInformation.EnvVariables,
		"\n", "<br />")
}

func writeReport(reportData ReportInfo, outputFile string, templateFile string) {
	funcMap := template.FuncMap{
		// Function required for inserting HTML code into the template.
		"safe": func(s string) template.HTML {
			return template.HTML(s) // #nosec
		},
	}
	tmpl, err := template.New(filepath.Base(templateFile)).Funcs(funcMap).ParseFiles(templateFile)
	checkError(err)
	err = os.MkdirAll(filepath.Dir(outputFile), os.ModePerm)
	checkError(err)
	reportFile, err := os.Create(outputFile)
	checkError(err)
	defer reportFile.Close()
	err = tmpl.ExecuteTemplate(reportFile, filepath.Base(templateFile), reportData)
	checkError(err)
}
