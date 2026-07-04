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

func TestServerInfoReportsStoreAndFormats(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", storeDir)

	csvPath := filepath.Join(t.TempDir(), "readings.csv")
	csv := "time,temp_c\n2026-01-01T00:00:00Z,12.4\n"
	if err := os.WriteFile(csvPath, []byte(csv), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	session := newConnectedSession(t)
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ingest_csv",
		Arguments: map[string]any{"path": csvPath},
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "server_info"})
	if err != nil {
		t.Fatalf("server_info: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error result: %+v", res)
	}

	data, _ := json.Marshal(res)
	s := string(data)
	for _, want := range []string{
		`"store_dir":"` + storeDir + `"`,
		`"storage_format":"parquet"`,
		`"supported_ingest_formats":["csv"]`,
		`catalog.json`,
		`readings.parquet`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("expected %s in %s", want, s)
		}
	}
}
