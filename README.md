# go-schedule
Go-schedule is the toolkit to schedule processes on the distributed infrastructure.

## Architecture 
Architecture consists of three main parts:

 - Management node 
 - Worker node
 - Schedctl 

Each of the parts is a Linux binary. 

In a nut shell 
schedctl is used for scheduling task/processes/jobs on distributed cluster of worker nodes via communication with management node.

The communication protocol for interaction between parts is gRPC. More on gRPC can be found [here](https://grpc.io/).

Etcd is used as the management-node storage solution. More on etcd can be found [here](https://github.com/etcd-io/etcd)


### Management node 

Management node is daemon that implements gRPC server functionality. It process requests from schedctl and worker nodes. 

It holds the state of connected worker nodes and tasks that have been distributed on the worker nodes.

All change state events triggers the storing of the 
specified information to the etcd storage.

### Worker node 

Worker node is daemon that implements function of gRPC client. In addition to that it executes tasks that were provided by management node via Worker node RPC.

Worker nodes can be placed behind NAT due to the fact that it initiate connection to management node.
Worker node provide RPC API via bi-directional gRPC stream. 


### Schedctl 

Schedctl is the command line utility that is used for interacting between operator and worker nodes cluster. 

### Call flow 

Here I wanted to highlight a common call flow of the 
go-schedule parts interaction.

1. Management node is setup. Listening on the address, connected to etcd.
1. Worker node setups and connects to management node via bi-directional stream connect method.
1. Once connection established management node starts periodically send ping message to worker node for connection state purpose.
1. Operator uses schedctl to schedule task on the workers. Schedule task call the rpc on management node.
1. Management node stores task with pending state and starts timeout at the same time choosing the worker to schedule and sending the task description to the worker. If timeout expires and there were no response from the worker management node stores task with dead state.
1. On positive reception from worker management node changes the state of task to scheduled and stores it in etcd.
1. Once task is done worker calls set task RPC on management node. And management node stores task with done state.

### Task state 
Here is the list of states that are used for defining state of the scheduled task.

1. pending - task is received by the management node. management node is waiting for response from worker.
1. scheduled - task is executed on the worker node 
1. done - task is done on the worker 
1. dead - task will never be executed 

### Worker state 
Here is the list of states that are used for defining 
state of the worker node.

1. connected - worker is connected to management node.
1. disconnected - worker is not connected to management node.


## How to 

### How to build solution locally

```
$ make build
```

binaries are placed to <go-schedule-part>/bin/

### How to run management node 


```
$ management-node/bin/management-node -f management-node/management-node.yaml
``` 

### How to run worker node 


```
$ worker-node/bin/worker-node -f worker-node/worker-node.yaml
``` 


### How to run schedctl 


```
$ schedctl/bin/schedctl -c schedctl/schedctl.yaml workers ls 
``` 

output example 

```
id                                   |address |state
c1c63fcc-402a-4896-bd6c-144b5544f191 |TBD     |disconnected
d76342d1-44e5-460a-9c42-28cf5d3ffed6 |TBD     |connected

```

More information can be found by typing

```
$ schedctl/bin/schedctl -c schedctl/schedctl.yaml 
```

output example

```
schedctl - is the utility that schedules running of arbitrary linux jobs
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
	are connected to the management node

```


## How to run unit tests for all of the components 

```
make test-unit
```

## In case you do not have infrastructure for GoLang

### How to build docker 

```
make docker-build
```

## How to run unit tests in docker 

```
make docker-run-unit
``` 

## TODOs:
Plan on how to develop further.

### Feature and architecture 

1. More unit tests
1. Worker reconnect
1. Security and mutual authentication of all parts
1. HA mode. Multiple etcd cluster. Multiple management node cluster 
1. More wise proto
1. End to end tests
1. Testbed in docker compose
1. Make schedctl better in terms of API
1. More complex scheduling algorithms
1. go routines leaking checks in unit tests
1. worker id and capability definition   

### Refactoring and style 

1. Few functions to truncate
1. Documentation on packages and missing functions
1. Refactoring of Makefile
1. More detailed documentation


## Requirements

### For solution to work

 - etcd

 ```
 / # etcd --version
etcd Version: 3.3.11
Git SHA: 2cf9e51d2
Go Version: go1.10.7
Go OS/Arch: linux/amd64
 ``` 
 or higher 

 - etcd client api v3

### For solution to compile 

 - protoc 

 ```
 protoc --version
libprotoc 3.6.1
 ```
 or higher 

 - protoc-gen-go 1.2.0 or higher   