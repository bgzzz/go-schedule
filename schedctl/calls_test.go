// +build unit

package main

import (
	"io/ioutil"
	"testing"
	"time"
)

var cfg = &SchedCtlConfig{
	ConnectionTimeout: 5 * time.Second,
}

func TestScheduleTasksErrorArgs(t *testing.T) {
	args := []string{"tasks"}

	if err := ScheduleTasks(args, cfg); err == nil {
		t.Errorf("There should be an error with this args %+v", args)
	}
}

func TestScheduleTasksErrorArgsFilePointer(t *testing.T) {
	args := []string{"tasks", "--file"}

	if err := ScheduleTasks(args, cfg); err == nil {
		t.Errorf("There should be an error with this args %+v", args)
	}
}

func TestScheduleTasksErrorArgsFileNotExist(t *testing.T) {
	args := []string{"--file", "--file"}

	if err := ScheduleTasks(args, cfg); err == nil {
		t.Errorf("There should be an error with this args %+v", args)
	}
}

const testWrongYaml = `# example of possible task list

# tasks array 
tasks:
  # task contains cmd descriptor
- cmd: sleep
  # and array of parameters
  - params: 
      - "30"
  - cmd: sleep
    params: 
      - "30"`

func TestScheduleTasksErrorWrongFileStruct(t *testing.T) {
	//create file

	data := []byte(testWrongYaml)
	err := ioutil.WriteFile("./TestScheduleTasksErrorWrongFileStruct.yaml", data, 0644)
	if err != nil {
		t.Error(err.Error())
		// or we can skip it
	}

	args := []string{"-f", "./TestScheduleTasksErrorWrongFileStruct.yaml"}

	if err := ScheduleTasks(args, cfg); err == nil {
		t.Errorf("There should be an error with this yaml %+v", string(data))
	}
}
