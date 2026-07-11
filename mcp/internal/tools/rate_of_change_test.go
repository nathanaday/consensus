package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestRateOfChangeReportsExtremes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv",
		SourceColumn: "temp_c", Unit: "celsius",
		RowCount: 3,
		Rows: []dataset.Row{
			{Timestamp: 0, Value: 0},
			{Timestamp: 1000, Value: 10},
			{Timestamp: 3000, Value: 5},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "rate_of_change", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("rate_of_change: %v", err)
	}
	if res.IsError {
		t.Fatalf("rate_of_change returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"row_count":3`,
		`"max_rise":{"rate":10,"timestamp":"1970-01-01T00:00:01Z"}`,
		`"max_fall":{"rate":-2.5,"timestamp":"1970-01-01T00:00:03Z"}`,
		`"mean_abs_rate":6.25`,
		`"unit":"celsius/second"`,
		`"median_sample_interval_seconds":1.5`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestRateOfChangeNeedsTwoRows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "single", Origin: "csv",
		SourceColumn: "v", RowCount: 1,
		Rows: []dataset.Row{{Timestamp: 0, Value: 1}},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "rate_of_change", Arguments: map[string]any{"id": "single"},
	})
	if res == nil || !res.IsError {
		t.Error("one-row dataset should be a tool error")
	}
}
