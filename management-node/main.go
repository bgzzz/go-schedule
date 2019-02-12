package main

import (
	"flag"
	"fmt"
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
	flag.Parse()

	err := parseCfgFile(*filePath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = common.InitLogging(Config.BasicConfig.LogLvl, Config.BasicConfig.LogPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	lis, err := net.Listen("tcp", Config.ListeningAddress)
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{Config.EtcdAddress},
		DialTimeout: time.Duration(Config.DialTimeout) * time.Second,
	})
	if err != nil {
		log.Error(err.Error())
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
	}

	// setting stated on db objects
	// tasks have to be dead or done
	// workers have to be disconnected
	err = mn.PrepareZeroState()
	if err != nil {
		log.Error(err.Error())
		os.Exit(5)
	}

	pb.RegisterSchedulerServer(s, mn)

	go func() {
		log.Info("Start serving on %s", Config.ListeningAddress)
		if err := s.Serve(lis); err != nil {
			log.Errorf("Failed to serve on %s: %s", Config.ListeningAddress, err.Error())
			os.Exit(6)
		}
	}()

	<-stop
}
