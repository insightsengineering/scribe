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
	"strings"
	yaml "gopkg.in/yaml.v3"
)

func getPackagesFileFromUrl(url string) {
	content, err := request(url)
	if err != nil {
		log.Errorf("Failed to get package content for URL %s", url)
	}
	// log.Debug(content)
	for _, lineGroup := range strings.Split(content, "\n\n") {
		firstLine := strings.Split(lineGroup, "\n")[0]
		packageName := strings.ReplaceAll(firstLine, "Package: ", "")
		log.Debug(packageName)
		m := make(map[string]string)
		err := yaml.Unmarshal([]byte(lineGroup), &m)
		if err != nil {
			log.Error("Error reading package data from PACKAGES: %v", err)
		}
		log.Debug(m)
	}
}
