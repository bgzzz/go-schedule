// +build int

package main

import (
	"context"
	"encoding/json"
	// "fmt"
	"reflect"
	"testing"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
)

const TestCfg = "./tests/management-node.yaml"

func prepareMn(t *testing.T) *ManagementNode {
	cfg, err := parseCfgFile(TestCfg)
	if err != nil {
		t.Error(err.Error())
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.EtcdAddress},
		DialTimeout: cfg.EtcdDialTimeout,
	})
	if err != nil {
		t.Error(err.Error())
	}

	//flush db
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	eCtx, cancel := context.WithTimeout(context.Background(),
		cfg.EtcdDialTimeout)
	_, err = cli.Delete(eCtx, EtcdWorkerPrefix, opts...)
	cancel()

	eCtx, cancel = context.WithTimeout(context.Background(),
		cfg.EtcdDialTimeout)
	_, err = cli.Delete(eCtx, EtcdTaskPrefix, opts...)
	cancel()

	return NewManagmentNode(cli, cfg)
}

func addWorker(mn *ManagementNode, id string, t *testing.T) {

	setObj := wrpc.NewWorkerNodeRPCClient(id, nil, newWorkerNode(id, common.WorkerNodeStateDisconnected), mn.cfg.SilenceTimeout)

	if err := mn.AddWorkerNode(context.Background(), setObj); err != nil {
		t.Error(err.Error())
	}
}

func getWorkerFromDb(MN *ManagementNode, id string, t *testing.T) *wrpc.WorkerRPCClient {
	eCtx, cancel := context.WithTimeout(context.Background(),
		MN.cfg.EtcdDialTimeout)
	gr, err := MN.etcd.Get(eCtx, id)
	cancel()
	if err != nil {
		t.Error(err.Error())
	}

	if len(gr.Kvs) == 0 {
		return nil
	}

	var w wrpc.WorkerRPCClient
	err = json.Unmarshal([]byte(gr.Kvs[0].Value), &w)
	if err != nil {
		t.Error(err.Error())
	}

	return &w
}

func newWorkerNode(id, state string) *pb.WorkerNode {
	return &pb.WorkerNode{
		Id:      id,
		State:   state,
		Address: "TBD",
	}
}

func TestAddWorkerNode(t *testing.T) {
	mn := prepareMn(t)

	id := uuid.New().String()

	addWorker(mn, id, t)

	w := getWorkerFromDb(mn, EtcdWorkerPrefix+id, t)
	if w == nil {
		t.Error("Empty response from db")
	}

	expectedObj := wrpc.NewWorkerNodeRPCClient(id, nil, newWorkerNode(id, common.WorkerNodeStateConnected), mn.cfg.SilenceTimeout)

	if !reflect.DeepEqual(w.WN, expectedObj.WN) {
		t.Errorf("should be equal (%+v != %+v)", w, expectedObj)
	}
}

func TestRmWorkerNode(t *testing.T) {
	id := uuid.New().String()
	mn := prepareMn(t)

	addWorker(mn, id, t)

	w := wrpc.NewWorkerNodeRPCClient(id, nil, newWorkerNode(id, common.WorkerNodeStateConnected), mn.cfg.SilenceTimeout)

	if err := mn.RmWorkerNode(context.Background(), w); err != nil {
		t.Error(err)
	}

	if len(mn.workerNodePool) != 0 {
		t.Errorf("Non empty worker pool %+v", mn.workerNodePool)
	}

	w = getWorkerFromDb(mn, id, t)
	if w != nil {
		t.Errorf("Non nil rsp from db %+v", w)
	}
}

func TestGetWorkerList(t *testing.T) {
	mn := prepareMn(t)

	var deletedId string
	ids := map[string]struct{}{}
	for i := 0; i < 3; i++ {
		id := uuid.New().String()

		addWorker(mn, id, t)

		ids[id] = struct{}{}
		deletedId = id

	}

	w := wrpc.NewWorkerNodeRPCClient(deletedId, nil,
		newWorkerNode(deletedId, common.WorkerNodeStateConnected), mn.cfg.SilenceTimeout)
	if err := mn.RmWorkerNode(context.Background(), w); err != nil {
		t.Error(err)
	}

	list, err := mn.GetWorkerList(context.Background(), &pb.DummyReq{})
	if err != nil {
		t.Error(err)
	}

	for _, node := range list.Nodes {
		_, ok := ids[node.Id]
		if !ok {
			t.Errorf("there is no id in the returned list (expected: %+v, returned %+v)",
				ids, list.Nodes)
		}

		if node.Id == deletedId && node.State != common.WorkerNodeStateDisconnected {
			t.Errorf("wrong state for id %s (should be disconnected) (expected: %+v, returned %+v) ",
				node.Id, ids, list.Nodes)
		} else if node.Id != deletedId && node.State != common.WorkerNodeStateConnected {
			t.Errorf("wrong state for id %s (should be connected) (expected: %+v, returned %+v) ",
				node.Id, ids, list.Nodes)
		}

	}
}

