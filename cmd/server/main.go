package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golubovicluka/CS320-PZ/internal/cluster"
	"github.com/golubovicluka/CS320-PZ/internal/engine"
	"github.com/golubovicluka/CS320-PZ/internal/logging"
	"github.com/golubovicluka/CS320-PZ/internal/scheduler"
	httptransport "github.com/golubovicluka/CS320-PZ/internal/transport/http"
)

type config struct {
	port                  string
	logLevel              string
	tickDuration          time.Duration
	heartbeatTimeoutTicks int64
	defaultScheduler      string
	maxNodes              int
	maxProcesses          int
	seed                  int64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configuration, err := loadConfig()
	if err != nil {
		return err
	}
	logger := logging.New(os.Stdout, configuration.logLevel)
	controller, err := cluster.New(cluster.Config{
		Scheduler:             configuration.defaultScheduler,
		Seed:                  configuration.seed,
		HeartbeatTimeoutTicks: configuration.heartbeatTimeoutTicks,
		MaxNodes:              configuration.maxNodes,
		MaxProcesses:          configuration.maxProcesses,
	})
	if err != nil {
		return err
	}
	simulationEngine, err := engine.New(controller, configuration.tickDuration)
	if err != nil {
		return err
	}
	defer simulationEngine.Close()
	handler, err := httptransport.NewHandler(controller, simulationEngine)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              ":" + configuration.port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("cluster simulator API started", "address", server.Addr, "scheduler", controller.SchedulerName())
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case serveErr := <-serverErrors:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info("server stopped")
	return nil
}

func loadConfig() (config, error) {
	configuration := config{
		port:             env("APP_PORT", "8080"),
		logLevel:         env("LOG_LEVEL", "info"),
		defaultScheduler: env("DEFAULT_SCHEDULER", scheduler.LeastLoadedName),
	}
	var err error
	if configuration.tickDuration, err = durationFromMilliseconds("TICK_DURATION_MS", 500); err != nil {
		return config{}, err
	}
	if configuration.heartbeatTimeoutTicks, err = int64FromEnv("HEARTBEAT_TIMEOUT_TICKS", 0); err != nil {
		return config{}, err
	}
	if configuration.maxNodes, err = intFromEnv("MAX_NODES", 100); err != nil {
		return config{}, err
	}
	if configuration.maxProcesses, err = intFromEnv("MAX_PROCESSES", 10_000); err != nil {
		return config{}, err
	}
	if configuration.seed, err = int64FromEnv("RANDOM_SEED", 42); err != nil {
		return config{}, err
	}
	return configuration, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func intFromEnv(name string, fallback int) (int, error) {
	value, err := strconv.Atoi(env(name, strconv.Itoa(fallback)))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return value, nil
}

func int64FromEnv(name string, fallback int64) (int64, error) {
	value, err := strconv.ParseInt(env(name, strconv.FormatInt(fallback, 10)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return value, nil
}

func durationFromMilliseconds(name string, fallback int64) (time.Duration, error) {
	value, err := int64FromEnv(name, fallback)
	if err != nil || value <= 0 {
		if err == nil {
			err = fmt.Errorf("must be positive")
		}
		return 0, fmt.Errorf("%s is invalid: %w", name, err)
	}
	return time.Duration(value) * time.Millisecond, nil
}
