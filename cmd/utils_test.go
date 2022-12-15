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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func Test_stringInSlice(t *testing.T) {
	var testSlice = []string{"a", "b", "c", "d"}
	assert.True(t, true, stringInSlice("a", testSlice))
}

func Test_writeJSON(t *testing.T) {
	var renvLock Renvlock
	getRenvLock("testdata/renv.lock.empty.json", &renvLock)
	numberOfBytes := writeJSON("testdata/test_output.json", renvLock)
	assert.Greater(t, numberOfBytes, 0)
	os.Remove("testdata/test_output.json")
}

func Test_execCommand(t *testing.T) {
	t.Skip("skipping integration test")
	res, err := execCommand("R CMD", false, false, nil, nil)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)
}

func Test_execCommandWithEnvs(t *testing.T) {
	t.Skip("skipping integration test")
	filePath := "Test_execCommandWithEnvs.log"
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		os.Remove(filePath)
	}

	logFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	assert.Nil(t, err)
	defer logFile.Close()

	res, err := execCommand(`R -s --vanilla -e ".libPaths()"`, false, false, []string{"R_LIBS=/usr/lib/R/library"}, logFile)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)

	content, err := ioutil.ReadFile(filePath)

	fmt.Print(content)
	assert.NotEmpty(t, content)
	assert.Nil(t, err)

}

func Test_tsort_many_packages(t *testing.T) {
	var deps map[string][]string
	jsonFile, _ := ioutil.ReadFile("testdata/deps.json")
	json.Unmarshal(jsonFile, &deps)
	ordered := tsort(deps)
	assert.NotEmpty(t, deps)
	assert.NotEmpty(t, ordered)

}

func Test_toEmptyMapString(t *testing.T) {
	testcases := []struct {
		slice   []string
		mapping map[string]string
	}{}

	mapeq := func(map1 map[string]string, map2 map[string]string) bool {
		if map1 == nil || map2 == nil {
			return false
		}
		if len(map1) != len(map2) {
			return false
		}

		for k, v := range map1 {
			v2, ok := map2[k]
			if !ok {
				return false
			}
			if v != v2 {
				return false
			}
		}

		for k, v := range map2 {
			v1, ok := map1[k]
			if !ok {
				return false
			}
			if v != v1 {
				return false
			}
		}
		return true
	}

	for _, c := range testcases {
		actual := toEmptyMapString(c.slice)

		if !mapeq(c.mapping, actual) {
			t.Fatalf("toEmptyMapString returns wrong value (%v). It should %v", actual, c.mapping)
		}
	}
}

func Test_tsort(t *testing.T) {

	testcases := []struct {
		testName      string
		g             map[string][]string
		expectedOrder []string
	}{
		{
			"All nodes are disconnected",
			map[string][]string{
				"B": {},
				"b": {},
				"A": {},
				"a": {},
				"2": {},
				"1": {},
				"3": {},
				"c": {},
				"C": {},
			},
			[]string{"1", "2", "3", "A", "B", "C", "a", "b", "c"},
		},

		{
			"Linear",
			map[string][]string{
				"2": {"5"},
				"3": {"7"},
				"4": {"1"},
				"1": {},
				"7": {"2"},
				"5": {"4"},
			},
			[]string{"1", "4", "5", "2", "7", "3"},
		},
		{
			"Small Binomial TREE",
			map[string][]string{

				"21": {"32", "31"},
				"22": {"34", "33"},
				"11": {"22", "21"},
			},
			[]string{"31", "32", "33", "34", "21", "22", "11"},
		},
		{
			"Small revert Binomial TREE",
			map[string][]string{

				"21": {"11"},
				"22": {"11"},

				"31": {"21"},
				"32": {"21"},

				"33": {"22"},
				"34": {"22"},
			},
			[]string{"11", "21", "22", "31", "32", "33", "34"},
		},
		{
			"Normal Binomial TREE + 2<->3 mix",
			map[string][]string{
				"11": {"21", "22"},

				"21": {"31", "32", "34", "33"},
				"22": {"33", "34", "32", "31"},

				"31": {"41", "42"},
				"32": {"43", "44"},
				"33": {"45", "46"},
				"34": {"47", "48"},
			},
			[]string{"41", "42", "43", "44", "45", "46", "47", "48", "31", "32", "33", "34", "21", "22", "11"},
		},
		{
			"Sample example 2",
			map[string][]string{
				"A": {"B", "F"},
				"B": {"H"},
				"G": {"A", "C"},
				"D": {"E", "C", "I"},
				"I": {"C"},
				"J": {"E"},
				"E": {"I"},
				"K": {"G", "D"},
			},
			[]string{"C", "F", "H", "B", "I", "E", "J", "A", "G", "D", "K"},
		},
		{
			"Sample example 3",
			map[string][]string{
				"E": {"K", "H"},
				"C": {"F", "I"},
				"D": {"G", "E"},
				"A": {"B", "C", "D"},
				"B": {"J"},
			},
			[]string{"F", "G", "H", "I", "J", "K", "B", "C", "E", "D", "A"},
		},
		{
			"Sample example 3",
			map[string][]string{
				"1": {},
				"2": {"1"},
				"3": {"2"},
				"4": {"1"},
				"5": {"4"},
				"6": {"1"},
				"7": {},
				"8": {"5"},
			},
			[]string{"1", "7", "2", "3", "4", "5", "6", "8"},
		},
	}
	for _, tc := range testcases {

		order := tsort(tc.g)
		assert.NotNil(t, order)
		if !slices.Equal(tc.expectedOrder, order) {
			t.Fatalf("[%s]\nactual:  %v\nexpected:%v", tc.testName, order, tc.expectedOrder)
		}
	}
}

func Test_fillEnvFromSystem(t *testing.T) {
	os.Setenv("LANG", "en_US.UTF-8")
	envs := fillEnvFromSystem([]string{"LANG"})
	assert.Equal(t, "LANG=en_US.UTF-8", envs[0])
}
