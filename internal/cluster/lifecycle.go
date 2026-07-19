package cluster

import (
	"errors"
	"fmt"
	"math"
	"slices"
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

func (c *Controller) WaitProcess(id string) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.processLocked(id)
	if err != nil {
		return nil, err
	}
	if process.State != domain.ProcessRunning {
		return nil, fmt.Errorf("%w: only a running process can enter waiting", domain.ErrInvalidStateTransition)
	}
	if err := c.releaseLocked(process); err != nil {
		return nil, err
	}
	if err := process.Transition(domain.ProcessWaiting); err != nil {
		return nil, err
	}
	c.emitLocked(domain.EventProcessWaiting, "info", process.ID, "", "process is waiting for simulated I/O")
	return process.Clone(), nil
}

func (c *Controller) WakeProcess(id string) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.processLocked(id)
	if err != nil {
		return nil, err
	}
	if process.State != domain.ProcessWaiting {
		return nil, fmt.Errorf("%w: only a waiting process can complete I/O", domain.ErrInvalidStateTransition)
	}
	if err := process.Transition(domain.ProcessReady); err != nil {
		return nil, err
	}
	process.LastReadyAtTick = c.state.CurrentTick
	c.emitLocked(domain.EventProcessIOCompleted, "info", process.ID, "", "simulated I/O completed")
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
	previousState := c.state.Clone()
	previousEvents := slices.Clone(c.events)
	previousSequence := c.eventSequence
	previousPending := clonePending(c.pendingProcesses)
	previousFailures := cloneFailures(c.scheduledFailures)
	rollback := func() {
		c.state = previousState
		c.events = previousEvents
		c.eventSequence = previousSequence
		c.pendingProcesses = previousPending
		c.scheduledFailures = previousFailures
	}
	if c.state.SimulationStatus != domain.SimulationRunning {
		c.state.SimulationStatus = domain.SimulationRunning
		c.emitLocked(domain.EventSimulationStarted, "info", "", "", "simulation started")
	}
	c.state.CurrentTick++
	if err := c.applyScheduledLocked(); err != nil {
		rollback()
		return err
	}
	c.detectMissedHeartbeatsLocked()
	c.scheduleReadyLocked()
	c.sampleUtilizationLocked()
	c.executeRunningLocked()
	if len(c.state.Processes) > 0 && c.allProcessesTerminalLocked() {
		c.finishLocked("all processes reached a terminal state")
	} else if c.state.MaxTicks > 0 && c.state.CurrentTick >= c.state.MaxTicks {
		c.finishLocked("maximum tick count reached")
	} else if c.noProgressPossibleLocked() {
		c.finishLocked("no progress is possible with available node capacities")
	}
	if err := c.validateInvariantsLocked(); err != nil {
		rollback()
		return err
	}
	return nil
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
	loads := make([]float64, 0, len(c.state.Nodes))
	for _, node := range c.state.Nodes {
		totalCPU += node.CPUCapacity
		allocatedCPU += node.CPUAllocated
		totalMemory += node.MemoryCapacityMB
		allocatedMemory += node.MemoryAllocatedMB
		loads = append(loads, 0.5*float64(node.CPUAllocated)/float64(node.CPUCapacity)+
			0.5*float64(node.MemoryAllocatedMB)/float64(node.MemoryCapacityMB))
	}
	if totalCPU > 0 {
		c.state.Statistics.CPUUtilizationSum += float64(allocatedCPU) / float64(totalCPU)
	}
	if totalMemory > 0 {
		c.state.Statistics.MemoryUtilizationSum += float64(allocatedMemory) / float64(totalMemory)
	}
	if len(loads) > 0 {
		var sum float64
		for _, load := range loads {
			sum += load
		}
		mean := sum / float64(len(loads))
		var squaredDifference float64
		for _, load := range loads {
			difference := load - mean
			squaredDifference += difference * difference
		}
		c.state.Statistics.LoadBalanceStdDevSum += math.Sqrt(squaredDifference / float64(len(loads)))
	}
	c.state.Statistics.UtilizationSamples++
}

func (c *Controller) allProcessesTerminalLocked() bool {
	if len(c.pendingProcesses) > 0 {
		return false
	}
	for _, process := range c.state.Processes {
		if !process.State.IsTerminal() {
			return false
		}
	}
	return true
}

func (c *Controller) noProgressPossibleLocked() bool {
	if len(c.pendingProcesses) > 0 {
		return false
	}
	ready := 0
	for _, process := range c.state.Processes {
		if process.State == domain.ProcessRunning {
			return false
		}
		if process.State != domain.ProcessReady {
			continue
		}
		ready++
		for _, node := range c.state.Nodes {
			if node.CPUCapacity >= process.CPURequest && node.MemoryCapacityMB >= process.MemoryRequestMB {
				return false
			}
		}
	}
	return ready > 0
}

func (c *Controller) finishLocked(reason string) {
	c.state.SimulationStatus = domain.SimulationCompleted
	c.state.FinishReason = reason
	c.emitLocked(domain.EventSimulationFinished, "info", "", "", reason)
}

func clonePending(source map[int64][]domain.ProcessDefinition) map[int64][]domain.ProcessDefinition {
	clone := make(map[int64][]domain.ProcessDefinition, len(source))
	for tick, definitions := range source {
		clone[tick] = slices.Clone(definitions)
	}
	return clone
}

func cloneFailures(source map[int64][]domain.FailureDefinition) map[int64][]domain.FailureDefinition {
	clone := make(map[int64][]domain.FailureDefinition, len(source))
	for tick, definitions := range source {
		clone[tick] = slices.Clone(definitions)
	}
	return clone
}
