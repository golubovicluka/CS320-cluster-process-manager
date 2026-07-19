package cluster

import (
	"strings"
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func TestStepRollsBackFailedScheduledAction(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	err := controller.LoadScenario(domain.Scenario{
		Name: "rollback", Scheduler: scheduler.RoundRobinName, MaxTicks: 5,
		Nodes: []domain.NodeDefinition{{ID: "n1", Name: "node", CPUCapacity: 2, MemoryCapacityMB: 128}},
		Processes: []domain.ProcessDefinition{
			{ID: "p1", Name: "short", CPURequest: 1, MemoryRequestMB: 32, TotalTicks: 1},
			{ID: "p2", Name: "long", CPURequest: 1, MemoryRequestMB: 32, TotalTicks: 5},
		},
		Failures: []domain.FailureDefinition{{Tick: 2, Type: domain.FailureProcess, ProcessID: "p1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	before := controller.Snapshot()
	beforeEvents := len(controller.Events())
	if err := controller.Step(); err == nil {
		t.Fatal("expected scheduled failure of terminal process to fail")
	}
	after := controller.Snapshot()
	if after.CurrentTick != before.CurrentTick || after.Processes["p2"].RemainingTicks != before.Processes["p2"].RemainingTicks || len(controller.Events()) != beforeEvents {
		t.Fatalf("failed step was not rolled back: before=%+v after=%+v", before, after)
	}
}

func TestLoadedScenarioStopsAtMaxTicks(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	err := controller.LoadScenario(domain.Scenario{
		Name: "max-ticks", Scheduler: scheduler.RoundRobinName, MaxTicks: 3,
		Nodes:     []domain.NodeDefinition{{ID: "n1", Name: "node", CPUCapacity: 1, MemoryCapacityMB: 64}},
		Processes: []domain.ProcessDefinition{{ID: "p1", Name: "job", CPURequest: 1, MemoryRequestMB: 32, TotalTicks: 10}},
		Failures:  []domain.FailureDefinition{{Tick: 1, Type: domain.FailureNode, NodeID: "n1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.Steps(10); err != nil {
		t.Fatal(err)
	}
	state := controller.Snapshot()
	if state.CurrentTick != 3 || state.SimulationStatus != domain.SimulationCompleted || state.FinishReason != "maximum tick count reached" {
		t.Fatalf("scenario did not stop at maxTicks: %+v", state)
	}
}

func TestStructurallyUnschedulableScenarioStops(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	err := controller.LoadScenario(domain.Scenario{
		Name: "no-progress", Scheduler: scheduler.RoundRobinName, MaxTicks: 50,
		Nodes:     []domain.NodeDefinition{{ID: "n1", Name: "node", CPUCapacity: 1, MemoryCapacityMB: 64}},
		Processes: []domain.ProcessDefinition{{ID: "p1", Name: "too-large", CPURequest: 2, MemoryRequestMB: 32, TotalTicks: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	state := controller.Snapshot()
	if state.CurrentTick != 1 || !strings.Contains(state.FinishReason, "no progress") {
		t.Fatalf("unexpected deadlock detection: %+v", state)
	}
}
