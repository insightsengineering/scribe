//go:build !darwin

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

// Functionality dependent on gosigar fails to build on macOS.

package cmd

import (
	"os"
	"runtime"
	"strings"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
	"github.com/gocarina/gocsv"
)

func getMiB(numberOfBytes uint64) uint64 {
	return numberOfBytes / (1024 * 1024)
}

type SystemMetrics struct {
	ElapsedTimeSeconds      float64 `csv:"elapsed_time_seconds" json:"elapsed_time_seconds"`
	RProcessesMemory        uint64  `csv:"r_processes_memory" json:"r_processes_memory"`
	ChromiumProcessesMemory uint64  `csv:"chromium_processes_memory" json:"chromium_processes_memory"`
	ScribeMemory            uint64  `csv:"scribe_memory" json:"scribe_memory"`
	OthersMemory            uint64  `csv:"others_memory" json:"others_memory"`
	TotalMemoryUsed         uint64  `csv:"total_memory_used" json:"total_memory_used"`
	SystemMemoryUsed        uint64  `csv:"system_memory_used" json:"system_memory_used"`
	SystemMemoryFree        uint64  `csv:"system_memory_free" json:"system_memory_free"`
	NumberOfGoroutines      int     `csv:"number_of_goroutines" json:"number_of_goroutines"`
	Load1                   float64 `csv:"load_1" json:"load_1"`
	Load5                   float64 `csv:"load_5" json:"load_5"`
	Load15                  float64 `csv:"load_15" json:"load_15"`
}

func systemMetricsRoutine(systemMetricsWaiter chan struct{}) {
	var timeElapsedMs uint64
	const samplingIntervalMs = 500
	var systemMetrics []SystemMetrics
system_metrics_loop:
	for {
		select {
		case <-systemMetricsWaiter:
			log.Info("Saving system metrics...")
			csvMetricsFile, err := os.OpenFile(systemMetricsCSVFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			checkError(err)
			defer csvMetricsFile.Close()
			err = gocsv.MarshalFile(systemMetrics, csvMetricsFile)
			checkError(err)
			writeJSON(systemMetricsJSONFileName, systemMetrics)
			log.Info("Exiting system metrics routine...")
			break system_metrics_loop
		default:
			pids := sigar.ProcList{}
			err := pids.Get()
			checkError(err)
			var rProcessesMemory uint64
			var chromiumProcessesMemory uint64
			var scribeMemory uint64
			var othersMemoryUsage uint64
			var totalMemoryUsage uint64
			for _, pid := range pids.List {
				state := sigar.ProcState{}
				mem := sigar.ProcMem{}
				args := sigar.ProcArgs{}
				if err = state.Get(pid); err != nil {
					continue
				}
				if err = mem.Get(pid); err != nil {
					continue
				}
				if err = args.Get(pid); err != nil {
					continue
				}
				othersMemory := true
			loop:
				// Try to identify type of process based on strings in process argument list.
				for _, processArgument := range args.List {
					switch {
					case strings.Contains(processArgument, "/usr/lib/R/"):
						rProcessesMemory += getMiB(mem.Resident - mem.Share)
						othersMemory = false
						break loop
					case strings.Contains(processArgument, "/usr/lib/chromium/"):
						chromiumProcessesMemory += getMiB(mem.Resident - mem.Share)
						othersMemory = false
						break loop
					case strings.Contains(processArgument, "./scribe"):
						scribeMemory += getMiB(mem.Resident - mem.Share)
						othersMemory = false
						break loop
					}
				}
				if othersMemory {
					othersMemoryUsage += getMiB(mem.Resident - mem.Share)
				}
				totalMemoryUsage += getMiB(mem.Resident - mem.Share)
			}
			mem := sigar.Mem{}
			err = mem.Get()
			checkError(err)
			actualUsedSystemMemory := getMiB(mem.ActualUsed)
			actualFreeSystemMemory := getMiB(mem.ActualFree)
			concreteSigar := sigar.ConcreteSigar{}
			avg, err := concreteSigar.GetLoadAverage()
			checkError(err)
			numberOfGoroutines := runtime.NumGoroutine()
			systemMetrics = append(systemMetrics, SystemMetrics{
				float64(timeElapsedMs) / 1000, rProcessesMemory, chromiumProcessesMemory, scribeMemory,
				othersMemoryUsage, totalMemoryUsage, actualUsedSystemMemory, actualFreeSystemMemory,
				numberOfGoroutines, avg.One, avg.Five, avg.Fifteen,
			})
			checkError(err)
			timeElapsedMs += samplingIntervalMs
			time.Sleep(samplingIntervalMs * time.Millisecond)
		}
	}
	// Signal to checkPackages() that metrics have been saved.
	systemMetricsWaiter <- struct{}{}
}
