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
	"strings"
	"unicode"
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
func WriteJSON(filename string, j interface{}) int {
	s, err := json.MarshalIndent(j, "", "  ")
	checkError(err)

	err = os.WriteFile(filename, s, 0644) //#nosec
	checkError(err)

	return len(s)
}

// Execute a system command
func execCommand(command string, showOutput bool, returnOutput bool) (string, error) {
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
		parts = append(parts, strings.Replace(part, "'", "", -1))
	}
	if returnOutput {
		data, err := exec.Command(parts[0], parts[1:]...).Output()
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command(parts[0], parts[1:]...)

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		log.Fatalln(err)
	}

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
	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	if err != nil {
		if showOutput {
			log.Println(errStr)
		}
		return "", err
	}
	if showOutput {
		log.Println(outStr)
	}

	return "", nil
}
