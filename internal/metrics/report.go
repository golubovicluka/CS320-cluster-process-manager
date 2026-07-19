package metrics

import "github.com/golubovicluka/CS320-PZ/internal/domain"

type Report struct {
	ScenarioName             string  `json:"scenarioName"`
	Seed                     int64   `json:"seed"`
	Scheduler                string  `json:"scheduler"`
	NodeCount                int     `json:"nodeCount"`
	ProcessCount             int     `json:"processCount"`
	ReadyProcesses           int     `json:"readyProcesses"`
	RunningProcesses         int     `json:"runningProcesses"`
	PausedProcesses          int     `json:"pausedProcesses"`
	TerminatedProcesses      int     `json:"terminatedProcesses"`
	FailedProcesses          int     `json:"failedProcesses"`
	KilledProcesses          int     `json:"killedProcesses"`
	Restarts                 int     `json:"restarts"`
	AverageWaitingTicks      float64 `json:"averageWaitingTicks"`
	MaximumWaitingTicks      int64   `json:"maximumWaitingTicks"`
	AverageTurnaroundTicks   float64 `json:"averageTurnaroundTicks"`
	ThroughputPerTick        float64 `json:"throughputPerTick"`
	SuccessRate              float64 `json:"successRate"`
	CurrentCPUUtilization    float64 `json:"currentCpuUtilization"`
	CurrentMemoryUtilization float64 `json:"currentMemoryUtilization"`
	AverageCPUUtilization    float64 `json:"averageCpuUtilization"`
	AverageMemoryUtilization float64 `json:"averageMemoryUtilization"`
	LoadBalanceStdDev        float64 `json:"loadBalanceStdDev"`
	SchedulingDeferrals      int64   `json:"schedulingDeferrals"`
	NodeFailures             int64   `json:"nodeFailures"`
	Reschedulings            int64   `json:"reschedulings"`
	TotalTicks               int64   `json:"totalTicks"`
	FinishReason             string  `json:"finishReason"`
}

func Build(cluster *domain.Cluster) Report {
	if cluster == nil {
		return Report{}
	}
	report := Report{
		ScenarioName:        cluster.ScenarioName,
		Seed:                cluster.Seed,
		Scheduler:           cluster.SchedulerName,
		NodeCount:           len(cluster.Nodes),
		ProcessCount:        len(cluster.Processes),
		SchedulingDeferrals: cluster.Statistics.SchedulingDeferred,
		NodeFailures:        cluster.Statistics.NodeFailures,
		Reschedulings:       cluster.Statistics.Reschedulings,
		TotalTicks:          cluster.CurrentTick,
		FinishReason:        cluster.FinishReason,
	}

	var totalWaiting int64
	var totalTurnaround int64
	waitingSamples := 0
	turnaroundSamples := 0
	for _, process := range cluster.Processes {
		report.Restarts += process.RestartCount
		if process.StartedAtTick != nil {
			totalWaiting += process.WaitingTicks
			waitingSamples++
			if process.WaitingTicks > report.MaximumWaitingTicks {
				report.MaximumWaitingTicks = process.WaitingTicks
			}
		}
		if process.FinishedAtTick != nil {
			totalTurnaround += *process.FinishedAtTick - process.SubmittedAtTick
			turnaroundSamples++
		}
		switch process.State {
		case domain.ProcessReady:
			report.ReadyProcesses++
		case domain.ProcessRunning:
			report.RunningProcesses++
		case domain.ProcessPaused, domain.ProcessWaiting:
			report.PausedProcesses++
		case domain.ProcessTerminated:
			report.TerminatedProcesses++
		case domain.ProcessFailed:
			report.FailedProcesses++
		case domain.ProcessKilled:
			report.KilledProcesses++
		}
	}
	if waitingSamples > 0 {
		report.AverageWaitingTicks = float64(totalWaiting) / float64(waitingSamples)
	}
	if turnaroundSamples > 0 {
		report.AverageTurnaroundTicks = float64(totalTurnaround) / float64(turnaroundSamples)
	}
	if cluster.CurrentTick > 0 {
		report.ThroughputPerTick = float64(report.TerminatedProcesses) / float64(cluster.CurrentTick)
	}
	if report.ProcessCount > 0 {
		report.SuccessRate = float64(report.TerminatedProcesses) / float64(report.ProcessCount)
	}
	if cluster.Statistics.UtilizationSamples > 0 {
		report.AverageCPUUtilization = cluster.Statistics.CPUUtilizationSum / float64(cluster.Statistics.UtilizationSamples)
		report.AverageMemoryUtilization = cluster.Statistics.MemoryUtilizationSum / float64(cluster.Statistics.UtilizationSamples)
		report.LoadBalanceStdDev = cluster.Statistics.LoadBalanceStdDevSum / float64(cluster.Statistics.UtilizationSamples)
	}
	report.CurrentCPUUtilization, report.CurrentMemoryUtilization = currentUtilization(cluster)
	return report
}

func currentUtilization(cluster *domain.Cluster) (float64, float64) {
	totalCPU, allocatedCPU := 0, 0
	totalMemory, allocatedMemory := 0, 0
	for _, node := range cluster.Nodes {
		totalCPU += node.CPUCapacity
		allocatedCPU += node.CPUAllocated
		totalMemory += node.MemoryCapacityMB
		allocatedMemory += node.MemoryAllocatedMB
	}
	var cpu, memory float64
	if totalCPU > 0 {
		cpu = float64(allocatedCPU) / float64(totalCPU)
	}
	if totalMemory > 0 {
		memory = float64(allocatedMemory) / float64(totalMemory)
	}
	return cpu, memory
}
