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
	"runtime"
	"fmt"
	"os"
	"strings"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
)

func getMiB(numberOfBytes uint64) uint64 {
	return numberOfBytes / (1024 * 1024)
}

func systemDebugRoutine(systemDebugWaiter chan struct{}) {
	var timeElapsedMs uint64
	const samplingIntervalMs = 500
	metricsFile, err := os.OpenFile("metrics.csv", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	checkError(err)
	defer metricsFile.Close()
	_, err = metricsFile.WriteString(
		"timeElapsedSeconds,rProcessesMemory,chromiumMemory,scribeMemory," +
		"othersMemory,actualUsedSystemMemory,actualFreeSystemMemory," +
		"numberOfGoroutines,load1,load5,load15,rCPUPercent,chromiumCPUPercent," +
		"scribeCPUPercent,othersCPUPercent\n",
	)
	var rPreviousProcTimeTotal float64
	var chromiumPreviousProcTimeTotal float64
	var scribePreviousProcTimeTotal float64
	var othersPreviousProcTimeTotal float64
system_debug_loop:
	for {
		select {
		case _ = <-systemDebugWaiter:
			log.Info("Exiting system debug routine...")
			break system_debug_loop
		default:
			pids := sigar.ProcList{}
			pids.Get()
			var rProcessesMemory uint64
			var chromiumProcessesMemory uint64
			var scribeMemory uint64
			var othersMemoryUsage uint64
			var totalMemoryUsage uint64
			var rCurrentProcPercent float64
			var rCurrentProcTimeTotal float64
			var chromiumCurrentProcPercent float64
			var chromiumCurrentProcTimeTotal float64
			var scribeCurrentProcPercent float64
			var scribeCurrentProcTimeTotal float64
			var othersCurrentProcPercent float64
			var othersCurrentProcTimeTotal float64
			for _, pid := range pids.List {
				state := sigar.ProcState{}
				mem := sigar.ProcMem{}
				args := sigar.ProcArgs{}
				procTime := sigar.ProcTime{}
				if err := state.Get(pid); err != nil {
					continue
				}
				if err := mem.Get(pid); err != nil {
					continue
				}
				if err := args.Get(pid); err != nil {
					continue
				}
				if err := procTime.Get(pid); err != nil {
					continue
				}
				othersMemory := true
				for _, processArgument := range args.List {
					if strings.Contains(processArgument, "/usr/lib/R/") {
						rProcessesMemory += (mem.Resident-mem.Share)/(1024*1024)
						rCurrentProcTimeTotal += float64(procTime.Total)/1000
						othersMemory = false
						break
					} else if strings.Contains(processArgument, "/usr/lib/chromium/") {
						chromiumProcessesMemory += (mem.Resident-mem.Share)/(1024*1024)
						chromiumCurrentProcTimeTotal += float64(procTime.Total)/1000
						othersMemory = false
						break
					} else if strings.Contains(processArgument, "./scribe") {
						scribeMemory += (mem.Resident-mem.Share)/(1024*1024)
						othersMemory = false
						scribeCurrentProcTimeTotal += float64(procTime.Total)/1000
						break
					}
				}
				if othersMemory {
					othersMemoryUsage += (mem.Resident-mem.Share)/(1024*1024)
					othersCurrentProcTimeTotal += float64(procTime.Total)/1000
				}
				totalMemoryUsage += (mem.Resident-mem.Share)/(1024*1024)
			}
			if rPreviousProcTimeTotal > 0.0001 {
				rCurrentProcPercent = (rCurrentProcTimeTotal - rPreviousProcTimeTotal) / (float64(samplingIntervalMs)/1000)
				rPreviousProcTimeTotal = rCurrentProcTimeTotal
			} else {
				rPreviousProcTimeTotal = rCurrentProcTimeTotal
			}
			if chromiumPreviousProcTimeTotal > 0.0001 {
				chromiumCurrentProcPercent = (chromiumCurrentProcTimeTotal - chromiumPreviousProcTimeTotal) / (float64(samplingIntervalMs)/1000)
				chromiumPreviousProcTimeTotal = chromiumCurrentProcTimeTotal
			} else {
				chromiumPreviousProcTimeTotal = chromiumCurrentProcTimeTotal
			}
			if scribePreviousProcTimeTotal > 0.0001 {
				scribeCurrentProcPercent = (scribeCurrentProcTimeTotal - scribePreviousProcTimeTotal) / (float64(samplingIntervalMs)/1000)
				scribePreviousProcTimeTotal = scribeCurrentProcTimeTotal
			} else {
				scribePreviousProcTimeTotal = scribeCurrentProcTimeTotal
			}
			if othersPreviousProcTimeTotal > 0.0001 {
				othersCurrentProcPercent = (othersCurrentProcTimeTotal - othersPreviousProcTimeTotal) / (float64(samplingIntervalMs)/1000)
				othersPreviousProcTimeTotal = othersCurrentProcTimeTotal
			} else {
				othersPreviousProcTimeTotal = othersCurrentProcTimeTotal
			}
			mem := sigar.Mem{}
			mem.Get()
			actualUsedSystemMemory := getMiB(mem.ActualUsed)
			actualFreeSystemMemory := getMiB(mem.ActualFree)
			concreteSigar := sigar.ConcreteSigar{}
			avg, err := concreteSigar.GetLoadAverage()
			checkError(err)
			numberOfGoroutines := runtime.NumGoroutine()
			_, err = metricsFile.WriteString(fmt.Sprintf(
				"%.1f,%d,%d,%d,%d,%d,%d,%d,%d,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f\n",
				float64(timeElapsedMs)/1000, rProcessesMemory, chromiumProcessesMemory, scribeMemory,
				othersMemoryUsage, totalMemoryUsage, actualUsedSystemMemory, actualFreeSystemMemory,
				numberOfGoroutines, avg.One, avg.Five, avg.Fifteen, rCurrentProcPercent,
				chromiumCurrentProcPercent, scribeCurrentProcPercent, othersCurrentProcPercent,
			))
			checkError(err)
			timeElapsedMs += samplingIntervalMs
			time.Sleep(samplingIntervalMs * time.Millisecond)
		}
	}
}
