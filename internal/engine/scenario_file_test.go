package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeScenarioRejectsUnknownFields(t *testing.T) {
	input := `{"name":"test","seed":1,"scheduler":"round-robin","maxTicks":1,"nodes":[{"id":"n1","name":"n","cpuCapacity":1,"memoryCapacityMB":1}],"processes":[],"surprise":true}`
	if _, err := DecodeScenario(strings.NewReader(input)); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestDecodeScenarioRejectsRemovedTimeQuantum(t *testing.T) {
	input := `{"name":"test","seed":1,"scheduler":"round-robin","maxTicks":1,"nodes":[{"id":"n1","name":"n","cpuCapacity":1,"memoryCapacityMB":1}],"processes":[{"id":"p1","name":"p","cpuRequest":1,"memoryRequestMB":1,"totalTicks":1,"timeQuantum":2}]}`
	if _, err := DecodeScenario(strings.NewReader(input)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected removed timeQuantum to be rejected, got %v", err)
	}
}

func TestRepositoryScenariosAreValid(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "scenarios", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 4 {
		t.Fatalf("got %d scenarios, want at least 4", len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			file, openErr := os.Open(path)
			if openErr != nil {
				t.Fatal(openErr)
			}
			defer file.Close()
			if _, decodeErr := DecodeScenario(file); decodeErr != nil {
				t.Fatal(decodeErr)
			}
		})
	}
}
