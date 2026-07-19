package domain

import (
	"fmt"
	"slices"
	"strings"
)

type Node struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Status            NodeStatus `json:"status"`
	CPUCapacity       int        `json:"cpuCapacity"`
	MemoryCapacityMB  int        `json:"memoryCapacityMB"`
	CPUAllocated      int        `json:"cpuAllocated"`
	MemoryAllocatedMB int        `json:"memoryAllocatedMB"`
	RunningProcessIDs []string   `json:"runningProcessIds"`
	LastHeartbeatTick int64      `json:"lastHeartbeatTick"`
}

type NodeDefinition struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	CPUCapacity      int    `json:"cpuCapacity"`
	MemoryCapacityMB int    `json:"memoryCapacityMB"`
}

type NodeSnapshot struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Status            NodeStatus `json:"status"`
	CPUCapacity       int        `json:"cpuCapacity"`
	MemoryCapacityMB  int        `json:"memoryCapacityMB"`
	CPUAllocated      int        `json:"cpuAllocated"`
	MemoryAllocatedMB int        `json:"memoryAllocatedMB"`
}

func NewNode(def NodeDefinition, tick int64) (*Node, error) {
	n := &Node{
		ID:                strings.TrimSpace(def.ID),
		Name:              strings.TrimSpace(def.Name),
		Status:            NodeOnline,
		CPUCapacity:       def.CPUCapacity,
		MemoryCapacityMB:  def.MemoryCapacityMB,
		RunningProcessIDs: []string{},
		LastHeartbeatTick: tick,
	}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func (n Node) Validate() error {
	if strings.TrimSpace(n.ID) == "" {
		return fmt.Errorf("%w: node id is required", ErrInvalidInput)
	}
	if strings.TrimSpace(n.Name) == "" {
		return fmt.Errorf("%w: node name is required", ErrInvalidInput)
	}
	if n.CPUCapacity <= 0 {
		return fmt.Errorf("%w: cpu capacity must be positive", ErrInvalidInput)
	}
	if n.MemoryCapacityMB <= 0 {
		return fmt.Errorf("%w: memory capacity must be positive", ErrInvalidInput)
	}
	if n.CPUAllocated < 0 || n.CPUAllocated > n.CPUCapacity {
		return fmt.Errorf("%w: cpu allocation is outside capacity", ErrInvalidInput)
	}
	if n.MemoryAllocatedMB < 0 || n.MemoryAllocatedMB > n.MemoryCapacityMB {
		return fmt.Errorf("%w: memory allocation is outside capacity", ErrInvalidInput)
	}
	return nil
}

func (n Node) Snapshot() NodeSnapshot {
	return NodeSnapshot{
		ID:                n.ID,
		Name:              n.Name,
		Status:            n.Status,
		CPUCapacity:       n.CPUCapacity,
		MemoryCapacityMB:  n.MemoryCapacityMB,
		CPUAllocated:      n.CPUAllocated,
		MemoryAllocatedMB: n.MemoryAllocatedMB,
	}
}

func (n NodeSnapshot) CanFit(p Process) bool {
	return n.Status == NodeOnline &&
		n.CPUCapacity-n.CPUAllocated >= p.CPURequest &&
		n.MemoryCapacityMB-n.MemoryAllocatedMB >= p.MemoryRequestMB
}

func (n *Node) Allocate(p Process) error {
	if n == nil || !n.Snapshot().CanFit(p) {
		return fmt.Errorf("%w: node cannot host process %s", ErrInsufficientResources, p.ID)
	}
	if slices.Contains(n.RunningProcessIDs, p.ID) {
		return fmt.Errorf("%w: process %s is already allocated", ErrInvalidInput, p.ID)
	}
	n.CPUAllocated += p.CPURequest
	n.MemoryAllocatedMB += p.MemoryRequestMB
	n.RunningProcessIDs = append(n.RunningProcessIDs, p.ID)
	return nil
}

func (n *Node) Release(p Process) error {
	if n == nil {
		return fmt.Errorf("%w: nil node", ErrInvalidInput)
	}
	index := slices.Index(n.RunningProcessIDs, p.ID)
	if index < 0 {
		return fmt.Errorf("%w: process %s is not allocated to node %s", ErrInvalidInput, p.ID, n.ID)
	}
	if n.CPUAllocated < p.CPURequest || n.MemoryAllocatedMB < p.MemoryRequestMB {
		return fmt.Errorf("%w: inconsistent allocation on node %s", ErrInvalidInput, n.ID)
	}
	n.CPUAllocated -= p.CPURequest
	n.MemoryAllocatedMB -= p.MemoryRequestMB
	n.RunningProcessIDs = slices.Delete(n.RunningProcessIDs, index, index+1)
	return nil
}

func (n Node) Clone() *Node {
	clone := n
	clone.RunningProcessIDs = slices.Clone(n.RunningProcessIDs)
	return &clone
}
