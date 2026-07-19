package scheduler

import (
	"slices"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

// PriorityAware treats larger priority values as more important. Aging adds one
// effective priority point for every AgingInterval ticks spent in READY state.
type PriorityAware struct {
	leastLoaded   *LeastLoaded
	AgingInterval int64
}

func NewPriorityAware(agingInterval int64) *PriorityAware {
	if agingInterval <= 0 {
		agingInterval = 5
	}
	return &PriorityAware{leastLoaded: NewLeastLoaded(), AgingInterval: agingInterval}
}

func (p *PriorityAware) Name() string {
	return PriorityAwareName
}

func (p *PriorityAware) SelectNode(process domain.Process, nodes []domain.NodeSnapshot) (string, error) {
	return p.leastLoaded.SelectNode(process, nodes)
}

func (p *PriorityAware) OrderReady(processes []*domain.Process, currentTick int64) {
	slices.SortStableFunc(processes, func(a, b *domain.Process) int {
		aPriority := p.effectivePriority(a, currentTick)
		bPriority := p.effectivePriority(b, currentTick)
		if aPriority > bPriority {
			return -1
		}
		if aPriority < bPriority {
			return 1
		}
		if a.SubmittedAtTick < b.SubmittedAtTick {
			return -1
		}
		if a.SubmittedAtTick > b.SubmittedAtTick {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
}

func (p *PriorityAware) effectivePriority(process *domain.Process, currentTick int64) int64 {
	waited := currentTick - process.LastReadyAtTick
	if waited < 0 {
		waited = 0
	}
	return int64(process.Priority) + waited/p.AgingInterval
}
