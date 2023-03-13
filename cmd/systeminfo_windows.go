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
	"fmt"
	"golang.org/x/sys/windows/registry"
	"strconv"
)

func getSystemDependentInfo(systemInfo *SystemInfo) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	checkError(err)
	defer k.Close()

	productName, _, err := k.GetStringValue("ProductName")
	systemInfo.PrettyName = productName
	checkError(err)
	majorVersionNumber, _, err := k.GetIntegerValue("CurrentMajorVersionNumber")
	checkError(err)
	minorVersionNumber, _, err := k.GetIntegerValue("CurrentMinorVersionNumber")
	checkError(err)
	currentBuild, _, err := k.GetStringValue("CurrentBuild")
	checkError(err)
	systemInfo.KernelVersion = fmt.Sprintf(
		"%s.%s.%s",
		strconv.Itoa(int(majorVersionNumber)), strconv.Itoa(int(minorVersionNumber)), currentBuild,
	)
}
