package main

/*
import (
	"fmt"
	"sync"
	"time"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WorkerNode the structure that holds information about
// connected worker node
type WorkerNode struct {
	WN *pb.WorkerNode `json:"wn"`

	stream pb.Scheduler_WorkerConnectServer

	// map of callbacks waiting for execution
	// on response from worker
	subscription    map[string]Callback
	subscriptionMtx sync.RWMutex

	// channels needed for interaction with
	// worker node
	stopSender chan struct{}
	send       chan *pb.MgmtReq

	// err holds the error value
	// during management to worker interaction
	err      error
	errorMtx sync.RWMutex

	// pingTicker is ticker that is used for
	// pinging the worker node
	pingTicker *time.Ticker

	// client to work with etcd db
	etcd *clientv3.Client
}

type Callback func(rsp *pb.WorkerRsp)

// StartSender setups eventloop for sync sending to
// stream
func (wn *WorkerNode) StartSender() {
	go func() {
		for {
			select {
			case <-wn.stopSender:
				{
					log.Debugf("sender is stopped for worker node %s", wn.WN.Id)
					return
				}
			case req := <-wn.send:
				{
					log.Debugf("tx: %s %s", req.Method, req.Id)

					err := wn.SendWithTimeout(req, time.Duration(Config.SilenceTimeout))
					if err != nil {
						log.Error(err.Error())
						wn.SetErr(err)
					}
				}
			}
		}
	}()
}

// StopSender stops sender infinite loop
func (wn *WorkerNode) StopSender() {
	wn.stopSender <- struct{}{}
	close(wn.stopSender)
}

// CheckErr thread safely checks the worker node
// error
func (wn *WorkerNode) CheckErr() error {
	wn.errorMtx.RLock()
	defer wn.errorMtx.RUnlock()

	err := wn.err
	return err
}

// SetErr thread safely sets the error
// to wn
func (wn *WorkerNode) SetErr(err error) {
	wn.errorMtx.Lock()
	defer wn.errorMtx.Unlock()

	if wn.err != nil {
		wn.err = fmt.Errorf("%s %s", wn.err, err)
	}
}

// SendWithHandlerTimeout sends request to sender
// and register callbacks on response and timer expiration
func (wn *WorkerNode) SendWithHandlerTimeout(req pb.MgmtReq,
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
func (wn *WorkerNode) Send(req pb.MgmtReq, cb Callback) {

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

	t := time.NewTimer(time.Duration(Config.SilenceTimeout) * time.Second)

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

// StartPinger setups ticker and sends ping to worker on
// every tick
func (wn *WorkerNode) StartPinger() {
	wn.pingTicker = time.NewTicker(time.Duration(Config.PingTimeout) * time.Second)

	log.Debug("Start pinger")
	go func() {
		for _ = range wn.pingTicker.C {
			req := pb.MgmtReq{
				Method: common.WorkerNodeRPCPing,
			}

			onRspHandler := func(rsp *pb.WorkerRsp) {

				if rsp.Reply != common.WorkerNodeRPCPingReply {
					err := fmt.Errorf("Wrong answer on ping with id %s", rsp.Id)
					log.Error(err.Error())
					wn.SetErr(err)
					return
				} else {
					log.Debugf("rx: pong %s", rsp.Id)
				}
				return
			}

			onTimerExpiredHandler := func() {
				err := fmt.Errorf("pong is delayed for %s: closing connection for %s", req.Id, wn.WN.Id)
				log.Error(err.Error())
				wn.SetErr(err)
			}

			wn.SendWithHandlerTimeout(req,
				onRspHandler, onTimerExpiredHandler, time.Duration(Config.SilenceTimeout))
		}
	}()
}

// StopPinger stops pinger ticker
func (wn *WorkerNode) StopPinger() {
	log.Debug("Stop pinger")
	wn.pingTicker.Stop()
}

// ProcessResponse execute the callback and delete it from
// subscription storage
func (wn *WorkerNode) ProcessResponse(rsp *pb.WorkerRsp) error {

	wn.subscriptionMtx.Lock()
	defer wn.subscriptionMtx.Unlock()

	call, ok := wn.subscription[rsp.Id]
	if !ok {
		err := fmt.Errorf("There is no handler for responce %s", rsp.Id)
		log.Error(err)
		return err
	}

	go call(rsp)

	delete(wn.subscription, rsp.Id)

	return nil
}

// RxMsg is helper struct for rx with timeout function
type RxMsg struct {
	err error
	rsp *pb.WorkerRsp
}

// RxWithTimeout prevents hanging on reception from stream
// return RxMsg with error and rsp
// TBD: make return with tow variables for the style and
// 		to get read off RxMsg
func (wn *WorkerNode) RxWithTimeout(d time.Duration) *RxMsg {
	msgChan := make(chan RxMsg, 1)
	go func() {
		rsp, err := wn.stream.Recv()
		msgChan <- RxMsg{
			err: err,
			rsp: rsp,
		}
		close(msgChan)
	}()
	t := time.NewTimer(d * time.Second)
	select {
	case <-t.C:
		return &RxMsg{
			err: status.Errorf(codes.DeadlineExceeded, "timer for rx of "+string(d)+" expired"),
			rsp: nil,
		}
	case msg := <-msgChan:
		if !t.Stop() {
			<-t.C
		}
		return &msg
	}
}

// SendWithTimeout prevents hanging on the sending to stream
// return error if timer expired
func (wn *WorkerNode) SendWithTimeout(req *pb.MgmtReq, d time.Duration) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- wn.stream.Send(req)
		close(errChan)
	}()
	t := time.NewTimer(d * time.Second)
	select {
	case <-t.C:
		return status.Errorf(codes.DeadlineExceeded, "timer for tx of "+string(d)+" expired")
	case err := <-errChan:
		if !t.Stop() {
			<-t.C
		}
		return err
	}

}
*/
