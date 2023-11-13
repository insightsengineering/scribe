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

	"github.com/jamiealquiza/envy"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.szostok.io/version/extension"
)

var cfgFile string
var logLevel string
var maskedEnvVars string
var renvLockFilename string
var checkPackageExpression string
var updatePackages string
var checkAllPackages bool
var maxDownloadRoutines int
var maxCheckRoutines int
var outputReportDirectory string
var numberOfWorkers uint
var clearCache bool
var includeSuggests bool
var failOnError bool
var buildOptions string
var checkOptions string
var installOptions string
var rCmdCheckFailRegex string

var log = logrus.New()

var temporalLibPath string
var rLibsPaths string

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
	customFormatter.ForceColors = true
	log.SetFormatter(customFormatter)
	log.SetReportCaller(false)
	customFormatter.FullTimestamp = false
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
}

func getExitStatus(allInstallInfo []InstallResultInfo, allCheckInfo []PackageCheckInfo) int {
	for _, p := range allInstallInfo {
		if p.BuildStatus == buildStatusFailed || p.Status == InstallResultInfoStatusFailed {
			return 1
		}
	}
	for _, p := range allCheckInfo {
		if p.ShouldFail {
			return 1
		}
		if p.MostSevereCheckItem == "ERROR" {
			return 1
		}
	}
	return 0
}

var rootCmd *cobra.Command

