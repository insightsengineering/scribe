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
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type SystemInfo struct {
	OperatingSystem string `json:"operatingSystem"`
	Architecture    string `json:"architecture"`
	KernelVersion   string `json:"kernelVersion"`
	PrettyName      string `json:"prettyName"`
	SystemPackages  string `json:"systemPackages"`
	RVersion        string `json:"rVersion"`
	Time            string `json:"time"`
	EnvVariables    string `json:"envVariables"`
	Hostname        string `json:"hostname"`
}

func getSystemRVersion() string {
	out, err := exec.Command(rExecutablePath, "--version").CombinedOutput()
	checkError(err)
	return strings.Split(string(out), "\n")[0]
}

// If regex is not equal to empty string, only environment variables
// with names NOT matching the regex will be returned.
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

func getHostname() string {
	out, err := exec.Command("hostname").CombinedOutput()
	checkError(err)
	return strings.TrimSuffix(string(out), "\n")
}

func getOsInformation(systemInfo *SystemInfo, maskingVariableRegex string) {
	systemInfo.OperatingSystem = runtime.GOOS
	systemInfo.Architecture = runtime.GOARCH
	systemInfo.Time = time.Now().Format("2006-01-02 15:04:05")
	systemInfo.Hostname = getHostname()
	systemInfo.EnvVariables = getEnvironmentVariables(maskingVariableRegex)
	systemInfo.RVersion = getSystemRVersion()
	getSystemDependentInfo(systemInfo)
}
