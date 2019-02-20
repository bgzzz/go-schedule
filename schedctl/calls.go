package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gravitational/trace"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/bgzzz/go-schedule/common"
	pb "github.com/bgzzz/go-schedule/proto"
)

// runRPC is wrapper for gRPC connection procedures
// it connects to management node and runs specific call
func runRPC(call func(client pb.SchedulerClient) error, cfg *SchedCtlConfig) error {

	conn, err := grpc.Dial(cfg.ServerAddress, grpc.WithInsecure())
	if err != nil {
		common.PrintDebugErr(fmt.Errorf("There is a problem with connection: %s\n", err.Error()))
		return err
	}
	defer conn.Close()
	client := pb.NewSchedulerClient(conn)

	err = call(client)
	if err != nil {
		common.PrintDebugErr(err)
		return err
	}

	return nil
}

// ScheduleTasks receives command line arguments
// for tasks schedule detective parses it in case of wrong
// usage and calls the schedule gRPC and prints the result
// to stdout
func ScheduleTasks(args []string, cfg *SchedCtlConfig) error {

	if len(args) < 2 {
		errStr := "There is not enought arguments for tasks schedule cmd"
		fmt.Println(errStr)
		err := fmt.Errorf(errStr)
		common.PrintDebugErr(err)
		return err
	}

	if !fileCheck(args[0]) {
		err := fmt.Errorf("There is no pointer to tasks file (-f, --file)")
		common.PrintDebugErr(err)
		return err
	}

	tl, err := ParseTasksFile(args[1])
	if err != nil {
		common.PrintDebugErr(err)
		return err
	}

	return runRPC(func(client pb.SchedulerClient) error {
		ctx, cancel := context.WithTimeout(context.Background(),
			cfg.ConnectionTimeout)

		log.Debug("Calling Schedule RPC")
		tl, err := client.Schedule(ctx, tl)
		cancel()
		if err != nil {
			return trace.Errorf("Connection problems: %s\n", err.Error())
		}

		printTasks(tl.Tasks)
		return nil
	}, cfg)
}

// ListTasks calls the GetTaskList and
// prints the result of received list of tasks that
// were registered to stdout
func ListTasks(cfg *SchedCtlConfig) error {
	return runRPC(func(client pb.SchedulerClient) error {
		ctx, cancel := context.WithTimeout(context.Background(),
			cfg.ConnectionTimeout)

		log.Debug("Calling ListTasks RPC")
		tl, err := client.GetTaskList(ctx, &pb.DummyReq{
			Id: "dummy",
		})

		cancel()

		if err != nil {
			return trace.Errorf("Connection problems: %s\n", err.Error())
		}

		printTasks(tl.Tasks)

		return nil
	}, cfg)
}

// ListWorkers calls the GetWorkerList gRPC and prints the
// result of workers that were registered on management node
// to stdout
func ListWorkers(cfg *SchedCtlConfig) error {
	return runRPC(func(client pb.SchedulerClient) error {
		ctx, cancel := context.WithTimeout(context.Background(),
			cfg.ConnectionTimeout)

		log.Debug("Calling ListWorkers RPC")
		wl, err := client.GetWorkerList(ctx, &pb.DummyReq{
			Id: "dummy",
		})
		cancel()

		if err != nil {
			return trace.Errorf("Connection problems: %s\n", err.Error())
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
		fmt.Fprintln(w, WorkersHeader)
		for _, v := range wl.Nodes {
			fmt.Fprintf(w, WorkersLineTemplate, v.Id, v.Address, v.State)
		}
		w.Flush()

		return nil
	}, cfg)
}