func newRootCommand() {
	rootCmd = &cobra.Command{
		Use:   "scribe",
		Short: "System Compatibility Report for Install & Build Evaluation",
		Long: `scribe (acronym for System Compatibility Report for Install & Build Evaluation)
		is a project that creates complete build, check and install reports
		for a collection of R packages that are defined in an
		[renv.lock](https://rstudio.github.io/renv/articles/lockfile.html) file.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initializeConfig()
		},
		Run: func(cmd *cobra.Command, args []string) {
			setLogLevel()

			// TODO this function should be used during renv.lock generation
			var allPackages []PackagesFile
			getPackagesFileFromURL("https://cloud.r-project.org/src/contrib/PACKAGES", &allPackages)
			writeJSON("packages.json", allPackages)

			fmt.Println("cfgfile =", cfgFile)
			fmt.Println("maskedEnvVars =", maskedEnvVars)
			fmt.Println("renvLockFilename =", renvLockFilename)
			fmt.Println("includeSuggests = ", includeSuggests)
			fmt.Println("checkPackage =", checkPackageExpression)
			fmt.Println("updatePackages =", updatePackages)
			fmt.Println("checkAllPackages =", checkAllPackages)
			fmt.Println("reportDir =", outputReportDirectory)
			fmt.Println("maxDownloadRoutines =", maxDownloadRoutines)
			fmt.Println("maxCheckRoutines =", maxCheckRoutines)
			fmt.Println("numberOfWorkers =", numberOfWorkers)
			fmt.Println("clearCache =", clearCache)
			fmt.Println("failOnError = ", failOnError)
			fmt.Println("buildOptions = ", buildOptions)
			fmt.Println("installOptions = ", installOptions)
			fmt.Println("checkOptions = ", checkOptions)
			fmt.Println("rCmdCheckFailRegex = ", rCmdCheckFailRegex)

			if maxDownloadRoutines < 1 {
				log.Warn("Maximum number of download routines set to less than 1. Setting the number to default value of 40.")
				maxDownloadRoutines = 40
			}
			if maxCheckRoutines < 1 {
				log.Warn("Maximum number of R CMD check routines set to less than 1. Setting the number to default value of 5.")
				maxCheckRoutines = 5
			}
			if int(numberOfWorkers) < 1 {
				log.Warn("Number of simultaneous installation processes should be greater than 0. Setting the default number of workers to 20.")
				numberOfWorkers = 20
			}

			if clearCache {
				clearCachedData()
			}

			temporalLibPath = os.Getenv("TMP") + `\tmp\scribe\installed_packages`
			rLibsPaths = os.Getenv("TMP") + `\tmp\scribe\installed_packages`

			logFile, logFileErr := os.OpenFile(os.Getenv("TMP") + `\tempLogFile`, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
			checkError(logFileErr)
			defer logFile.Close()
			output, err3 := execCommand(`powershell.exe -noexit "$env:PATH; $env:R_LIBS; $env:LANG"`, false,
				[]string{
					"R_LIBS=" + rLibsPaths,
					"LANG=en_US.UTF-8",
				}, logFile)
			checkError(err3)
			fmt.Println("output =", output)

			var systemInfo SystemInfo
			getOsInformation(&systemInfo, maskedEnvVars)
			var renvLock Renvlock
			var renvLockOld Renvlock
			var renvLockFilenameOld string
			getRenvLock(renvLockFilename, &renvLock)
			validateRenvLock(renvLock)
			if updatePackages != "" {
				renvLockFilenameOld = renvLockFilename
				renvLockFilename += ".updated"
				updatePackagesRenvLock(&renvLock, renvLockFilename, updatePackages)
				// updatePackagesRenvLock modified the original structure in place.
				// Therefore, we make a copy to show both renv.lock contents in the report.
				getRenvLock(renvLockFilenameOld, &renvLockOld)
			}

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
				installPackages(renvLock, &allDownloadInfo, &allInstallInfo, includeSuggests, buildOptions, installOptions)
			}

			// Perform R CMD check, except when cache contains JSON with previous check results.
			checkInfoFile := filepath.Join(tempCacheDirectory, "checkInfo.json")
			var allCheckInfo []PackageCheckInfo
			if _, err := os.Stat(checkInfoFile); err == nil {
				readJSON(checkInfoFile, &allCheckInfo)
			} else {
				log.Infof("%s doesn't exist.", checkInfoFile)
				checkPackages(checkInfoFile, checkOptions)
				// If no packages were checked (because of e.g. not matching the CLI parameter)
				// the file with check results will not be generated, so we're checking
				// its existence once again.
				if _, err := os.Stat(checkInfoFile); err == nil {
					readJSON(checkInfoFile, &allCheckInfo)
				}
			}

			// Generate report.
			var reportData ReportInfo
			processReportData(allDownloadInfo, allInstallInfo, allCheckInfo, &systemInfo, &reportData,
				renvLock, renvLockOld, renvLockFilenameOld)
			err := os.RemoveAll(filepath.Join(outputReportDirectory, "logs"))
			checkError(err)
			err = os.MkdirAll(filepath.Join(outputReportDirectory, "logs"), os.ModePerm)
			checkError(err)
			// Copy log files so that they can be accessed from the HTML report.
			copyFiles(packageLogPath, "install-", filepath.Join(outputReportDirectory, "logs"))
			copyFiles(buildLogPath, "build-", filepath.Join(outputReportDirectory, "logs"))
			copyFiles(checkLogPath, "check-", filepath.Join(outputReportDirectory, "logs"))
			writeReport(reportData, filepath.Join(outputReportDirectory, "index.html"))

			if failOnError {
				exitStatus := getExitStatus(allInstallInfo, allCheckInfo)
				os.Exit(exitStatus)
			}
		},
	}
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.scribe.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "logLevel", "info",
		"Logging level (trace, debug, info, warn, error). "+
			"Typically info log level is used for relevant information. "+
			"Use debug or trace for more detailed debugging information.")
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
	rootCmd.PersistentFlags().StringVar(&updatePackages, "updatePackages", "",
		"Expression with wildcards indicating which packages should be updated to the newest version. "+
			"The expression follows the same pattern as checkPackage flag. "+
			"This is currently only supported for packages downloaded from git repositories.")
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
	rootCmd.PersistentFlags().BoolVar(&clearCache, "clearCache", false,
		"Use this flag if you want to clear scribe internal cache directory structure. This will cause "+
			"all packages to be downloaded, installed, built, and checked from scratch.")
	rootCmd.PersistentFlags().BoolVar(&includeSuggests, "includeSuggests", false,
		"Use this flag if you also want to install packages from the 'Suggests' field in the "+
			"dependencies' DESCRIPTION files.")
	rootCmd.PersistentFlags().BoolVar(&failOnError, "failOnError", false,
		"Use this flag to make scribe return exit code 1 in case of check errors or build failures.")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().StringVar(&buildOptions, "buildOptions", "",
		"Extra options to pass to R CMD build. Options must be supplied in double quoted string.")
	rootCmd.PersistentFlags().StringVar(&installOptions, "installOptions", "",
		"Extra options to pass to R CMD INSTALL. Options must be supplied in double quoted string.")
	rootCmd.PersistentFlags().StringVar(&checkOptions, "checkOptions", "",
		"Extra options to pass to R CMD check. Options must be supplied in double quoted string.")
	rootCmd.PersistentFlags().StringVar(&rCmdCheckFailRegex, "rCmdCheckFailRegex", "",
		"Regex which when encountered as part of R CMD check NOTE or WARNING, should cause scribe to fail "+
			"(only when failOnError is true).")

	// Add version command.
	rootCmd.AddCommand(extension.NewVersionCobraCmd())

	cfg := envy.CobraConfig{
		Prefix:     "SCRIBE",
		Persistent: true,
	}
	envy.ParseCobra(rootCmd, cfg)
}

func init() {
	cobra.OnInitialize(initConfig)
}

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
	} else {
		fmt.Println(err)
	}
}

func Execute() {
	newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initializeConfig() {
	for _, v := range []string{
		"logLevel",
		"maskedEnvVars",
		"renvLockFilename",
		"checkPackage",
		"updatePackages",
		"checkAllPackages",
		"reportDir",
		"maxDownloadRoutines",
		"maxCheckRoutines",
		"numberOfWorkers",
		"clearCache",
		"includeSuggests",
		"failOnError",
		"buildOptions",
		"installOptions",
		"checkOptions",
		"rCmdCheckFailRegex",
	} {
		// If the flag has not been set in newRootCommand() and it has been set in initConfig().
		// In other words: if it's not been provided in command line, but has been
		// provided in config file.
		// Helpful project where it's explained:
		// https://github.com/carolynvs/stingoftheviper
		if !rootCmd.PersistentFlags().Lookup(v).Changed && viper.IsSet(v) {
			err := rootCmd.PersistentFlags().Set(v, fmt.Sprintf("%v", viper.Get(v)))
			checkError(err)
		}
	}
}
