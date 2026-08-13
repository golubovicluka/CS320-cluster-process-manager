package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type client struct {
	baseURL string
	http    *http.Client
	output  io.Writer
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, output io.Writer) error {
	global := flag.NewFlagSet("clusterctl", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	server := global.String("server", "http://localhost:8080", "cluster API URL")
	if err := global.Parse(arguments); err != nil {
		return err
	}
	args := global.Args()
	if len(args) < 2 {
		return usageError()
	}
	current := &client{
		baseURL: strings.TrimRight(*server, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
		output:  output,
	}
	switch args[0] {
	case "node":
		return current.node(args[1], args[2:])
	case "process":
		return current.process(args[1], args[2:])
	case "scheduler":
		return current.scheduler(args[1], args[2:])
	case "simulation":
		return current.simulation(args[1], args[2:])
	case "report":
		return current.report(args[1], args[2:])
	case "events":
		return current.request(http.MethodGet, "/api/v1/events", nil, "")
	case "metrics":
		return current.request(http.MethodGet, "/api/v1/metrics", nil, "")
	default:
		return usageError()
	}
}

func (c *client) node(action string, args []string) error {
	switch action {
	case "add":
		flags := newFlags("node add")
		id := flags.String("id", "", "node id")
		name := flags.String("name", "", "node name")
		cpu := flags.Int("cpu", 0, "CPU capacity")
		memory := flags.Int("memory", 0, "memory capacity in MB")
		if err := flags.Parse(args); err != nil {
			return err
		}
		if *name == "" {
			*name = *id
		}
		return c.request(http.MethodPost, "/api/v1/nodes", map[string]any{
			"id": *id, "name": *name, "cpuCapacity": *cpu, "memoryCapacityMB": *memory,
		}, "")
	case "list":
		return c.request(http.MethodGet, "/api/v1/nodes", nil, "")
	case "fail", "recover", "heartbeat":
		id, err := requiredArgument(args, "node id")
		if err != nil {
			return err
		}
		return c.request(http.MethodPost, "/api/v1/nodes/"+id+"/"+action, nil, "")
	case "remove":
		id, err := requiredArgument(args, "node id")
		if err != nil {
			return err
		}
		return c.request(http.MethodDelete, "/api/v1/nodes/"+id, nil, "")
	case "status":
		flags := newFlags("node status")
		status := flags.String("status", "", "ONLINE, DRAINING, OFFLINE, or FAILED")
		if err := flags.Parse(args); err != nil {
			return err
		}
		id, err := requiredArgument(flags.Args(), "node id")
		if err != nil {
			return err
		}
		return c.request(http.MethodPatch, "/api/v1/nodes/"+id+"/status", map[string]string{"status": strings.ToUpper(*status)}, "")
	default:
		return usageError()
	}
}

func (c *client) process(action string, args []string) error {
	switch action {
	case "submit":
		flags := newFlags("process submit")
		id := flags.String("id", "", "process id")
		name := flags.String("name", "", "process name")
		cpu := flags.Int("cpu", 0, "CPU request")
		memory := flags.Int("memory", 0, "memory request in MB")
		ticks := flags.Int("ticks", 0, "duration in ticks")
		priority := flags.Int("priority", 0, "priority; larger values run first")
		restart := flags.String("restart", "NEVER", "NEVER or ON_FAILURE")
		maxRestarts := flags.Int("max-restarts", 0, "maximum restarts")
		if err := flags.Parse(args); err != nil {
			return err
		}
		if *name == "" {
			*name = *id
		}
		return c.request(http.MethodPost, "/api/v1/processes", map[string]any{
			"id": *id, "name": *name, "cpuRequest": *cpu, "memoryRequestMB": *memory,
			"totalTicks": *ticks, "priority": *priority,
			"restartPolicy": strings.ToUpper(*restart), "maxRestarts": *maxRestarts,
		}, "")
	case "list":
		return c.request(http.MethodGet, "/api/v1/processes", nil, "")
	case "pause", "resume", "wait", "wake", "kill", "fail":
		id, err := requiredArgument(args, "process id")
		if err != nil {
			return err
		}
		return c.request(http.MethodPost, "/api/v1/processes/"+id+"/"+action, nil, "")
	default:
		return usageError()
	}
}

func (c *client) scheduler(action string, args []string) error {
	switch action {
	case "get":
		return c.request(http.MethodGet, "/api/v1/scheduler", nil, "")
	case "set":
		name, err := requiredArgument(args, "scheduler name")
		if err != nil {
			return err
		}
		return c.request(http.MethodPut, "/api/v1/scheduler", map[string]string{"name": name}, "")
	default:
		return usageError()
	}
}

func (c *client) simulation(action string, args []string) error {
	switch action {
	case "step":
		flags := newFlags("simulation step")
		ticks := flags.Int("ticks", 1, "number of ticks")
		if err := flags.Parse(args); err != nil {
			return err
		}
		return c.request(http.MethodPost, "/api/v1/simulation/step", map[string]int{"ticks": *ticks}, "")
	case "start", "pause", "reset":
		return c.request(http.MethodPost, "/api/v1/simulation/"+action, nil, "")
	case "status":
		return c.request(http.MethodGet, "/api/v1/simulation/status", nil, "")
	case "load":
		path, err := requiredArgument(args, "scenario path")
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var scenario any
		if err := json.Unmarshal(data, &scenario); err != nil {
			return err
		}
		return c.request(http.MethodPost, "/api/v1/simulation/scenarios", scenario, "")
	default:
		return usageError()
	}
}

func (c *client) report(action string, args []string) error {
	switch action {
	case "show":
		return c.request(http.MethodGet, "/api/v1/reports/summary", nil, "")
	case "export":
		flags := newFlags("report export")
		format := flags.String("format", "json", "json or csv")
		output := flags.String("output", "", "output file")
		if err := flags.Parse(args); err != nil {
			return err
		}
		return c.request(http.MethodGet, "/api/v1/reports/export?format="+*format, nil, *output)
	default:
		return usageError()
	}
}

func (c *client) request(method, path string, body any, outputPath string) error {
	var requestBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, c.baseURL+path, requestBody)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("API returned %s: %s", response.Status, strings.TrimSpace(string(data)))
	}
	if outputPath != "" {
		return os.WriteFile(outputPath, data, 0o644)
	}
	if len(data) == 0 {
		fmt.Fprintln(c.output, "ok")
		return nil
	}
	if strings.Contains(response.Header.Get("Content-Type"), "json") {
		var formatted bytes.Buffer
		if err := json.Indent(&formatted, data, "", "  "); err == nil {
			data = formatted.Bytes()
		}
	}
	_, err = fmt.Fprintln(c.output, string(data))
	return err
}

func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func requiredArgument(args []string, name string) (string, error) {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return args[0], nil
}

func usageError() error {
	return errors.New("usage: clusterctl [--server URL] <node|process|scheduler|simulation|report|events|metrics> <action> [options]")
}
