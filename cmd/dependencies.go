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
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func getMapKeyDiffOrEmpty(originMap map[string]bool, mapskeysToRemove map[string][]string) map[string]bool {
	newmap := make(map[string]bool)
	for k, v := range originMap {
		if mapskeysToRemove[k] == nil || len(mapskeysToRemove[k]) == 0 ||
			(len(mapskeysToRemove[k]) == 1 && mapskeysToRemove[k][0] == "") {
			newmap[k] = v
		}
	}

	return newmap
}

func parseDescriptionFile(descriptionFilePath string) map[string]string {
	log.Tracef("Parsing DESCRIPTION file: %s", descriptionFilePath)
	jsonFile, err := os.ReadFile(descriptionFilePath)
	if err != nil {
		log.Errorf("Error reading DESCRIPTION file %v", err)
		return map[string]string{}
	}
	return parseDescription(string(jsonFile))
}

func parseDescription(description string) map[string]string {
	cleaned := cleanDescription(description)
	m := make(map[string]string)
	err := yaml.Unmarshal([]byte(cleaned), &m)
	if err != nil {
		log.Fatalf("Error in parseDescription: %v", err)
	}
	return m
}

func cleanDescription(description string) string {
	lines := strings.Split(description, "\n")
	filterFields := []string{"Package:", "Version:", "Depends:", "Imports:", "Suggests:", "LinkingTo:"}
	continuation := false
	content := ""
	for _, line := range lines {
		findField := false
		for _, filed := range filterFields {
			if strings.HasPrefix(line, filed) {
				content += line + "\n"
				continuation = true
				findField = true
				break
			}
		}
		if findField {
			continue
		}

		// nolint: gocritic
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

func request(url string) (string, error) {
	tr := &http.Transport{ // #nosec
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec
	} // #nosec
	client := &http.Client{Transport: tr}
	log.Tracef("Requesting %s" + url)
	resp, errc := client.Get(url)
	checkError(errc)

	if errc == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			b, errr := io.ReadAll(resp.Body)
			if errr != nil {
				log.Error(errr)
				return "", errr
			}
			return string(b), nil
		}
	}
	return "", errc
}
func getPackageContent() string {
	url := "https://cloud.r-project.org/src/contrib/PACKAGES"
	content, err := request(url)
	if err != nil {
		log.Errorf("Failed to get package content for URL %s", url)
	}
	return content
}

func removePackageVersionConstraints(packageVersion string) string {
	noseprator := strings.Split(strings.TrimSpace(packageVersion), " ")[0]
	bracket := strings.Split(noseprator, "(")[0]
	return strings.Split(bracket, ">")[0]
}

func getPackageDepsFromDescriptionFileContent(descriptionFileContent string, includeSuggests bool) []string {
	deps := make([]string, 0)
	if descriptionFileContent != "" {
		desc := parseDescription(descriptionFileContent)
		fields := getDependenciesFields(includeSuggests)
		for _, field := range fields {
			fieldLine := desc[field]
			for _, pversionConstraints := range strings.Split(fieldLine, ",") {
				p := removePackageVersionConstraints(pversionConstraints)
				if p != "" {
					deps = append(deps, p)
				}
			}
		}
	} else {
		log.Warnf("Cannot get package dependencies from empty string")
	}
	return deps
}

func getPackageDepsFromSinglePackageLocation(repoLocation string, includeSuggests bool) []string {
	descFilePath := filepath.Join(repoLocation, "DESCRIPTION")
	deps := make([]string, 0)
	if _, err := os.Stat(descFilePath); !os.IsNotExist(err) {
		descriptionFileData, err := os.ReadFile(descFilePath)
		if err != nil {
			log.Errorf("Cannot read DESCRIPTION file %s", descFilePath)
		}
		deps = getPackageDepsFromDescriptionFileContent(string(descriptionFileData), includeSuggests)
	}
	// TODO: What does this mean?
	log.Tracef("Filled %d packages with dependencies from SinglePackageLocation", len(deps))
	return deps
}

func getDependenciesFields(includeSuggests bool) []string {
	res := []string{"Depends", "Imports", "LinkingTo"}
	if includeSuggests {
		res = append(res, "Suggests")
	}
	return res
}

func getDescriptionFileContentFromTargz(tarGzFilePath string) string {
	res := ""
	f, err := os.Open(tarGzFilePath)
	if err != nil {
		log.Error("Cannot open %s", tarGzFilePath, ". Error:", err)
		log.Error(err)
	} else {
		defer f.Close()
		gzf, err := gzip.NewReader(f)
		if err != nil {
			log.Tracef("Cannot read %v", f)
			log.Error(err)
		} else {
			tarReader := tar.NewReader(gzf)
			for {
				header, err := tarReader.Next()
				if err == io.EOF || err != nil {
					log.Tracef("Reached EOF for %v %s", tarReader, tarGzFilePath)
					log.Error(err)
					break
				}
				name := header.Name
				if name != "" && strings.HasSuffix(name, "DESCRIPTION") {
					if header.Typeflag == tar.TypeReg {
						for {
							data := make([]byte, header.Size)
							_, err := tarReader.Read(data)
							res += strings.Trim(string(data), "\x00")
							if err != nil {
								if err != io.EOF {
									log.Tracef("Cannot read DESCRIPTION file from %s", tarGzFilePath)
									log.Error(err)
								}
								return res
							}
						}
					}
				}
			}
		}
	}
	return res
}

