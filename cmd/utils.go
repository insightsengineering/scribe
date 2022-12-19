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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slices"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func checkError(err error) {
	if err != nil {
		log.Error(err)
	}
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		checkError(err)
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// Returns number of bytes written to a file
func writeJSON(filename string, j interface{}) int {
	s, err := json.MarshalIndent(j, "", "  ")
	checkError(err)

	err = os.WriteFile(filename, s, 0644) //#nosec
	checkError(err)

	return len(s)
}

func fillEnvFromSystem(envs []string) []string {
	for i, env := range envs {
		if env != "" && !strings.Contains(env, "=") {
			value := os.Getenv(env)
			envs[i] = env + "=" + value
		}
	}
	return envs
}

// Execute a system command
func execCommand(command string, showOutput bool, returnOutput bool, envs []string, file *os.File) (string, error) {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	var parts []string
	preParts := strings.FieldsFunc(command, f)
	for i := range preParts {
		part := preParts[i]
		parts = append(parts, strings.ReplaceAll(part, "'", ""))
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()

	for _, env := range fillEnvFromSystem(envs) {
		if env != "" {
			cmd.Env = append(cmd.Env, env)
		}
	}
	if returnOutput {
		data, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	var stdout, stderr io.Writer
	if file != nil {
		stdout = io.MultiWriter(&stdoutBuf, file)
		stderr = io.MultiWriter(&stderrBuf, file)
	}
	err := cmd.Start()
	if err != nil {
		log.Error(err)
	}

	if file != nil {
		go func() {
			_, errStdout = io.Copy(stdout, stdoutIn)
		}()

		go func() {
			_, errStderr = io.Copy(stderr, stderrIn)
		}()

		if errStdout != nil || errStderr != nil {
			if showOutput {
				log.Fatalln("Failed to capture stdout or stderr!")
			}
			if errStdout != nil {
				return "", errStdout
			}
			return "", errStderr
		}

		err = cmd.Wait()
		outStr, errStr := string(stdoutBuf.String()), string(stderrBuf.Bytes())
		if err != nil {
			if showOutput {
				log.Println(errStr + outStr)
			}
			return errStr + outStr, err
		}
		if showOutput {
			log.Println(outStr)
		}
	}
	return "", nil
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

	//for to, degree := range outdegree {
	//	if degree == 0 {
	//		resultOrder = append(resultOrder, to)
	//	}
	//}
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

func toEmptyMapString(slice []string) map[string]string {
	rmap := make(map[string]string)
	for _, s := range slice {
		rmap[s] = ""
	}
	return rmap
}
