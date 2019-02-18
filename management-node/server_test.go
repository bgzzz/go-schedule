// +build func

package main

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
	"github.com/bgzzz/go-schedule/wrpc"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
)

const TestCfg = "./test/cfg"

func prepareMn(t *testing.T) *ManagementNode {
	cfg, err := parseCfgFile(TestCfg)
	if err != nil {
		t.Error(err.Error())
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.EtcdAddress},
		DialTimeout: cfg.EtcdDialTimeout,
	})

	return NewManagmentNode(cli, cfg)
}

func TestAddWorkerNode(t *testing.T) {
	MN := prepareMn(t)

	id := uuid.New().String()
	wn := &pb.WorkerNode{
		Id:      id,
		Address: "TBD",
		State:   common.WorkerNodeStateDisconnected,
	}

	setObj := wrpc.NewWorkerNodeRPCClient(id, nil, wn, MN.cfg.SilenceTimeout)

	wnExp := &pb.WorkerNode{
		Id:      id,
		Address: "TBD",
		State:   common.WorkerNodeStateDisconnected,
	}
	expectedObj := wrpc.NewWorkerNodeRPCClient(id, nil, wnExp, MN.cfg.SilenceTimeout)

	eCtx, cancel := context.WithTimeout(context.Background(),
		MN.cfg.EtcdDialTimeout)
	gr, err := MN.etcd.Get(eCtx, EtcdWorkerPrefix+setObj.WN.Id, nil)
	cancel()
	if err != nil {
		t.Error(err.Error())
	}

	if len(gr.Kvs) == 0 {
		t.Error("Empty rsp from etcd")
	}

	var w wrpc.WorkerRPCClient
	err = json.Unmarshal([]byte(gr.Kvs[0].Value), &w)
	if err != nil {
		t.Error(err.Error())
	}

	if err := MN.AddWorkerNode(context.Background(), setObj); err != nil {
		t.Error(err.Error())
	}

	if !reflect.DeepEqual(w, *expectedObj) {
		t.Errorf("should be equal (%+v != %+v)", w, *expectedObj)
	}
}
