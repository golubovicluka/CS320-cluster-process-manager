package metrics

import "github.com/golubovicluka/CS320-PZ/internal/domain"

type Report struct {
	ScenarioName                    string  `json:"scenarioName"`
	Seed                            int64   `json:"seed"`
	Scheduler                       string  `json:"scheduler"`
	NodeCount                       int     `json:"nodeCount"`
	SubmittedProcesses              int     `json:"submittedProcesses"`
	StartedProcesses                int     `json:"startedProcesses"`
	NeverStartedProcesses           int     `json:"neverStartedProcesses"`
	ReadyProcesses                  int     `json:"readyProcesses"`
	RunningProcesses                int     `json:"runningProcesses"`
	PausedProcesses                 int     `json:"pausedProcesses"`
	WaitingProcesses                int     `json:"waitingProcesses"`
	TerminatedProcesses             int     `json:"terminatedProcesses"`
	FailedProcesses                 int     `json:"failedProcesses"`
	KilledProcesses                 int     `json:"killedProcesses"`
	Restarts                        int     `json:"restarts"`
	AverageWaitingTicksStarted      float64 `json:"averageWaitingTicksStarted"`
	AverageWaitingTicksAllSubmitted float64 `json:"averageWaitingTicksAllSubmitted"`
	MaximumWaitingTicksStarted      int64   `json:"maximumWaitingTicksStarted"`
	MaximumWaitingTicksAllSubmitted int64   `json:"maximumWaitingTicksAllSubmitted"`
	AverageTurnaroundTicks          float64 `json:"averageTurnaroundTicks"`
	ThroughputPerTick               float64 `json:"throughputPerTick"`
	SuccessRate                     float64 `json:"successRate"`
	CurrentCPUUtilization           float64 `json:"currentCpuUtilization"`
	CurrentMemoryUtilization        float64 `json:"currentMemoryUtilization"`
	AverageCPUUtilization           float64 `json:"averageCpuUtilization"`
	AverageMemoryUtilization        float64 `json:"averageMemoryUtilization"`
	LoadBalanceStdDev               float64 `json:"loadBalanceStdDev"`
	SchedulingDeferrals             int64   `json:"schedulingDeferrals"`
	NodeFailures                    int64   `json:"nodeFailures"`
	Reschedulings                   int64   `json:"reschedulings"`
	TotalTicks                      int64   `json:"totalTicks"`
	FinishReason                    string  `json:"finishReason"`
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
		SubmittedProcesses:  len(cluster.Processes),
		SchedulingDeferrals: cluster.Statistics.SchedulingDeferred,
		NodeFailures:        cluster.Statistics.NodeFailures,
		Reschedulings:       cluster.Statistics.Reschedulings,
		TotalTicks:          cluster.CurrentTick,
		FinishReason:        cluster.FinishReason,
	}

	var totalWaitingStarted int64
	var totalWaitingAllSubmitted int64
	var totalTurnaround int64
	turnaroundSamples := 0
	for _, process := range cluster.Processes {
		report.Restarts += process.RestartCount
		waitingTicks := observedWaitingTicks(process, cluster.CurrentTick)
		totalWaitingAllSubmitted += waitingTicks
		if waitingTicks > report.MaximumWaitingTicksAllSubmitted {
			report.MaximumWaitingTicksAllSubmitted = waitingTicks
		}
		if process.StartedAtTick != nil {
			report.StartedProcesses++
			totalWaitingStarted += waitingTicks
			if waitingTicks > report.MaximumWaitingTicksStarted {
				report.MaximumWaitingTicksStarted = waitingTicks
			}
		} else {
			report.NeverStartedProcesses++
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
		case domain.ProcessPaused:
			report.PausedProcesses++
		case domain.ProcessWaiting:
			report.WaitingProcesses++
		case domain.ProcessTerminated:
			report.TerminatedProcesses++
		case domain.ProcessFailed:
			report.FailedProcesses++
		case domain.ProcessKilled:
			report.KilledProcesses++
		}
	}
	if report.StartedProcesses > 0 {
		report.AverageWaitingTicksStarted = float64(totalWaitingStarted) / float64(report.StartedProcesses)
	}
	if report.SubmittedProcesses > 0 {
		report.AverageWaitingTicksAllSubmitted = float64(totalWaitingAllSubmitted) / float64(report.SubmittedProcesses)
	}
	if turnaroundSamples > 0 {
		report.AverageTurnaroundTicks = float64(totalTurnaround) / float64(turnaroundSamples)
	}
	if cluster.CurrentTick > 0 {
		report.ThroughputPerTick = float64(report.TerminatedProcesses) / float64(cluster.CurrentTick)
	}
	if report.SubmittedProcesses > 0 {
		report.SuccessRate = float64(report.TerminatedProcesses) / float64(report.SubmittedProcesses)
	}
	if cluster.Statistics.UtilizationSamples > 0 {
		report.AverageCPUUtilization = cluster.Statistics.CPUUtilizationSum / float64(cluster.Statistics.UtilizationSamples)
		report.AverageMemoryUtilization = cluster.Statistics.MemoryUtilizationSum / float64(cluster.Statistics.UtilizationSamples)
		report.LoadBalanceStdDev = cluster.Statistics.LoadBalanceStdDevSum / float64(cluster.Statistics.UtilizationSamples)
	}
	report.CurrentCPUUtilization, report.CurrentMemoryUtilization = currentUtilization(cluster)
	return report
}

func observedWaitingTicks(process *domain.Process, currentTick int64) int64 {
	waitingTicks := process.WaitingTicks
	if process.State == domain.ProcessReady && currentTick > process.LastReadyAtTick {
		waitingTicks += currentTick - process.LastReadyAtTick
	}
	if process.StartedAtTick == nil && process.State != domain.ProcessReady {
		endTick := currentTick
		if process.FinishedAtTick != nil {
			endTick = *process.FinishedAtTick
		}
		if elapsed := endTick - process.SubmittedAtTick; elapsed > waitingTicks {
			waitingTicks = elapsed
		}
	}
	if waitingTicks < 0 {
		return 0
	}
	return waitingTicks
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
