package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func Test_cleanDescription(t *testing.T) {
	cleanedDescription := cleanDescription(descriptionContent)
	assert.True(t, strings.Contains(cleanedDescription, "Imports"))
	assert.False(t, strings.Contains(cleanedDescription, "BugReports"))
	assert.False(t, strings.Contains(cleanedDescription, "Roxygen"))
	assert.Equal(t, cleanDescriptionContent, cleanedDescription)
}

func Test_parseDescription(t *testing.T) {
	kv := parseDescription(descriptionContent)
	assert.Equal(t, "methods, optimx, parallel,", kv["Imports"])
}

func Test_getPackageContent(t *testing.T) {
	content := getPackageContent()
	assert.True(t, strings.Contains(content, "Package:"))
}

func Test_getPackageDepsFromCrandb(t *testing.T) {
	pkgs := []string{"ggplot2", "dplyr"}
	packDeps := getPackageDepsFromCrandb(pkgs)

	assert.NotEmpty(t, packDeps)
	assert.Contains(t, packDeps["dplyr"], "rlang")
}

func Test_getPackageDepsFromCrandbWithChunk(t *testing.T) {
	pkgs := []string{"childsds", "ini", "teal.logger", "withr", "BiocFileCache", "contrast", "spatial", "stringr", "biomaRt", "ggrepel", "ggstance", "glue", "knitrBootstrap", "leaps", "pastecs", "cmprsk", "descr", "gdtools", "table1", "NonCompart", "RJDBC", "BiocGenerics", "eulerr", "fontLiberation", "hdrcde", "mirt", "teal.connectors.rice", "GenomeInfoDbData", "gsubfn", "lotri", "SparseM", "RBesT", "extraDistr", "mondate", "rapportools", "tidygraph", "xfun", "KFAS", "rstanarm", "corrplot", "deldir", "labelled", "pbapply", "stringfish", "vegan", "UpSetR", "spelling", "forestplot", "etm", "fitdistrplus", "maditr", "packrat", "r2d3", "argparse", "scda.2022", "gdata", "jquerylib", "reprex", "riceutils", "rversions", "ICSNP", "teal.modules.general", "vars", "npsurv", "dynCorr", "farver", "n1qn1", "pkgbuild", "slam", "PreciseSums", "coin", "fdrtool", "progress", "rpact", "rvg", "brglm", "RcppProgress", "sendmailR", "patchwork", "sfsmisc", "shinydashboard", "pool", "sparklyr", "xaringan", "zeallot", "RMySQL", "ggtext", "htmlwidgets", "lmom", "mapproj", "pkgKitten", "rle", "CommonJavaJars", "dbplyr", "filelock", "glmnet", "gridGraphics", "prettyunits", "simstudy", "colorspace", "rworldmap", "truncdist", "GlobalOptions", "pROC", "reporttools", "statnet.common", "ROCR", "lattice", "proto", "udunits2", "dimRed", "GenomeInfoDb", "KEGGREST", "diagram", "drc", "glasso", "gss", "mongolite", "CVST", "tzdb", "spatstat.data", "future.apply", "RUnit", "Rook", "mlbench", "BiocStyle", "EnvStats", "pkgconfig", "Ecfun", "waveslim", "igraph", "mitools", "pracma", "rappdirs", "xopen", "fastmap", "compare", "isoband", "kableExtra", "lambda.r", "rjags", "Hmisc", "png", "gridBase", "CePa", "mvnfast", "ragg", "vroom", "BiocBaseUtils", "lifecycle", "meta", "prabclus", "sourcetools", "R.oo", "clinfun", "cyclocomp", "feather", "Gviz", "ModelMetrics", "MALDIquant", "nor1mix"}
	packDeps := getPackageDepsFromCrandbWithChunk(pkgs)

	assert.NotEmpty(t, packDeps)
	assert.Contains(t, packDeps["dplyr"], "rlang")
}

func Test_getCrandbUrl(t *testing.T) {
	pkgs := []string{"ggplot2"}
	url := getCrandbUrl(pkgs)
	assert.True(t, strings.Contains(url, "ggplot2"))
}

