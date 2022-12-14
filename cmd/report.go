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
)

type ReportInfo struct {
	PackageName         string `json:"packageName"`
	PackageVersion      string `json:"packageVersion"`
	DownloadStatusText  string `json:"downloadStatusText"`
	DownloadHelp        string `json:"downloadHelp"`
}

func preprocessReportData(allDownloadInfo []DownloadInfo, reportData *[]ReportInfo) {
	var downloadStatusText string
	var downloadHelp string
	for _, p := range allDownloadInfo {
		if p.StatusCode < 0 {
			downloadStatusText = "❌"
			downloadHelp = "data-toggle=\"tooltip\" data-placement=\"right\" title=\"" + fmt.Sprint(p.StatusCode) + "\""
		} else {
			downloadStatusText = "✅"
			downloadHelp = ""
		}
		*reportData = append(
			*reportData,
			ReportInfo{p.PackageName, p.PackageVersion, downloadStatusText, downloadHelp},
		)
	}
}

func writeReport(reportData []ReportInfo, outputFile string) {
	funcMap := template.FuncMap{
		"safe": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
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
