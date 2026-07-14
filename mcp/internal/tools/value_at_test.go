package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func seedValueAtRows(t *testing.T, dir string) {
	t.Helper()
	rows := []dataset.Row{
		{Timestamp: 0, Value: 10},
		{Timestamp: 100000, Value: 20},
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", Unit: "celsius", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestValueAtInterpolates(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedValueAtRows(t, dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "value_at", Arguments: map[string]any{"id": "readings", "at": "1970-01-01T00:00:25Z"},
	})
	if err != nil {
		t.Fatalf("value_at: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"interpolated":12.5`, `"offset_seconds":-25`, `"value":10`, `"unit":"celsius"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestValueAtOutsideSpanCaveat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedValueAtRows(t, dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "value_at", Arguments: map[string]any{"id": "readings", "at": "1970-01-01T01:00:00Z"},
	})
	if err != nil {
		t.Fatalf("value_at: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "outside the dataset") {
		t.Errorf("expected outside-span caveat in %s", s)
	}
	if strings.Contains(s, `"interpolated"`) {
		t.Errorf("interpolated should be omitted outside the span, got %s", s)
	}
}

func TestValueAtMaxDistanceErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedValueAtRows(t, dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "value_at", Arguments: map[string]any{
			"id": "readings", "at": "1970-01-01T01:00:00Z", "max_distance": "1m",
		},
	})
	if err != nil {
		t.Fatalf("value_at: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want error when the nearest sample is too far, got %+v", res)
	}
}

func TestValueAtRejectsBadTimestamp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedValueAtRows(t, dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "value_at", Arguments: map[string]any{"id": "readings", "at": "3pm yesterday"},
	})
	if err != nil {
		t.Fatalf("value_at: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want error for a malformed timestamp, got %+v", res)
	}
}
