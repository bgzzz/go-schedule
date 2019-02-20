package main

import (
	"flag"
	"os"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	DefaultCfgFilePath = "/etc/go-schedule/worker-node.yaml"
)

func main() {

	verbose := flag.Bool("v", false, "-v for verbose debug output ")
	filePath := flag.String("f", DefaultCfgFilePath, "-f worker-node.yaml, provides config file path")
	flag.Parse()

	cfg, err := parseClientCfgFile(*filePath)
	if err != nil {
		common.PrintErr(err, *verbose, "")
		os.Exit(1)
	}

	if *verbose {
		cfg.BasicConfig.LogLvl = "debug"
	}

	err = common.InitLogging(&cfg.BasicConfig)
	if err != nil {
		common.PrintErr(err, *verbose, "")
		os.Exit(2)
	}

	// Set up a connection to the server.

	for {
		conn, err := grpc.Dial(cfg.ServerAddress, grpc.WithInsecure())
		if err != nil {
			common.PrintDebugErr(err)
			os.Exit(3)
		}
		client := pb.NewSchedulerClient(conn)

		if err := RunWorkerRPCServer(client, cfg); err != nil {
			common.PrintDebugErr(err)
		}

		conn.Close()

		log.Info("Reconnecting ...")
		time.Sleep(cfg.ReconnectTimeout)
	}

}
