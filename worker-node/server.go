package main

import (
	"context"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/google/uuid"
	"github.com/gravitational/trace"
)

func RunWorkerRPCServer(client pb.SchedulerClient, cfg *WorkerNodeConfig) error {

	ctx := context.Background()

	stream, err := client.WorkerConnect(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	// for now generating uuid
	// has to be fixed in future
	id := uuid.New().String()

	wRPCServer := wrpc.NewWorkerNodeRPCServer(id, stream, client, cfg.SilenceTimeout)

	c, cancel := context.WithCancel(ctx)
	wRPCServer.StartSender(c)

	// initiation rsp
	rsp := &pb.WorkerRsp{
		Reply: id,
	}
	wRPCServer.Send(c, rsp)

	if err := wRPCServer.InitLoop(c); err != nil {
		common.PrintDebugErr(err)
	}

	wRPCServer.StopSender(cancel)

	if err != nil {
		return trace.Wrap(err)
	}

	return nil

}
