package cluster

import (
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func TestNodeFailureRestartsAndReschedulesProcess(t *testing.T) {
	controller := newTestController(t, scheduler.LeastLoadedName)
	addTestNode(t, controller, "n1", 1)
	addTestNode(t, controller, "n2", 1)
	submitTestProcess(t, controller, "p1", 3, domain.RestartOnFailure, 1)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	process, _ := controller.Process("p1")
	failedNode := process.NodeID
	if _, err := controller.FailNode(failedNode); err != nil {
		t.Fatal(err)
	}
	process, _ = controller.Process("p1")
	if process.State != domain.ProcessReady || process.RestartCount != 1 || process.RemainingTicks != process.TotalTicks {
		t.Fatalf("unexpected restarted process: %+v", process)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	process, _ = controller.Process("p1")
	if process.State != domain.ProcessRunning || process.NodeID == failedNode {
		t.Fatalf("process was not rescheduled: %+v", process)
	}
	if err := controller.ValidateInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestRestartNeverLeavesProcessFailed(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	addTestNode(t, controller, "n1", 1)
	submitTestProcess(t, controller, "p1", 2, domain.RestartNever, 0)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.FailProcess("p1", "simulated crash"); err != nil {
		t.Fatal(err)
	}
	process, _ := controller.Process("p1")
	if process.State != domain.ProcessFailed || process.FinishedAtTick == nil {
		t.Fatalf("unexpected failed process: %+v", process)
	}
}

func TestProcessDoesNotExceedMaxRestarts(t *testing.T) {
	controller := newTestController(t, scheduler.RoundRobinName)
	addTestNode(t, controller, "n1", 1)
	submitTestProcess(t, controller, "p1", 5, domain.RestartOnFailure, 1)
	if _, err := controller.FailProcess("p1", "first failure"); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.FailProcess("p1", "second failure"); err != nil {
		t.Fatal(err)
	}
	process, _ := controller.Process("p1")
	if process.RestartCount != 1 || process.State != domain.ProcessFailed {
		t.Fatalf("restart limit was not enforced: %+v", process)
	}
}

func TestHeartbeatTimeoutFailsNode(t *testing.T) {
	controller, err := New(Config{Scheduler: scheduler.RoundRobinName, HeartbeatTimeoutTicks: 1})
	if err != nil {
		t.Fatal(err)
	}
	addTestNode(t, controller, "n1", 1)
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	if err := controller.Step(); err != nil {
		t.Fatal(err)
	}
	node, _ := controller.Node("n1")
	if node.Status != domain.NodeFailed {
		t.Fatalf("got status %s, want FAILED", node.Status)
	}
	if _, err := controller.RecoverNode("n1"); err != nil {
		t.Fatal(err)
	}
	node, _ = controller.Node("n1")
	if node.Status != domain.NodeOnline {
		t.Fatalf("got status %s, want ONLINE", node.Status)
	}
}
