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
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxInstallRoutines = 40

const temporalLibPath = "/tmp/scribe/installed_packages"

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

func installSinglePackage(packageName string) error {

	return nil
}
func InstallPackages(allDownloadInfo *[]DownloadInfo) {
	err := os.MkdirAll(temporalLibPath, os.ModePerm)
	checkError(err)

	for i := 0; i < len(*allDownloadInfo); i++ {
		v := (*allDownloadInfo)[i]
		log.Debug(v)
		log.Info("Package location is", v.OutputLocation)
		cmd := "R CMD INSTALL " + v.OutputLocation + " -l " + temporalLibPath
		log.Debug(cmd)
		result, err := execCommand(cmd, true, true)
		log.Error(result)
		if err != nil {
			log.Error(err)
		}
	}

	log.Info("Done")
}

func parseDescriptionFile(descriptionFilePath string) map[string]string {
	jsonFile, _ := ioutil.ReadFile(descriptionFilePath)
	return parseDescription(string(jsonFile))
}

func parseDescription(description string) map[string]string {
	cleaned := cleanDescription(description)
	m := make(map[string]string)
	err := yaml.Unmarshal([]byte(cleaned), &m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return m
}

func cleanDescription(description string) string {
	lines := strings.Split(description, "\n")
	filterFields := []string{"Version", "Depends", "Imports", "Suggests"}
	continuation := false
	content := ""
	for _, line := range lines {
		for _, filed := range filterFields {
			if strings.HasPrefix(line, filed) {
				content += line + "\n"
				continuation = true
				break
			}
		}

		if continuation && strings.HasPrefix(line, " ") {
			content += line + "\n"
		} else if line == "\n" {
			content += "\n"
		} else {
			continuation = false
		}
	}
	return content
}

func getPackageContent() (string, error) {
	url := "https://cloud.r-project.org/src/contrib/PACKAGES"

	tr := &http.Transport{ // #nosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec
	} // #nosec
	client := &http.Client{Transport: tr}
	resp, err := client.Get(url)
	checkError(err)

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}
			return string(b), nil
		}
	}
	return "", err
}

func getPackageDepsFromPackagesFile(packages []string) {
	_, err := getPackageContent()
	checkError(err)

}
