package main

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	// log.SetLevel(logrus.DebugLevel)
	log.SetLevel(logrus.InfoLevel)
	customFormatter.FullTimestamp = true
	var renv_lock Renvlock
	GetRenvLock("renv.lock", &renv_lock)
	ValidateRenvLock(renv_lock)
	DownloadPackages(renv_lock)
	// WriteRenvLock("test-renv", renv_lock)
}