func getPackageDepsFromTarGz(tarGzFilePath string) []string {
	log.Tracef("Getting dependencies from %s", tarGzFilePath)
	descContent := getDescriptionFileContentFromTargz(tarGzFilePath)
	return getPackageDepsFromDescriptionFileContent(descContent, false)
}

func getPackageDepsFromCrandbWithChunk(packagesWithVersion map[string]string) map[string][]string {
	chunkMaxSize := 100
	deps := make(map[string][]string)
	chunkCounter := 0
	chunkSize := 0
	packagesWithVersionInChunk := make(map[string]string)
	lastChunkNumber := len(packagesWithVersion) / chunkMaxSize
	lastChunkSize := len(packagesWithVersion) % chunkMaxSize
	for p, v := range packagesWithVersion {

		packagesWithVersionInChunk[p] = v
		chunkSize++

		if chunkSize >= chunkMaxSize || (lastChunkNumber == chunkCounter && lastChunkSize == chunkSize) {
			log.Tracef("Getting deps from CRANDB service. Chunk #%d. Packages: %d", chunkCounter, len(packagesWithVersionInChunk))
			depsInChunk := getPackageDepsFromCrandb(packagesWithVersionInChunk)
			for k, v := range depsInChunk {
				deps[k] = v
			}
			chunkCounter++
			chunkSize = 0
			packagesWithVersionInChunk = make(map[string]string)
		}
	}
	// TODO: What does this mean?
	log.Tracef("Filled %d packages with dependencies from CrandbWithChunk", len(deps))
	return deps
}

func getPackageDepsFromCrandb(packagesWithVersion map[string]string) map[string][]string {
	depsFields := getDependenciesFields(false)
	url := getCrandbURL(packagesWithVersion)
	log.Trace("Requesting dependency info from: ", url)
	depsJSON, err := request(url)
	checkError(err)
	deps := make(map[string][]string)
	var m map[string]map[string]map[string]string
	errunmarshal := json.Unmarshal([]byte(depsJSON), &m)
	if errunmarshal != nil {
		log.Error(errunmarshal)
	}

	for k, v := range packagesWithVersion {
		p := k
		if v != "" {
			p += "-" + v
		}
		if m[p] != nil {
			for _, df := range depsFields {
				if m[p][df] != nil {
					for d := range m[p][df] {
						deps[k] = append(deps[k], d)
					}
				}
			}
		}
	}
	// TODO: Clean this up
	log.Tracef("Get getPackageDepsFromCrandb %v", deps)
	return deps
}

func getPackageDepsFromRepositoryURLs(repositoryURLs []string, packages map[string]bool) map[string][]string {
	deps := make(map[string][]string)
	for _, url := range repositoryURLs {
		depsForURL := getPackageDepsFromRepositoryURL(url, packages)
		for k, v := range depsForURL {
			deps[k] = v
		}
	}
	// TODO: What does this mean?
	log.Tracef("Filled %d packages with dependencies from Repository URLs", len(deps))
	return deps
}

func getPackageDepsFromRepositoryURL(repositoryURL string, packages map[string]bool) map[string][]string {
	endPoint := "src/contrib/PACKAGES"
	if !strings.HasSuffix(repositoryURL, endPoint) {
		repositoryURL = repositoryURL + "/" + endPoint
	}
	content, err := request(repositoryURL)
	if err != nil {
		log.Error(err)
	} else {
		deps := getPackageDepsFromPackagesFileContent(content, packages)
		return deps
	}

	return nil
}

func getPackageDepsFromPackagesFileContent(packagesFileContent string, packages map[string]bool) map[string][]string {
	deps := make(map[string][]string)
	depFields := getDependenciesFields(false)
	for _, linegroup := range strings.Split(packagesFileContent, "\n\n") {
		firstLine := strings.Split(linegroup, "\n")[0]
		packageName := strings.ReplaceAll(firstLine, "Package: ", "")
		if _, ok := packages[packageName]; ok {
			m := make(map[string]string)
			err := yaml.Unmarshal([]byte(linegroup), &m)
			if err != nil {
				// TODO: Clean this up
				log.Fatalf("error in getPackageDepsFromPackagesFileContent: %v", err)
			} else if len(m) > 1 {
				packageDep := make([]string, 0)
				for _, field := range depFields {
					fieldLine := m[field]
					for _, pversionConstraints := range strings.Split(fieldLine, ",") {
						p := removePackageVersionConstraints(pversionConstraints)
						if p != "" {
							packageDep = append(packageDep, p)
						}
					}
				}
				deps[packageName] = packageDep
			}
		}
	}
	// TODO: What does this mean?
	log.Tracef("Filled %d packages with dependencies from PackagesFileContent", len(deps))
	return deps
}

