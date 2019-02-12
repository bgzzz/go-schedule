package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	pb "github.com/bgzzz/go-schedule/proto"
)

// output templates
const (
	TasksHeader       = "id\tstate\tworker\tstdout\tstderr"
	TasksLineTemplate = "%s\t%s\t%s\t%s\t%s\n"

	DefaultRightBound = 20
)

const (
	WorkersHeader       = "id\taddress\tstate"
	WorkersLineTemplate = "%s\t%s\t%s\n"
)

// getRightBound checks the default bound for long
// string printing
func getRightBound(bound int) int {
	if bound > DefaultRightBound {
		return DefaultRightBound
	}

	return bound
}

// printTasks prints tasks array to stdout
func printTasks(tasks []*pb.Task) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, TasksHeader)
	for _, v := range tasks {
		fmt.Fprintf(w, TasksLineTemplate, v.Id, v.State, v.WorkerId, v.Stdout[:getRightBound(len(v.Stdout))], v.Stderr[:getRightBound(len(v.Stderr))])
	}
	w.Flush()
}
