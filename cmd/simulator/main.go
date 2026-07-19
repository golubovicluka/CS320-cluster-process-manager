package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golubovicluka/CS320-PZ/internal/cluster"
	"github.com/golubovicluka/CS320-PZ/internal/engine"
	"github.com/golubovicluka/CS320-PZ/internal/metrics"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("simulator", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	scenarioPath := flags.String("scenario", "scenarios/balanced.json", "path to scenario JSON")
	format := flags.String("format", "json", "report format: json or csv")
	outputPath := flags.String("output", "", "optional report output path")
	schedulerOverride := flags.String("scheduler", "", "optional scheduler override")
	seedOverride := flags.Int64("seed", 0, "optional non-zero seed override")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	scenario, err := engine.LoadScenarioFile(*scenarioPath)
	if err != nil {
		return err
	}
	if *schedulerOverride != "" {
		scenario.Scheduler = *schedulerOverride
	}
	if *seedOverride != 0 {
		scenario.Seed = *seedOverride
	}
	controller, err := cluster.New(cluster.Config{Scheduler: scenario.Scheduler, Seed: scenario.Seed})
	if err != nil {
		return err
	}
	simulationEngine, err := engine.New(controller, time.Duration(scenario.TickDurationMS)*time.Millisecond)
	if err != nil {
		return err
	}
	defer simulationEngine.Close()
	result, err := simulationEngine.RunScenario(scenario)
	if err != nil {
		return err
	}
	report := metrics.Build(result)

	output := stdout
	var file *os.File
	if *outputPath != "" {
		file, err = os.Create(*outputPath)
		if err != nil {
			return fmt.Errorf("create report: %w", err)
		}
		defer file.Close()
		output = file
	}
	switch *format {
	case "json":
		return metrics.WriteJSON(output, report)
	case "csv":
		return metrics.WriteCSV(output, report)
	default:
		return fmt.Errorf("format must be json or csv")
	}
}
