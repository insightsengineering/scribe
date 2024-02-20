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
	"time"
)

// Since functionality dependent on gosigar fails to build on macOS, the definition
// of system metrics routine for macOS does nothing, except for interacting
// with the channel in a way expected by checkPackages().

func systemMetricsRoutine(systemMetricsWaiter chan struct{}) {
system_metrics_loop:
	for {
		select {
		case _ = <-systemMetricsWaiter:
			break system_metrics_loop
		default:
			time.Sleep(1 * time.Second)
		}
	}
	// Signal to checkPackages() that processing has finished.
	systemMetricsWaiter <- struct{}{}
}
