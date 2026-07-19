package cluster

import (
	"errors"
	"sync"
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func newTestController(t *testing.T, schedulerName string) *Controller {
	t.Helper()
	controller, err := New(Config{Scheduler: schedulerName, Seed: 42})
	if err != nil {
		t.Fatal(err)
	}
	return controller
}

func addTestNode(t *testing.T, controller *Controller, id string, cpu int) {
	t.Helper()
	_, err := controller.RegisterNode(domain.NodeDefinition{ID: id, Name: id, CPUCapacity: cpu, MemoryCapacityMB: 1024})
	if err != nil {
		t.Fatal(err)
	}
}

func submitTestProcess(t *testing.T, controller *Controller, id string, ticks int, policy domain.RestartPolicy, maxRestarts int) {
	t.Helper()
	_, err := controller.SubmitProcess(domain.ProcessDefinition{
		ID: id, Name: id, CPURequest: 1, MemoryRequestMB: 128, TotalTicks: ticks,
		RestartPolicy: policy, MaxRestarts: maxRestarts,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterAndSubmitRejectDuplicates(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	addTestNode(t, controller, "n1", 2)
	if _, err := controller.RegisterNode(domain.NodeDefinition{ID: "n1", Name: "duplicate", CPUCapacity: 1, MemoryCapacityMB: 1}); !errors.Is(err, domain.ErrDuplicateNode) {
		t.Fatalf("expected duplicate node, got %v", err)
	}
	submitTestProcess(t, controller, "p1", 2, domain.RestartNever, 0)
	if _, err := controller.SubmitProcess(domain.ProcessDefinition{ID: "p1", Name: "duplicate", CPURequest: 1, MemoryRequestMB: 1, TotalTicks: 1}); !errors.Is(err, domain.ErrDuplicateProcess) {
		t.Fatalf("expected duplicate process, got %v", err)
	}
}

func TestStepSchedulesCompletesAndReleasesResources(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	addTestNode(t, controller, "n1", 1)
	submitTestProcess(t, controller, "p1", 2, domain.RestartNever, 0)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	process, _ := controller.Process("p1")
	if process.State != domain.ProcessRunning || process.RemainingTicks != 1 || process.NodeID != "n1" {
		t.Fatalf("unexpected process after first tick: %+v", process)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	process, _ = controller.Process("p1")
	node, _ := controller.Node("n1")
	if process.State != domain.ProcessTerminated || process.NodeID != "" || node.CPUAllocated != 0 || len(node.RunningProcessIDs) != 0 {
		t.Fatalf("completion did not release resources: process=%+v node=%+v", process, node)
	}
	if err := controller.ValidateInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestCapacityDefersThenSchedulesWaitingProcess(t *testing.T) {
	controller := newTestController(t, scheduler.LeastLoadedName)
	addTestNode(t, controller, "n1", 1)
	submitTestProcess(t, controller, "p1", 1, domain.RestartNever, 0)
	submitTestProcess(t, controller, "p2", 1, domain.RestartNever, 0)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	p2, _ := controller.Process("p2")
	if p2.State != domain.ProcessReady {
		t.Fatalf("expected p2 to remain ready, got %s", p2.State)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	p2, _ = controller.Process("p2")
	if p2.State != domain.ProcessTerminated || p2.WaitingTicks != 2 {
		t.Fatalf("unexpected second process result: %+v", p2)
	}
}

func TestPauseResumeAndKill(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	addTestNode(t, controller, "n1", 1)
	submitTestProcess(t, controller, "p1", 5, domain.RestartNever, 0)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.PauseProcess("p1"); err != nil {
		t.Fatal(err)
	}
	node, _ := controller.Node("n1")
	if node.CPUAllocated != 0 {
		t.Fatal("pause did not release resources")
	}
	if _, err := controller.ResumeProcess("p1"); err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.KillProcess("p1"); err != nil {
		t.Fatal(err)
	}
	process, _ := controller.Process("p1")
	if process.State != domain.ProcessKilled || process.NodeID != "" {
		t.Fatalf("unexpected killed process: %+v", process)
	}
}

func TestConcurrentSubmissionsAreSafe(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	var wait sync.WaitGroup
	for index := range 100 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			id := "p-" + string(rune(index+1000))
			_, _ = controller.SubmitProcess(domain.ProcessDefinition{ID: id, Name: id, CPURequest: 1, MemoryRequestMB: 1, TotalTicks: 1})
		}()
	}
	wait.Wait()
	if got := len(controller.Processes()); got != 100 {
		t.Fatalf("got %d processes, want 100", got)
	}
}
