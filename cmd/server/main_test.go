package main

import (
	"testing"
	"time"
)

func TestLoadConfigDefaultsAndOverrides(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("TICK_DURATION_MS", "25")
	t.Setenv("MAX_NODES", "12")
	configuration, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.port != "9090" || configuration.tickDuration != 25*time.Millisecond || configuration.maxNodes != 12 {
		t.Fatalf("unexpected config: %+v", configuration)
	}
}

func TestLoadConfigRejectsInvalidNumber(t *testing.T) {
	t.Setenv("MAX_PROCESSES", "many")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected invalid configuration")
	}
}
