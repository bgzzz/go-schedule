package main

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

// ManagementNode main structure holding
// management node grpc server logic
type ManagementNode struct {
	// workerNodePool is sync map of WorkerNodes
	// that are active and connected to management
	// node
	workerNodePool    map[string]*wrpc.WorkerRPCClient
	workerNodePoolMtx sync.RWMutex

	// etcd is link to db
	etcd *clientv3.Client

	// scheduledTasks is sync map of tasks that are
	// scheduled at the moment
	scheduledTasks    map[string]*Task
	scheduledTasksMtx sync.RWMutex

	// cfg holds server config
	cfg *ServerConfig
}

// Etcd prefixes that are used for storing the objects to
// etcd
const (
	EtcdWorkerPrefix = "worker_"
	EtcdTaskPrefix   = "task_"
)

// AddWorkerNode thread safely adds worker node to worker pool and etcd
// errors in case of data inconsistency
func (mn *ManagementNode) AddWorkerNode(ctx context.Context,
	wn *wrpc.WorkerRPCClient) error {
	mn.workerNodePoolMtx.Lock()

	if _, ok := mn.workerNodePool[wn.WN.Id]; ok {
		mn.workerNodePoolMtx.Unlock()
		return trace.Errorf("There is a worker node with the same id %s in the worker node pool", wn.WN.Id)
	}

	log.Debugf("Add worker node %s to the worker node pool", wn.WN.Id)
	mn.workerNodePool[wn.WN.Id] = wn

	wn.WN.State = common.WorkerNodeStateConnected

	mn.workerNodePoolMtx.Unlock()

	err := mn.SetToDb(ctx, wn, EtcdWorkerPrefix+wn.WN.Id)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// RmWorkerNode thread safely removes worker node from the pool
// and changes state of the worker as disconnected
func (mn *ManagementNode) RmWorkerNode(ctx context.Context, wn *wrpc.WorkerRPCClient) error {
	mn.workerNodePoolMtx.Lock()

	if _, ok := mn.workerNodePool[wn.WN.Id]; !ok {
		mn.workerNodePoolMtx.Unlock()
		return trace.Errorf("There is no worker node with id %s in the worker node pool", wn.WN.Id)
	}

	log.Debugf("Rm worker node %s from the worker node pool", wn.WN.Id)

	delete(mn.workerNodePool, wn.WN.Id)

	wn.WN.State = common.WorkerNodeStateDisconnected

	mn.workerNodePoolMtx.Unlock()
	err := mn.SetToDb(ctx, wn, EtcdWorkerPrefix+wn.WN.Id)
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// WorkerConnect is gRPC method that holds bidirectional stream for
// management to worker node communication
func (mn *ManagementNode) WorkerConnect(stream pb.Scheduler_WorkerConnectServer) error {
	log.Info("WorkerConnect is called")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TBD NewWorkerNode func style
	// creating worker node that has connected
	wRPCClient := wrpc.NewWorkerNodeRPCClient("", stream, &pb.WorkerNode{
		Id:      "",
		Address: "TBD",
		State:   common.WorkerNodeStateConnected,
	}, mn.cfg.SilenceTimeout)

	// by architecture decision worker connects and send first
	// message to via stream
	// message have to contain its id in reply field
	msg, err := wRPCClient.RxWithTimeout(ctx)
	if err != nil {
		common.PrintDebugErr(err)
		return err
	}

	rsp := msg.(*pb.WorkerRsp)
	wRPCClient.SetId(rsp.Reply)

	err = mn.AddWorkerNode(ctx, wRPCClient)
	if err != nil {
		common.PrintDebugErr(err)
		return nil
	}

	c, cancel := context.WithCancel(ctx)

	// to make async communication between Worker and Manager
	// because sending via stream is possible only synchronously
	wRPCClient.StartSender(c)

	// launch pinger to check connection state
	wRPCClient.StartPinger(c, mn.cfg.PingTimer)

	// launch main eventloop
	if err = wRPCClient.InitLoop(c); err != nil {
		common.PrintDebugErr(err)
	}

	rmErr := mn.RmWorkerNode(c, wRPCClient)

	if rmErr != nil {
		//concat errors in case there are problems with db
		err = trace.Errorf("%s %s", err.Error(), rmErr.Error())
		common.PrintDebugErr(err)
	}

	wRPCClient.StopPinger()
	wRPCClient.StopSender(cancel)

	return err
}

// GetWorkerList gRPC that is called by schedctl with parameters
// of DummyReq (for now). In future can be used for proper querying
// and returns list of workers that reside in db
// TBD: can be done with pagination and limiting on rsp
//		or even changed to stream
func (md *ManagementNode) GetWorkerList(ctx context.Context, req *pb.DummyReq) (*pb.WorkerNodeList, error) {

	log.Info("GetWorkerList")
	// for now it gets all recorded workers
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	eCtx, cancel := context.WithTimeout(ctx,
		md.cfg.EtcdDialTimeout)
	gr, err := md.etcd.Get(eCtx, EtcdWorkerPrefix, opts...)
	cancel()
	if err != nil {
		common.PrintDebugErr(err)
		return nil, err
	}

	wnl := pb.WorkerNodeList{}

	for _, item := range gr.Kvs {
		var wn wrpc.WorkerRPCClient
		err := json.Unmarshal([]byte(item.Value), &wn)
		if err != nil {
			common.PrintDebugErr(err)
			return nil, err
		}
		wnl.Nodes = append(wnl.Nodes, wn.WN)
	}

	return &wnl, nil
}

// GetTaskList is gRPC that is called by schedctl with parameters
// of DummyReq (for now). In future can be used for proper querying
// and returns list of tasks that reside in db
// TBD: can be done with pagination and limiting on rsp
//		or even changed to stream
func (md *ManagementNode) GetTaskList(ctx context.Context, req *pb.DummyReq) (*pb.TaskList, error) {

	log.Info("GetTaskList")
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	eCtx, cancel := context.WithTimeout(ctx,
		md.cfg.EtcdDialTimeout)
	gr, err := md.etcd.Get(eCtx, EtcdTaskPrefix, opts...)
	cancel()
	if err != nil {
		common.PrintDebugErr(err)
		return nil, err
	}

	tl := pb.TaskList{}

	for _, item := range gr.Kvs {
		var task pb.Task
		err := json.Unmarshal([]byte(item.Value), &task)
		if err != nil {
			common.PrintDebugErr(err)
			return nil, err
		}
		tl.Tasks = append(tl.Tasks, &task)
	}

	return &tl, nil
}

// Schedule is gRPC that is called by schedctl to schedule tasks
// among connected workers. As parameters it receives list of tasks.
// As return value it return list of tasks and their intermediate state.
// Task call flow is following: received by Scheduled method ->
// stored in db with pending state -> send to worker -> rxed confirmation ->
// change state to scheduled , start dead timer -> store in db -> return
// TBD: function is too big, split on smaller parts

// change it like this
// store in db and return
// send to workers
// store in db
// rx timoeout or call
// store in db
func (md *ManagementNode) Schedule(ctx context.Context, req *pb.TaskList) (*pb.TaskList, error) {
	for _, task := range req.Tasks {
		task.State = common.TaskStatePending
		task.Id = uuid.New().String()
	}

	if err := md.setTasksToDb(ctx, req.Tasks); err != nil {
		common.PrintDebugErr(err)
		return nil, err
	}

	go md.schedule(req.Tasks)

	return req, nil
}

func (md *ManagementNode) setTasksDead(ctx context.Context, tasks []*pb.Task) {
	for _, task := range tasks {
		task.State = common.TaskStateDead
		task.Id = uuid.New().String()

		if err := md.SetToDb(ctx, task, EtcdTaskPrefix+task.Id); err != nil {
			common.PrintDebugErr(err)
			return
		}

	}
}

func (md *ManagementNode) schedule(tasks []*pb.Task) {
	// get workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	workers, err := md.GetWorkerList(ctx, &pb.DummyReq{})
	if err != nil {
		common.PrintDebugErr(err)
		return
	}

	activeNodes := []*pb.WorkerNode{}
	for _, node := range workers.Nodes {
		if node.State == common.WorkerNodeStateConnected {
			activeNodes = append(activeNodes, node)
		}
	}

	if len(activeNodes) == 0 {
		log.Warning("There are no connected workers: all tasks are dead")
		md.setTasksDead(ctx, tasks)
		return
	}

	// choose workers to execurte the task
	for i, task := range tasks {

		worker := activeNodes[0]
		switch md.cfg.SchedulerAlgo {
		case SchedAlgoRR:
			{
				worker = activeNodes[int(math.
					Mod(float64(i),
						float64(len(activeNodes))))]
			}
		case SchedAlgoRand:
			{
				s := rand.NewSource(time.Now().UnixNano())
				r := rand.New(s)

				worker = activeNodes[r.Intn(len(activeNodes))]
			}
		default:
			{
				log.Error("Unsupported scheduling algorithm is used")
				return
			}
		}

		task.WorkerId = worker.Id

		// setDb
		if err := md.SetToDb(ctx, task, EtcdTaskPrefix+task.Id); err != nil {
			common.PrintDebugErr(err)
			return
		}

		// execute worker
		go md.executeWorker(*task)
	}

}

func (md *ManagementNode) executeWorker(t pb.Task) {

	ctx := context.Background()

	// start timer
	// setup dead timeout
	md.SetTaskAndRunDeadTimeout(ctx, &Task{
		task: &t,
		cfg:  md.cfg,
		rxed: make(chan struct{}),
	})

	// handlers
	onRspHandler := func(c context.Context, rsp *pb.WorkerRsp) {

		// TBD: validate reply

		t.State = rsp.Reply
		if err := md.SetToDb(c, &t, EtcdTaskPrefix+t.Id); err != nil {
			common.PrintDebugErr(err)
		}
	}

	onTimerExpiredHandler := func(c context.Context) {
		err := trace.Errorf("timer is expired for rsp on task %s", t.Id)
		common.PrintDebugErr(err)

		if err := md.StopTaskDeadTimeout(t.Id); err != nil {
			common.PrintDebugErr(err)
		}

		md.DelTask(t.Id)

		t.State = common.TaskStateDead
		t.WorkerError = trace.DebugReport(err)

		if err := md.SetToDb(c, &t, EtcdTaskPrefix+t.Id); err != nil {
			common.PrintDebugErr(err)
		}
	}

	md.workerNodePoolMtx.RLock()
	defer md.workerNodePoolMtx.RUnlock()

	//check if it still active
	worker, ok := md.workerNodePool[t.WorkerId]
	if !ok {
		md.taskScheduledOnDisconnectedWorker(ctx, worker, t)
		return
	}

	// prepare req
	// combine parameters with cmd definition
	params := append([]string{t.Cmd}, t.Parameters...)

	// prepare request
	req := pb.MgmtReq{
		Id:     t.Id,
		Method: common.WorkerNodeRPCExec,
		Params: params,
	}

	worker.SendWithHandlerTimeout(ctx, req, onRspHandler,
		onTimerExpiredHandler,
		md.cfg.SilenceTimeout)
}

func (md *ManagementNode) taskScheduledOnDisconnectedWorker(ctx context.Context, w *wrpc.WorkerRPCClient, t pb.Task) {
	err := trace.Errorf("Worker is not active %s", w.WN.Id)
	common.PrintDebugErr(err)

	if err := md.StopTaskDeadTimeout(t.Id); err != nil {
		common.PrintDebugErr(err)
	}

	md.DelTask(t.Id)

	t.State = common.TaskStateDead
	t.WorkerError = trace.DebugReport(err)

	go func(c context.Context) {
		if err := md.SetToDb(c, &t, EtcdTaskPrefix+t.Id); err != nil {
			common.PrintDebugErr(err)
		}
	}(ctx)
	return
}

// SetToDb stores subj to etcd by id
func (md *ManagementNode) SetToDb(ctx context.Context,
	subj interface{}, id string) error {
	b, err := json.Marshal(subj)
	if err != nil {
		return trace.Wrap(err)
	}

	c, cancel := context.WithTimeout(ctx,
		md.cfg.EtcdDialTimeout)
	_, err = md.etcd.Put(c, id, string(b))
	cancel()
	if err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// SetTaskState is gRPC that is called by worker node
// to set state of the task that was previously executed
// it stops dead timeout
// and sets task to db
func (md *ManagementNode) SetTaskState(ctx context.Context, task *pb.Task) (*pb.Empty, error) {

	// TBD: check of existed object might be needed
	// might be dead before state change

	log.Infof("SetTaskState with params %+v", task)

	if err := md.StopTaskDeadTimeout(task.Id); err != nil {
		common.PrintDebugErr(err)
		return nil, err
	}

	if err := md.SetToDb(ctx, task, EtcdTaskPrefix+task.Id); err != nil {
		common.PrintDebugErr(err)
		return nil, err
	}

	return &pb.Empty{}, nil
}

// setTasksToDb is helper function for setting bunch of tasks
// to etcd
func (md *ManagementNode) setTasksToDb(ctx context.Context,
	tasks []*pb.Task) error {
	for _, task := range tasks {
		err := md.SetToDb(ctx, task, EtcdTaskPrefix+task.Id)
		if err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

// PrepareZeroState sets proper state for workers and tasks
// in db
// workers are all set to disconnected
// task with pending and scheduled are all set to dead
func (md *ManagementNode) PrepareZeroState() error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tl, err := md.GetTaskList(ctx, &pb.DummyReq{})
	if err != nil {
		return trace.Wrap(err)
	}

	for _, task := range tl.Tasks {
		if task.State != common.TaskStateDone && task.State != common.TaskStateDead {
			task.State = common.TaskStateDead
			err := md.SetToDb(ctx, task, EtcdTaskPrefix+task.Id)
			if err != nil {
				return trace.Wrap(err)
			}
		}
	}

	wl, err := md.GetWorkerList(ctx, &pb.DummyReq{})
	if err != nil {
		return trace.Wrap(err)
	}

	for _, worker := range wl.Nodes {
		if worker.State == common.WorkerNodeStateConnected {
			worker.State = common.WorkerNodeStateDisconnected

			wrapper := &wrpc.WorkerRPCClient{
				WN: worker,
			}

			err := md.SetToDb(ctx, wrapper, EtcdWorkerPrefix+worker.Id)
			if err != nil {
				return trace.Wrap(err)
			}
		}
	}

	return nil
}

// SetTaskAndRunDeadTimeout sets task to the scheduled tasks
// storage and setups the dead timeout for this task
func (md *ManagementNode) SetTaskAndRunDeadTimeout(ctx context.Context,
	t *Task) {
	md.scheduledTasksMtx.Lock()

	md.scheduledTasks[t.task.Id] = t

	md.scheduledTasksMtx.Unlock()

	t.StartDeadTimeout(ctx,
		func(ctx context.Context) {
			t.task.State = common.TaskStateDead
			err := md.SetToDb(ctx, t.task, EtcdTaskPrefix+t.task.Id)
			if err != nil {
				common.PrintDebugErr(err)
			}

			md.DelTask(EtcdTaskPrefix + t.task.Id)
		})
}

// StopTaskDeadTimeout stops the dead timeout of the task
// with id
func (md *ManagementNode) StopTaskDeadTimeout(id string) error {

	md.scheduledTasksMtx.RLock()
	defer md.scheduledTasksMtx.RUnlock()

	task, ok := md.scheduledTasks[id]
	if !ok {
		return trace.Errorf("There is no task %s in the scheduled task storage", id)
	}

	task.StopDeadTimeout()

	return nil

}

// DelTask is thread safely deletes the task from schedule
// task storage
func (md *ManagementNode) DelTask(id string) {
	md.scheduledTasksMtx.Lock()
	defer md.scheduledTasksMtx.Unlock()

	delete(md.scheduledTasks, id)

}
