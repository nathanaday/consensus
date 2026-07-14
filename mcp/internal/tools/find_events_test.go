package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func seedEventRows(t *testing.T, dir string) {
	t.Helper()
	values := []float64{5, 6, 12, 15, 11, 6, 5, 6, 20, 5}
	rows := make([]dataset.Row, len(values))
	for i, v := range values {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: v}
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", Unit: "celsius", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestFindEventsAboveThreshold(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedEventRows(t, dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "find_events", Arguments: map[string]any{"id": "readings", "condition": "above", "threshold": 10},
	})
	if err != nil {
		t.Fatalf("find_events: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"total_events":2`, `"points_matching":4`, `"pct_points":40`,
		`"duration_seconds":120`, `"peak_value":15`, `"time_in_events_seconds":120`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestFindEventsHonorsLimit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedEventRows(t, dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "find_events", Arguments: map[string]any{"id": "readings", "condition": "above", "threshold": 10, "limit": 1},
	})
	if err != nil {
		t.Fatalf("find_events: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, `"total_events":2`) {
		t.Errorf("total_events should stay the full count, got %s", s)
	}
	if strings.Count(s, `"direction"`) != 1 {
		t.Errorf("want exactly 1 returned event, got %s", s)
	}
}

func TestFindEventsMissingBoundsErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedEventRows(t, dir)
	session := newConnectedSession(t)
	for _, args := range []map[string]any{
		{"id": "readings", "condition": "above"},
		{"id": "readings", "condition": "between", "lower": 1},
		{"id": "readings", "condition": "sideways", "threshold": 1},
	} {
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "find_events", Arguments: args,
		})
		if err != nil {
			t.Fatalf("find_events: %v", err)
		}
		if !res.IsError {
			t.Errorf("want error for %v, got %+v", args, res)
		}
	}
}
