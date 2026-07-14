package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestDataQualityReportsGap(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 2},
		{Timestamp: 120000, Value: 3},
		{Timestamp: 120000 + 30*60000, Value: 4},
		{Timestamp: 120000 + 31*60000, Value: 5},
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", Unit: "celsius", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "data_quality", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("data_quality: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"total_gaps":1`, `"median_interval_seconds":60`, `"duration_seconds":1800`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestDataQualityFewPointsCaveat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "data_quality", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("data_quality: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "fewer than 3 points") {
		t.Errorf("expected few-points caveat in %s", s)
	}
}
