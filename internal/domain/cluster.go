package domain

type Statistics struct {
	CPUUtilizationSum    float64 `json:"cpuUtilizationSum"`
	MemoryUtilizationSum float64 `json:"memoryUtilizationSum"`
	UtilizationSamples   int64   `json:"utilizationSamples"`
	SchedulingDeferred   int64   `json:"schedulingDeferred"`
	NodeFailures         int64   `json:"nodeFailures"`
	Reschedulings        int64   `json:"reschedulings"`
}

type Cluster struct {
	Nodes            map[string]*Node    `json:"nodes"`
	Processes        map[string]*Process `json:"processes"`
	SchedulerName    string              `json:"scheduler"`
	CurrentTick      int64               `json:"currentTick"`
	SimulationStatus SimulationStatus    `json:"simulationStatus"`
	ScenarioName     string              `json:"scenarioName,omitempty"`
	Seed             int64               `json:"seed"`
	FinishReason     string              `json:"finishReason,omitempty"`
	Statistics       Statistics          `json:"statistics"`
}

func NewCluster(schedulerName string, seed int64) *Cluster {
	return &Cluster{
		Nodes:            make(map[string]*Node),
		Processes:        make(map[string]*Process),
		SchedulerName:    schedulerName,
		SimulationStatus: SimulationIdle,
		Seed:             seed,
	}
}

func (c Cluster) Clone() *Cluster {
	clone := c
	clone.Nodes = make(map[string]*Node, len(c.Nodes))
	for id, node := range c.Nodes {
		clone.Nodes[id] = node.Clone()
	}
	clone.Processes = make(map[string]*Process, len(c.Processes))
	for id, process := range c.Processes {
		clone.Processes[id] = process.Clone()
	}
	return &clone
}
