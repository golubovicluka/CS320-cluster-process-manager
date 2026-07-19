package store

import (
	"bytes"
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func TestMemorySnapshotsAreIsolated(t *testing.T) {
	memory := NewMemory("round-robin", 42)
	snapshot := memory.Snapshot()
	node, _ := domain.NewNode(domain.NodeDefinition{ID: "n1", Name: "worker", CPUCapacity: 2, MemoryCapacityMB: 128}, 0)
	snapshot.Nodes[node.ID] = node
	if len(memory.Snapshot().Nodes) != 0 {
		t.Fatal("mutating snapshot changed stored state")
	}
	if err := memory.Replace(snapshot); err != nil {
		t.Fatal(err)
	}
	if len(memory.Snapshot().Nodes) != 1 {
		t.Fatal("replacement was not stored")
	}
}

func TestSnapshotJSONRoundTrip(t *testing.T) {
	cluster := domain.NewCluster("least-loaded", 9)
	node, _ := domain.NewNode(domain.NodeDefinition{ID: "n1", Name: "worker", CPUCapacity: 2, MemoryCapacityMB: 128}, 0)
	cluster.Nodes[node.ID] = node
	var output bytes.Buffer
	if err := WriteSnapshot(&output, cluster); err != nil {
		t.Fatal(err)
	}
	restored, err := ReadSnapshot(&output)
	if err != nil {
		t.Fatal(err)
	}
	if restored.SchedulerName != "least-loaded" || restored.Nodes["n1"].CPUCapacity != 2 {
		t.Fatalf("unexpected restored snapshot: %+v", restored)
	}
}
