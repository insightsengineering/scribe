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
}

func Test_execCommand(t *testing.T) {
	res, err := execCommand("R CMD", true, false, nil, nil)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)
}

func Test_execCommandWithEnvs(t *testing.T) {

	res, err := execCommand(`R -s --vanilla -e ".libPaths()"`, true, false, []string{"R_LIBS=/usr/lib/R/library"}, nil)
	assert.NotEmpty(t, res)
	assert.Nil(t, err)
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
		// Normal Rev Binomial TREE
		/*
			g := map[string][]string{

				"21": {"11"},
				"22": {"11"},

				"31": {"21"},
				"32": {"21"},

				"33": {"22"},
				"34": {"22"},

				"41": {"31"},
				"42": {"31"},

				"43": {"32"},
				"44": {"32"},

				"45": {"33"},
				"46": {"33"},

				"47": {"34"},
				"48": {"34"},
			}
			expectedOrder := []string{"11", "21", "22", "31", "32", "33", "34", "41", "42", "43", "44", "45", "46", "47", "48"}
		*/
		// Big Binomial TREE
		/*
			g := map[string][]string{
				"11": {"21", "22"},

				"21": {"31", "32"},
				"22": {"33", "34"},

				"31": {"41", "42"},
				"32": {"43", "44"},
				"33": {"45", "46"},
				"34": {"47", "48"},

				"41": {"51", "52"},
				"42": {"53", "54"},
				"43": {"55", "56"},
				"44": {"57", "58"},
				"45": {"59", "510"},
				"46": {"511", "512"},
				"47": {"513", "514"},
				"48": {"515", "516"},

				"51":  {"61", "62"},
				"52":  {"63", "64"},
				"53":  {"65", "66"},
				"54":  {"67", "68"},
				"55":  {"69", "610"},
				"56":  {"611", "612"},
				"57":  {"613", "614"},
				"58":  {"615", "616"},
				"59":  {"617", "618"},
				"510": {"619", "620"},
				"511": {"621", "622"},
				"512": {"623", "624"},
				"513": {"625", "626"},
				"514": {"627", "628"},
				"515": {"629", "630"},
				"516": {"631", "632"},
			}
			expectedOrder := []string{"61", "610", "611", "612", "613", "614", "615", "616", "617", "618", "619", "62", "620", "621", "622", "623", "624", "625", "626", "627", "628", "629", "63", "630", "631", "632", "64", "65", "66", "67", "68", "69"}
		*/

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
			[]string{"1", "7", "2", "4"},
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
