package engine

import (
	"context"
	"testing"
	"time"

	"github.com/golubovicluka/CS320-PZ/internal/cluster"
	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func scenario() domain.Scenario {
	return domain.Scenario{
		Name: "deterministic", Seed: 42, Scheduler: scheduler.RoundRobinName, MaxTicks: 20,
		Nodes: []domain.NodeDefinition{
			{ID: "n1", Name: "one", CPUCapacity: 1, MemoryCapacityMB: 128},
			{ID: "n2", Name: "two", CPUCapacity: 1, MemoryCapacityMB: 128},
		},
		Processes: []domain.ProcessDefinition{
			{ID: "p1", Name: "one", CPURequest: 1, MemoryRequestMB: 64, TotalTicks: 3, RestartPolicy: domain.RestartOnFailure, MaxRestarts: 1},
			{ID: "p2", Name: "two", CPURequest: 1, MemoryRequestMB: 64, TotalTicks: 2, SubmitAtTick: 2},
		},
		Failures: []domain.FailureDefinition{{Tick: 2, Type: domain.FailureNode, NodeID: "n1"}},
	}
}

func newEngine(t *testing.T, duration time.Duration) (*cluster.Controller, *Engine) {
	t.Helper()
	controller, err := cluster.New(cluster.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	current, err := New(controller, duration)
	if err != nil {
		t.Fatal(err)
	}
	return controller, current
}

func TestRunScenarioIsDeterministic(t *testing.T) {
	controller, current := newEngine(t, time.Millisecond)
	first, err := current.RunScenario(scenario())
	if err != nil {
		t.Fatal(err)
	}
	firstEvents := controller.Events()
	second, err := current.RunScenario(scenario())
	if err != nil {
		t.Fatal(err)
	}
	secondEvents := controller.Events()
	if first.CurrentTick != second.CurrentTick || first.Statistics != second.Statistics || len(firstEvents) != len(secondEvents) {
		t.Fatalf("runs differ: first=%+v second=%+v events=%d/%d", first.Statistics, second.Statistics, len(firstEvents), len(secondEvents))
	}
	if first.Processes["p1"].RestartCount != 1 || first.Processes["p1"].State != domain.ProcessTerminated {
		t.Fatalf("scheduled failure did not recover: %+v", first.Processes["p1"])
	}
	if first.Processes["p2"].SubmittedAtTick != 2 {
		t.Fatalf("future process was submitted at tick %d", first.Processes["p2"].SubmittedAtTick)
	}
}

func TestRealtimeEngineStopsOnCompletion(t *testing.T) {
	controller, current := newEngine(t, time.Millisecond)
	if err := controller.LoadScenario(domain.Scenario{
		Name: "short", Scheduler: scheduler.RoundRobinName, MaxTicks: 2,
		Nodes:     []domain.NodeDefinition{{ID: "n1", Name: "one", CPUCapacity: 1, MemoryCapacityMB: 64}},
		Processes: []domain.ProcessDefinition{{ID: "p1", Name: "one", CPURequest: 1, MemoryRequestMB: 32, TotalTicks: 1}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := current.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(time.Second)
	for controller.Snapshot().SimulationStatus != domain.SimulationCompleted {
		select {
		case <-deadline:
			t.Fatal("real-time engine did not finish")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	current.Close()
}
