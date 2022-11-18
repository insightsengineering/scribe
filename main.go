package main

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// TODO this has to be replaced with actual checking whether we're running in a pipeline
const Interactive = false

func main() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	// log.SetLevel(logrus.DebugLevel)
	log.SetLevel(logrus.InfoLevel)
	customFormatter.FullTimestamp = true
	var renv_lock Renvlock
	GetRenvLock("renv.lock", &renv_lock)
	if Interactive {
		file, err := os.OpenFile("scribe.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			log.Out = file
		} else {
			log.Out = os.Stdout
			log.Info("Failed to log to file, using default stdout")
		}
	}

	ValidateRenvLock(renv_lock)
	DownloadPackages(renv_lock)
	// WriteRenvLock("test-renv", renv_lock)
}
