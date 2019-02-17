package wrpc

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

type WorkerRPCServer struct {
	*Worker

	client pb.SchedulerClient

	err      error
	errorMtx sync.RWMutex
}

func NewWorkerNodeRPCServer(id string,
	stream pb.Scheduler_WorkerConnectClient,
	client pb.SchedulerClient,
	silenceTimeout time.Duration) *WorkerRPCServer {
	return &WorkerRPCServer{
		client: client,
		Worker: &Worker{
			id:             id,
			streamServer:   nil,
			streamClient:   stream,
			send:           make(chan interface{}, 1),
			silenceTimeout: silenceTimeout,
		},
	}
}

// execute is the worker RPC server's RPC that is called
// by management node in order to execute task
// function execute task described in MgmtReq
// and calls the SetTaskState after execution is successfully ended
func (wrpc *WorkerRPCServer) execute(ctx context.Context, req *pb.MgmtReq) {

	var task pb.Task
	if len(req.Params) == 0 {
		task = pb.Task{
			Id:       req.Id,
			State:    common.TaskStateDone,
			WorkerId: wrpc.id,
		}
	} else {

		cmd := exec.Command(req.Params[0], req.Params[1:]...)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		//could be that we need to check for the error type here
		if err != nil {
			log.Warningf("cmd.Run() failed with %s\n", err.Error())
		}
		outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())

		task = pb.Task{
			Id:         req.Id,
			Cmd:        req.Params[0],
			Parameters: req.Params[1:],
			State:      common.TaskStateDone,
			Stdout:     outStr,
			Stderr:     errStr,
			WorkerId:   wrpc.id,
		}
	}

	// task is done setting task state to management node
	c, cancel := context.WithTimeout(ctx,
		wrpc.silenceTimeout*time.Second)
	_, err := wrpc.client.SetTaskState(c, &task)
	cancel()

	if err != nil {
		wrpc.SetErr(trace.Wrap(err))
	}
}

func (wrpc *WorkerRPCServer) ProcessRequest(ctx context.Context, r interface{}) error {
	req := r.(*pb.MgmtReq)
	var err error
	switch req.Method {
	case common.WorkerNodeRPCExec:
		{
			go func() {
				fmt.Printf("Executing %s with params %+v\n", req.Method, req.Params)

				rsp := &pb.WorkerRsp{
					Id:    req.Id,
					Reply: common.TaskStateScheduled,
				}

				wrpc.Send(ctx, rsp)

				wrpc.execute(ctx, req)

			}()

		}
	case common.WorkerNodeRPCPing:
		{
			log.Debugf("rx: ping %s", req.Id)
			go func() {
				rsp := &pb.WorkerRsp{
					Id:    req.Id,
					Reply: common.WorkerNodeRPCPingReply,
				}
				wrpc.Send(ctx, rsp)
			}()

		}
	default:
		{
			err = trace.Errorf("There is no such method as %s", req.Method)
		}
	}
	return err
}

func (wrpc *WorkerRPCServer) Send(ctx context.Context, rsp *pb.WorkerRsp) {
	c, cancel := context.WithTimeout(ctx, wrpc.silenceTimeout*time.Second)
	defer cancel()

	select {
	case wrpc.send <- rsp:
		{

		}
		//this one is done to prevent go routines leaking
	case <-c.Done():
		{
			log.Warningf("Timeouted to send rsp %s", rsp.Id)
		}
	}
}

func (wrpc *WorkerRPCServer) InitLoop(ctx context.Context) error {
	return wrpc.initLoop(ctx, wrpc.ProcessRequest)
}
