package domain

//RunStatus is the lifecycle state of a single execution.
type RunStatus string

const (
	RunQueued		RunStatus = "QUEUED"
	RunPreparing	RunStatus = "PREPARING"
	RunRunning		RunStatus = "RUNNING"
	RunSucceeded	RunStatus = "SUCCEEDED"
	RunFailed		RunStatus = "FAILED"
	RunTimedOut		RunStatus = "TIMED_OUT"
	RunCancelled	RunStatus = "CANCELLED"
)
