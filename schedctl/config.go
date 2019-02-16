package main

import (
	"github.com/bgzzz/go-schedule/common"

	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/gravitational/trace"
)

var Config *SchedCtlConfig

type SchedCtlConfig struct {
	// BasicConfig config that is common for all parts
	// of the solution
	BasicConfig common.Config `yaml:"basic_config"`

	// ServerAddress of the management node
	ServerAddress string `yaml:"server_address"`

	// ConnectionTimeout is timeout for RPC call
	ConnectionTimeout int `yaml:"connection_timeout"`
}

// parseClientCfgFile parses the yaml file and
// stores to global config variable
func parseClientCfgFile(filePath string) error {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return trace.Wrap(err)
	}

	var cfg SchedCtlConfig
	err = yaml.Unmarshal(dat, &cfg)
	if err != nil {
		return trace.Wrap(err)
	}

	Config = &cfg
	return nil
}
