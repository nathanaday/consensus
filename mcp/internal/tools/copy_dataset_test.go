package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestCopyDatasetCreatesChild(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv",
		SourceColumn: "temp_c", Unit: "celsius",
		RowCount: 1,
		Rows:     []dataset.Row{{Timestamp: 1, Value: 1.5}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "copy_dataset", Arguments: map[string]any{"id": "readings"}})
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if res.IsError {
		t.Fatalf("copy returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, `"id":"readings-2"`) {
		t.Errorf("expected new id readings-2 in %s", s)
	}
	if !strings.Contains(s, `"parent":{"id":"readings","origin":"csv"}`) {
		t.Errorf("expected parent edge to readings in %s", s)
	}
	if !strings.Contains(s, `"children":[]`) {
		t.Errorf("expected empty children in %s", s)
	}
	if _, err := os.Stat(filepath.Join(dir, "readings-2.parquet")); err != nil {
		t.Errorf("copied parquet not written: %v", err)
	}
}
