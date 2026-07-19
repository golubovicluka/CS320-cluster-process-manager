package cluster

import (
	"fmt"
	"sort"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
)

func (c *Controller) LoadScenario(scenario domain.Scenario) error {
	if err := scenario.Validate(); err != nil {
		return err
	}
	selected, err := scheduler.New(scenario.Scheduler)
	if err != nil {
		return err
	}
	if len(scenario.Nodes) > c.maxNodes || len(scenario.Processes) > c.maxProcesses {
		return fmt.Errorf("%w: scenario exceeds configured limits", domain.ErrInvalidInput)
	}

	state := domain.NewCluster(selected.Name(), scenario.Seed)
	state.ScenarioName = scenario.Name
	for _, definition := range scenario.Nodes {
		node, createErr := domain.NewNode(definition, 0)
		if createErr != nil {
			return createErr
		}
		if _, exists := state.Nodes[node.ID]; exists {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateNode, node.ID)
		}
		state.Nodes[node.ID] = node
	}

	pending := make(map[int64][]domain.ProcessDefinition)
	processIDs := make(map[string]bool)
	for _, definition := range scenario.Processes {
		process, createErr := domain.NewProcess(definition, max(definition.SubmitAtTick, 0))
		if createErr != nil {
			return createErr
		}
		if processIDs[process.ID] {
			return fmt.Errorf("%w: %s", domain.ErrDuplicateProcess, process.ID)
		}
		processIDs[process.ID] = true
		if definition.SubmitAtTick > 0 {
			pending[definition.SubmitAtTick] = append(pending[definition.SubmitAtTick], definition)
			continue
		}
		if transitionErr := process.Transition(domain.ProcessReady); transitionErr != nil {
			return transitionErr
		}
		state.Processes[process.ID] = process
	}

	failures := make(map[int64][]domain.FailureDefinition)
	for _, failure := range scenario.Failures {
		if failure.Type == domain.FailureNode {
			if _, exists := state.Nodes[failure.NodeID]; !exists {
				return fmt.Errorf("%w: scheduled failure references node %s", domain.ErrNodeNotFound, failure.NodeID)
			}
		} else if !processIDs[failure.ProcessID] {
			return fmt.Errorf("%w: scheduled failure references process %s", domain.ErrProcessNotFound, failure.ProcessID)
		}
		failures[failure.Tick] = append(failures[failure.Tick], failure)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
	c.scheduler = selected
	c.pendingProcesses = pending
	c.scheduledFailures = failures
	c.events = c.events[:0]
	c.eventSequence = 0
	nodeIDs := make([]string, 0, len(state.Nodes))
	for id := range state.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)
	for _, id := range nodeIDs {
		c.emitLocked(domain.EventNodeRegistered, "info", "", id, "node loaded from scenario")
	}
	processes := make([]string, 0, len(state.Processes))
	for id := range state.Processes {
		processes = append(processes, id)
	}
	sort.Strings(processes)
	for _, id := range processes {
		c.emitLocked(domain.EventProcessSubmitted, "info", id, "", "process loaded from scenario")
	}
	c.emitLocked(domain.EventScenarioLoaded, "info", "", "", "scenario loaded: "+scenario.Name)
	return nil
}

func (c *Controller) applyScheduledLocked() error {
	if definitions := c.pendingProcesses[c.state.CurrentTick]; len(definitions) > 0 {
		sort.SliceStable(definitions, func(i, j int) bool { return definitions[i].ID < definitions[j].ID })
		for _, definition := range definitions {
			if _, err := c.submitProcessLocked(definition); err != nil {
				return err
			}
		}
		delete(c.pendingProcesses, c.state.CurrentTick)
	}
	definitions := c.scheduledFailures[c.state.CurrentTick]
	for _, failure := range definitions {
		reason := failure.Message
		if reason == "" {
			reason = "scheduled scenario failure"
		}
		switch failure.Type {
		case domain.FailureNode:
			node, exists := c.state.Nodes[failure.NodeID]
			if !exists {
				return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, failure.NodeID)
			}
			if err := c.failNodeLocked(node, reason); err != nil {
				return err
			}
		case domain.FailureProcess:
			process, exists := c.state.Processes[failure.ProcessID]
			if !exists {
				return fmt.Errorf("%w: %s", domain.ErrProcessNotFound, failure.ProcessID)
			}
			if err := c.failProcessLocked(process, reason); err != nil {
				return err
			}
		}
	}
	delete(c.scheduledFailures, c.state.CurrentTick)
	return nil
}
