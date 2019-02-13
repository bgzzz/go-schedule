package main

import (
	"fmt"
	"time"

	pb "github.com/bgzzz/go-schedule/proto"

	log "github.com/sirupsen/logrus"
)

// Task is struct that holds information about scheduled
// jobd/task
type Task struct {
	task *pb.Task

	deadTimer *time.Timer
	// channel for harnfull stop of on deadTimerexpired
	// go routine
	stop chan struct{}
}

//TBD: Add new task function

// StartDeadTimeout start timeout
// and runs cb when expired
func (t *Task) StartDeadTimeout(cb func()) {
	fmt.Println("START DEAD")
	t.deadTimer = time.NewTimer(time.Duration(Config.DeadTimeout) * time.Second)
	go func() {
		select {
		case <-t.deadTimer.C:
			{
				cb()
			}
		case <-t.stop:
			{
				if !t.deadTimer.Stop() {
					<-t.deadTimer.C
				}
				log.Debugf("Dead timeout stopped for task %s", t.task.Id)
			}
		}
	}()
}

// StopDeadTimeout stops dead timeout for task
func (t *Task) StopDeadTimeout() {
	select {
	case t.stop <- struct{}{}:
		{
			log.Debugf("Dead timeout is stopped for task %s",
				t.task.Id)
		}
	default:
		{
			log.Warningf("Blocked dead timeout stop for task %s", t.task.Id)
		}
	}
}
