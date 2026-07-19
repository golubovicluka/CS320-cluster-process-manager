package domain

import (
	"errors"
	"testing"
)

func TestNewProcessValidation(t *testing.T) {
	tests := []struct {
		name string
		def  ProcessDefinition
	}{
		{name: "missing id", def: ProcessDefinition{Name: "worker", CPURequest: 1, MemoryRequestMB: 64, TotalTicks: 1}},
		{name: "zero cpu", def: ProcessDefinition{ID: "p1", Name: "worker", MemoryRequestMB: 64, TotalTicks: 1}},
		{name: "zero memory", def: ProcessDefinition{ID: "p1", Name: "worker", CPURequest: 1, TotalTicks: 1}},
		{name: "zero duration", def: ProcessDefinition{ID: "p1", Name: "worker", CPURequest: 1, MemoryRequestMB: 64}},
		{name: "invalid policy", def: ProcessDefinition{ID: "p1", Name: "worker", CPURequest: 1, MemoryRequestMB: 64, TotalTicks: 1, RestartPolicy: "SOMETIMES"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewProcess(test.def, 0)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

func TestProcessTransitions(t *testing.T) {
	p, err := NewProcess(ProcessDefinition{
		ID: "p1", Name: "worker", CPURequest: 1, MemoryRequestMB: 64, TotalTicks: 2,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []ProcessState{ProcessReady, ProcessRunning, ProcessPaused, ProcessReady, ProcessKilled} {
		if err := p.Transition(state); err != nil {
			t.Fatalf("transition to %s: %v", state, err)
		}
	}
	if err := p.Transition(ProcessReady); !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestNodeAllocationAndRelease(t *testing.T) {
	node, err := NewNode(NodeDefinition{ID: "n1", Name: "worker", CPUCapacity: 2, MemoryCapacityMB: 256}, 0)
	if err != nil {
		t.Fatal(err)
	}
	process, err := NewProcess(ProcessDefinition{ID: "p1", Name: "job", CPURequest: 2, MemoryRequestMB: 128, TotalTicks: 1}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := node.Allocate(*process); err != nil {
		t.Fatal(err)
	}
	if node.CPUAllocated != 2 || node.MemoryAllocatedMB != 128 || len(node.RunningProcessIDs) != 1 {
		t.Fatalf("unexpected allocation: %+v", node)
	}
	if err := node.Allocate(*process); !errors.Is(err, ErrInsufficientResources) {
		t.Fatalf("expected insufficient resources, got %v", err)
	}
	if err := node.Release(*process); err != nil {
		t.Fatal(err)
	}
	if node.CPUAllocated != 0 || node.MemoryAllocatedMB != 0 || len(node.RunningProcessIDs) != 0 {
		t.Fatalf("resources were not released: %+v", node)
	}
}

func TestNewNodeValidation(t *testing.T) {
	tests := []NodeDefinition{
		{Name: "missing-id", CPUCapacity: 1, MemoryCapacityMB: 1},
		{ID: "n1", Name: "zero-cpu", MemoryCapacityMB: 1},
		{ID: "n1", Name: "zero-memory", CPUCapacity: 1},
	}
	for _, definition := range tests {
		if _, err := NewNode(definition, 0); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected invalid node input for %+v, got %v", definition, err)
		}
	}
}

func TestScenarioRejectsFailureBeforeSubmission(t *testing.T) {
	scenario := Scenario{
		Name: "invalid", Scheduler: "round-robin", MaxTicks: 10,
		Nodes:     []NodeDefinition{{ID: "n1", Name: "node", CPUCapacity: 1, MemoryCapacityMB: 1}},
		Processes: []ProcessDefinition{{ID: "p1", Name: "job", CPURequest: 1, MemoryRequestMB: 1, TotalTicks: 1, SubmitAtTick: 5}},
		Failures:  []FailureDefinition{{Tick: 2, Type: FailureProcess, ProcessID: "p1"}},
	}
	if err := scenario.Validate(); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid scenario, got %v", err)
	}
}

func TestClusterCloneDoesNotShareMutableState(t *testing.T) {
	cluster := NewCluster("round-robin", 42)
	node, _ := NewNode(NodeDefinition{ID: "n1", Name: "worker", CPUCapacity: 1, MemoryCapacityMB: 64}, 0)
	cluster.Nodes[node.ID] = node
	clone := cluster.Clone()
	clone.Nodes["n1"].Status = NodeFailed
	if cluster.Nodes["n1"].Status != NodeOnline {
		t.Fatal("clone shares node state with source")
	}
}
