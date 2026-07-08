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

func newConnectedSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
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
	t.Cleanup(func() { session.Close() })
	return session
}

func TestListDatasetsReturnsCatalogMetadata(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", storeDir)

	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c,humidity\n2026-01-01T00:00:00Z,12.4,5.1\n2026-01-01T00:05:00Z,12.6,5.0\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	session := newConnectedSession(t)
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath, "units": map[string]any{"temp_c": "celsius"}},
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_datasets"})
	if err != nil {
		t.Fatalf("list_datasets: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error result: %+v", res)
	}

	data, _ := json.Marshal(res)
	s := string(data)
	for _, want := range []string{
		`"id":"readings/temp_c"`,
		`"id":"readings/humidity"`,
		`"source_column":"temp_c"`,
		`"unit":"celsius"`,
		`"row_count":2`,
		`"start":"2026-01-01T00:00:00Z"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %s in %s", want, s)
		}
	}
	if strings.Contains(s, `"size_bytes":0`) {
		t.Errorf("expected a non-zero size_bytes in %s", s)
	}
}

func TestListDatasetsEmptyStore(t *testing.T) {
	ctx := context.Background()
	t.Setenv("CONSENSUS_STORE_DIR", t.TempDir())

	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_datasets"})
	if err != nil {
		t.Fatalf("list_datasets: %v", err)
	}
	data, _ := json.Marshal(res)
	if !strings.Contains(string(data), `"datasets":[]`) {
		t.Errorf("expected empty datasets array, got %s", data)
	}
}