func TestGetTaskList(t *testing.T) {
	mn := prepareMn(t)

	tasks := map[string]*pb.Task{}
	for i := 0; i < 3; i++ {

		id := uuid.New().String()

		task := &pb.Task{
			Id:         id,
			Cmd:        "bash",
			Parameters: []string{"-xe", "sh"},
			State:      common.TaskStateScheduled,
		}

		if err := mn.SetToDb(context.Background(), task, EtcdTaskPrefix+task.Id); err != nil {
			t.Error(err)
		}

		tasks[id] = task
	}

	tl, err := mn.GetTaskList(context.Background(), &pb.DummyReq{})
	if err != nil {
		t.Error(err)
	}

	for _, task := range tl.Tasks {
		tsk, ok := tasks[task.Id]
		if !ok {
			t.Errorf("There should not be task with %s in the storage", task.Id)
		}

		if !reflect.DeepEqual(tsk, task) {
			t.Errorf("tasks should be equal %+v != %+v", tsk, task)
		}
	}

}

func TestSetTasksDead(t *testing.T) {
	mn := prepareMn(t)

	taskArr := []*pb.Task{}
	for i := 0; i < 3; i++ {

		task := &pb.Task{
			Cmd:        "bash",
			Parameters: []string{"-xe", "sh"},
			State:      common.TaskStateScheduled,
		}

		taskArr = append(taskArr, task)
	}

	mn.setTasksDead(context.Background(), taskArr)

	tl, err := mn.GetTaskList(context.Background(), &pb.DummyReq{})
	if err != nil {
		t.Error(err)
	}

	for _, task := range tl.Tasks {

		if task.State != common.TaskStateDead {
			t.Errorf("Wrong task state %+v", task)
		}
	}
}

func TestSetToDb(t *testing.T) {
	mn := prepareMn(t)

	type SomeObj struct {
		A int
		B string
	}

	obj := &SomeObj{
		A: 11111,
		B: "something",
	}

	objId := "test_obj_id"
	err := mn.SetToDb(context.Background(), obj, objId)
	if err != nil {
		t.Error(err)
	}

	eCtx, cancel := context.WithTimeout(context.Background(),
		mn.cfg.EtcdDialTimeout)
	gr, err := mn.etcd.Get(eCtx, objId)
	cancel()
	if err != nil {
		t.Error(err)
	}

	rxObj := &SomeObj{}
	err = json.Unmarshal([]byte(gr.Kvs[0].Value), &rxObj)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(rxObj, obj) {
		t.Errorf("Object are not equal %+v %+v", rxObj, obj)
	}

	eCtx, cancel = context.WithTimeout(context.Background(),
		mn.cfg.EtcdDialTimeout)
	_, err = mn.etcd.Delete(eCtx, objId)
	cancel()
	if err != nil {
		t.Error(err)
	}
}

func TestSetTaskAndRunDeadTimeout(t *testing.T) {
	mn := prepareMn(t)

	mn.cfg.DeadTimeout = time.Duration(1) * time.Second

	task := Task{
		cfg: mn.cfg,
		task: &pb.Task{
			Id:    uuid.New().String(),
			State: common.TaskStatePending,
		},
	}

	mn.SetTaskAndRunDeadTimeout(context.Background(), &task)

	time.Sleep(2 * time.Second)

	eCtx, cancel := context.WithTimeout(context.Background(),
		mn.cfg.EtcdDialTimeout)
	gr, err := mn.etcd.Get(eCtx, EtcdTaskPrefix+task.task.Id)
	cancel()
	if err != nil {
		t.Error(err)
	}

	var newTask pb.Task
	err = json.Unmarshal([]byte(gr.Kvs[0].Value), &newTask)
	if err != nil {
		t.Error(err)
	}

	if newTask.State != common.TaskStateDead {
		t.Errorf("There should be dead state for %+v", newTask)
	}

}

func TestStopDeadAndDelTimeout(t *testing.T) {
	mn := prepareMn(t)

	mn.cfg.DeadTimeout = time.Duration(10) * time.Second

	task := NewTask(&pb.Task{
		Id:    uuid.New().String(),
		State: common.TaskStatePending,
	}, mn.cfg)

	mn.SetTaskAndRunDeadTimeout(context.Background(), task)

	err := mn.StopTaskDeadTimeout(context.Background(), task.task.Id)
	if err != nil {
		t.Error(err.Error())
	}

	mn.DelTask(task.task.Id)

	if len(mn.scheduledTasks) != 0 {
		t.Errorf("There should not be any tasks in scheduled tasks %+v",
			mn.scheduledTasks)
	}

}
