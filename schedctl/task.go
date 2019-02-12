package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"

	pb "github.com/bgzzz/go-schedule/proto"
)

// TaskYaml task description yaml mapper
type TaskYaml struct {
	Cmd    string   `yaml:"cmd"`
	Params []string `yaml:"params"`
}

// TasksYaml tasks description yaml mapper
type TasksYaml struct {
	Tasks []TaskYaml `yaml:"tasks"`
}

// ParseTasksFile reads yaml file and transforms return
// to tasklist
func ParseTasksFile(filePath string) (*pb.TaskList, error) {
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("There is a problem with reading tasks file: %s\n", err.Error())
		log.Error(err.Error())
		return nil, err
	}

	var tasks TasksYaml
	err = yaml.Unmarshal(dat, &tasks)
	if err != nil {
		fmt.Println("There is a problem with yaml structure")
		log.Error(err.Error())
		return nil, err
	}

	taskList := &pb.TaskList{}
	for _, v := range tasks.Tasks {
		taskList.Tasks = append(taskList.Tasks, &pb.Task{
			Id:         "dummy",
			Cmd:        v.Cmd,
			Parameters: v.Params,
		})
	}

	return taskList, nil
}
