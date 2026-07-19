package domain

import (
	"fmt"
	"strings"
)

type FailureType string

const (
	FailureNode    FailureType = "NODE"
	FailureProcess FailureType = "PROCESS"
)

type FailureDefinition struct {
	Tick      int64       `json:"tick"`
	Type      FailureType `json:"type"`
	NodeID    string      `json:"nodeId,omitempty"`
	ProcessID string      `json:"processId,omitempty"`
	Message   string      `json:"message,omitempty"`
}

type Scenario struct {
	Name           string              `json:"name"`
	Seed           int64               `json:"seed"`
	Scheduler      string              `json:"scheduler"`
	TickDurationMS int                 `json:"tickDurationMs"`
	MaxTicks       int                 `json:"maxTicks"`
	Nodes          []NodeDefinition    `json:"nodes"`
	Processes      []ProcessDefinition `json:"processes"`
	Failures       []FailureDefinition `json:"failures,omitempty"`
}

func (s Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("%w: scenario name is required", ErrInvalidInput)
	}
	if strings.TrimSpace(s.Scheduler) == "" {
		return fmt.Errorf("%w: scenario scheduler is required", ErrInvalidInput)
	}
	if s.MaxTicks <= 0 {
		return fmt.Errorf("%w: max ticks must be positive", ErrInvalidInput)
	}
	if len(s.Nodes) == 0 {
		return fmt.Errorf("%w: scenario requires at least one node", ErrInvalidInput)
	}
	nodeIDs := make(map[string]bool, len(s.Nodes))
	for _, definition := range s.Nodes {
		node, err := NewNode(definition, 0)
		if err != nil {
			return err
		}
		if nodeIDs[node.ID] {
			return fmt.Errorf("%w: %s", ErrDuplicateNode, node.ID)
		}
		nodeIDs[node.ID] = true
	}
	processes := make(map[string]ProcessDefinition, len(s.Processes))
	for _, definition := range s.Processes {
		process, err := NewProcess(definition, max(definition.SubmitAtTick, 0))
		if err != nil {
			return err
		}
		if _, exists := processes[process.ID]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateProcess, process.ID)
		}
		if definition.SubmitAtTick > int64(s.MaxTicks) {
			return fmt.Errorf("%w: process %s is submitted after maxTicks", ErrInvalidInput, process.ID)
		}
		processes[process.ID] = definition
	}
	failureKeys := make(map[string]bool, len(s.Failures))
	for _, failure := range s.Failures {
		if failure.Tick <= 0 {
			return fmt.Errorf("%w: failure tick must be positive", ErrInvalidInput)
		}
		if failure.Tick > int64(s.MaxTicks) {
			return fmt.Errorf("%w: failure tick cannot exceed maxTicks", ErrInvalidInput)
		}
		key := fmt.Sprintf("%s:%s:%s:%d", failure.Type, failure.NodeID, failure.ProcessID, failure.Tick)
		if failureKeys[key] {
			return fmt.Errorf("%w: duplicate scheduled failure at tick %d", ErrInvalidInput, failure.Tick)
		}
		failureKeys[key] = true
		switch failure.Type {
		case FailureNode:
			if failure.NodeID == "" {
				return fmt.Errorf("%w: node failure requires nodeId", ErrInvalidInput)
			}
			if !nodeIDs[failure.NodeID] {
				return fmt.Errorf("%w: scheduled failure references node %s", ErrNodeNotFound, failure.NodeID)
			}
		case FailureProcess:
			if failure.ProcessID == "" {
				return fmt.Errorf("%w: process failure requires processId", ErrInvalidInput)
			}
			definition, exists := processes[failure.ProcessID]
			if !exists {
				return fmt.Errorf("%w: scheduled failure references process %s", ErrProcessNotFound, failure.ProcessID)
			}
			if failure.Tick < max(definition.SubmitAtTick, 1) {
				return fmt.Errorf("%w: process %s fails before submission", ErrInvalidInput, failure.ProcessID)
			}
		default:
			return fmt.Errorf("%w: unsupported failure type %q", ErrInvalidInput, failure.Type)
		}
	}
	return nil
}
