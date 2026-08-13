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
	if state.CurrentTick != 1 || !strings.Contains(state.FinishReason, "NO_ONLINE_CAPACITY") {
		t.Fatalf("unexpected deadlock detection: %+v", state)
	}
}

func TestReadyProcessStopsWhenNoNodeAcceptsWork(t *testing.T) {
	statuses := []domain.NodeStatus{domain.NodeDraining, domain.NodeOffline, domain.NodeFailed}
	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			controller := newTestController(t, scheduler.RoundRobinName)
			addTestNode(t, controller, "n1", 1)
			if _, err := controller.ChangeNodeStatus("n1", status); err != nil {
				t.Fatal(err)
			}
			submitTestProcess(t, controller, "p1", 1, domain.RestartNever, 0)
			if err := controller.Step(); err != nil {
				t.Fatal(err)
			}
			state := controller.Snapshot()
			if !strings.Contains(state.FinishReason, "NO_ONLINE_CAPACITY") {
				t.Fatalf("unexpected finish reason: %q", state.FinishReason)
			}
		})
	}
}

func TestWaitingOrPausedProcessesStopAsExternallyBlocked(t *testing.T) {
	tests := map[string]func(*Controller, string) error{
		"waiting": func(controller *Controller, id string) error {
			_, err := controller.WaitProcess(id)
			return err
		},
		"paused": func(controller *Controller, id string) error {
			_, err := controller.PauseProcess(id)
			return err
		},
	}
	for name, block := range tests {
		t.Run(name, func(t *testing.T) {
			controller := newTestController(t, scheduler.RoundRobinName)
			addTestNode(t, controller, "n1", 1)
			submitTestProcess(t, controller, "p1", 5, domain.RestartNever, 0)
			if err := controller.Step(); err != nil {
				t.Fatal(err)
			}
			if err := block(controller, "p1"); err != nil {
				t.Fatal(err)
			}
			if err := controller.Step(); err != nil {
				t.Fatal(err)
			}
			state := controller.Snapshot()
			if !strings.Contains(state.FinishReason, "EXTERNALLY_BLOCKED") {
				t.Fatalf("unexpected finish reason: %q", state.FinishReason)
			}
		})
	}
}

func TestReadyCapacityBlockTakesPrecedenceOverExternalBlock(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	addTestNode(t, controller, "n1", 1)
	submitTestProcess(t, controller, "waiting", 5, domain.RestartNever, 0)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.WaitProcess("waiting"); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.SubmitProcess(domain.ProcessDefinition{
		ID: "too-large", Name: "too-large", CPURequest: 2, MemoryRequestMB: 128, TotalTicks: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if reason := controller.Snapshot().FinishReason; !strings.Contains(reason, "NO_ONLINE_CAPACITY") {
		t.Fatalf("unexpected finish reason: %q", reason)
	}
}

func TestFutureSubmissionPreventsEarlyNoProgressStop(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	err := controller.LoadScenario(domain.Scenario{
		Name: "future-submission", Scheduler: scheduler.RoundRobinName, MaxTicks: 10,
		Nodes: []domain.NodeDefinition{{ID: "n1", Name: "node", CPUCapacity: 1, MemoryCapacityMB: 64}},
		Processes: []domain.ProcessDefinition{{
			ID: "p1", Name: "later", CPURequest: 1, MemoryRequestMB: 32, TotalTicks: 1, SubmitAtTick: 3,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	state := controller.Snapshot()
	if state.SimulationStatus == domain.SimulationCompleted {
		t.Fatalf("future submission was ignored: %+v", state)
	}
}

func TestFutureFailurePreventsEarlyNoProgressStop(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	err := controller.LoadScenario(domain.Scenario{
		Name: "future-failure", Scheduler: scheduler.RoundRobinName, MaxTicks: 10,
		Nodes:     []domain.NodeDefinition{{ID: "n1", Name: "node", CPUCapacity: 1, MemoryCapacityMB: 64}},
		Processes: []domain.ProcessDefinition{{ID: "p1", Name: "too-large", CPURequest: 2, MemoryRequestMB: 32, TotalTicks: 1}},
		Failures:  []domain.FailureDefinition{{Tick: 3, Type: domain.FailureProcess, ProcessID: "p1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if state := controller.Snapshot(); state.SimulationStatus == domain.SimulationCompleted {
		t.Fatalf("future failure was ignored: %+v", state)
	}
	if err := controller.Steps(2); err != nil {
		t.Fatal(err)
	}
	state := controller.Snapshot()
	if state.SimulationStatus != domain.SimulationCompleted || state.Processes["p1"].State != domain.ProcessFailed {
		t.Fatalf("scheduled failure did not complete the scenario: %+v", state)
	}
}
