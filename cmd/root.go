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
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var scribeVersion string
var logLevel string
var interactive bool

var log = logrus.New()

// within below directory:
// tar.gz packages are downloaded to package_archives subdirectory
// GitHub repositories are cloned into github subdirectory
// GitLab repositories are cloned into gitlab subdirectory
const localOutputDirectory = "/tmp/scribe/downloaded_packages"

var bioconductorCategories = [4]string{"bioc", "data/experiment", "data/annotation", "workflows"}

func setLogLevel() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	fmt.Println("Loglevel =", logLevel)
	switch logLevel {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
	if interactive {
		// Save the log to a file instead of outputting it to stdout.
		file, err := os.OpenFile("scribe.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			log.Out = file
		} else {
			log.Out = os.Stdout
			log.Info("Failed to log to file, using default stdout")
		}
	}
}

func getTempCheckOutput(filename string) string {
	checkOutput, err := os.ReadFile(filename)
	checkError(err)
	return string(checkOutput)
}

func parseCheck() {
	var allCheckInfo []CheckInfo
	outputString := getTempCheckOutput("cmd/testdata/r_cmd_check.txt")
	parseCheckOutput(outputString, &allCheckInfo)
	for _, check := range allCheckInfo {
		fmt.Println(check.CheckItemType)
		fmt.Println(check.CheckItemContent)
	}
}

var rootCmd = &cobra.Command{
	Use:   "scribe",
	Short: "System Compatibility Report for Install & Build Evaluation",
	Long: `scribe (acronym for System Compatibility Report for Install & Build Evaluation)
	is a project that creates complete build, check and install reports
	for a collection of R packages that are defined in an
	[renv.lock](https://rstudio.github.io/renv/articles/lockfile.html) file.`,
	Version: scribeVersion,
	Run: func(cmd *cobra.Command, args []string) {
		setLogLevel()
		parseCheck()
		// TODO getting renv lock here is just temporary
		// we'll have to figure out how to use that together with other components
		// var renvLock Renvlock
		// getRenvLock("renv.lock", &renvLock)
		// validateRenvLock(renvLock)
		// var allDownloadInfo []DownloadInfo
		// downloadPackages(renvLock, &allDownloadInfo, downloadFile, cloneGitRepo)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.scribe.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "logLevel", "info",
		"Logging level (trace, debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&interactive, "interactive", false,
		"Is scribe running in interactive environment (as opposed to e.g. CI pipeline)?")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".scribe" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".scribe")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
