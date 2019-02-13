package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	DefaultCfgFilePath = "/etc/go-schedule/worker-node.yaml"
)

func main() {

	filePath := flag.String("f", DefaultCfgFilePath, "-f worker-node.yaml, provides config file path")
	flag.Parse()

	err := parseClientCfgFile(*filePath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = common.InitLogging(&Config.BasicConfig)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	// Set up a connection to the server.

	conn, err := grpc.Dial(Config.ServerAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewSchedulerClient(conn)

	if err := RunWorkerRPCServer(client); err != nil {
		log.Error(err.Error())
	}

}
