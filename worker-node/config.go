package main

import (
	"github.com/bgzzz/go-schedule/common"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

const (
	DefaultTmpFolderPath = "/tmp/go-schedule/worker-node/"
)

var Config *WorkerNodeConfig

type WorkerNodeConfig struct {
	// BasicConfig is common config for all
	// parts of the solution
	BasicConfig common.Config `yaml:"basic_config"`

	//client specific config guration
	ServerAddress string `yaml:"server_address"`

	// ConnectionTimeout is used for gRPC context setup
	ConnectionTimeout int `yaml:"connetion_timeout"`

	SilenceTimeout int `yaml:"silence_timeout"`
}

// parseClientCfgFile parses config yaml file to
// global config varaible
func parseClientCfgFile(filePath string) error {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	var cfg WorkerNodeConfig
	err = yaml.Unmarshal(dat, &cfg)
	if err != nil {
		return err
	}

	Config = &cfg
	return nil
}
