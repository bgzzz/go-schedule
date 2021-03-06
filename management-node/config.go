package main

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/bgzzz/go-schedule/common"
	"github.com/gravitational/trace"
)

// Scheduling algorithms that can be used for
// distributed scheduling
const (
	// SchedAlgoRR is the algorithm that equally
	// distributes jobs/tasks among connected
	// worker nodes
	SchedAlgoRR = "rr"
	// SchedAlgoRand is the algorithm that randomly
	// selects connected worker node to execute the
	// task/job
	SchedAlgoRand = "rand"
)

// ServerConfig is configuration of management node
type ServerConfig struct {
	// BasicConfig is basic configuration part of config
	// is is common for all parts of solution
	BasicConfig common.Config `yaml:"basic_config"`

	// Server and connection handling related configuration
	ListeningAddress string `yaml:"listening_address"`
	// PingTimer is period of sending ping messages to
	// worker via stream. Management node sends ping message
	// to worker node each PingTimer seconds
	PingTimer time.Duration `yaml:"ping_timer"`
	// SilenceTimeout is timeout that is used for checking
	// silence periods during bi-directional streaming
	// session. Server terminates stream if SilenceTimeout
	// expired
	SilenceTimeout time.Duration `yaml:"silence_timeout"`

	// Etcd related configuration
	// EtcdAddress is the address of db
	EtcdAddress string `yaml:"etcd_address"`
	// DialTimeout is timeout used for etcd querying
	EtcdDialTimeout time.Duration `yaml:"etcd_dial_timeout"`

	// Scheduler related config
	// SchedulerAlgo algorithm that is used for
	// scheduling
	SchedulerAlgo string `yaml:"scheduler_algo"`
	// DeadTimeout is timeout that is used to handle that
	// task will never end its execution. If timer expired
	// Task is defined as dead.
	DeadTimeout time.Duration `yaml:"dead_timeout"`
}

// parseCfgFile is parsing yaml file to config structure
func parseCfgFile(filePath string) (*ServerConfig, error) {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	var cfg ServerConfig
	err = yaml.Unmarshal(dat, &cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	cfg.EtcdDialTimeout = cfg.EtcdDialTimeout * time.Second
	cfg.DeadTimeout = cfg.DeadTimeout * time.Second
	cfg.PingTimer = cfg.PingTimer * time.Second
	cfg.SilenceTimeout = cfg.SilenceTimeout * time.Second

	return cfgValidate(&cfg)
}

// cfgValidate validates some config fields
func cfgValidate(cfg *ServerConfig) (*ServerConfig, error) {

	if cfg.SchedulerAlgo != SchedAlgoRR &&
		cfg.SchedulerAlgo != SchedAlgoRand {

		return nil,
			trace.Errorf("There is unknown scheduler algo (%s)", cfg.SchedulerAlgo)
	}

	// cfg.PingTimer has to be <= cfg.SilenceTimeout
	// due to async architecture
	if int(cfg.PingTimer) > int(cfg.SilenceTimeout) {
		return nil,
			trace.Errorf("Ping timer is > then silence timeout")
	}

	return cfg, nil
}
