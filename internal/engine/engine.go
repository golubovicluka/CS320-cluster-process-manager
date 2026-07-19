package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golubovicluka/CS320-PZ/internal/cluster"
	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

type Engine struct {
	controller   *cluster.Controller
	tickDuration time.Duration

	controlMu sync.Mutex
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
}

func New(controller *cluster.Controller, tickDuration time.Duration) (*Engine, error) {
	if controller == nil {
		return nil, fmt.Errorf("%w: controller cannot be nil", domain.ErrInvalidInput)
	}
	if tickDuration <= 0 {
		tickDuration = 500 * time.Millisecond
	}
	return &Engine{controller: controller, tickDuration: tickDuration}, nil
}

func (e *Engine) Start(parent context.Context) error {
	if parent == nil {
		return fmt.Errorf("%w: context cannot be nil", domain.ErrInvalidInput)
	}
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	e.mu.Lock()
	if e.cancel != nil {
		e.mu.Unlock()
		return fmt.Errorf("%w: real-time engine is already running", domain.ErrInvalidStateTransition)
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	e.cancel = cancel
	e.done = done
	e.mu.Unlock()

	e.controller.Start()
	go e.loop(ctx, done)
	return nil
}

func (e *Engine) loop(ctx context.Context, done chan struct{}) {
	ticker := time.NewTicker(e.tickDuration)
	defer ticker.Stop()
	defer func() {
		e.mu.Lock()
		if e.done == done {
			e.cancel = nil
			e.done = nil
		}
		e.mu.Unlock()
		close(done)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.controller.Step(); err != nil {
				return
			}
			if e.controller.Snapshot().SimulationStatus == domain.SimulationCompleted {
				return
			}
		}
	}
}

func (e *Engine) Pause() {
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	e.stopRealtime()
	e.controller.PauseSimulation()
}

func (e *Engine) Close() {
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	e.stopRealtime()
}

func (e *Engine) stopRealtime() {
	e.mu.Lock()
	cancel := e.cancel
	done := e.done
	e.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func (e *Engine) RunSteps(ticks int) error {
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	e.stopRealtime()
	return e.controller.Steps(ticks)
}

func (e *Engine) LoadScenario(scenario domain.Scenario) error {
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	if err := e.controller.ValidateScenario(scenario); err != nil {
		return err
	}
	e.stopRealtime()
	return e.controller.LoadScenario(scenario)
}

func (e *Engine) Reset() {
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	e.stopRealtime()
	e.controller.Reset()
}

func (e *Engine) RunScenario(scenario domain.Scenario) (*domain.Cluster, error) {
	e.controlMu.Lock()
	defer e.controlMu.Unlock()
	if err := e.controller.ValidateScenario(scenario); err != nil {
		return nil, err
	}
	e.stopRealtime()
	if err := e.controller.LoadScenario(scenario); err != nil {
		return nil, err
	}
	for range scenario.MaxTicks {
		if err := e.controller.Step(); err != nil {
			return nil, err
		}
		if e.controller.Snapshot().SimulationStatus == domain.SimulationCompleted {
			return e.controller.Snapshot(), nil
		}
	}
	e.controller.Finish("maximum tick count reached")
	return e.controller.Snapshot(), nil
}
