package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// seedTwoPeriods stores one dataset whose first half averages ~10 and second
// half ~20, so a per-side windowed compare shows a +100% mean change.
func seedTwoPeriods(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	var rows []dataset.Row
	for i := 0; i < 10; i++ {
		rows = append(rows, dataset.Row{Timestamp: int64(i) * 60000, Value: 10})
	}
	for i := 10; i < 20; i++ {
		rows = append(rows, dataset.Row{Timestamp: int64(i) * 60000, Value: 20})
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", Unit: "celsius", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestCompareDatasetsPeriods(t *testing.T) {
	ctx := context.Background()
	seedTwoPeriods(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "compare_datasets", Arguments: map[string]any{
			"id_a": "readings", "end_a": "1970-01-01T00:09:00Z",
			"id_b": "readings", "start_b": "1970-01-01T00:10:00Z",
		},
	})
	if err != nil {
		t.Fatalf("compare_datasets: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"mean_difference":10`, `"mean_pct_change":100`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestCompareDatasetsUnitMismatchCaveat(t *testing.T) {
	ctx := context.Background()
	seedTwoChannels(t) // temp (celsius) and pressure (kpa)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "compare_datasets", Arguments: map[string]any{"id_a": "temp", "id_b": "pressure"},
	})
	if err != nil {
		t.Fatalf("compare_datasets: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "units differ") {
		t.Errorf("expected unit-mismatch caveat in %s", s)
	}
}
