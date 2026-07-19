package scheduler

import (
	"fmt"
	"math"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

type LeastLoaded struct{}

func NewLeastLoaded() *LeastLoaded {
	return &LeastLoaded{}
}

func (l *LeastLoaded) Name() string {
	return LeastLoadedName
}

func (l *LeastLoaded) SelectNode(process domain.Process, nodes []domain.NodeSnapshot) (string, error) {
	selectedID := ""
	selectedScore := math.Inf(1)
	for _, node := range nodes {
		if !node.CanFit(process) {
			continue
		}
		score := loadScore(node)
		if score < selectedScore || (score == selectedScore && (selectedID == "" || node.ID < selectedID)) {
			selectedID = node.ID
			selectedScore = score
		}
	}
	if selectedID == "" {
		return "", fmt.Errorf("%w: process %s", domain.ErrNoSchedulableNode, process.ID)
	}
	return selectedID, nil
}

func loadScore(node domain.NodeSnapshot) float64 {
	cpu := float64(node.CPUAllocated) / float64(node.CPUCapacity)
	memory := float64(node.MemoryAllocatedMB) / float64(node.MemoryCapacityMB)
	return 0.5*cpu + 0.5*memory
}