func Test_getDependenciesFileds(t *testing.T) {
	cases := []struct{ included bool }{
		{true},
		{false},
	}
	var fileds []string
	for _, v := range cases {
		fileds = getDependenciesFileds(v.included)
		assert.True(t, slices.Contains(fileds, "Suggests") == v.included)

	}
}

func Test_tsort(t *testing.T) {
	/*
		g := map[string][]string{
			"B": {},
			"b": {},
			"A": {},
			"a": {},
			"2": {},
			"1": {},
			"3": {},
			"c": {},
			"C": {},
		}
		expectedOrder := []string{"1", "2", "3", "A", "B", "C", "a", "b", "c"}
	*/

	/*
		g := map[string][]string{
			"2": {"5"},
			"3": {"7"},
			"4": {"1"},
			"1": {},
			"7": {"2"},
			"5": {"4"},
		}
		expectedOrder := []string{"1", "4", "5", "2", "7", "3"}
	*/
	// Small Binominal TREE
	/*
		g := map[string][]string{

			"21": {"32", "31"},
			"22": {"34", "33"},
			"11": {"22", "21"},
		}
		expectedOrder := []string{"31", "32", "33", "34", "21", "22", "11"}
	*/
	// Small rEVERT Binominal TREE
	/*
		g := map[string][]string{

			"21": {"11"},
			"22": {"11"},

			"31": {"21"},
			"32": {"21"},

			"33": {"22"},
			"34": {"22"},
		}
		expectedOrder := []string{"11", "21", "22", "31", "32", "33", "34"}
	*/

	// Normal Binominal TREE
	/*
		g := map[string][]string{
			"11": {"21", "22"},

			"21": {"31", "32"},
			"22": {"33", "34"},

			"31": {"41", "42"},
			"32": {"43", "44"},
			"33": {"45", "46"},
			"34": {"47", "48"},
		}
		expectedOrder := []string{"41", "42", "43", "44", "45", "46", "47", "48", "31", "32", "33", "34", "21", "22", "11"}
	*/
	// Normal Rev Binominal TREE
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
	// Big Binominal TREE
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

	/*
		g := map[string][]string{
			"A": {"B", "F"},
			"B": {"H"},
			"G": {"A", "C"},
			"D": {"E", "C", "I"},
			"I": {"C"},
			"J": {"E"},
			"E": {"I"},
			"K": {"G", "D"},
		}
		expectedOrder := []string{"C", "F", "H", "B", "I", "E", "J", "A", "G", "D", "K"} //to change
	*/
	g := map[string][]string{
		"E": {"K", "H"},
		"C": {"F", "I"},
		"D": {"G", "E"},
		"A": {"B", "C", "D"},
		"B": {"J"},
	}
	expectedOrder := []string{"F", "G", "H", "I", "J", "K", "B", "C", "E", "D", "A"}

	/*
		g := map[string][]string{
			"1": {},
			"2": {"1"},
			"3": {"2"},
			"4": {"1"},
			"5": {"4"},
			"6": {"1"},
			"7": {},
			"8": {"5"},
		}
		expectedOrder := []string{"1", "7", "2", "4"}
	*/
	order := tsort(g)
	fmt.Println(order)
	assert.NotNil(t, order)
	assert.Equal(t, expectedOrder, order)
}

const cleanDescriptionContent = `Package: tern
Version: 0.7.6.9037
Depends:
    R (>= 3.6),
Imports:
    methods,
    optimx,
    parallel,
Suggests:
    knitr,
    testthat (>= 2.0)
`
const descriptionContent = `Package: tern
Title: Create Common TLGs used in Clinical Trials
Version: 0.7.6.9037
Date: 2022-01-27
Authors@R: c(
    person("NEST", , , "basel.nestcicd@roche.com", role = c("aut", "cre")),
  )
Description: Table, Listings, and Graphs (TLG) library for common outputs
    used in clinical trials.
License: Apache License 2.0 | file LICENSE
URL: https://github.com/insightsengineering/tern
BugReports: https://github.com/insightsengineering/tern/issues
Depends:
    R (>= 3.6),
Imports:
    methods,
    optimx,
    parallel,
Suggests:
    knitr,
    testthat (>= 2.0)
VignetteBuilder:
    knitr
Encoding: UTF-8
LazyData: true
Roxygen: list(markdown = TRUE)
RoxygenNote: 7.1.2
Collate:
    'formats.R'
`
