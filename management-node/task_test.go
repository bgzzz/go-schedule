// +build unit

package main

import (
	"context"
	"testing"
	"time"

	pb "github.com/bgzzz/go-schedule/proto"
)

func TestDeadTimeoutExpiration(t *testing.T) {
	task := Task{
		task: &pb.Task{},
		rxed: make(chan struct{}),
		cfg: &ServerConfig{
			DeadTimeout: 1 * time.Second,
		},
	}

	waitMsg := make(chan struct{}, 1)
	cb := func(ctx context.Context) {
		c, cancel := context.WithCancel(ctx)
		defer cancel()

		select {
		case waitMsg <- struct{}{}:
			{
			}
		case <-c.Done():
			{
				t.Log("timeout already stopped")
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	task.StartDeadTimeout(ctx, cb)

	select {
	case <-waitMsg:
		{

		}
	case <-ctx.Done():
		{
			t.Error("Dead timer for 1 second is not expired")
		}
	}
}

func TestStopDeadTimeout(t *testing.T) {
	task := &Task{
		task: &pb.Task{},
		rxed: make(chan struct{}),
		cfg: &ServerConfig{
			DeadTimeout: 100 * time.Second,
		},
	}

	waitMsg := make(chan struct{}, 1)
	cb := func(ctx context.Context) {
		c, cancel := context.WithCancel(ctx)
		defer cancel()

		select {
		case <-c.Done():
			{
				t.Log("stopped timer")
			}
		case waitMsg <- struct{}{}:
			{

			}
		}

	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(4)*time.Second)
	defer cancel()

	task.StartDeadTimeout(ctx, cb)
	task.StopDeadTimeout(ctx)

	select {
	case <-waitMsg:
		{
			t.Error("Dead timer of 10 seconds has to be stopped")
		}
	case <-ctx.Done():
		{
		}
	}
}
