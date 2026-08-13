package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/golubovicluka/CS320-PZ/internal/metrics"
)

func TestRunScenarioCommand(t *testing.T) {
	var output bytes.Buffer
	path := filepath.Join("..", "..", "scenarios", "balanced.json")
	if err := run([]string{"-scenario", path, "-format", "json"}, &output); err != nil {
		t.Fatal(err)
	}
	var report metrics.Report
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.SubmittedProcesses != 12 || report.StartedProcesses != 12 || report.NeverStartedProcesses != 0 ||
		report.TerminatedProcesses != 12 || report.Scheduler != "round-robin" {
		t.Fatalf("unexpected report: %+v", report)
	}
}
