package metrics

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func TestBuildReport(t *testing.T) {
	finishedAt := int64(7)
	startedAt := int64(2)
	cluster := domain.NewCluster("least-loaded", 42)
	cluster.ScenarioName = "balanced"
	cluster.CurrentTick = 10
	cluster.FinishReason = "done"
	cluster.Statistics = domain.Statistics{
		CPUUtilizationSum: 3, MemoryUtilizationSum: 2, UtilizationSamples: 10,
		SchedulingDeferred: 4, NodeFailures: 1, Reschedulings: 1,
	}
	cluster.Nodes["n1"] = &domain.Node{ID: "n1", CPUCapacity: 4, MemoryCapacityMB: 100, CPUAllocated: 2, MemoryAllocatedMB: 25}
	cluster.Processes["p1"] = &domain.Process{
		ID: "p1", State: domain.ProcessTerminated, SubmittedAtTick: 0, StartedAtTick: &startedAt,
		FinishedAtTick: &finishedAt, WaitingTicks: 2, RestartCount: 1,
	}

	report := Build(cluster)
	if report.TerminatedProcesses != 1 || report.Restarts != 1 || report.AverageWaitingTicks != 2 || report.AverageTurnaroundTicks != 7 {
		t.Fatalf("unexpected process metrics: %+v", report)
	}
	if report.ThroughputPerTick != 0.1 || report.SuccessRate != 1 || report.AverageCPUUtilization != 0.3 || report.CurrentCPUUtilization != 0.5 {
		t.Fatalf("unexpected aggregate metrics: %+v", report)
	}
}

func TestExports(t *testing.T) {
	report := Report{ScenarioName: "test", Scheduler: "round-robin", ProcessCount: 2, TerminatedProcesses: 2}
	var jsonOutput bytes.Buffer
	if err := WriteJSON(&jsonOutput, report); err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(jsonOutput.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ScenarioName != report.ScenarioName {
		t.Fatalf("unexpected JSON report: %+v", decoded)
	}

	var csvOutput bytes.Buffer
	if err := WriteCSV(&csvOutput, report); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(&csvOutput).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[1][0] != "test" || records[1][4] != "2" {
		t.Fatalf("unexpected CSV: %v", records)
	}
}
