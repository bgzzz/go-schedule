package main

import (
	"context"
	"time"

	pb "github.com/bgzzz/go-schedule/proto"

	log "github.com/sirupsen/logrus"
)

// Task is struct that holds information about scheduled
// jobd/task
type Task struct {
	task *pb.Task

	deadTimer *time.Timer
	// channel for harmfull stop of on deadTimerexpired
	// go routine
	rxed chan struct{}

	cfg *ServerConfig
}

func NewTask(task *pb.Task, cfg *ServerConfig) *Task {
	return &Task{
		task: task,
		cfg:  cfg,
		rxed: make(chan struct{}),
	}
}

// StartDeadTimeout start timeout
// and runs cb when expired
func (t *Task) StartDeadTimeout(ctx context.Context,
	cb func(ctx context.Context)) {
	go func() {
		c, cancel := context.WithTimeout(ctx, t.cfg.DeadTimeout)
		defer cancel()

		select {
		case <-c.Done():
			{
				cb(ctx)
			}
		case <-t.rxed:
			{
				log.Debugf("Dead timeout stopped for task %s", t.task.Id)
			}
		}
	}()
}

// StopDeadTimeout stops dead timeout for task
func (t *Task) StopDeadTimeout(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	select {
	case t.rxed <- struct{}{}:
		{
			log.Debugf("Dead timeout is stopped for task %s",
				t.task.Id)
		}
	case <-c.Done():
		{
			log.Warningf("context declined for task %s", t.task.Id)
		}
	}
}
