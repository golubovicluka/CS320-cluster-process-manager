package scheduler

import (
	"fmt"
	"slices"
	"sync"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

type RoundRobin struct {
	mu     sync.Mutex
	cursor int
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (r *RoundRobin) Name() string {
	return RoundRobinName
}

func (r *RoundRobin) SelectNode(process domain.Process, nodes []domain.NodeSnapshot) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ordered := slices.Clone(nodes)
	slices.SortFunc(ordered, func(a, b domain.NodeSnapshot) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	if len(ordered) == 0 {
		return "", fmt.Errorf("%w: no nodes registered", domain.ErrNoSchedulableNode)
	}
	start := r.cursor % len(ordered)
	for offset := range ordered {
		index := (start + offset) % len(ordered)
		if ordered[index].CanFit(process) {
			r.cursor = (index + 1) % len(ordered)
			return ordered[index].ID, nil
		}
	}
	return "", fmt.Errorf("%w: process %s", domain.ErrNoSchedulableNode, process.ID)
}
