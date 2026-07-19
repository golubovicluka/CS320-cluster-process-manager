package scheduler

import (
	"fmt"
	"strings"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

const (
	RoundRobinName    = "round-robin"
	LeastLoadedName   = "least-loaded"
	PriorityAwareName = "priority-aware"
)

type Scheduler interface {
	Name() string
	SelectNode(process domain.Process, nodes []domain.NodeSnapshot) (string, error)
}

type ProcessOrderer interface {
	OrderReady(processes []*domain.Process, currentTick int64)
}

func New(name string) (Scheduler, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", RoundRobinName, "rr":
		return NewRoundRobin(), nil
	case LeastLoadedName, "least_loaded", "leastloaded":
		return NewLeastLoaded(), nil
	case PriorityAwareName, "priority", "priority_aware":
		return NewPriorityAware(5), nil
	default:
		return nil, fmt.Errorf("%w: unknown scheduler %q", domain.ErrInvalidInput, name)
	}
}

func Available() []string {
	return []string{RoundRobinName, LeastLoadedName, PriorityAwareName}
}
