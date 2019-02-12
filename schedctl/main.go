package main

import (
	"flag"
	"fmt"

	common "github.com/bgzzz/go-schedule/common"
	log "github.com/sirupsen/logrus"
	"os"
)

const (
	DefaultCfgFilePath = "/etc/go-schedule/schedctl.yaml"
	Usage              = `schedctl - is the utility that schedules running of arbitrary linux jobs
on the cluster of remote linux machines (workers).

usage: 
	schedctl [options] [target] [target cmd] [target cmd parameters] 

options:
	-c - config file path. usage: -c /etc/go-schedule/schedctl.yaml, 
	by default(if no -c provided) utility checks /etc/go-schedule/schedctl.yaml
	-v - add debug verbose level to standard output  

target: 
	workers - remote servers that are used for running tasks
	tasks - tasks that can be scheduled on the cluster of workers
	for more information for each of the target type: schedctl [target] help

example: 
	schedctl workers ls - shows all the remote linux machines that were and 
	are connected to the management node`

	UsageWorkers = `schedctl workers - utility that queries managment nodes 
to get information about workers that are and were connected to the managment nodes

usage: 
	schedctl workers [cmd]

cmd:
	ls - list of all workers that are and were connected to the management node
	help, -h, --help, ? - show the usage of the schedctl workers 

example:
	schedctl workers ls - shows all the remote linux machines that were and 
	are connected to the management node`

	UsageTasks = `schedctl tasks - utility that schedules arbitrary linux task
on the cluster of worker nodes connected to management node

usage: 
	schedctl tasks [cmd] [cmd parameters]

cmd:
	ls - list of all tasks that are and were scheduled on the cluster of worker nodes 
	schedule [schedule parameters] - schedules the tasks on the remote cluster 
	of worker nodes
		schedule parameters:
			--file, -f [tasks_file_path] - provides path to file that describes tasks
			have to be scheduled on the remote cluster of workers
	help, -h, --help, ? - show the usage of the schedctl tasks

example:
	schedctl tasks schedule -f tasks.yaml - schedules tasks described in tasks.yaml
	on the cluster of remote worker nodes`
)

const (
	WorkersCmd = "workers"
	TasksCmd   = "tasks"

	LsCmd       = "ls"
	ScheduleCmd = "schedule"
)

var (
	HelpCmds = []string{"-h", "--help", "?", "help"}

	FileCmds = []string{"-f", "--file"}
)

func main() {

	verbose := flag.Bool("v", false, "-v to add debug information to stdout messages")
	configPath := flag.String("c", DefaultCfgFilePath, "-c /etc/go-schedule/schedctl.yaml provides config file path")

	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		fmt.Println(Usage)
		os.Exit(1)
	}

	err := parseClientCfgFile(*configPath)
	if err != nil {
		if *verbose {
			fmt.Println(err.Error())
		}
		fmt.Println(Usage)
		os.Exit(1)
	}

	logLvl := Config.BasicConfig.LogLvl
	if *verbose {
		logLvl = "debug"
	}

	err = common.InitLogging(logLvl, Config.BasicConfig.LogPath)
	if err != nil {
		if *verbose {
			fmt.Println(err.Error())
		}
		fmt.Println(Usage)
		os.Exit(1)
	}

	err = parseCmd("schedctl", args)
	if err != nil {
		log.Error(err.Error())
		fmt.Println(Usage)
		os.Exit(1)
	}
}
