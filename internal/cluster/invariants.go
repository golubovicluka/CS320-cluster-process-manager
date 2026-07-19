package cluster

import (
	"fmt"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func (c *Controller) ValidateInvariants() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.validateInvariantsLocked()
}

func (c *Controller) validateInvariantsLocked() error {
	placements := make(map[string]int)
	for nodeID, node := range c.state.Nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("node %s: %w", nodeID, err)
		}
		expectedCPU := 0
		expectedMemory := 0
		seen := make(map[string]bool)
		for _, processID := range node.RunningProcessIDs {
			if seen[processID] {
				return fmt.Errorf("node %s lists process %s more than once", nodeID, processID)
			}
			seen[processID] = true
			process, exists := c.state.Processes[processID]
			if !exists {
				return fmt.Errorf("node %s refers to unknown process %s", nodeID, processID)
			}
			if process.State != domain.ProcessRunning || process.NodeID != nodeID {
				return fmt.Errorf("node %s has inconsistent process %s", nodeID, processID)
			}
			expectedCPU += process.CPURequest
			expectedMemory += process.MemoryRequestMB
			placements[processID]++
		}
		if node.CPUAllocated != expectedCPU || node.MemoryAllocatedMB != expectedMemory {
			return fmt.Errorf("node %s resource counters do not match running processes", nodeID)
		}
	}
	for processID, process := range c.state.Processes {
		if process.State == domain.ProcessRunning {
			if process.NodeID == "" || placements[processID] != 1 {
				return fmt.Errorf("running process %s must have exactly one node", processID)
			}
		} else if process.NodeID != "" {
			return fmt.Errorf("non-running process %s retains node %s", processID, process.NodeID)
		}
	}
	return nil
}
