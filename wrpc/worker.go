package wrpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/bgzzz/go-schedule/proto"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
)

type Worker struct {
	id           string
	send         chan interface{}
	streamServer pb.Scheduler_WorkerConnectServer
	streamClient pb.Scheduler_WorkerConnectClient

	err      error
	errorMtx sync.RWMutex

	silenceTimeout time.Duration
}

func (w *Worker) StartSender(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				{
					log.Debugf("sender is stopped for worker node ")
					return
				}
			case msg := <-w.send:
				{
					log.Debugf("tx: %+v", msg)
					err := w.SendWithTimeout(ctx, msg)
					if err != nil {
						w.SetErr(trace.Wrap(err))
					}
				}
			}
		}
	}()
}

func (w *Worker) StopSender(cancel context.CancelFunc) {
	cancel()
}

func (w *Worker) CheckErr() error {
	w.errorMtx.RLock()
	defer w.errorMtx.RUnlock()

	err := w.err
	return err
}

func (w *Worker) SetErr(err error) {
	w.errorMtx.Lock()
	defer w.errorMtx.Unlock()

	if w.err != nil {
		w.err = fmt.Errorf("%s %s", w.err, err)
	}

	err = trace.Wrap(err)
}

// RxMsg is helper struct
type rxMsg struct {
	err error
	msg interface{}
}

func (w *Worker) RxWithTimeout(ctx context.Context) (interface{}, error) {
	msgChan := make(chan rxMsg, 1)
	go func() {
		var msg interface{}
		var err error
		if w.streamServer != nil {
			msg, err = w.streamServer.Recv()
		} else {
			msg, err = w.streamClient.Recv()
		}
		msgChan <- rxMsg{
			err: err,
			msg: msg,
		}
		close(msgChan)
	}()

	c, cancel := context.WithTimeout(ctx, w.silenceTimeout)
	defer cancel()

	select {
	case <-c.Done():
		return nil, trace.Errorf("timer for rx of %d expired", w.silenceTimeout)
	case msg := <-msgChan:
		return msg.msg, trace.Wrap(msg.err)
	}
}

func (w *Worker) SendWithTimeout(ctx context.Context, msg interface{}) error {
	errChan := make(chan error, 1)
	go func() {
		if w.streamServer != nil {
			errChan <- w.streamServer.Send(msg.(*pb.MgmtReq))
		} else {
			errChan <- w.streamClient.Send(msg.(*pb.WorkerRsp))
		}
		close(errChan)
	}()

	c, cancel := context.WithTimeout(ctx, w.silenceTimeout)
	defer cancel()

	select {
	case <-c.Done():
		return trace.Errorf("timer for tx of " + string(w.silenceTimeout) + " expired")
	case err := <-errChan:
		return trace.Wrap(err)
	}

}

func (w *Worker) initLoop(ctx context.Context,
	processCB func(ctx context.Context, msg interface{}) error) error {
	for {

		msg, err := w.RxWithTimeout(ctx)
		if err != nil {
			return trace.Wrap(err)
		}

		if err := processCB(ctx, msg); err != nil {
			return trace.Wrap(err)
		}

		if err := w.CheckErr(); err != nil {
			return trace.Wrap(err)
		}
	}
}
