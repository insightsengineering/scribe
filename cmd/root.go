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
var maxDownloadRoutines int
var maxCheckRoutines int
var outputReportDirectory string
var numberOfWorkers uint

var log = logrus.New()

// within below directory:
// tar.gz packages are downloaded to package_archives subdirectory
// GitHub repositories are cloned into github subdirectory
// GitLab repositories are cloned into gitlab subdirectory
const localOutputDirectory = "/tmp/scribe/downloaded_packages"

const tempCacheDirectory = "/tmp/scribe/cache"

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

		if maxDownloadRoutines < 1 {
			log.Warn("Maximum number of download routines set to less than 1. Setting the number to default value of 40.")
			maxDownloadRoutines = 40
		}
		if maxCheckRoutines < 1 {
			log.Warn("Maximum number of R CMD check routines set to less than 1. Setting the number to default value of 5.")
			maxCheckRoutines = 5
		}
		if int(numberOfWorkers) < 1 {
			log.Error("Number of simultaneous installation processes should be greater than 0")
			return
		}
		var systemInfo SystemInfo
		getOsInformation(&systemInfo, maskedEnvVars)
		var renvLock Renvlock
		getRenvLock(renvLockFilename, &renvLock)
		validateRenvLock(renvLock)

		mkdirerr := os.MkdirAll(tempCacheDirectory, os.ModePerm)
		if mkdirerr != nil {
			log.Errorf("Cannot make dir %s %v", tempCacheDirectory, mkdirerr)
		}

		// Perform package download, except when cache contains JSON with previous
		// download results.
		downloadInfoFile := filepath.Join(tempCacheDirectory, "downloadInfo.json")
		var allDownloadInfo []DownloadInfo
		if _, err := os.Stat(downloadInfoFile); err == nil {
			// File with downloaded packages information is already present.
			readJSON(downloadInfoFile, &allDownloadInfo)
		} else {
			log.Infof("%s doesn't exist.", downloadInfoFile)
			downloadPackages(renvLock, &allDownloadInfo, downloadFile, cloneGitRepo)
			writeJSON(downloadInfoFile, &allDownloadInfo)
		}

		// Perform package installation, except when cache contains JSON with previous
		// installation results.
		err2 := os.MkdirAll(buildLogPath, os.ModePerm)
		checkError(err2)
		installInfoFile := filepath.Join(tempCacheDirectory, "installResultInfos.json")
		var allInstallInfo []InstallResultInfo
		if _, err := os.Stat(installInfoFile); err == nil {
			readJSON(installInfoFile, &allInstallInfo)
		} else {
			log.Infof("%s doesn't exist.", installInfoFile)
			installPackages(renvLock, &allDownloadInfo, &allInstallInfo)
		}

		// Perform R CMD check, except when cache contains JSON with previous check results.
		checkInfoFile := filepath.Join(tempCacheDirectory, "checkInfo.json")
		var allCheckInfo []PackageCheckInfo
		if _, err := os.Stat(checkInfoFile); err == nil {
			readJSON(checkInfoFile, &allCheckInfo)
		} else {
			log.Infof("%s doesn't exist.", checkInfoFile)
			checkPackages(allInstallInfo, checkInfoFile)
			readJSON(checkInfoFile, &allCheckInfo)
		}

		// Generate report.
		var reportData ReportInfo
		processReportData(allDownloadInfo, allInstallInfo, allCheckInfo, &systemInfo, &reportData)
		err := os.RemoveAll(filepath.Join(outputReportDirectory, "logs"))
		checkError(err)
		err = os.MkdirAll(filepath.Join(outputReportDirectory, "logs"), os.ModePerm)
		checkError(err)
		// Copy log files so that they can be accessed from the HTML report.
		copyFiles(packageLogPath, "install-", filepath.Join(outputReportDirectory, "logs"))
		copyFiles(buildLogPath, "build-", filepath.Join(outputReportDirectory, "logs"))
		copyFiles(checkLogPath, "check-", filepath.Join(outputReportDirectory, "logs"))
		writeReport(reportData, filepath.Join(outputReportDirectory, "index.html"))
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
		"Logging level (trace, debug, info, warn, error). "+
			"Typically info log level is used for relevant information. "+
			"Use debug or trace for more detailed debugging information.")
	// TODO this should probably be reversed: the flag called --noninteractive
	// and the flag would be used in CI or when user wants to see whole output.
	rootCmd.PersistentFlags().BoolVar(&interactive, "interactive", false,
		"Use this flag if you want to see only progress bars for downloading, installing, etc. "+
			"If this flag is not used (e.g. in CI pipeline), detailed progress output is shown.")
	rootCmd.PersistentFlags().StringVar(&maskedEnvVars, "maskedEnvVars", "",
		"Regular expression defining which environment variables should be masked in the output report. "+
			"Typically variables with sensitive data should be masked. Example: "+`'sensitiveValue1|sensitiveValue2'`)
	rootCmd.PersistentFlags().StringVar(&renvLockFilename, "renvLockFilename", "renv.lock",
		"Path to renv.lock file to be processed")
	rootCmd.PersistentFlags().StringVar(&checkPackageExpression, "checkPackage", "",
		"Expression with wildcards indicating which packages should be R CMD checked. "+
			"The expression follows the pattern: \"expression1,expression2,...\" where \"expressionN\" can be: "+
			"literal package name and/or * symbol(s) meaning any set of characters. Example: "+
			`'package*,*abc,a*b,someOtherPackage'`)
	rootCmd.PersistentFlags().BoolVar(&checkAllPackages, "checkAllPackages", false,
		"Use this flag to check all installed packages.")
	rootCmd.PersistentFlags().StringVar(&outputReportDirectory, "reportDir", "outputReport",
		"The name of directory where the output report should be saved.")
	rootCmd.PersistentFlags().IntVar(&maxDownloadRoutines, "maxDownloadRoutines", 40,
		"Maximum number of concurrently running download goroutines.")
	rootCmd.PersistentFlags().IntVar(&maxCheckRoutines, "maxCheckRoutines", 5,
		"Maximum number of concurrently running R CMD check goroutines.")
	rootCmd.PersistentFlags().UintVar(&numberOfWorkers, "numberOfWorkers", 20,
		"Number of simultaneous installation processes.")
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
