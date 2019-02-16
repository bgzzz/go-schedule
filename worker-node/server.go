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

	wRPCServer.StartSender()

	// initiation rsp
	rsp := &pb.WorkerRsp{
		Reply: id,
	}
	wRPCServer.Send(rsp)

	if err := wRPCServer.InitLoop(); err != nil {
		common.PrintDebugErr(err)
	}

	wRPCServer.StopSender()

	if err != nil {
		return trace.Wrap(err)
	}

	return nil

}
