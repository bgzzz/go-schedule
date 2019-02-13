package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
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
}

// Etcd prefixes that are used for storing the objects to
// etcd
const (
	EtcdWorkerPrefix = "worker_"
	EtcdTaskPrefix   = "task_"
)

// AddWorkerNode thread safely adds worker node to worker pool and etcd
// errors in case of data inconsistency
func (mn *ManagementNode) AddWorkerNode(wn *wrpc.WorkerRPCClient) error {
	mn.workerNodePoolMtx.Lock()
	defer mn.workerNodePoolMtx.Unlock()

	if _, ok := mn.workerNodePool[wn.WN.Id]; ok {
		return fmt.Errorf("There is a worker node with the same id %s in the worker node pool", wn.WN.Id)
	}

	log.Debugf("Add worker node %s to the worker node pool", wn.WN.Id)
	mn.workerNodePool[wn.WN.Id] = wn

	wn.WN.State = common.WorkerNodeStateConnected

	err := mn.SetToDb(wn, EtcdWorkerPrefix+wn.WN.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}

// RmWorkerNode thread safely removes worker node from the pool
// and changes state of the worker as disconnected
func (mn *ManagementNode) RmWorkerNode(wn *wrpc.WorkerRPCClient) error {
	mn.workerNodePoolMtx.Lock()
	defer mn.workerNodePoolMtx.Unlock()

	if _, ok := mn.workerNodePool[wn.WN.Id]; !ok {
		return fmt.Errorf("There is no worker node with id %s in the worker node pool", wn.WN.Id)
	}

	log.Debugf("Rm worker node %s from the worker node pool", wn.WN.Id)

	delete(mn.workerNodePool, wn.WN.Id)

	wn.WN.State = common.WorkerNodeStateDisconnected

	err := mn.SetToDb(wn, EtcdWorkerPrefix+wn.WN.Id)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}

// WorkerConnect is gRPC method that holds bidirectional stream for
// management to worker node communication
func (mn *ManagementNode) WorkerConnect(stream pb.Scheduler_WorkerConnectServer) error {
	log.Info("WorkerConnect is called")

	// TBD NewWorkerNode func style
	// creating worker node that has connected
	wRPCClient := wrpc.NewWorkerNodeRPCClient("", stream, &pb.WorkerNode{
		Id:      "",
		Address: "TBD",
		State:   common.WorkerNodeStateConnected,
	}, Config.SilenceTimeout)

	// by architecture decision worker connects and send first
	// message to via stream
	// message have to contain its id in reply field
	msg, err := wRPCClient.RxWithTimeout()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	rsp := msg.(*pb.WorkerRsp)
	wRPCClient.SetId(rsp.Reply)

	err = mn.AddWorkerNode(wRPCClient)
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	// to make async communication between Worker and Manager
	// because sending via stream is possible only synchronously
	wRPCClient.StartSender()

	// launch pinger to check connection state
	wRPCClient.StartPinger(time.Duration(Config.PingTimer))

	// launch main eventloop
	if err := wRPCClient.InitLoop(); err != nil {
		log.Error(err.Error())
	}

	wRPCClient.StopPinger()
	wRPCClient.StopSender()

	rmErr := mn.RmWorkerNode(wRPCClient)
	if rmErr != nil {
		//concat errors in case there are problems with db
		err = fmt.Errorf("%s %s", err.Error(), rmErr.Error())
		log.Error(err.Error())
	}

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

	eCtx, cancel := context.WithTimeout(context.Background(),
		time.Duration(Config.DialTimeout)*time.Second)
	gr, err := md.etcd.Get(eCtx, EtcdWorkerPrefix, opts...)
	cancel()
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	wnl := pb.WorkerNodeList{}

	for _, item := range gr.Kvs {
		var wn wrpc.WorkerRPCClient
		err := json.Unmarshal([]byte(item.Value), &wn)
		if err != nil {
			log.Error(err.Error())
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

	eCtx, cancel := context.WithTimeout(context.Background(),
		time.Duration(Config.DialTimeout)*time.Second)
	gr, err := md.etcd.Get(eCtx, EtcdTaskPrefix, opts...)
	cancel()
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	tl := pb.TaskList{}

	for _, item := range gr.Kvs {
		var task pb.Task
		err := json.Unmarshal([]byte(item.Value), &task)
		if err != nil {
			log.Error(err.Error())
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
func (md *ManagementNode) Schedule(ctx context.Context, req *pb.TaskList) (*pb.TaskList, error) {
	md.workerNodePoolMtx.RLock()
	defer md.workerNodePoolMtx.RUnlock()

	//make slice of connected workers
	var workers []string
	for k, _ := range md.workerNodePool {
		workers = append(workers, k)
	}

	// if no connected workers set state of tasks requested to schedule to
	// dead
	// TBD: make it in waiting for active workers to be scheduled
	if len(workers) == 0 {

		log.Warning("There are no connected workers: all tasks are dead")
		for _, task := range req.Tasks {
			task.State = common.TaskStateDead
			task.Id = uuid.New().String()
		}

		if err := md.setTasksToDb(req.Tasks); err != nil {
			log.Error(err.Error())
			return nil, err
		}

		return req, nil
	}

	// msg type for internal use
	type msg struct {
		err  error
		task *pb.Task
	}

	// channel for connecting onRsp handlers
	// to the current go routine
	messanger := make(chan msg, 1)

	for i, task := range req.Tasks {

		// all the ids are generated by management node
		id := uuid.New().String()
		workerId := workers[0]

		// RR worker id calculation
		if Config.SchedulerAlgo == SchedAlgoRR {
			workerId = workers[int(math.
				Mod(float64(i),
					float64(len(workers))))]
		} else if Config.SchedulerAlgo == SchedAlgoRand {
			s := rand.NewSource(time.Now().UnixNano())
			r := rand.New(s)

			workerId = workers[r.Intn(len(workers))]

		}

		// set task state to pending
		task.Id = id
		task.State = common.TaskStatePending
		task.WorkerId = workerId

		if err := md.SetToDb(task, EtcdTaskPrefix+id); err != nil {
			log.Error(err.Error())
			return nil, err
		}

		// setup handlers
		// copy object to handler
		t := *task
		onRspHandler := func(rsp *pb.WorkerRsp) {

			// TBD: validate reply

			t.State = rsp.Reply
			var err error
			if err = md.SetToDb(&t, EtcdTaskPrefix+t.Id); err != nil {
				log.Error(err.Error())
			}

			messanger <- msg{
				task: &t,
				err:  err,
			}
		}

		onTimerExpiredHandler := func() {
			err := fmt.Errorf("timer is expired for rsp on task %s", task.Id)
			log.Error(err.Error())
			messanger <- msg{
				err:  err,
				task: &t,
			}
		}

		// combine parameters with cmd definition
		params := append([]string{task.Cmd}, task.Parameters...)

		// prepare request
		req := pb.MgmtReq{
			Id:     task.Id,
			Method: common.WorkerNodeRPCExec,
			Params: params,
		}

		// task is scheduled
		// setup dead timeout
		md.SetTaskAndRunDeadTimeout(&Task{
			task: task,
			stop: make(chan struct{}, 1),
		})

		// sending to worker node
		md.workerNodePool[workerId].
			SendWithHandlerTimeout(req, onRspHandler,
				onTimerExpiredHandler,
				time.Duration(Config.SilenceTimeout))
	}

	// prepare task objects to response to schedctl
	tasks := []*pb.Task{}
	for _ = range req.Tasks {
		m := <-messanger

		// check if it responded with error
		if m.err != nil {
			log.Error(m.err)

			// set error to worker error field
			m.task.WorkerError = m.err.Error()
			if err := md.SetToDb(m.task, EtcdTaskPrefix+m.task.Id); err != nil {
				log.Error(err.Error())
				return nil, err
			}
			tasks = append(tasks, m.task)
			continue
		}

		tasks = append(tasks, m.task)
	}

	close(messanger)

	return &pb.TaskList{Tasks: tasks}, nil

}

// SetToDb stores subj to etcd by id
func (md *ManagementNode) SetToDb(subj interface{}, id string) error {
	b, err := json.Marshal(subj)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(Config.DialTimeout)*time.Second)
	_, err = md.etcd.Put(ctx, id, string(b))
	cancel()
	if err != nil {
		log.Error(err.Error())
		return err
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
		log.Error(err.Error())
		return nil, err
	}

	if err := md.SetToDb(task, EtcdTaskPrefix+task.Id); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return &pb.Empty{}, nil
}

// setTasksToDb is helper function for setting bunch of tasks
// to etcd
func (md *ManagementNode) setTasksToDb(tasks []*pb.Task) error {
	for _, task := range tasks {
		err := md.SetToDb(task, EtcdTaskPrefix+task.Id)
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}
	return nil
}

// PrepareZeroState sets proper state for workers and tasks
// in db
// workers are all set to disconnected
// task with pending and scheduled are all set to dead
func (md *ManagementNode) PrepareZeroState() error {

	tl, err := md.GetTaskList(context.Background(), &pb.DummyReq{})
	if err != nil {
		log.Error(err.Error())
		return err
	}

	for _, task := range tl.Tasks {
		if task.State != common.TaskStateDone && task.State != common.TaskStateDead {
			task.State = common.TaskStateDead
			err := md.SetToDb(task, EtcdTaskPrefix+task.Id)
			if err != nil {
				log.Error(err.Error())
				return err
			}
		}
	}

	wl, err := md.GetWorkerList(context.Background(), &pb.DummyReq{})
	if err != nil {
		log.Error(err.Error())
		return err
	}

	for _, worker := range wl.Nodes {
		if worker.State == common.WorkerNodeStateConnected {
			worker.State = common.WorkerNodeStateDisconnected

			wrapper := &wrpc.WorkerRPCClient{
				WN: worker,
			}

			err := md.SetToDb(wrapper, EtcdWorkerPrefix+worker.Id)
			if err != nil {
				log.Error(err.Error())
				return err
			}
		}
	}

	return nil
}

// SetTaskAndRunDeadTimeout sets task to the scheduled tasks
// storage and setups the dead timeout for this task
func (md *ManagementNode) SetTaskAndRunDeadTimeout(t *Task) {
	md.scheduledTasksMtx.Lock()

	md.scheduledTasks[t.task.Id] = t

	md.scheduledTasksMtx.Unlock()

	t.StartDeadTimeout(func() {
		t.task.State = common.TaskStateDead
		err := md.SetToDb(t.task, EtcdTaskPrefix+t.task.Id)
		if err != nil {
			log.Error(err.Error())
		}
	})
}

// StopTaskDeadTimeout stops the dead timeout of the task
// with id
func (md *ManagementNode) StopTaskDeadTimeout(id string) error {

	md.scheduledTasksMtx.RLock()
	md.scheduledTasksMtx.RUnlock()

	task, ok := md.scheduledTasks[id]
	if !ok {
		err := fmt.Errorf("There is no task %s in the scheduled task storage", id)
		log.Error(err.Error())
		return err
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
