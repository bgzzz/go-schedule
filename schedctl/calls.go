package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/bgzzz/go-schedule/proto"
)

// runRPC is wrapper for gRPC connection procedures
// it connects to management node and runs specific call
func runRPC(call func(client pb.SchedulerClient) error) error {

	conn, err := grpc.Dial(Config.ServerAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("There is a problem with connection: %s\n", err.Error())
		log.Error("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	client := pb.NewSchedulerClient(conn)

	err = call(client)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return nil
}

// ScheduleTasks receives command line arguments
// for tasks schedule detective parses it in case of wrong
// usage and calls the schedule gRPC and prints the result
// to stdout
func ScheduleTasks(args []string) error {

	if len(args) < 2 {
		errStr := "There is not enought arguments for tasks schedule cmd"
		fmt.Println(errStr)
		err := fmt.Errorf(errStr)
		log.Error(err.Error())
		return err
	}

	if !fileCheck(args[0]) {
		err := fmt.Errorf("There is no pointer to tasks file (-f, --file)")
		log.Error(err.Error())
		return err
	}

	tl, err := ParseTasksFile(args[1])
	if err != nil {
		log.Error(err.Error())
		return err
	}

	return runRPC(func(client pb.SchedulerClient) error {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(Config.ConnectionTimeout)*time.Second)

		log.Debug("Calling Schedule RPC")
		tl, err := client.Schedule(ctx, tl)
		cancel()
		if err != nil {
			fmt.Printf("Connection problems: %s\n", err.Error())
			log.Error(err.Error())
			return err
		}

		printTasks(tl.Tasks)
		return nil
	})
}

// ListTasks calls the GetTaskList and
// prints the result of received list of tasks that
// were registered to stdout
func ListTasks() error {
	return runRPC(func(client pb.SchedulerClient) error {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(Config.ConnectionTimeout)*time.Second)

		log.Debug("Calling ListTasks RPC")
		tl, err := client.GetTaskList(ctx, &pb.DummyReq{
			Id: "dummy",
		})

		cancel()

		if err != nil {
			log.Error(err.Error())
			fmt.Printf("Connection problems: %s\n", err.Error())
			return err
		}

		printTasks(tl.Tasks)

		return nil
	})
}

// ListWorkers calls the GetWorkerList gRPC and prints the
// result of workers that were registered on management node
// to stdout
func ListWorkers() error {
	return runRPC(func(client pb.SchedulerClient) error {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(Config.ConnectionTimeout)*time.Second)

		log.Debug("Calling ListWorkers RPC")
		wl, err := client.GetWorkerList(ctx, &pb.DummyReq{
			Id: "dummy",
		})
		cancel()

		if err != nil {
			log.Error(err.Error())
			fmt.Printf("Connection problems: %s\n", err.Error())
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
		fmt.Fprintln(w, WorkersHeader)
		for _, v := range wl.Nodes {
			fmt.Fprintf(w, WorkersLineTemplate, v.Id, v.Address, v.State)
		}
		w.Flush()

		return nil
	})
}
