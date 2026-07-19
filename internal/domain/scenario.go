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
	for _, failure := range s.Failures {
		if failure.Tick <= 0 {
			return fmt.Errorf("%w: failure tick must be positive", ErrInvalidInput)
		}
		switch failure.Type {
		case FailureNode:
			if failure.NodeID == "" {
				return fmt.Errorf("%w: node failure requires nodeId", ErrInvalidInput)
			}
		case FailureProcess:
			if failure.ProcessID == "" {
				return fmt.Errorf("%w: process failure requires processId", ErrInvalidInput)
			}
		default:
			return fmt.Errorf("%w: unsupported failure type %q", ErrInvalidInput, failure.Type)
		}
	}
	return nil
}
