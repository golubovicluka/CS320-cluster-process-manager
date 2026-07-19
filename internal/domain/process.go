package domain

import (
	"fmt"
	"strings"
)

type Process struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	State           ProcessState  `json:"state"`
	Priority        int           `json:"priority"`
	CPURequest      int           `json:"cpuRequest"`
	MemoryRequestMB int           `json:"memoryRequestMB"`
	TotalTicks      int           `json:"totalTicks"`
	RemainingTicks  int           `json:"remainingTicks"`
	TimeQuantum     int           `json:"timeQuantum"`
	NodeID          string        `json:"nodeId,omitempty"`
	RestartPolicy   RestartPolicy `json:"restartPolicy"`
	RestartCount    int           `json:"restartCount"`
	MaxRestarts     int           `json:"maxRestarts"`
	SubmittedAtTick int64         `json:"submittedAtTick"`
	StartedAtTick   *int64        `json:"startedAtTick,omitempty"`
	FinishedAtTick  *int64        `json:"finishedAtTick,omitempty"`
	LastReadyAtTick int64         `json:"lastReadyAtTick"`
	WaitingTicks    int64         `json:"waitingTicks"`
	LastError       string        `json:"lastError,omitempty"`
}

type ProcessDefinition struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Priority        int           `json:"priority"`
	CPURequest      int           `json:"cpuRequest"`
	MemoryRequestMB int           `json:"memoryRequestMB"`
	TotalTicks      int           `json:"totalTicks"`
	TimeQuantum     int           `json:"timeQuantum"`
	RestartPolicy   RestartPolicy `json:"restartPolicy"`
	MaxRestarts     int           `json:"maxRestarts"`
	SubmitAtTick    int64         `json:"submitAtTick,omitempty"`
}

func NewProcess(def ProcessDefinition, tick int64) (*Process, error) {
	if def.RestartPolicy == "" {
		def.RestartPolicy = RestartNever
	}
	if def.TimeQuantum == 0 {
		def.TimeQuantum = 1
	}
	p := &Process{
		ID:              strings.TrimSpace(def.ID),
		Name:            strings.TrimSpace(def.Name),
		State:           ProcessNew,
		Priority:        def.Priority,
		CPURequest:      def.CPURequest,
		MemoryRequestMB: def.MemoryRequestMB,
		TotalTicks:      def.TotalTicks,
		RemainingTicks:  def.TotalTicks,
		TimeQuantum:     def.TimeQuantum,
		RestartPolicy:   def.RestartPolicy,
		MaxRestarts:     def.MaxRestarts,
		SubmittedAtTick: tick,
		LastReadyAtTick: tick,
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p Process) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("%w: process id is required", ErrInvalidInput)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: process name is required", ErrInvalidInput)
	}
	if p.CPURequest <= 0 {
		return fmt.Errorf("%w: cpu request must be positive", ErrInvalidInput)
	}
	if p.MemoryRequestMB <= 0 {
		return fmt.Errorf("%w: memory request must be positive", ErrInvalidInput)
	}
	if p.TotalTicks <= 0 {
		return fmt.Errorf("%w: total ticks must be positive", ErrInvalidInput)
	}
	if p.TimeQuantum <= 0 {
		return fmt.Errorf("%w: time quantum must be positive", ErrInvalidInput)
	}
	if p.MaxRestarts < 0 {
		return fmt.Errorf("%w: max restarts cannot be negative", ErrInvalidInput)
	}
	switch p.RestartPolicy {
	case RestartNever, RestartOnFailure, RestartAlways:
	default:
		return fmt.Errorf("%w: unsupported restart policy %q", ErrInvalidInput, p.RestartPolicy)
	}
	return nil
}

var processTransitions = map[ProcessState]map[ProcessState]bool{
	ProcessNew: {
		ProcessReady: true,
	},
	ProcessReady: {
		ProcessRunning: true,
		ProcessFailed:  true,
		ProcessKilled:  true,
	},
	ProcessRunning: {
		ProcessReady:      true,
		ProcessWaiting:    true,
		ProcessPaused:     true,
		ProcessTerminated: true,
		ProcessFailed:     true,
		ProcessKilled:     true,
	},
	ProcessWaiting: {
		ProcessReady:  true,
		ProcessFailed: true,
		ProcessKilled: true,
	},
	ProcessPaused: {
		ProcessReady:  true,
		ProcessFailed: true,
		ProcessKilled: true,
	},
	ProcessFailed: {
		ProcessReady: true,
	},
}

func (p *Process) Transition(to ProcessState) error {
	if p == nil || !processTransitions[p.State][to] {
		var from ProcessState
		if p != nil {
			from = p.State
		}
		return fmt.Errorf("%w: process %s -> %s", ErrInvalidStateTransition, from, to)
	}
	p.State = to
	return nil
}

func (p Process) Clone() *Process {
	clone := p
	if p.StartedAtTick != nil {
		started := *p.StartedAtTick
		clone.StartedAtTick = &started
	}
	if p.FinishedAtTick != nil {
		finished := *p.FinishedAtTick
		clone.FinishedAtTick = &finished
	}
	return &clone
}

func (p Process) CanRestart() bool {
	return (p.RestartPolicy == RestartOnFailure || p.RestartPolicy == RestartAlways) && p.RestartCount < p.MaxRestarts
}
