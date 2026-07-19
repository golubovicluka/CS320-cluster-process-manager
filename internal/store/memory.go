package store

import (
	"fmt"
	"sync"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

type Memory struct {
	mu      sync.RWMutex
	cluster *domain.Cluster
}

func NewMemory(schedulerName string, seed int64) *Memory {
	return &Memory{cluster: domain.NewCluster(schedulerName, seed)}
}

func (m *Memory) Snapshot() *domain.Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cluster.Clone()
}

func (m *Memory) Replace(cluster *domain.Cluster) error {
	if cluster == nil {
		return fmt.Errorf("%w: cluster cannot be nil", domain.ErrInvalidInput)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cluster = cluster.Clone()
	return nil
}

func (m *Memory) Reset(schedulerName string, seed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cluster = domain.NewCluster(schedulerName, seed)
}
