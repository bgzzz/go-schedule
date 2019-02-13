package main

import (
	"fmt"
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
func parseCmd(target string, args []string) error {

	if len(args) == 0 {
		err := fmt.Errorf("There is no parameters of the cmd %s", target)
		return err
	}

	if helpCheck(args[0]) {
		switch target {
		case WorkersCmd:
			fmt.Println(UsageWorkers)
		case TasksCmd:
			fmt.Println(UsageTasks)
		default:
			return fmt.Errorf("Unsupported target %s", target)
		}
		return nil
	}

	var err error
	switch args[0] {
	case WorkersCmd:
		err = parseCmd(WorkersCmd, args[1:])
	case TasksCmd:
		err = parseCmd(TasksCmd, args[1:])
	case LsCmd:
		switch target {
		case WorkersCmd:
			err = ListWorkers()
		case TasksCmd:
			err = ListTasks()
		default:
			err = fmt.Errorf("Wrong target for ls (%s)", target)
		}
	case ScheduleCmd:
		if target != TasksCmd {
			err = fmt.Errorf("Wrong target for the cmd (%s)", target)
			break
		}

		err = ScheduleTasks(args[1:])
	default:
		err = fmt.Errorf("Wrong cmd paramater (%s)", args[0])
	}

	return err
}
