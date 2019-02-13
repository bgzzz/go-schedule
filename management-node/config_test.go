// +build unit

package main

import (
	"testing"

	"github.com/bgzzz/go-schedule/common"
)

func TestConfigValidate(t *testing.T) {
	cfg := ServerConfig{
		DeadTimeout:      90,
		SilenceTimeout:   10,
		PingTimer:        5,
		SchedulerAlgo:    "rr",
		ListeningAddress: "127.0.0.1:1234",
		EtcdAddress:      "127.0.0.1:1235",
		BasicConfig: common.Config{
			LogLvl:  "debug",
			LogPath: "/some/path",
		},
		DialTimeout: 5,
	}

	err := cfgValidate(&cfg)
	if err != nil {
		t.Errorf("Non nil error %s", err.Error())
	}
}

func TestConfigValidateWithTimers(t *testing.T) {
	cfg := ServerConfig{
		DeadTimeout:      90,
		SilenceTimeout:   10,
		PingTimer:        55,
		SchedulerAlgo:    "rr",
		ListeningAddress: "127.0.0.1:1234",
		EtcdAddress:      "127.0.0.1:1235",
		BasicConfig: common.Config{
			LogLvl:  "debug",
			LogPath: "/some/path",
		},
		DialTimeout: 5,
	}

	err := cfgValidate(&cfg)
	if err != nil {
		t.Errorf("Non nil error %s", err.Error())
	}

	if cfg.SilenceTimeout != cfg.PingTimer {
		t.Errorf("silence timeout != ping timeout (%d != %d)",
			cfg.SilenceTimeout, cfg.PingTimer)
	}
}

func TestConfigValidateWrongAlgo(t *testing.T) {
	cfg := ServerConfig{
		DeadTimeout:      90,
		SilenceTimeout:   10,
		PingTimer:        55,
		SchedulerAlgo:    "some",
		ListeningAddress: "127.0.0.1:1234",
		EtcdAddress:      "127.0.0.1:1235",
		BasicConfig: common.Config{
			LogLvl:  "debug",
			LogPath: "/some/path",
		},
		DialTimeout: 5,
	}

	err := cfgValidate(&cfg)
	if err == nil {
		t.Errorf("Nil error for config with algo %+v", cfg)
	}
}
