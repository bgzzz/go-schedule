package main

import (
	"flag"
	"os"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"

	"google.golang.org/grpc"
)

const (
	DefaultCfgFilePath = "/etc/go-schedule/worker-node.yaml"
)

func main() {

	verbose := flag.Bool("v", false, "-v for verbose debug output ")
	filePath := flag.String("f", DefaultCfgFilePath, "-f worker-node.yaml, provides config file path")
	flag.Parse()

	err := parseClientCfgFile(*filePath)
	if err != nil {
		common.PrintErr(err, *verbose, "")
		os.Exit(1)
	}

	if *verbose {
		Config.BasicConfig.LogLvl = "debug"
	}

	err = common.InitLogging(&Config.BasicConfig)
	if err != nil {
		common.PrintErr(err, *verbose, "")
		os.Exit(2)
	}

	// Set up a connection to the server.

	conn, err := grpc.Dial(Config.ServerAddress, grpc.WithInsecure())
	if err != nil {
		common.PrintDebugErr(err)
		os.Exit(3)
	}
	defer conn.Close()
	client := pb.NewSchedulerClient(conn)

	if err := RunWorkerRPCServer(client); err != nil {
		common.PrintDebugErr(err)
	}

}
