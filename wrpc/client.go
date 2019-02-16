package wrpc

import (
	"sync"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"

	"github.com/google/uuid"
	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

type Callback func(rsp *pb.WorkerRsp)

// WorkerRPCClient the structure that holds information about
// connected worker node
type WorkerRPCClient struct {
	*Worker
	WN *pb.WorkerNode `json:"wn"`

	// map of callbacks waiting for execution
	// on response from worker
	subscription    map[string]Callback
	subscriptionMtx sync.RWMutex

	// pingTicker is ticker that is used for
	// pinging the worker node
	pingTicker *time.Ticker
}

func NewWorkerNodeRPCClient(id string,
	stream pb.Scheduler_WorkerConnectServer,
	wn *pb.WorkerNode,
	silenceTimeout int) *WorkerRPCClient {
	return &WorkerRPCClient{
		WN: wn,
		Worker: &Worker{
			id:             id,
			streamServer:   stream,
			streamClient:   nil,
			send:           make(chan interface{}, 1),
			stopSender:     make(chan struct{}, 1),
			silenceTimeout: time.Duration(silenceTimeout),
		},
		subscription: make(map[string]Callback),
	}
}

// SendWithHandlerTimeout sends request to sender
// and register callbacks on response and timer expiration
func (wn *WorkerRPCClient) SendWithHandlerTimeout(req pb.MgmtReq,
	onRspHandler Callback,
	onTimerExpiredHandler func(),
	timoute time.Duration) {

	go func() {
		t := time.NewTimer(timoute * time.Second)
		// channel for connection between callback
		// and on rsp go routine
		rxChannel := make(chan struct{}, 1)

		cb := func(rsp *pb.WorkerRsp) {
			rxChannel <- struct{}{}

			onRspHandler(rsp)
		}

		// sending request to sender
		wn.Send(req, cb)

		select {
		case <-rxChannel:
			{
				if !t.Stop() {
					<-t.C
				}
			}
		case <-t.C:
			{
				// remove subscription cb if timer expired
				wn.subscriptionMtx.Lock()
				delete(wn.subscription, req.Id)
				wn.subscriptionMtx.Unlock()

				onTimerExpiredHandler()
			}
		}
	}()
}

// Send send request to sender and registers cb in the
// subscription map
func (wn *WorkerRPCClient) Send(req pb.MgmtReq, cb Callback) {

	//check if it is already set
	// if request Id set by uuid string
	// for now uuid equals to any string with len 35
	// we do not create req id
	if len(req.Id) < 35 {
		id := uuid.New().String()
		req.Id = id
	}

	wn.subscriptionMtx.Lock()
	wn.subscription[req.Id] = cb
	wn.subscriptionMtx.Unlock()

	t := time.NewTimer(wn.silenceTimeout * time.Second)

	select {
	case wn.send <- &req:
		{
			if !t.Stop() {
				<-t.C
			}
		}
		// this one is done to prevent go routines leaking
		// can be done with default
	case <-t.C:
		{
			log.Warningf("Timeouted to send req %s", req.Id)
		}
	}
}

// ProcessResponse execute the callback and delete it from
// subscription storage
func (wn *WorkerRPCClient) ProcessResponse(r interface{}) error {

	rsp := r.(*pb.WorkerRsp)
	wn.subscriptionMtx.Lock()
	defer wn.subscriptionMtx.Unlock()

	call, ok := wn.subscription[rsp.Id]
	if !ok {
		return trace.Errorf("There is no handler for responce %s", rsp.Id)
	}

	go call(rsp)

	delete(wn.subscription, rsp.Id)

	return nil
}

func (wn *WorkerRPCClient) InitLoop() error {
	return wn.initLoop(wn.ProcessResponse)
}

func (wn *WorkerRPCClient) SetId(id string) {
	wn.WN.Id = id
}

// StartPinger setups ticker and sends ping to worker on
// every tick
func (wn *WorkerRPCClient) StartPinger(d time.Duration) {
	wn.pingTicker = time.NewTicker(d * time.Second)

	log.Debug("Start pinger")
	go func() {
		for _ = range wn.pingTicker.C {
			req := pb.MgmtReq{
				Method: common.WorkerNodeRPCPing,
			}

			onRspHandler := func(rsp *pb.WorkerRsp) {

				if rsp.Reply != common.WorkerNodeRPCPingReply {
					err := trace.Errorf("Wrong answer on ping with id %s", rsp.Id)
					wn.SetErr(err)
					return
				} else {
					log.Debugf("rx: pong %s", rsp.Id)
				}
				return
			}

			onTimerExpiredHandler := func() {
				err := trace.Errorf("pong is delayed for %s: closing connection for %s", req.Id, wn.WN.Id)
				wn.SetErr(err)
			}

			wn.SendWithHandlerTimeout(req,
				onRspHandler, onTimerExpiredHandler, wn.silenceTimeout)
		}
	}()
}

// StopPinger stops pinger ticker
func (wn *WorkerRPCClient) StopPinger() {
	log.Debug("Stop pinger")
	wn.pingTicker.Stop()
}
