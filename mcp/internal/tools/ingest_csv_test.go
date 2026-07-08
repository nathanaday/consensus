package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestIngestCSVStoresDatasetAndSummarizes(t *testing.T) {
	ctx := context.Background()

	storeDir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", storeDir)

	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c,humidity\n2026-01-01T00:00:00Z,12.4,5.1\n2026-01-01T00:05:00Z,12.6,5.0\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "consensus", Version: "test"}, nil)
	Register(server)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath, "units": map[string]any{"temp_c": "celsius"}},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error result: %+v", res)
	}

	data, _ := json.Marshal(res)
	s := string(data)
	for _, want := range []string{
		`"group":"readings"`,
		`"dataset_id":"readings/temp_c"`,
		`"dataset_id":"readings/humidity"`,
		`"unit":"celsius"`,
		`"row_count":2`,
		`"timestamp_column":"time"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %s in %s", want, s)
		}
	}

	for _, p := range []string{"readings/temp_c.parquet", "readings/humidity.parquet", "catalog.json"} {
		if _, err := os.Stat(filepath.Join(storeDir, filepath.FromSlash(p))); err != nil {
			t.Errorf("%s not stored: %v", p, err)
		}
	}
}

func TestIngestCSVMissingFileErrors(t *testing.T) {
	ctx := context.Background()
	t.Setenv("CONSENSUS_STORE_DIR", t.TempDir())

	server := mcp.NewServer(&mcp.Implementation{Name: "consensus", Version: "test"}, nil)
	Register(server)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, _ := client.Connect(ctx, clientTransport, nil)
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": "/no/such/file.csv"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected error result for missing file")
	}
}
