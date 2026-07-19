package cluster

import (
	"errors"
	"fmt"
	"sort"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func (c *Controller) SubmitProcess(def domain.ProcessDefinition) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.submitProcessLocked(def)
}

func (c *Controller) submitProcessLocked(def domain.ProcessDefinition) (*domain.Process, error) {
	if len(c.state.Processes) >= c.maxProcesses {
		return nil, fmt.Errorf("%w: process limit %d reached", domain.ErrInvalidInput, c.maxProcesses)
	}
	process, err := domain.NewProcess(def, c.state.CurrentTick)
	if err != nil {
		return nil, err
	}
	if _, exists := c.state.Processes[process.ID]; exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrDuplicateProcess, process.ID)
	}
	if err := process.Transition(domain.ProcessReady); err != nil {
		return nil, err
	}
	c.state.Processes[process.ID] = process
	if c.state.SimulationStatus == domain.SimulationCompleted {
		c.state.SimulationStatus = domain.SimulationIdle
		c.state.FinishReason = ""
	}
	c.emitLocked(domain.EventProcessSubmitted, "info", process.ID, "", "process submitted")
	return process.Clone(), nil
}

func (c *Controller) Process(id string) (*domain.Process, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	process, exists := c.state.Processes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrProcessNotFound, id)
	}
	return process.Clone(), nil
}

func (c *Controller) Processes() []*domain.Process {
	c.mu.RLock()
	defer c.mu.RUnlock()
	processes := make([]*domain.Process, 0, len(c.state.Processes))
	for _, process := range c.state.Processes {
		processes = append(processes, process.Clone())
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].ID < processes[j].ID })
	return processes
}

func (c *Controller) PauseProcess(id string) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.processLocked(id)
	if err != nil {
		return nil, err
	}
	if process.State != domain.ProcessRunning {
		return nil, fmt.Errorf("%w: only a running process can be paused", domain.ErrInvalidStateTransition)
	}
	if err := c.releaseLocked(process); err != nil {
		return nil, err
	}
	if err := process.Transition(domain.ProcessPaused); err != nil {
		return nil, err
	}
	c.emitLocked(domain.EventProcessPaused, "info", process.ID, "", "process paused")
	return process.Clone(), nil
}

func (c *Controller) ResumeProcess(id string) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.processLocked(id)
	if err != nil {
		return nil, err
	}
	if process.State != domain.ProcessPaused {
		return nil, fmt.Errorf("%w: only a paused process can be resumed", domain.ErrInvalidStateTransition)
	}
	if err := process.Transition(domain.ProcessReady); err != nil {
		return nil, err
	}
	process.LastReadyAtTick = c.state.CurrentTick
	c.emitLocked(domain.EventProcessResumed, "info", process.ID, "", "process returned to ready queue")
	return process.Clone(), nil
}

func (c *Controller) KillProcess(id string) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.processLocked(id)
	if err != nil {
		return nil, err
	}
	if process.State.IsTerminal() {
		return nil, fmt.Errorf("%w: process %s is already terminal", domain.ErrInvalidStateTransition, id)
	}
	if process.State == domain.ProcessRunning {
		if err := c.releaseLocked(process); err != nil {
			return nil, err
		}
	}
	if err := process.Transition(domain.ProcessKilled); err != nil {
		return nil, err
	}
	finished := c.state.CurrentTick
	process.FinishedAtTick = &finished
	c.emitLocked(domain.EventProcessKilled, "warning", process.ID, "", "process killed")
	return process.Clone(), nil
}

func (c *Controller) Step() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.SimulationStatus == domain.SimulationCompleted {
		return fmt.Errorf("%w: simulation has completed", domain.ErrInvalidStateTransition)
	}
	if c.state.SimulationStatus != domain.SimulationRunning {
		c.state.SimulationStatus = domain.SimulationRunning
		c.emitLocked(domain.EventSimulationStarted, "info", "", "", "simulation started")
	}
	c.state.CurrentTick++
	c.detectMissedHeartbeatsLocked()
	c.scheduleReadyLocked()
	c.sampleUtilizationLocked()
	c.executeRunningLocked()
	if len(c.state.Processes) > 0 && c.allProcessesTerminalLocked() {
		c.state.SimulationStatus = domain.SimulationCompleted
		c.state.FinishReason = "all processes reached a terminal state"
		c.emitLocked(domain.EventSimulationFinished, "info", "", "", c.state.FinishReason)
	}
	return c.validateInvariantsLocked()
}

func (c *Controller) Steps(count int) error {
	if count <= 0 {
		return fmt.Errorf("%w: tick count must be positive", domain.ErrInvalidInput)
	}
	for range count {
		err := c.Step()
		if errors.Is(err, domain.ErrInvalidStateTransition) && c.Snapshot().SimulationStatus == domain.SimulationCompleted {
			return nil
		}
		if err != nil {
			return err
		}
		if c.Snapshot().SimulationStatus == domain.SimulationCompleted {
			return nil
		}
	}
	return nil
}

