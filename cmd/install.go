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

	return nil
}
func InstallPackages(allDownloadInfo *[]DownloadInfo) {
	err := os.MkdirAll(temporalLibPath, os.ModePerm)
	checkError(err)

	for i := 0; i < len(*allDownloadInfo); i++ {
		v := (*allDownloadInfo)[i]
		log.Debug(v)
		log.Info("Packge location is", v.OutputLocation)
		cmd := "R CMD INSTALL " + v.OutputLocation + " -l " + temporalLibPath
		log.Debug(cmd)
		result, err := execCommand(cmd, true, true)
		if err == nil {
			log.Info(result)
		} else {
			log.Error(err)
		}
	}

	log.Info("Done")
}

func parseDescriptionFile(descriptionFilePath string) map[string]string {
	jsonFile, _ := ioutil.ReadFile(descriptionFilePath)
	return parseDescription(jsonFile)
}

func parseDescription(description string) map[string]string {

}
