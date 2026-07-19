package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNodeAddCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/nodes" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"n1"}`))
	}))
	defer server.Close()
	var output bytes.Buffer
	err := run([]string{"--server", server.URL, "node", "add", "--id", "n1", "--cpu", "2", "--memory", "1024"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"id": "n1"`) {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestCommandReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()
	if err := run([]string{"--server", server.URL, "process", "list"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected API error")
	}
}
