package main

import (
	"context"

	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func RunWorkerRPCServer(client pb.SchedulerClient) error {

	ctx := context.Background()

	stream, err := client.WorkerConnect(ctx)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// for now generatin uuid
	// has to be fixed in future
	id := uuid.New().String()

	wRPCServer := wrpc.NewWorkerNodeRPCServer(id, stream, client, Config.SilenceTimeout)

	wRPCServer.StartSender()

	// initiation rsp
	rsp := &pb.WorkerRsp{
		Reply: id,
	}
	wRPCServer.Send(rsp)

	if err := wRPCServer.InitLoop(); err != nil {
		log.Error(err.Error())
	}

	wRPCServer.StopSender()

	if err != nil {
		log.Error(err.Error())
		return err
	}

	return nil

}
