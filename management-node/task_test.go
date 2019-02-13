// +build unit

package main

import (
	"testing"
	"time"

	pb "github.com/bgzzz/go-schedule/proto"
)

func TestDeadTimeoutExpiration(t *testing.T) {
	task := Task{
		task: &pb.Task{},
		stop: make(chan struct{}),
	}

	Config = &ServerConfig{
		DeadTimeout: 1,
	}

	waitMsg := make(chan struct{}, 1)
	cb := func() {
		waitMsg <- struct{}{}
	}

	task.StartDeadTimeout(cb)
	timer := time.NewTimer(time.Duration(2) * time.Second)

	select {
	case <-waitMsg:
		{
			if !timer.Stop() {
				<-timer.C
			}
		}
	case <-timer.C:
		{
			t.Error("Dead timer for 1 second is not expired")
		}
	}
}

func TestStopDeadTimeout(t *testing.T) {
	task := Task{
		task: &pb.Task{},
		stop: make(chan struct{}, 1),
	}

	Config = &ServerConfig{
		DeadTimeout: 10,
	}

	waitMsg := make(chan struct{}, 1)
	cb := func() {
		waitMsg <- struct{}{}
	}

	task.StartDeadTimeout(cb)
	timer := time.NewTimer(time.Duration(1) * time.Second)
	task.StopDeadTimeout()

	select {
	case <-waitMsg:
		{
			t.Error("Dead timer of 2 seconds has to be stopped")
		}
	case <-timer.C:
		{
		}
	}
}
