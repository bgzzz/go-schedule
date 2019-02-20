// +build unit

package main

import (
	"testing"
	"time"

	"github.com/bgzzz/go-schedule/common"
)

func TestConfigValidate(t *testing.T) {
	cfg := ServerConfig{
		DeadTimeout:      90 * time.Second,
		SilenceTimeout:   10 * time.Second,
		PingTimer:        5 * time.Second,
		SchedulerAlgo:    "rr",
		ListeningAddress: "127.0.0.1:1234",
		EtcdAddress:      "127.0.0.1:1235",
		BasicConfig: common.Config{
			LogLvl:  "debug",
			LogPath: "/some/path",
		},
		EtcdDialTimeout: 5 * time.Second,
	}

	_, err := cfgValidate(&cfg)
	if err != nil {
		t.Errorf("Non nil error %s", err.Error())
	}
}

func TestConfigValidateWithTimers(t *testing.T) {
	cfg := ServerConfig{
		DeadTimeout:      90 * time.Second,
		SilenceTimeout:   10 * time.Second,
		PingTimer:        55 * time.Second,
		SchedulerAlgo:    "rr",
		ListeningAddress: "127.0.0.1:1234",
		EtcdAddress:      "127.0.0.1:1235",
		BasicConfig: common.Config{
			LogLvl:  "debug",
			LogPath: "/some/path",
		},
		EtcdDialTimeout: 5 * time.Second,
	}

	_, err := cfgValidate(&cfg)
	if err == nil {
		t.Errorf("Should be en error because ping timer > silence timer")
	}

}

func TestConfigValidateWrongAlgo(t *testing.T) {
	cfg := ServerConfig{
		DeadTimeout:      90 * time.Second,
		SilenceTimeout:   10 * time.Second,
		PingTimer:        55 * time.Second,
		SchedulerAlgo:    "some",
		ListeningAddress: "127.0.0.1:1234",
		EtcdAddress:      "127.0.0.1:1235",
		BasicConfig: common.Config{
			LogLvl:  "debug",
			LogPath: "/some/path",
		},
		EtcdDialTimeout: 5 * time.Second,
	}

	_, err := cfgValidate(&cfg)
	if err == nil {
		t.Errorf("Nil error for config with algo %+v", cfg)
	}
}
