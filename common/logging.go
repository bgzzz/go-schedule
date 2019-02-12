package common

import (
	log "github.com/sirupsen/logrus"
	"os"
)

// InitLogging sets up logger with
// appropriate logging level and path for log file
// return error in case wrong log level parcing
func InitLogging(lvlStr string, logPath string) error {

	lvl, err := log.ParseLevel(lvlStr)
	if err != nil {
		return err
	}

	log.SetFormatter(&log.JSONFormatter{})

	// if logPath != "" {
	// 	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE, 0755)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.SetOutput(f)
	// }

	if lvl == log.DebugLevel {
		log.SetOutput(os.Stdout)
	}

	log.SetLevel(lvl)

	log.SetReportCaller(true)

	return nil

}
