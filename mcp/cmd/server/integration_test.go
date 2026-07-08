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
	if !strings.Contains(string(data), `"dataset_id":"readings/temp_c"`) {
		t.Fatalf("expected dataset_id readings/temp_c, got %s", data)
	}
	if _, err := os.Stat(filepath.Join(storeDir, "readings", "temp_c.parquet")); err != nil {
		t.Fatalf("parquet not stored: %v", err)
	}
}

func TestServerIntrospectionOverStdio(t *testing.T) {
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

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	}); err != nil {
		t.Fatalf("ingest_csv: %v", err)
	}

	listRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_datasets"})
	if err != nil {
		t.Fatalf("list_datasets: %v", err)
	}
	if listRes.IsError {
		t.Fatalf("list_datasets error result: %+v", listRes)
	}
	listData, _ := json.Marshal(listRes)
	if !strings.Contains(string(listData), `"id":"readings/temp_c"`) {
		t.Fatalf("expected dataset readings/temp_c, got %s", listData)
	}

	infoRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "server_info"})
	if err != nil {
		t.Fatalf("server_info: %v", err)
	}
	if infoRes.IsError {
		t.Fatalf("server_info error result: %+v", infoRes)
	}
	infoData, _ := json.Marshal(infoRes)
	if !strings.Contains(string(infoData), `"storage_format":"parquet"`) {
		t.Fatalf("expected storage_format parquet, got %s", infoData)
	}
}

func TestServerLineageOverStdio(t *testing.T) {
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

	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	}); err != nil {
		t.Fatalf("ingest_csv: %v", err)
	}

	copyRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "copy_dataset",
		Arguments: map[string]any{"id": "readings/temp_c"},
	})
	if err != nil {
		t.Fatalf("copy_dataset: %v", err)
	}
	if copyRes.IsError {
		t.Fatalf("copy_dataset error result: %+v", copyRes)
	}
	if !strings.Contains(string(mustMarshal(copyRes)), `"id":"readings/temp_c-2"`) {
		t.Fatalf("expected copy id readings/temp_c-2, got %s", mustMarshal(copyRes))
	}

	descRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "describe_dataset",
		Arguments: map[string]any{"id": "readings/temp_c-2"},
	})
	if err != nil {
		t.Fatalf("describe_dataset: %v", err)
	}
	if !strings.Contains(string(mustMarshal(descRes)), `"parent":{"id":"readings/temp_c","origin":"csv"}`) {
		t.Fatalf("expected parent edge, got %s", mustMarshal(descRes))
	}

	graphRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "dataset_graph",
		Arguments: map[string]any{"format": "mermaid"},
	})
	if err != nil {
		t.Fatalf("dataset_graph: %v", err)
	}
	// Assert on the copy edge's label + target, which contain no JSON-escaped
	// characters (the "-->" arrow's ">" is HTML-escaped on the wire, so match
	// the unescaped part of the edge instead).
	if !strings.Contains(string(mustMarshal(graphRes)), `|copy| readings_temp_c_2`) {
		t.Fatalf("expected copy edge in mermaid, got %s", mustMarshal(graphRes))
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
