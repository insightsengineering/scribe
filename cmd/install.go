package cmd

import (
	"os"
)

const maxInstallRoutines = 40

const temporalLibPath = "/tmp/scribe/installed_packages"

type InstallInfo struct {
	StatusCode     int    `json:"statusCode"`
	Message        string `json:"message"`
	OutputLocation string `json:"outputLocation"`
}

func installSinglePackage(packageName string) error {
	err := os.MkdirAll(temporalLibPath, os.ModePerm)
	checkError(err)
	return nil
}
func InstallPackages(allDownloadInfo *[]DownloadInfo) {
	for i := 0; i < len(*allDownloadInfo); i++ {
		v := (*allDownloadInfo)[i]
		log.Info("%s", v.OutputLocation)
	}

	log.Info("Done")
}
