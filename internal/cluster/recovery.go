package cluster

import (
	"fmt"
	"sort"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func (c *Controller) ChangeNodeStatus(id string, status domain.NodeStatus) (*domain.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, exists := c.state.Nodes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrNodeNotFound, id)
	}
	if status == domain.NodeFailed {
		if err := c.failNodeLocked(node, "node failed by user request"); err != nil {
			return nil, err
		}
		return node.Clone(), nil
	}
	if !validNodeTransition(node.Status, status) {
		return nil, fmt.Errorf("%w: node %s -> %s", domain.ErrInvalidStateTransition, node.Status, status)
	}
	if status == domain.NodeOffline && len(node.RunningProcessIDs) > 0 {
		return nil, fmt.Errorf("%w: running processes must finish or node must fail", domain.ErrInvalidStateTransition)
	}
	node.Status = status
	if status == domain.NodeOnline {
		node.LastHeartbeatTick = c.state.CurrentTick
	}
	c.emitLocked(domain.EventNodeStatusChanged, "info", "", node.ID, "node status changed to "+string(status))
	return node.Clone(), nil
}

func (c *Controller) FailNode(id string) (*domain.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, exists := c.state.Nodes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrNodeNotFound, id)
	}
	if err := c.failNodeLocked(node, "node failed by user request"); err != nil {
		return nil, err
	}
	return node.Clone(), nil
}

func (c *Controller) RecoverNode(id string) (*domain.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, exists := c.state.Nodes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrNodeNotFound, id)
	}
	if node.Status != domain.NodeFailed && node.Status != domain.NodeOffline {
		return nil, fmt.Errorf("%w: only failed or offline nodes can recover", domain.ErrInvalidStateTransition)
	}
	node.Status = domain.NodeRecovering
	c.emitLocked(domain.EventNodeStatusChanged, "info", "", node.ID, "node is recovering")
	node.Status = domain.NodeOnline
	node.LastHeartbeatTick = c.state.CurrentTick
	c.emitLocked(domain.EventNodeRecovered, "info", "", node.ID, "node recovered")
	return node.Clone(), nil
}

func (c *Controller) Heartbeat(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, exists := c.state.Nodes[id]
	if !exists {
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, id)
	}
	if node.Status != domain.NodeOnline && node.Status != domain.NodeDraining {
		return fmt.Errorf("%w: node %s cannot heartbeat while %s", domain.ErrInvalidStateTransition, id, node.Status)
	}
	node.LastHeartbeatTick = c.state.CurrentTick
	return nil
}

func (c *Controller) FailProcess(id, reason string) (*domain.Process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	process, err := c.processLocked(id)
	if err != nil {
		return nil, err
	}
	if err := c.failProcessLocked(process, reason); err != nil {
		return nil, err
	}
	return process.Clone(), nil
}

func (c *Controller) failNodeLocked(node *domain.Node, reason string) error {
	if node.Status == domain.NodeFailed {
		return fmt.Errorf("%w: node %s is already failed", domain.ErrInvalidStateTransition, node.ID)
	}
	runningIDs := append([]string(nil), node.RunningProcessIDs...)
	sort.Strings(runningIDs)
	node.Status = domain.NodeFailed
	node.CPUAllocated = 0
	node.MemoryAllocatedMB = 0
	node.RunningProcessIDs = node.RunningProcessIDs[:0]
	c.state.Statistics.NodeFailures++
	c.emitLocked(domain.EventNodeFailed, "error", "", node.ID, reason)
	for _, processID := range runningIDs {
		process := c.state.Processes[processID]
		process.NodeID = ""
		if err := c.failProcessLocked(process, "hosting node "+node.ID+" failed"); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) failProcessLocked(process *domain.Process, reason string) error {
	if process.State.IsTerminal() {
		return fmt.Errorf("%w: process %s is terminal", domain.ErrInvalidStateTransition, process.ID)
	}
	nodeID := process.NodeID
	if process.State == domain.ProcessRunning && nodeID != "" {
		if node, exists := c.state.Nodes[nodeID]; exists && node.Status != domain.NodeFailed {
			if err := c.releaseLocked(process); err != nil {
				return err
			}
		} else {
			process.NodeID = ""
		}
	}
	if err := process.Transition(domain.ProcessFailed); err != nil {
		return err
	}
	process.LastError = reason
	c.emitLocked(domain.EventProcessFailed, "error", process.ID, nodeID, reason)
	if process.CanRestart() {
		process.RestartCount++
		process.RemainingTicks = process.TotalTicks
		process.FinishedAtTick = nil
		if err := process.Transition(domain.ProcessReady); err != nil {
			return err
		}
		process.LastReadyAtTick = c.state.CurrentTick
		c.emitLocked(domain.EventProcessRestarted, "warning", process.ID, "", "process queued for restart")
		return nil
	}
	finished := c.state.CurrentTick
	process.FinishedAtTick = &finished
	return nil
}

func (c *Controller) detectMissedHeartbeatsLocked() {
	if c.heartbeatTimeoutTicks <= 0 {
		return
	}
	ids := make([]string, 0)
	for id, node := range c.state.Nodes {
		if node.Status == domain.NodeOnline && c.state.CurrentTick-node.LastHeartbeatTick > c.heartbeatTimeoutTicks {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		node := c.state.Nodes[id]
		c.emitLocked(domain.EventNodeHeartbeatMissed, "error", "", id, "heartbeat timeout exceeded")
		_ = c.failNodeLocked(node, "heartbeat timeout exceeded")
	}
}

func validNodeTransition(from, to domain.NodeStatus) bool {
	if from == to {
		return true
	}
	allowed := map[domain.NodeStatus]map[domain.NodeStatus]bool{
		domain.NodeOnline: {
			domain.NodeDraining: true,
			domain.NodeOffline:  true,
		},
		domain.NodeDraining: {
			domain.NodeOnline:  true,
			domain.NodeOffline: true,
		},
		domain.NodeOffline: {
			domain.NodeOnline:     true,
			domain.NodeRecovering: true,
		},
		domain.NodeRecovering: {
			domain.NodeOnline: true,
		},
	}
	return allowed[from][to]
}
