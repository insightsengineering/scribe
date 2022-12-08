package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func getMapKeyDiff(originMap map[string]bool, mapskeysToRemove map[string][]string) map[string]bool {
	newmap := make(map[string]bool)
	for k, v := range originMap {
		if mapskeysToRemove[k] == nil {
			newmap[k] = v
		}
	}

	return newmap
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
	log.Trace("Requesting" + url)
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
func getPackageContent() string {
	url := "https://cloud.r-project.org/src/contrib/PACKAGES"
	content, _ := request(url)
	return content
}

func removePackageVersionConstraints(packageVersion string) string {
	noseprator := strings.Split(strings.TrimSpace(packageVersion), " ")[0]
	bracket := strings.Split(noseprator, "(")[0]
	return strings.Split(bracket, ">")[0]
}

func getPackageDepsFromDescriptionFileContent(descriptionFileContent string, includeSuggests bool) []string {
	deps := make([]string, 0)
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
	//if len(deps) == 0 {
	//	deps = append(deps, "R")
	//}
	return deps
}

func getPackageDepsFromSinglePackageLocation(repoLocation string, includeSuggests bool) []string {
	descFilePath := filepath.Join(repoLocation, "DESCRIPTION")
	deps := make([]string, 0)
	if _, err := os.Stat(descFilePath); !os.IsNotExist(err) {
		descriptionFileData, _ := ioutil.ReadFile(descFilePath)
		deps = getPackageDepsFromDescriptionFileContent(string(descriptionFileData), includeSuggests)
	}
	//if len(deps) == 0 {
	//	deps = append(deps, "R")
	//}
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
	f, err := os.Open(tarGzFilePath)
	if err != nil {
		log.Error(err)
	} else {
		defer f.Close()
		gzf, err := gzip.NewReader(f)
		if err != nil {
			log.Error(err)
		} else {
			tarReader := tar.NewReader(gzf)
			for true {
				header, err := tarReader.Next()
				if err == io.EOF || err != nil {
					log.Error(err)
					break
				}
				name := header.Name
				if name != "" && strings.HasSuffix(name, "DESCRIPTION") {
					if header.Typeflag == tar.TypeReg {
						data := make([]byte, header.Size)
						_, err := tarReader.Read(data)
						if err != nil {
							log.Error(err)
						}
						return string(data)
					}
				}
			}
		}
	}
	return ""
}

func getPackageDepsFromTarGz(tarGzFilePath string) []string {
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
		log.Debugf("Getting deps from Crandb service. Chunk #%d", chunkCounter)

		packagesWithVersionInChunk[p] = v
		chunkSize++

		if chunkSize >= chunkMaxSize || (lastChunkNumber == chunkCounter && lastChunkSize == chunkSize) {
			depsInChunk := getPackageDepsFromCrandb(packagesWithVersionInChunk)
			for k, v := range depsInChunk {
				deps[k] = v
			}

			chunkCounter++
			chunkSize = 0
			packagesWithVersionInChunk = make(map[string]string)
		}
	}
	return deps
}

func getPackageDepsFromCrandb(packagesWithVersion map[string]string) map[string][]string {
	depsFields := getDependenciesFields(false)
	url := getCrandbUrl(packagesWithVersion)
	log.Trace("Request for package deps from CranDB on URL: " + url)
	depsJson, _ := request(url)
	var m map[string]map[string]map[string]string
	json.Unmarshal([]byte(depsJson), &m)
	deps := make(map[string][]string)
	for p := range packagesWithVersion {
		if m[p] != nil {
			for _, df := range depsFields {
				if m[p][df] != nil {
					for k := range m[p][df] {
						deps[p] = append(deps[p], k)
					}
				}
			}
			// if len(deps[p]) == 0 {
			// 	deps[p] = append(deps[p], "R")
			// }
		}
	}
	return deps
}

func getPackageDepsFromRepositoryURLs(repositoryUrls []string, packages map[string]bool) map[string][]string {
	deps := make(map[string][]string)
	for _, url := range repositoryUrls {
		depsForUrl := getPackageDepsFromRepositoryURL(url, packages)
		for k, v := range depsForUrl {
			deps[k] = v
		}
	}
	return deps
}

func getPackageDepsFromRepositoryURL(repositoryUrl string, packages map[string]bool) map[string][]string {
	endPoint := "src/contrib/PACKAGES"
	if !strings.HasSuffix(repositoryUrl, endPoint) {
		repositoryUrl = repositoryUrl + "/" + endPoint
	}
	content, err := request(repositoryUrl)
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
				log.Fatalf("error: %v", err)
			} else {
				if len(m) > 1 {
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
					if len(packageDep) == 0 {
						packageDep = append(packageDep, "R")
					}
					deps[packageName] = packageDep
				}
			}

		}
	}
	return deps
}

func getPackageDepsFromPackagesFile(packagesFilePath string, packages map[string]bool) map[string][]string {
	packagesContent, _ := ioutil.ReadFile(packagesFilePath)
	return getPackageDepsFromPackagesFileContent(string(packagesContent), packages)

}

func getPackageDepsFromBioconductor(packages map[string]bool, bioconductorVersion string) map[string][]string {
	deps := make(map[string][]string)

	for _, biocCategory := range bioconductorCategories {
		packageFileLocation := localOutputDirectory + "/package_files/BIOC_PACKAGES_" + strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_"))
		depsBiocCategory := make(map[string][]string)
		if _, err := os.Stat(packageFileLocation); !os.IsNotExist(err) {
			depsBiocCategory = getPackageDepsFromPackagesFile(packageFileLocation, packages)
		} else {
			log.Warnf("File %s doesn't exists", packageFileLocation)
			url := bioConductorURL + "/" + bioconductorVersion + "/" + biocCategory
			depsBiocCategory = getPackageDepsFromRepositoryURL(url, packages)
		}

		for k := range depsBiocCategory {
			deps[k] = depsBiocCategory[k]
		}
	}
	return deps
}

func getPackageDeps(
	packages []string,
	bioconductorVersion string,
	allDownloadInfo *[]DownloadInfo,
	reposUrls []string,
	packagesLocation map[string]struct{ PackageType, Location string },
) map[string][]string {

	packagesSet := make(map[string]bool)
	for _, p := range packages {
		packagesSet[p] = true
	}

	deps := getPackageDepsFromCrandbWithChunk(toEmptyMapString(packages))
	depsBioc := getPackageDepsFromBioconductor(packagesSet, bioconductorVersion)
	for k, v := range depsBioc {
		deps[k] = v
	}

	depsRepos := getPackageDepsFromRepositoryURLs(reposUrls, packagesSet)
	for k, v := range depsRepos {
		deps[k] = v
	}

	for pName, pInfo := range packagesLocation {
		if pInfo.PackageType == "git" {
			if _, err := os.Stat(pInfo.Location); !os.IsNotExist(err) {
				packageDeps := getPackageDepsFromSinglePackageLocation(pInfo.Location, true)
				deps[pName] = packageDeps
			} else {
				log.Errorf("Directory %s for package %s does not exist", pInfo.PackageType, pInfo.Location)
			}
		}
	}

	packagesNoDeps := getMapKeyDiff(packagesSet, deps)
	for k := range packagesNoDeps {
		info := packagesLocation[k]
		if info.PackageType == "tar.gz" {
			targzDeps := getPackageDepsFromTarGz(info.Location)
			deps[k] = targzDeps
		}
	}
	return deps
}

func getCrandbUrl(packagesWithVersion map[string]string) string {
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
