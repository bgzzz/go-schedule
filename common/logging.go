package common

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

// InitLogging sets up logger with
// appropriate logging level and path for log file
// return error in case wrong log level parcing
func InitLogging(cfg *Config) error {

	lvl, err := log.ParseLevel(cfg.LogLvl)
	if err != nil {
		return err
	}

	log.SetFormatter(&log.JSONFormatter{})

	var f *os.File
	if cfg.LogPath != "" {
		f, err = os.OpenFile(cfg.LogPath, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
	}

	if cfg.Stdout {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

	if lvl == log.DebugLevel {
		log.SetOutput(os.Stdout)
	}

	log.SetLevel(lvl)

	if lvl == log.DebugLevel {
		log.SetReportCaller(true)
	}
	return nil

}
