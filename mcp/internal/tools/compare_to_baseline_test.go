package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// seedBaselineDataset stores 20 quiet points (~30) then 3 spikes (~110),
// all one minute apart, so a subject window over the spikes finds an episode
// and the default baseline is the quiet period before it.
func seedBaselineDataset(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := make([]dataset.Row, 0, 23)
	for i := 0; i < 20; i++ {
		v := 29.0
		if i%2 == 1 {
			v = 31
		}
		rows = append(rows, dataset.Row{Timestamp: int64(i) * 60000, Value: v})
	}
	rows = append(rows,
		dataset.Row{Timestamp: 20 * 60000, Value: 108},
		dataset.Row{Timestamp: 21 * 60000, Value: 109},
		dataset.Row{Timestamp: 22 * 60000, Value: 112},
	)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", SourceColumn: "temp_c", Unit: "celsius",
		RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestCompareToBaselineFindsEpisode(t *testing.T) {
	ctx := context.Background()
	seedBaselineDataset(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "compare_to_baseline", Arguments: map[string]any{
			"id":    "readings",
			"start": "1970-01-01T00:20:00Z", // first spike
		},
	})
	if err != nil {
		t.Fatalf("compare_to_baseline: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"total_episodes":1`, `"direction":"above"`, `"peak_value":112`, `"points_outside":3`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestCompareToBaselineEmptyBaselineErrors(t *testing.T) {
	ctx := context.Background()
	seedBaselineDataset(t)
	session := newConnectedSession(t)
	// subject starts at the very first row -> no history before it.
	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "compare_to_baseline", Arguments: map[string]any{
			"id": "readings", "start": "1970-01-01T00:00:00Z",
		},
	})
	if res == nil || !res.IsError {
		t.Error("empty default baseline should be a tool error")
	}
}
