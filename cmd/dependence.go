package cmd

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"math"
	"path/filepath"
	"sort"

	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

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
	filterFields := []string{"Package:", "Version:", "Depends:", "Imports:", "Suggests:"}
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
	return strings.Split(strings.TrimSpace(packageVersion), " ")[0]
}

func getPackageDepsFromSinglePackageLocation(repoLocation string, includeSuggests bool) []string {
	descFilePath := filepath.Join(repoLocation, "DESCRIPTION")
	deps := make([]string, 0)
	if _, err := os.Stat(descFilePath); !os.IsNotExist(err) {
		desc := parseDescriptionFile(descFilePath)
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
		if len(deps) == 0 {
			deps = append(deps, "R")
		}
	}
	return deps
}

func getDependenciesFields(includeSuggests bool) []string {
	res := []string{"Depends", "Imports"}
	if includeSuggests {
		res = append(res, "Suggests")
	}
	return res
}

func getPackageDepsFromCrandbWithChunk(packages []string) map[string][]string {
	chunkSize := 100
	deps := make(map[string][]string)
	for i := 0; i < (len(packages)/chunkSize)+1; i++ {
		fromI := chunkSize * i
		toI := int(math.Min(float64(chunkSize*(i+1)), float64(len(packages))))
		packagesInChunk := packages[fromI:toI]
		depsInChunk := getPackageDepsFromCrandb(packagesInChunk)
		for k, v := range depsInChunk {
			deps[k] = v
		}
	}
	return deps
}

func getPackageDepsFromCrandb(packages []string) map[string][]string {
	depsFields := getDependenciesFields(false)
	url := getCrandbUrl(packages)
	log.Trace("Request for package deps from CranDB on URL: " + url)
	depsJson, _ := request(url)
	var m map[string]map[string]map[string]string
	json.Unmarshal([]byte(depsJson), &m)
	deps := make(map[string][]string)
	for _, p := range packages {
		if m[p] != nil {
			for _, df := range depsFields {
				if m[p][df] != nil {
					for k := range m[p][df] {
						deps[p] = append(deps[p], k)
					}
				}
			}
			if len(deps[p]) == 0 {
				deps[p] = append(deps[p], "R")
			}
		}
	}
	return deps
}

func getPackageDepsFromPackagesFile(packagesFilePath string, packages map[string]struct{}) map[string][]string {
	depFields := getDependenciesFields(false)
	deps := make(map[string][]string)
	if _, err := os.Stat(packagesFilePath); !os.IsNotExist(err) {
		packagesContent, _ := ioutil.ReadFile(packagesFilePath)
		for _, linegroup := range strings.Split(string(packagesContent), "\n\n") {
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

	} else {
		log.Warnf("File %s doesn't exists", packagesFilePath)
	}
	return deps
}

func getPackageDepsFromBioconductor(packages []string) map[string][]string {
	deps := make(map[string][]string)
	packagesSet := make(map[string]struct{}, 0)
	for _, v := range packages {
		packagesSet[v] = struct{}{}
	}
	for _, biocCategory := range bioconductorCategories {
		packageFileLocation := localOutputDirectory + "/package_files/BIOC_PACKAGES_" + strings.ToUpper(strings.ReplaceAll(biocCategory, "/", "_"))
		depsBiocCategory := getPackageDepsFromPackagesFile(packageFileLocation, packagesSet)
		for k := range depsBiocCategory {
			deps[k] = depsBiocCategory[k]
		}
	}
	return deps
}

func getCrandbUrl(packages []string) string {
	acc := ""
	for _, v := range packages {
		if len(acc) > 0 {
			acc += ","
		}
		acc += "%22" + v + "%22"
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

func tsort(graph map[string][]string) (resultOrder []string) {

	allNodesSet := mapset.NewSet[string]()
	revGraph := map[string][]string{}
	for from, tos := range graph {
		allNodesSet.Add(from)
		if len(tos) == 0 {
			resultOrder = append(resultOrder, from)
		} else {
			for _, to := range tos {
				allNodesSet.Add(to)
				revGraph[to] = append(revGraph[to], from)
			}
		}
	}

	allNodes := allNodesSet.ToSlice()
	indegree := make(map[string]int)
	outdegree := make(map[string]int)
	for _, n := range allNodes {
		indegree[n] = 0
		outdegree[n] = 0
	}
	for from, tos := range graph {
		outdegree[from] = len(tos)
	}
	for from, tos := range revGraph {
		indegree[from] = len(tos)
	}

	for to, degree := range outdegree {
		if degree == 0 {
			resultOrder = append(resultOrder, to)
		}
	}
	sort.Strings(resultOrder)

	stack := []string{}

	var dfs func(node string, fvisited map[string]bool, fstack *[]string)
	dfs = func(node string, fvisited map[string]bool, fstack *[]string) {
		fvisited[node] = true
		for _, to := range sortByCounter(outdegree, graph[node]) {
			if fvisited[to] == false {
				dfs(to, fvisited, &*fstack)
			}
		}
		*fstack = append(*fstack, node)
	}

	visited := make(map[string]bool)
	for _, node := range resultOrder {
		visited[node] = true
	}

	allNodes = sortByCounter(outdegree, allNodes)

	for _, node := range allNodes {
		if visited[node] == false {
			dfs(node, visited, &stack)
		}

	}

	for i := 0; i < len(stack); i++ {
		if !slices.Contains(resultOrder, stack[i]) {
			resultOrder = append(resultOrder, stack[i])
		}
	}

	return resultOrder
}
