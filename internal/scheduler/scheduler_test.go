package scheduler

import (
	"errors"
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func testProcess() domain.Process {
	return domain.Process{ID: "p1", CPURequest: 1, MemoryRequestMB: 64}
}

func testNodes() []domain.NodeSnapshot {
	return []domain.NodeSnapshot{
		{ID: "n2", Status: domain.NodeOnline, CPUCapacity: 4, MemoryCapacityMB: 512},
		{ID: "n1", Status: domain.NodeOnline, CPUCapacity: 4, MemoryCapacityMB: 512},
		{ID: "n3", Status: domain.NodeOnline, CPUCapacity: 4, MemoryCapacityMB: 512},
	}
}

func TestRoundRobinCyclesDeterministically(t *testing.T) {
	roundRobin := NewRoundRobin()
	want := []string{"n1", "n2", "n3", "n1"}
	for index, expected := range want {
		got, err := roundRobin.SelectNode(testProcess(), testNodes())
		if err != nil {
			t.Fatalf("selection %d: %v", index, err)
		}
		if got != expected {
			t.Fatalf("selection %d: got %s, want %s", index, got, expected)
		}
	}
}

func TestRoundRobinSkipsUnavailableAndFullNodes(t *testing.T) {
	nodes := testNodes()
	nodes[1].Status = domain.NodeFailed
	nodes[0].CPUAllocated = nodes[0].CPUCapacity
	got, err := NewRoundRobin().SelectNode(testProcess(), nodes)
	if err != nil {
		t.Fatal(err)
	}
	if got != "n3" {
		t.Fatalf("got %s, want n3", got)
	}
}

func TestLeastLoadedUsesRelativeCPUAndMemoryLoad(t *testing.T) {
	nodes := []domain.NodeSnapshot{
		{ID: "busy", Status: domain.NodeOnline, CPUCapacity: 8, CPUAllocated: 6, MemoryCapacityMB: 1000, MemoryAllocatedMB: 500},
		{ID: "free", Status: domain.NodeOnline, CPUCapacity: 4, CPUAllocated: 1, MemoryCapacityMB: 500, MemoryAllocatedMB: 50},
	}
	got, err := NewLeastLoaded().SelectNode(testProcess(), nodes)
	if err != nil {
		t.Fatal(err)
	}
	if got != "free" {
		t.Fatalf("got %s, want free", got)
	}
}

func TestSchedulersRejectUnschedulableProcess(t *testing.T) {
	nodes := []domain.NodeSnapshot{{ID: "small", Status: domain.NodeOnline, CPUCapacity: 1, MemoryCapacityMB: 32}}
	for _, current := range []Scheduler{NewRoundRobin(), NewLeastLoaded(), NewPriorityAware(5)} {
		t.Run(current.Name(), func(t *testing.T) {
			_, err := current.SelectNode(testProcess(), nodes)
			if !errors.Is(err, domain.ErrNoSchedulableNode) {
				t.Fatalf("expected no schedulable node, got %v", err)
			}
		})
	}
}

func TestPriorityAwareOrdersByPriorityThenAge(t *testing.T) {
	processes := []*domain.Process{
		{ID: "normal", Priority: 5, LastReadyAtTick: 9, SubmittedAtTick: 1},
		{ID: "high", Priority: 10, LastReadyAtTick: 9, SubmittedAtTick: 2},
		{ID: "aged", Priority: 1, LastReadyAtTick: 0, SubmittedAtTick: 0},
	}
	NewPriorityAware(1).OrderReady(processes, 10)
	want := []string{"aged", "high", "normal"}
	for index, expected := range want {
		if processes[index].ID != expected {
			t.Fatalf("position %d: got %s, want %s", index, processes[index].ID, expected)
		}
	}
}

func TestFactory(t *testing.T) {
	for _, name := range Available() {
		current, err := New(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if current.Name() != name {
			t.Fatalf("got %s, want %s", current.Name(), name)
		}
	}
	if _, err := New("unknown"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
