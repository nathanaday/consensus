package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerSpeaksMCPOverStdio(t *testing.T) {
	ctx := context.Background()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	transport := &mcp.CommandTransport{Command: exec.Command("go", "run", ".")}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "greet",
		Arguments: map[string]any{"name": "Grace"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned an error result: %+v", res)
	}

	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if !strings.Contains(string(data), "Hi Grace") {
		t.Fatalf("expected result to contain %q, got %s", "Hi Grace", data)
	}
}

func TestServerIngestsCSVOverStdio(t *testing.T) {
	ctx := context.Background()

	storeDir := t.TempDir()
	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c\n2026-01-01T00:00:00Z,12.4\n2026-01-01T00:05:00Z,12.6\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Env = append(os.Environ(), "CONSENSUS_STORE_DIR="+storeDir)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to server subprocess: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error result: %+v", res)
	}

	data, _ := json.Marshal(res)
	if !strings.Contains(string(data), `"dataset_id":"readings"`) {
		t.Fatalf("expected dataset_id readings, got %s", data)
	}
	if _, err := os.Stat(filepath.Join(storeDir, "readings.parquet")); err != nil {
		t.Fatalf("parquet not stored: %v", err)
	}
}
