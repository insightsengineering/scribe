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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	locksmith "github.com/insightsengineering/locksmith/cmd"
	yaml "gopkg.in/yaml.v3"
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

func clearCachedData() {
	err := os.RemoveAll("/tmp/scribe")
	checkError(err)
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

func readJSON(filename string, j interface{}) {
	log.Debug("Reading ", filename)
	byteValue, err := os.ReadFile(filename)
	checkError(err)

	err = json.Unmarshal(byteValue, j)
	checkError(err)
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

func getTimeMinutesAndSeconds(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	remainderSeconds := seconds % 60
	return fmt.Sprintf("%dm%ds", minutes, remainderSeconds)
}

// Execute a system command
// nolint: gocyclo
func execCommand(command string, returnOutput bool, envs []string, file *os.File, escapeHTMLTags bool) (string, error) {
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

	// nolint: gosec
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()

	for _, env := range fillEnvFromSystem(envs) {
		if env != "" {
			cmd.Env = append(cmd.Env, env)
		}
	}
	if returnOutput {
		data, err := cmd.Output()
		return string(data), err
	}

	log.Trace("Command to execute: ", cmd)
	out, errCombinedOutput := cmd.CombinedOutput()
	checkError(errCombinedOutput)

	outStr := string(out)

	if escapeHTMLTags {
		outStr = strings.ReplaceAll(outStr, "<", "&lt;")
		outStr = strings.ReplaceAll(outStr, ">", "&gt;")
	}

	_, errWriteString := file.WriteString(outStr)
	checkError(errWriteString)

	return outStr, errCombinedOutput
}

func parseDescriptionFile(descriptionFilePath string) map[string]string {
	jsonFile, err := os.ReadFile(descriptionFilePath)
	checkError(err)
	cleaned := locksmith.CleanDescriptionOrPackagesEntry(string(jsonFile), true)
	packageMap := make(map[string]string)
	err = yaml.Unmarshal([]byte(cleaned), &packageMap)
	checkError(err)
	return packageMap
}
