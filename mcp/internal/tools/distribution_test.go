package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestDistributionReportsHistogramAndPercentiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := make([]dataset.Row, 100)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: float64(i)}
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", Unit: "celsius", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "distribution", Arguments: map[string]any{"id": "readings", "bins": 10},
	})
	if err != nil {
		t.Fatalf("distribution: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"row_count":100`, `"p50":49.5`, `"p95":94.05`, `"min":0`, `"max":99`,
		`"unit":"celsius"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
	if strings.Count(s, `"lower"`) != 10 {
		t.Errorf("want 10 histogram bins, got %s", s)
	}
}

func TestDistributionRejectsOverCapBins(t *testing.T) {
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
		Name: "distribution", Arguments: map[string]any{"id": "readings", "bins": 100},
	})
	if err != nil {
		t.Fatalf("distribution: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want error for over-cap bins, got %+v", res)
	}
}

func TestDistributionConstantSeriesCaveat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := []dataset.Row{
		{Timestamp: 0, Value: 7}, {Timestamp: 60000, Value: 7}, {Timestamp: 120000, Value: 7},
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "distribution", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("distribution: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "all values are identical") {
		t.Errorf("expected constant-series caveat in %s", s)
	}
}
