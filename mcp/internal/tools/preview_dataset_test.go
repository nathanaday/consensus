package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestPreviewDatasetRespectsLimit(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv",
		Series:   []dataset.Series{{ID: "temp_c"}},
		RowCount: 3,
		Rows: []dataset.Row{
			{Timestamp: 1, SeriesID: "temp_c", Value: 1},
			{Timestamp: 2, SeriesID: "temp_c", Value: 2},
			{Timestamp: 3, SeriesID: "temp_c", Value: 3},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	session := newConnectedSession(t)

	// Default limit returns all 3 rows.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "preview_dataset", Arguments: map[string]any{"id": "readings"}})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, `"returned":3`) || !strings.Contains(s, `"row_count":3`) {
		t.Errorf("expected returned 3 / row_count 3 in %s", s)
	}

	// Explicit limit caps the rows returned.
	res2, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "preview_dataset", Arguments: map[string]any{"id": "readings", "limit": 1}})
	if err != nil {
		t.Fatalf("preview limit: %v", err)
	}
	s2 := string(mustJSON(res2))
	if !strings.Contains(s2, `"returned":1`) {
		t.Errorf("expected returned 1 in %s", s2)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
