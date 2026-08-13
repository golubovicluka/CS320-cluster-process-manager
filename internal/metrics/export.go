package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteCSV(w io.Writer, report Report) error {
	writer := csv.NewWriter(w)
	header := []string{
		"scenario", "seed", "scheduler", "nodes", "submitted_processes", "started_processes", "never_started_processes",
		"terminated", "failed", "killed", "restarts", "average_waiting_ticks_started",
		"average_waiting_ticks_all_submitted", "maximum_waiting_ticks_started",
		"maximum_waiting_ticks_all_submitted", "average_turnaround_ticks",
		"throughput_per_tick", "success_rate", "average_cpu_utilization", "average_memory_utilization",
		"load_balance_stddev", "scheduling_deferrals", "node_failures", "reschedulings", "total_ticks", "finish_reason",
	}
	row := []string{
		report.ScenarioName,
		strconv.FormatInt(report.Seed, 10),
		report.Scheduler,
		strconv.Itoa(report.NodeCount),
		strconv.Itoa(report.SubmittedProcesses),
		strconv.Itoa(report.StartedProcesses),
		strconv.Itoa(report.NeverStartedProcesses),
		strconv.Itoa(report.TerminatedProcesses),
		strconv.Itoa(report.FailedProcesses),
		strconv.Itoa(report.KilledProcesses),
		strconv.Itoa(report.Restarts),
		formatFloat(report.AverageWaitingTicksStarted),
		formatFloat(report.AverageWaitingTicksAllSubmitted),
		strconv.FormatInt(report.MaximumWaitingTicksStarted, 10),
		strconv.FormatInt(report.MaximumWaitingTicksAllSubmitted, 10),
		formatFloat(report.AverageTurnaroundTicks),
		formatFloat(report.ThroughputPerTick),
		formatFloat(report.SuccessRate),
		formatFloat(report.AverageCPUUtilization),
		formatFloat(report.AverageMemoryUtilization),
		formatFloat(report.LoadBalanceStdDev),
		strconv.FormatInt(report.SchedulingDeferrals, 10),
		strconv.FormatInt(report.NodeFailures, 10),
		strconv.FormatInt(report.Reschedulings, 10),
		strconv.FormatInt(report.TotalTicks, 10),
		report.FinishReason,
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}
	if err := writer.Write(row); err != nil {
		return fmt.Errorf("write CSV row: %w", err)
	}
	writer.Flush()
	return writer.Error()
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 6, 64)
}
