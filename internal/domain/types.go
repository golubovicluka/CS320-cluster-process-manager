package domain

type ProcessState string

const (
	ProcessNew        ProcessState = "NEW"
	ProcessReady      ProcessState = "READY"
	ProcessRunning    ProcessState = "RUNNING"
	ProcessWaiting    ProcessState = "WAITING"
	ProcessPaused     ProcessState = "PAUSED"
	ProcessTerminated ProcessState = "TERMINATED"
	ProcessFailed     ProcessState = "FAILED"
	ProcessKilled     ProcessState = "KILLED"
)

func (s ProcessState) IsTerminal() bool {
	return s == ProcessTerminated || s == ProcessFailed || s == ProcessKilled
}

type RestartPolicy string

const (
	RestartNever     RestartPolicy = "NEVER"
	RestartOnFailure RestartPolicy = "ON_FAILURE"
	RestartAlways    RestartPolicy = "ALWAYS"
)

type NodeStatus string

const (
	NodeOnline     NodeStatus = "ONLINE"
	NodeDraining   NodeStatus = "DRAINING"
	NodeOffline    NodeStatus = "OFFLINE"
	NodeFailed     NodeStatus = "FAILED"
	NodeRecovering NodeStatus = "RECOVERING"
)

type SimulationStatus string

const (
	SimulationIdle      SimulationStatus = "IDLE"
	SimulationRunning   SimulationStatus = "RUNNING"
	SimulationPaused    SimulationStatus = "PAUSED"
	SimulationCompleted SimulationStatus = "COMPLETED"
)

type EventType string

const (
	EventNodeRegistered      EventType = "NODE_REGISTERED"
	EventNodeRemoved         EventType = "NODE_REMOVED"
	EventNodeStatusChanged   EventType = "NODE_STATUS_CHANGED"
	EventNodeHeartbeatMissed EventType = "NODE_HEARTBEAT_MISSED"
	EventNodeFailed          EventType = "NODE_FAILED"
	EventNodeRecovered       EventType = "NODE_RECOVERED"
	EventProcessSubmitted    EventType = "PROCESS_SUBMITTED"
	EventProcessScheduled    EventType = "PROCESS_SCHEDULED"
	EventProcessStarted      EventType = "PROCESS_STARTED"
	EventProcessPreempted    EventType = "PROCESS_PREEMPTED"
	EventProcessPaused       EventType = "PROCESS_PAUSED"
	EventProcessResumed      EventType = "PROCESS_RESUMED"
	EventProcessWaiting      EventType = "PROCESS_WAITING"
	EventProcessIOCompleted  EventType = "PROCESS_IO_COMPLETED"
	EventProcessCompleted    EventType = "PROCESS_COMPLETED"
	EventProcessFailed       EventType = "PROCESS_FAILED"
	EventProcessRestarted    EventType = "PROCESS_RESTARTED"
	EventProcessKilled       EventType = "PROCESS_KILLED"
	EventSchedulingDeferred  EventType = "SCHEDULING_DEFERRED"
	EventResourceAllocated   EventType = "RESOURCE_ALLOCATED"
	EventResourceReleased    EventType = "RESOURCE_RELEASED"
	EventSimulationStarted   EventType = "SIMULATION_STARTED"
	EventSimulationPaused    EventType = "SIMULATION_PAUSED"
	EventSimulationReset     EventType = "SIMULATION_RESET"
	EventSimulationFinished  EventType = "SIMULATION_FINISHED"
	EventSchedulerChanged    EventType = "SCHEDULER_CHANGED"
	EventScenarioLoaded      EventType = "SCENARIO_LOADED"
)

type Event struct {
	ID        string    `json:"id"`
	Tick      int64     `json:"tick"`
	Type      EventType `json:"type"`
	Severity  string    `json:"severity"`
	ProcessID string    `json:"processId,omitempty"`
	NodeID    string    `json:"nodeId,omitempty"`
	Message   string    `json:"message"`
}
