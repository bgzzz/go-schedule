package main

import (
	"flag"
	"net"
	"os"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/coreos/etcd/clientv3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	DefaultCfgFilePath = "/etc/go-schedule/management-node.yaml"
)

func main() {

	filePath := flag.String("f", DefaultCfgFilePath, "-f management-node-cfg.yaml, provides config file path")
	verbose := flag.Bool("v", false, "-v , verbose all the output: equal to debug in config file")

	flag.Parse()

	cfg, err := parseCfgFile(*filePath)
	if err != nil {
		common.PrintErr(err, *verbose, "")
		os.Exit(1)
	}

	if *verbose {
		cfg.BasicConfig.LogLvl = "debug"
	}

	err = common.InitLogging(&cfg.BasicConfig)
	if err != nil {
		common.PrintErr(err, *verbose, cfg.BasicConfig.LogLvl)
		os.Exit(2)
	}

	lis, err := net.Listen("tcp", cfg.ListeningAddress)
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.EtcdAddress},
		DialTimeout: cfg.EtcdDialTimeout * time.Second,
	})
	if err != nil {
		common.PrintDebugErr(err)
		os.Exit(4)
	}

	// channel for blocking main
	stop := make(chan struct{})

	// TBD: change to NewManagementNode function style
	// creating grpc server
	s := grpc.NewServer()
	mn := &ManagementNode{
		etcd:           cli,
		workerNodePool: make(map[string]*wrpc.WorkerRPCClient),
		scheduledTasks: make(map[string]*Task),
		cfg:            cfg,
	}

	// setting stated on db objects
	// tasks have to be dead or done
	// workers have to be disconnected
	err = mn.PrepareZeroState()
	if err != nil {
		common.PrintDebugErr(err)
		os.Exit(5)
	}

	pb.RegisterSchedulerServer(s, mn)

	go func() {
		log.Infof("Start serving on %s", cfg.ListeningAddress)
		if err := s.Serve(lis); err != nil {
			log.Errorf("Failed to serve on %s: %s", cfg.ListeningAddress, err.Error())
			os.Exit(6)
		}
	}()

	<-stop
}
