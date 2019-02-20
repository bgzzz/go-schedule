package common

const (
	// TaskStatePending is the state of the task when the
	// Task is registered in db but has not sent
	// to worker node
	TaskStatePending = "pending"

	// TaskStateScheduled is the state of the task when the
	// Task is sent to worker node and
	// worker node started the execution
	TaskStateScheduled = "scheduled"

	// TaskStateDone is the state of the task when the
	// Task is executed and done on
	// the worker node. Information about
	// status change is registered in db
	TaskStateDone = "done"

	// TaskStateDead is the state of the task when the
	// Task is registered is dead, meaning that
	// it will never be executed.
	// It happened when the are no active
	// workers connected to management or worker
	// that has been used for execution is not responding
	// within certain period
	TaskStateDead = "dead"
)
