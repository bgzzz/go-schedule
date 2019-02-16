package main

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/bgzzz/go-schedule/common"

	"github.com/gravitational/trace"
)

const (
	DefaultTmpFolderPath = "/tmp/go-schedule/worker-node/"
)

type WorkerNodeConfig struct {
	// BasicConfig is common config for all
	// parts of the solution
	BasicConfig common.Config `yaml:"basic_config"`

	//client specific config guration
	ServerAddress string `yaml:"server_address"`

	// ConnectionTimeout is used for gRPC context setup
	ConnectionTimeout time.Duration `yaml:"connetion_timeout"`

	SilenceTimeout time.Duration `yaml:"silence_timeout"`
}

// parseClientCfgFile parses config yaml file to
// global config varaible
func parseClientCfgFile(filePath string) (*WorkerNodeConfig, error) {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var cfg WorkerNodeConfig
	err = yaml.Unmarshal(dat, &cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &cfg, nil
}
