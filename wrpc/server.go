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
			stopSender:     make(chan struct{}, 1),
			silenceTimeout: silenceTimeout,
		},
	}
}

// execute is the worker RPC server's RPC that is called
// by management node in order to execute task
// function execute task described in MgmtReq
// and calls the SetTaskState after execution is successfully ended
func (wrpc *WorkerRPCServer) execute(req *pb.MgmtReq) {

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
	ctx, cancel := context.WithTimeout(context.Background(),
		wrpc.silenceTimeout*time.Second)
	_, err := wrpc.client.SetTaskState(ctx, &task)
	cancel()

	if err != nil {
		wrpc.SetErr(trace.Wrap(err))
	}
}

func (wrpc *WorkerRPCServer) ProcessRequest(r interface{}) error {
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

				wrpc.Send(rsp)

				wrpc.execute(req)

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
				wrpc.Send(rsp)
			}()

		}
	default:
		{
			err = trace.Errorf("There is no such method as %s", req.Method)
		}
	}
	return err
}

func (wrpc *WorkerRPCServer) Send(rsp *pb.WorkerRsp) {
	t := time.NewTimer(wrpc.silenceTimeout * time.Second)

	select {
	case wrpc.send <- rsp:
		{
			if !t.Stop() {
				<-t.C
			}
		}
		//this one is done to prevent go routines leaking
	case <-t.C:
		{
			log.Warningf("Timeouted to send rsp %s", rsp.Id)
		}
	}
}

func (wrpc *WorkerRPCServer) InitLoop() error {
	return wrpc.initLoop(wrpc.ProcessRequest)
}
