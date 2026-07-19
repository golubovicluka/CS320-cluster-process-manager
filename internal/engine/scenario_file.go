package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/golubovicluka/CS320-PZ/internal/domain"
)

func DecodeScenario(reader io.Reader) (domain.Scenario, error) {
	var scenario domain.Scenario
	decoder := json.NewDecoder(io.LimitReader(reader, 4<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&scenario); err != nil {
		return domain.Scenario{}, fmt.Errorf("decode scenario: %w", err)
	}
	if err := scenario.Validate(); err != nil {
		return domain.Scenario{}, err
	}
	return scenario, nil
}

func LoadScenarioFile(path string) (domain.Scenario, error) {
	file, err := os.Open(path)
	if err != nil {
		return domain.Scenario{}, fmt.Errorf("open scenario: %w", err)
	}
	defer file.Close()
	return DecodeScenario(file)
}
