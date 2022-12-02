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
	"bufio"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type SystemInfo struct {
	OperatingSystem string
	Architecture    string
	KernelVersion   string
	PrettyName      string
	SystemPackages  string
	RVersion        string
	Time            string
	EnvVariables    string
}

func parseEtcReleaseFile() string {
	etcRelease, err := os.Open("/etc/os-release")
	checkError(err)
	if err != nil {
		return ""
	}
	defer etcRelease.Close()

	scanner := bufio.NewScanner(etcRelease)
	var prettyName string
	for scanner.Scan() {
		newLine := scanner.Text()
		if strings.HasPrefix(newLine, "PRETTY_NAME=") {
			prettyName = strings.Split(newLine, "=")[1]
			// Remove surrounding quotes
			prettyName = prettyName[1 : len(prettyName)-1]
		}
	}
	return prettyName
}

func parseProcVersionFile() string {
	procVersion, err := os.Open("/proc/version")
	checkError(err)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(procVersion)
	for scanner.Scan() {
		newLine := scanner.Text()
		return newLine
	}
	return ""
}

func getSystemPackages(prettyName string) string {
	if strings.Contains(prettyName, "Ubuntu") ||
		strings.Contains(prettyName, "Debian") {
		out, err := exec.Command("dpkg-query", "-l").CombinedOutput()
		checkError(err)
		return string(out)
	} else if strings.Contains(prettyName, "Fedora") ||
		strings.Contains(prettyName, "CentOS") {
		out, err := exec.Command("yum", "list", "installed").CombinedOutput()
		checkError(err)
		return string(out)
	}
	return ""
}

func getSystemRVersion() string {
	out, err := exec.Command("R", "--version").CombinedOutput()
	checkError(err)
	RVersion := strings.Split(string(out), "\n")[0]
	return RVersion
}

// if regex is not equal to empty string, only environment variables
// with names NOT matching the regex will be returned
func getEnvironmentVariables(regex string) string {
	r, err := regexp.Compile(regex)
	checkError(err)
	var envVariables strings.Builder
	for _, e := range os.Environ() {
		values := strings.Split(e, "=")
		variableName := values[0]
		if regex != "" && r.MatchString(variableName) {
			log.Info("Masking environment variable ", variableName)
		} else {
			envVariables.WriteString(e + "\n")
		}
	}
	return envVariables.String()
}

func getOsInformation(maskingVariableRegex string) {
	var systemInfo SystemInfo
	systemInfo.OperatingSystem = runtime.GOOS
	systemInfo.Architecture = runtime.GOARCH
	systemInfo.Time = time.Now().Format("2006-01-02 15:04:05")
	if systemInfo.OperatingSystem == "linux" {
		systemInfo.KernelVersion = parseProcVersionFile()
		systemInfo.PrettyName = parseEtcReleaseFile()
		systemInfo.SystemPackages = getSystemPackages(systemInfo.PrettyName)
		systemInfo.RVersion = getSystemRVersion()
		systemInfo.EnvVariables = getEnvironmentVariables(maskingVariableRegex)

	} else if systemInfo.OperatingSystem == "windows" {

	}
	writeJSON("systemInfo.json", systemInfo)
}
