package common

const (
	// WorkerNodeRPCExec is RPC method that is called by management node
	// on worker node. It executes arbitrary Linux job on
	// the worker node
	WorkerNodeRPCExec = "exec"

	// WorkerNodeRPCPing is RPC method that is called by management node
	// on worker node. It is used for connection check via
	// bi-directional streaming
	WorkerNodeRPCPing = "ping"
)

const (

	// WorkerNodeRPCPingReply is message that is used as
	// reply on ping RPC execution on the worker nodes
	WorkerNodeRPCPingReply = "pong"
)

// Worker node states
const (
	WorkerNodeStateConnected    = "connected"
	WorkerNodeStateDisconnected = "disconnected"
)
