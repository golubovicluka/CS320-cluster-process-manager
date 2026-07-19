package cluster

import (
	"fmt"
	"slices"
	"sort"
	"sync"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

type Config struct {
	Scheduler             string
	Seed                  int64
	HeartbeatTimeoutTicks int64
	MaxNodes              int
	MaxProcesses          int
}

func DefaultConfig() Config {
	return Config{
		Scheduler:    scheduler.LeastLoadedName,
		Seed:         42,
		MaxNodes:     100,
		MaxProcesses: 10_000,
	}
}

type Controller struct {
	mu                    sync.RWMutex
	state                 *domain.Cluster
	scheduler             scheduler.Scheduler
	events                []domain.Event
	eventSequence         uint64
	heartbeatTimeoutTicks int64
	maxNodes              int
	maxProcesses          int
}

func New(config Config) (*Controller, error) {
	defaults := DefaultConfig()
	if config.Scheduler == "" {
		config.Scheduler = defaults.Scheduler
	}
	if config.MaxNodes == 0 {
		config.MaxNodes = defaults.MaxNodes
	}
	if config.MaxProcesses == 0 {
		config.MaxProcesses = defaults.MaxProcesses
	}
	if config.MaxNodes < 1 || config.MaxProcesses < 1 || config.HeartbeatTimeoutTicks < 0 {
		return nil, fmt.Errorf("%w: invalid controller limits", domain.ErrInvalidInput)
	}
	selected, err := scheduler.New(config.Scheduler)
	if err != nil {
		return nil, err
	}
	return &Controller{
		state:                 domain.NewCluster(selected.Name(), config.Seed),
		scheduler:             selected,
		events:                make([]domain.Event, 0),
		heartbeatTimeoutTicks: config.HeartbeatTimeoutTicks,
		maxNodes:              config.MaxNodes,
		maxProcesses:          config.MaxProcesses,
	}, nil
}

func (c *Controller) Snapshot() *domain.Cluster {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state.Clone()
}

func (c *Controller) Events() []domain.Event {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return slices.Clone(c.events)
}

func (c *Controller) RegisterNode(def domain.NodeDefinition) (*domain.Node, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.state.Nodes) >= c.maxNodes {
		return nil, fmt.Errorf("%w: node limit %d reached", domain.ErrInvalidInput, c.maxNodes)
	}
	node, err := domain.NewNode(def, c.state.CurrentTick)
	if err != nil {
		return nil, err
	}
	if _, exists := c.state.Nodes[node.ID]; exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrDuplicateNode, node.ID)
	}
	c.state.Nodes[node.ID] = node
	c.emitLocked(domain.EventNodeRegistered, "info", "", node.ID, "node registered")
	return node.Clone(), nil
}

func (c *Controller) Node(id string) (*domain.Node, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	node, exists := c.state.Nodes[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", domain.ErrNodeNotFound, id)
	}
	return node.Clone(), nil
}

func (c *Controller) Nodes() []*domain.Node {
	c.mu.RLock()
	defer c.mu.RUnlock()
	nodes := make([]*domain.Node, 0, len(c.state.Nodes))
	for _, node := range c.state.Nodes {
		nodes = append(nodes, node.Clone())
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

func (c *Controller) RemoveNode(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, exists := c.state.Nodes[id]
	if !exists {
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, id)
	}
	if len(node.RunningProcessIDs) != 0 || node.CPUAllocated != 0 || node.MemoryAllocatedMB != 0 {
		return fmt.Errorf("%w: node %s is not empty", domain.ErrInvalidStateTransition, id)
	}
	delete(c.state.Nodes, id)
	c.emitLocked(domain.EventNodeRemoved, "info", "", id, "node removed")
	return nil
}

func (c *Controller) SetScheduler(name string) error {
	selected, err := scheduler.New(name)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scheduler = selected
	c.state.SchedulerName = selected.Name()
	c.emitLocked(domain.EventSchedulerChanged, "info", "", "", "scheduler changed to "+selected.Name())
	return nil
}

func (c *Controller) SchedulerName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.scheduler.Name()
}

func (c *Controller) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.SimulationStatus == domain.SimulationRunning {
		return
	}
	c.state.SimulationStatus = domain.SimulationRunning
	c.state.FinishReason = ""
	c.emitLocked(domain.EventSimulationStarted, "info", "", "", "simulation started")
}

func (c *Controller) PauseSimulation() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.SimulationStatus != domain.SimulationRunning {
		return
	}
	c.state.SimulationStatus = domain.SimulationPaused
	c.emitLocked(domain.EventSimulationPaused, "info", "", "", "simulation paused")
}

func (c *Controller) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = domain.NewCluster(c.scheduler.Name(), c.state.Seed)
	c.events = c.events[:0]
	c.eventSequence = 0
	c.emitLocked(domain.EventSimulationReset, "info", "", "", "simulation reset")
}

func (c *Controller) emitLocked(eventType domain.EventType, severity, processID, nodeID, message string) {
	c.eventSequence++
	c.events = append(c.events, domain.Event{
		ID:        fmt.Sprintf("event-%06d", c.eventSequence),
		Tick:      c.state.CurrentTick,
		Type:      eventType,
		Severity:  severity,
		ProcessID: processID,
		NodeID:    nodeID,
		Message:   message,
	})
}
