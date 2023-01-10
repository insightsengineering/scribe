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
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var scribeVersion string
var logLevel string
var interactive bool
var maskedEnvVars string
var renvLockFilename string
var checkPackageExpression string
var checkAllPackages bool

var log = logrus.New()

// within below directory:
// tar.gz packages are downloaded to package_archives subdirectory
// GitHub repositories are cloned into github subdirectory
// GitLab repositories are cloned into gitlab subdirectory
const localOutputDirectory = "/tmp/scribe/downloaded_packages"

const temporalCacheDirectory = "/tmp/scribe/cache"

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
		var systemInfo SystemInfo
		getOsInformation(&systemInfo, maskedEnvVars)
		var renvLock Renvlock
		getRenvLock(renvLockFilename, &renvLock)
		validateRenvLock(renvLock)
		var allDownloadInfo []DownloadInfo

		mkdirerr := os.MkdirAll(temporalCacheDirectory, os.ModePerm)
		if mkdirerr != nil {
			log.Errorf("Cannot make dir %s %v", temporalCacheDirectory, mkdirerr)
		}
		downloadInfoFile := filepath.Join(temporalCacheDirectory, "downloadInfo.json")
		if _, err := os.Stat(downloadInfoFile); err == nil {
			// File with downloaded packages information is already present.
			log.Info("Reading", downloadInfoFile)
			jsonFile, errr := os.ReadFile(downloadInfoFile)
			checkError(errr)
			errUnmarshal := json.Unmarshal(jsonFile, &allDownloadInfo)
			checkError(errUnmarshal)
		} else {
			log.Infof("No %s", downloadInfoFile)
			downloadPackages(renvLock, &allDownloadInfo, downloadFile, cloneGitRepo)
			writeJSON(downloadInfoFile, &allDownloadInfo)
		}

		installResultInfos := make([]InstallResultInfo, 0)
		InstallPackages(renvLock, &allDownloadInfo, &installResultInfos)
		checkPackages(installResultInfos)

		// Generate report.
		readJSON(downloadInfoFile, &allDownloadInfo)
		var allInstallInfo []InstallResultInfo
		readJSON(temporalCacheDirectory+"/installResultInfos.json", &allInstallInfo)
		var reportData ReportInfo
		preprocessReportData(allDownloadInfo, allInstallInfo, &systemInfo, &reportData)
		err := os.MkdirAll("outputReport", os.ModePerm)
		checkError(err)
		writeReport(reportData, "outputReport/index.html", "cmd/report/index.html")
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
	rootCmd.PersistentFlags().StringVar(&maskedEnvVars, "maskedEnvVars", "",
		"Regular expression for which environment variables should be masked in system information report")
	rootCmd.PersistentFlags().StringVar(&renvLockFilename, "renvLockFilename", "renv.lock",
		"Path to renv.lock file to be processed")
	rootCmd.PersistentFlags().StringVar(&checkPackageExpression, "checkPackage", "",
		"Expression with wildcards indicating which packages should be R CMD checked")
	rootCmd.PersistentFlags().BoolVar(&checkAllPackages, "checkAllPackages", false,
		"Should R CMD check be run on all installed packages?")
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
