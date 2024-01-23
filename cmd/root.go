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
	"runtime"
	"strconv"

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
var rExecutablePath string

var log = logrus.New()

var temporaryLibPath string
var rLibsPaths string
var rExecutable string

// Within localOutputDirectory:
// tar.gz packages are downloaded to package_archives subdirectory
// GitHub repositories are cloned into github subdirectory
// GitLab repositories are cloned into gitlab subdirectory
var localOutputDirectory string

const tempCacheDirectory = "/tmp/scribe/cache"
const defaultDownloadDirectory = "/tmp/scribe/downloaded_packages"

func setLogLevel() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.ForceColors = true
	log.SetFormatter(customFormatter)
	log.SetReportCaller(false)
	customFormatter.FullTimestamp = false
	fmt.Println(`logLevel = "` + logLevel + `"`)
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

			fmt.Println(`cfgfile = "` + cfgFile + `"`)
			fmt.Println(`maskedEnvVars = "` + maskedEnvVars + `"`)
			fmt.Println(`renvLockFilename = "` + renvLockFilename + `"`)
			fmt.Println(`checkPackage = "` + checkPackageExpression + `"`)
			fmt.Println(`updatePackages = "` + updatePackages + `"`)
			fmt.Println(`reportDir = "` + outputReportDirectory + `"`)
			fmt.Println(`buildOptions = "` + buildOptions + `"`)
			fmt.Println(`installOptions = "` + installOptions + `"`)
			fmt.Println(`checkOptions = "` + checkOptions + `"`)
			fmt.Println(`rCmdCheckFailRegex = "` + rCmdCheckFailRegex + `"`)
			fmt.Println(`rExecutablePath = "` + rExecutablePath + `"`)
			fmt.Println(`includeSuggests = ` + strconv.FormatBool(includeSuggests))
			fmt.Println(`checkAllPackages = ` + strconv.FormatBool(checkAllPackages))
			fmt.Println(`clearCache = ` + strconv.FormatBool(clearCache))
			fmt.Println(`failOnError = ` + strconv.FormatBool(failOnError))
			fmt.Println(`maxDownloadRoutines = ` + strconv.Itoa(maxDownloadRoutines))
			fmt.Println(`maxCheckRoutines = ` + strconv.Itoa(maxCheckRoutines))
			fmt.Println(`numberOfWorkers = ` + strconv.Itoa(int(numberOfWorkers)))

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

			if runtime.GOOS == windows {
				temporaryLibPath = os.Getenv("TMP") + `\tmp\scribe\installed_packages`
				rLibsPaths = os.Getenv("TMP") + `\tmp\scribe\installed_packages`
				localOutputDirectory = os.Getenv("TMP") + `\tmp\scribe\downloaded_packages`
				rExecutable = `'` + rExecutablePath + `'`
			} else {
				temporaryLibPath = "/tmp/scribe/installed_packages"
				rLibsPaths = "/tmp/scribe/installed_packages:/usr/local/lib/R/site-library:/usr/lib/R/site-library:/usr/lib/R/library"
				localOutputDirectory = defaultDownloadDirectory
				rExecutable = rExecutablePath
			}

			var systemInfo SystemInfo
			getOsInformation(&systemInfo, maskedEnvVars)
			var renvLock Renvlock
			var renvLockOld Renvlock
			var renvLockFilenameOld string
			var erroneousRepositoryNames []string
			getRenvLock(renvLockFilename, &renvLock)
			validateRenvLock(renvLock, &erroneousRepositoryNames)
			if updatePackages != "" {
				renvLockFilenameOld = renvLockFilename
				renvLockFilename += ".updated"
				updatePackagesRenvLock(&renvLock, renvLockFilename, updatePackages)
				// updatePackagesRenvLock modified the original structure in place.
				// Therefore, we make a copy to show both renv.lock contents in the report.
				getRenvLock(renvLockFilenameOld, &renvLockOld)
			}

			err := os.MkdirAll(tempCacheDirectory, os.ModePerm)
			checkError(err)

			// Perform package download, except when cache contains JSON with previous
			// download results.
			downloadInfoFile := filepath.Join(tempCacheDirectory, "downloadInfo.json")
			var allDownloadInfo []DownloadInfo
			if _, err = os.Stat(downloadInfoFile); err == nil {
				// File with downloaded packages information is already present.
				readJSON(downloadInfoFile, &allDownloadInfo)
			} else {
				log.Info(downloadInfoFile, " doesn't exist.")
				downloadPackages(renvLock, &allDownloadInfo, downloadFile, cloneGitRepo)
				writeJSON(downloadInfoFile, &allDownloadInfo)
			}

			// Perform package installation, except when cache contains JSON with previous
			// installation results.
			err = os.MkdirAll(buildLogPath, os.ModePerm)
			checkError(err)
			installInfoFile := filepath.Join(tempCacheDirectory, "installResultInfo.json")
			var allInstallInfo []InstallResultInfo
			if _, err = os.Stat(installInfoFile); err == nil {
				readJSON(installInfoFile, &allInstallInfo)
			} else {
				log.Info(installInfoFile, " doesn't exist.")
				installPackages(renvLock, &allDownloadInfo, &allInstallInfo, buildOptions,
					installOptions, erroneousRepositoryNames)
			}

			// Perform R CMD check, except when cache contains JSON with previous check results.
			checkInfoFile := filepath.Join(tempCacheDirectory, "checkInfo.json")
			var allCheckInfo []PackageCheckInfo
			if _, err = os.Stat(checkInfoFile); err == nil {
				readJSON(checkInfoFile, &allCheckInfo)
			} else {
				log.Info(checkInfoFile, " doesn't exist.")
				checkPackages(checkInfoFile, checkOptions)
				// If no packages were checked (e.g. because their names didn't match the CLI parameter)
				// the file with check results will not be generated, so we're checking
				// its existence once again.
				if _, err = os.Stat(checkInfoFile); err == nil {
					readJSON(checkInfoFile, &allCheckInfo)
				}
			}

			// Generate report.
			var reportData ReportInfo
			processReportData(allDownloadInfo, allInstallInfo, allCheckInfo, &systemInfo, &reportData,
				renvLock, renvLockOld, renvLockFilenameOld)
			err = os.RemoveAll(filepath.Join(outputReportDirectory, "logs"))
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
		"Extra options to pass to R CMD build. Options must be supplied in single quoted string.")
	rootCmd.PersistentFlags().StringVar(&installOptions, "installOptions", "",
		"Extra options to pass to R CMD INSTALL. Options must be supplied in single quoted string.")
	rootCmd.PersistentFlags().StringVar(&checkOptions, "checkOptions", "",
		"Extra options to pass to R CMD check. Options must be supplied in single quoted string.")
	rootCmd.PersistentFlags().StringVar(&rCmdCheckFailRegex, "rCmdCheckFailRegex", "",
		"Regex which when encountered as part of R CMD check NOTE or WARNING, should cause scribe to fail "+
			"(only when failOnError is true).")
	rootCmd.PersistentFlags().StringVar(&rExecutablePath, "rExecutablePath", "R",
		"Path to the R executable.")

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

		// Search for config in home directory.
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".scribe")
	}
	// Read in environment variables that match.
	viper.AutomaticEnv()

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
		"logLevel", "maskedEnvVars", "renvLockFilename", "checkPackage", "updatePackages",
		"checkAllPackages", "reportDir", "maxDownloadRoutines", "maxCheckRoutines", "numberOfWorkers",
		"clearCache", "includeSuggests", "failOnError", "buildOptions", "installOptions",
		"checkOptions", "rCmdCheckFailRegex", "rExecutablePath",
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
