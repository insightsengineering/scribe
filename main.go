package main

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	var renvLock Renvlock
	GetRenvLock("renv.lock", &renvLock)
	ValidateRenvLock(renvLock)
	WriteRenvLock("test-renv", renvLock)
}
