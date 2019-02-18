package main

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/bgzzz/go-schedule/common"

	"github.com/gravitational/trace"
)

type SchedCtlConfig struct {
	// BasicConfig config that is common for all parts
	// of the solution
	BasicConfig common.Config `yaml:"basic_config"`

	// ServerAddress of the management node
	ServerAddress string `yaml:"server_address"`

	// ConnectionTimeout is timeout for RPC call
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
}

// parseClientCfgFile parses the yaml file and
// stores to global config variable
func parseClientCfgFile(filePath string) (*SchedCtlConfig, error) {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var cfg SchedCtlConfig
	err = yaml.Unmarshal(dat, &cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	cfg.ConnectionTimeout = cfg.ConnectionTimeout * time.Second

	return &cfg, nil
}
