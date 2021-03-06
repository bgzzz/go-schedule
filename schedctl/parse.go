package main

import (
	"fmt"

	"github.com/gravitational/trace"
)

// helpCheck check if cmd is a help detective
func helpCheck(cmd string) bool {
	for _, v := range HelpCmds {
		if v == cmd {
			return true
		}
	}
	return false
}

// fileCheck check if cmd is a file detective
func fileCheck(cmd string) bool {
	for _, v := range FileCmds {
		if v == cmd {
			return true
		}
	}
	return false
}

// parseCmd parses cmd arguments and call specific
// gRPCs
func parseCmd(target string, args []string, cfg *SchedCtlConfig) error {

	if len(args) == 0 {
		return trace.Errorf("There is no parameters of the cmd %s", target)
	}

	if helpCheck(args[0]) {
		switch target {
		case WorkersCmd:
			fmt.Println(UsageWorkers)
		case TasksCmd:
			fmt.Println(UsageTasks)
		default:
			return trace.Errorf("Unsupported target %s", target)
		}
		return nil
	}

	var err error
	switch args[0] {
	case WorkersCmd:
		err = parseCmd(WorkersCmd, args[1:], cfg)
	case TasksCmd:
		err = parseCmd(TasksCmd, args[1:], cfg)
	case LsCmd:
		switch target {
		case WorkersCmd:
			err = ListWorkers(cfg)
		case TasksCmd:
			err = ListTasks(cfg)
		default:
			err = fmt.Errorf("Wrong target for ls (%s)", target)
		}
	case ScheduleCmd:
		if target != TasksCmd {
			err = fmt.Errorf("Wrong target for the cmd (%s)", target)
			break
		}

		err = ScheduleTasks(args[1:], cfg)
	default:
		err = fmt.Errorf("Wrong cmd paramater (%s)", args[0])
	}

	return trace.Wrap(err)
}
