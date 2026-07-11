package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// seedAnalysisDataset stores values {1,2,3,4,100} one minute apart, unit
// celsius. Shared by the analysis tool tests.
func seedAnalysisDataset(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv",
		SourceColumn: "temp_c", Unit: "celsius",
		RowCount: 5,
		Rows: []dataset.Row{
			{Timestamp: 0, Value: 1},
			{Timestamp: 60000, Value: 2},
			{Timestamp: 120000, Value: 3},
			{Timestamp: 180000, Value: 4},
			{Timestamp: 240000, Value: 100},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestSummaryStatsWholeDataset(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("summary_stats: %v", err)
	}
	if res.IsError {
		t.Fatalf("summary_stats returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"row_count":5`,
		`"mean":22`,
		`"median":3`,
		`"min":{"timestamp":"1970-01-01T00:00:00Z","value":1}`,
		`"max":{"timestamp":"1970-01-01T00:04:00Z","value":100}`,
		`"analyzed_range":{"end":"1970-01-01T00:04:00Z","start":"1970-01-01T00:00:00Z"}`,
		`"unit":"celsius"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestSummaryStatsWindowed(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{
			"id":    "readings",
			"start": "1970-01-01T00:01:00Z",
			"end":   "1970-01-01T00:03:00Z",
		},
	})
	if err != nil {
		t.Fatalf("summary_stats: %v", err)
	}
	if res.IsError {
		t.Fatalf("summary_stats returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"row_count":3`, `"mean":3`, `"median":3`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestSummaryStatsErrors(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)

	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{"id": "readings", "start": "yesterday"},
	})
	if res == nil || !res.IsError {
		t.Error("malformed start should be a tool error")
	}

	res, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{
			"id": "readings", "start": "1980-01-01T00:00:00Z",
		},
	})
	if res == nil || !res.IsError {
		t.Error("empty window should be a tool error")
	}

	res, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "summary_stats", Arguments: map[string]any{"id": "nope"},
	})
	if res == nil || !res.IsError {
		t.Error("unknown dataset should be a tool error")
	}
}