func (c *Controller) scheduleReadyLocked() {
	ready := make([]*domain.Process, 0)
	for _, process := range c.state.Processes {
		if process.State == domain.ProcessReady {
			ready = append(ready, process)
		}
	}
	sort.SliceStable(ready, func(i, j int) bool {
		if ready[i].SubmittedAtTick != ready[j].SubmittedAtTick {
			return ready[i].SubmittedAtTick < ready[j].SubmittedAtTick
		}
		return ready[i].ID < ready[j].ID
	})
	if orderer, ok := c.scheduler.(scheduler.ProcessOrderer); ok {
		orderer.OrderReady(ready, c.state.CurrentTick)
	}
	nodes := c.nodeSnapshotsLocked()
	for _, process := range ready {
		nodeID, err := c.scheduler.SelectNode(*process, nodes)
		if err != nil {
			c.state.Statistics.SchedulingDeferred++
			c.emitLocked(domain.EventSchedulingDeferred, "info", process.ID, "", "no online node has enough resources")
			continue
		}
		node := c.state.Nodes[nodeID]
		if err := node.Allocate(*process); err != nil {
			c.state.Statistics.SchedulingDeferred++
			c.emitLocked(domain.EventSchedulingDeferred, "warning", process.ID, nodeID, err.Error())
			continue
		}
		if err := process.Transition(domain.ProcessRunning); err != nil {
			_ = node.Release(*process)
			continue
		}
		process.NodeID = nodeID
		process.WaitingTicks += c.state.CurrentTick - process.LastReadyAtTick
		if process.StartedAtTick == nil {
			started := c.state.CurrentTick
			process.StartedAtTick = &started
			c.emitLocked(domain.EventProcessStarted, "info", process.ID, nodeID, "process started")
		} else {
			c.state.Statistics.Reschedulings++
		}
		c.emitLocked(domain.EventProcessScheduled, "info", process.ID, nodeID, "process scheduled")
		c.emitLocked(domain.EventResourceAllocated, "info", process.ID, nodeID, "resources allocated")
		for index := range nodes {
			if nodes[index].ID == nodeID {
				nodes[index] = node.Snapshot()
				break
			}
		}
	}
}

func (c *Controller) executeRunningLocked() {
	ids := make([]string, 0)
	for id, process := range c.state.Processes {
		if process.State == domain.ProcessRunning {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		process := c.state.Processes[id]
		process.RemainingTicks--
		if process.RemainingTicks > 0 {
			continue
		}
		if err := c.releaseLocked(process); err != nil {
			continue
		}
		if err := process.Transition(domain.ProcessTerminated); err != nil {
			continue
		}
		finished := c.state.CurrentTick
		process.FinishedAtTick = &finished
		c.emitLocked(domain.EventProcessCompleted, "info", process.ID, "", "process completed")
	}
}

func (c *Controller) releaseLocked(process *domain.Process) error {
	nodeID := process.NodeID
	node, exists := c.state.Nodes[nodeID]
	if !exists {
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, nodeID)
	}
	if err := node.Release(*process); err != nil {
		return err
	}
	process.NodeID = ""
	c.emitLocked(domain.EventResourceReleased, "info", process.ID, nodeID, "resources released")
	return nil
}

func (c *Controller) processLocked(id string) (*domain.Process, error) {
	process, exists := c.state.Processes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrProcessNotFound, id)
	}
	return process, nil
}

func (c *Controller) nodeSnapshotsLocked() []domain.NodeSnapshot {
	nodes := make([]domain.NodeSnapshot, 0, len(c.state.Nodes))
	for _, node := range c.state.Nodes {
		nodes = append(nodes, node.Snapshot())
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

func (c *Controller) sampleUtilizationLocked() {
	totalCPU := 0
	allocatedCPU := 0
	totalMemory := 0
	allocatedMemory := 0
	for _, node := range c.state.Nodes {
		totalCPU += node.CPUCapacity
		allocatedCPU += node.CPUAllocated
		totalMemory += node.MemoryCapacityMB
		allocatedMemory += node.MemoryAllocatedMB
	}
	if totalCPU > 0 {
		c.state.Statistics.CPUUtilizationSum += float64(allocatedCPU) / float64(totalCPU)
	}
	if totalMemory > 0 {
		c.state.Statistics.MemoryUtilizationSum += float64(allocatedMemory) / float64(totalMemory)
	}
	c.state.Statistics.UtilizationSamples++
}

func (c *Controller) allProcessesTerminalLocked() bool {
	for _, process := range c.state.Processes {
		if !process.State.IsTerminal() {
			return false
		}
	}
	return true
}
