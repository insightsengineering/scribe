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
	"os"
	"path/filepath"
	"fmt"
	"net/http"
	"strings"
)

type PackagesData struct {
	PackageName         string `json:"packageName"`
	PackageVersion      string `json:"packageVersion"`
	DownloadStatusText  string `json:"downloadStatusText"`
}

type ReportInfo struct {
	PackagesInformation []PackagesData `json:"packagesInformation"`
	SystemInformation   *SystemInfo   `json:"systemInformation"`
}

func preprocessReportData(allDownloadInfo []DownloadInfo, systemInfo *SystemInfo, reportOutput *ReportInfo) {
	var downloadStatusText string
	for _, p := range allDownloadInfo {
		if p.StatusCode != http.StatusOK {
			var statusDescription string
			switch p.StatusCode {
			case -1: statusDescription = "package version could not be found in any BioConductor repository"
			case -2: statusDescription = "error during cloning of GitHub repository"
			case -3: statusDescription = "error during cloning of GitLab repository"
			case -4: statusDescription = "network error during package download"
			case 404: statusDescription = "package not found"
			}
			downloadStatusText = "<button type=\"button\" class=\"btn\"" +
				"data-bs-toggle=\"tooltip\" data-bs-placement=\"left\" title=\"" +
				fmt.Sprintf("Status %s: %s", fmt.Sprint(p.StatusCode), statusDescription) + "\">❌</button>"
		} else {
			downloadStatusText = "<button type=\"button\" class=\"btn\" disabled>✅</button>"
		}
		reportOutput.PackagesInformation = append(
			reportOutput.PackagesInformation,
			PackagesData{p.PackageName, p.PackageVersion, downloadStatusText},
		)
	}
	reportOutput.SystemInformation = systemInfo
	reportOutput.SystemInformation.SystemPackages = strings.Replace(
		reportOutput.SystemInformation.SystemPackages,
		"\n", "<br />", -1)
	reportOutput.SystemInformation.EnvVariables = strings.Replace(
		reportOutput.SystemInformation.EnvVariables,
		"\n", "<br />", -1)
}

func writeReport(reportData ReportInfo, outputFile string) {
	funcMap := template.FuncMap{
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		 },
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles("cmd/report/index.html")
	checkError(err)
	err = os.MkdirAll(filepath.Dir(outputFile), os.ModePerm)
	checkError(err)
	reportFile, err := os.Create(outputFile)
	checkError(err)
	defer reportFile.Close()
	err = tmpl.ExecuteTemplate(reportFile, "index.html", reportData)
	checkError(err)
}