func getPackageDepsFromPackagesFile(packagesFilePath string, packages map[string]bool) map[string][]string {
	packagesContent, err := os.ReadFile(packagesFilePath)
	if err != nil {
		log.Errorf("Cannot read file %s", packagesFilePath)
	}
	return getPackageDepsFromPackagesFileContent(string(packagesContent), packages)

}

func getPackageDepsFromBioconductor(packages map[string]bool, bioconductorVersion string) map[string][]string {
	deps := make(map[string][]string)

	for _, biocCategory := range bioconductorCategories {
		packageFileLocation := localOutputDirectory + "/package_files/BIOC_PACKAGES_" + strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_"))
		var depsBiocCategory map[string][]string
		if _, err := os.Stat(packageFileLocation); !os.IsNotExist(err) {
			depsBiocCategory = getPackageDepsFromPackagesFile(packageFileLocation, packages)
		} else {
			log.Warnf("File %s does not exist", packageFileLocation)
			url := bioConductorURL + "/" + bioconductorVersion + "/" + biocCategory
			depsBiocCategory = getPackageDepsFromRepositoryURL(url, packages)
		}

		for k := range depsBiocCategory {
			deps[k] = depsBiocCategory[k]
		}
	}
	// TODO: What does this mean?
	log.Tracef("Filled %d packages with dependencies from Bioconductor", len(deps))
	return deps
}

func getPackageDeps(
	rpackages map[string]Rpackage,
	bioconductorVersion string,
	reposURLs []string,
	packagesLocation map[string]struct{ PackageType, Location string },
) map[string][]string {
	log.Debugf("Getting package dependencies for %d packages", len(rpackages))
	packagesSet := make(map[string]bool)
	packagesWithVersion := make(map[string]string)
	for k, v := range rpackages {
		packagesSet[k] = true
		packagesWithVersion[k] = v.Version
	}

	deps := getPackageDepsFromCrandbWithChunk(packagesWithVersion)
	depsBioc := getPackageDepsFromBioconductor(packagesSet, bioconductorVersion)
	for k, v := range depsBioc {
		deps[k] = v
	}

	depsRepos := getPackageDepsFromRepositoryURLs(reposURLs, packagesSet)
	for k, v := range depsRepos {
		// it will not overwrite if package already exists
		if _, ok := deps[k]; !ok {
			deps[k] = v
		}
	}

	for pName, pInfo := range packagesLocation {
		if pInfo.PackageType == gitConst {
			if _, err := os.Stat(pInfo.Location); !os.IsNotExist(err) {
				packageDeps := getPackageDepsFromSinglePackageLocation(pInfo.Location, true)
				deps[pName] = packageDeps
			} else {
				log.Errorf("Directory %s for package %s does not exist", pInfo.PackageType, pInfo.Location)
			}
		}
	}

	packagesNoDeps := getMapKeyDiffOrEmpty(packagesSet, deps)
	for k := range packagesNoDeps {
		info := packagesLocation[k]
		if info.PackageType == targzExtensionFile {
			log.Debug("Getting dependencies for ", k, " packages")
			targzDeps := getPackageDepsFromTarGz(info.Location)
			deps[k] = targzDeps
		}
	}
	// TODO: What does this mean?
	log.Debugf("Found %d packages with dependencies", len(deps))
	return deps
}

func getCrandbURL(packagesWithVersion map[string]string) string {
	acc := ""
	for k, v := range packagesWithVersion {
		if len(acc) > 0 {
			acc += ","
		}
		if v != "" {
			acc += "%22" + k + "-" + v + "%22"
		} else {
			acc += "%22" + k + "%22"
		}
	}
	return "https://crandb.r-pkg.org/-/versions?keys=[" + acc + "]"
}

func sortByCounter(counter map[string]int, nodes []string) []string {
	sort.Slice(nodes, func(i, j int) bool {
		if counter[nodes[i]] == counter[nodes[j]] {
			return nodes[i] < nodes[j]
		}
		return counter[nodes[i]] < counter[nodes[j]]
	})
	return nodes
}

func isDependencyFulfilled(packageName string, dependency map[string][]string, installedPackagesWithVersion map[string]string) bool {
	// TODO: What does this mean?
	log.Tracef("Checking if package %s has fulfilled dependencies", packageName)
	deps := dependency[packageName]
	if len(deps) > 0 {
		for _, dep := range deps {
			if _, ok := installedPackagesWithVersion[dep]; !ok {
				log.Tracef("Not all dependencies are installed for package %s. Dependency not installed: %s", packageName, dep)
				return false
			}
		}
	}
	return true
}
