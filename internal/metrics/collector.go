package metrics

import (
	"fmt"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

type SnapshotProvider interface {
	Snapshot() *domain.Cluster
}

type Collector struct {
	provider SnapshotProvider
}

func NewCollector(provider SnapshotProvider) (*Collector, error) {
	if provider == nil {
		return nil, fmt.Errorf("%w: snapshot provider cannot be nil", domain.ErrInvalidInput)
	}
	return &Collector{provider: provider}, nil
}

func (c *Collector) Current() Report {
	return Build(c.provider.Snapshot())
}
