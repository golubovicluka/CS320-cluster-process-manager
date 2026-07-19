package store

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func WriteSnapshot(w io.Writer, cluster *domain.Cluster) error {
	if cluster == nil {
		return fmt.Errorf("%w: cluster cannot be nil", domain.ErrInvalidInput)
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cluster)
}

func ReadSnapshot(r io.Reader) (*domain.Cluster, error) {
	var cluster domain.Cluster
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cluster); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	if cluster.Nodes == nil {
		cluster.Nodes = make(map[string]*domain.Node)
	}
	if cluster.Processes == nil {
		cluster.Processes = make(map[string]*domain.Process)
	}
	return cluster.Clone(), nil
}
