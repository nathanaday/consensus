package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func seedTwoChannels(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	var ra, rb []dataset.Row
	for i := 0; i < 12; i++ {
		ts := int64(i) * 60000
		ra = append(ra, dataset.Row{Timestamp: ts, Value: float64(i)})
		rb = append(rb, dataset.Row{Timestamp: ts, Value: 2*float64(i) + 3})
	}
	for _, d := range []struct {
		id   string
		rows []dataset.Row
		unit string
	}{{"temp", ra, "celsius"}, {"pressure", rb, "kpa"}} {
		if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
			NameOverride: d.id, Origin: "csv", Unit: d.unit, RowCount: len(d.rows), Rows: d.rows,
		}); err != nil {
			t.Fatalf("seed %s: %v", d.id, err)
		}
	}
}

func TestCorrelateStrongPositive(t *testing.T) {
	ctx := context.Background()
	seedTwoChannels(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "correlate", Arguments: map[string]any{"id_a": "temp", "id_b": "pressure", "bucket": "1m"},
	})
	if err != nil {
		t.Fatalf("correlate: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"aligned_samples":12`, `"pearson":1`, `"spearman":1`, `"unit_a":"celsius"`, `"unit_b":"kpa"`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestCorrelateUnknownDatasetErrors(t *testing.T) {
	ctx := context.Background()
	seedTwoChannels(t)
	session := newConnectedSession(t)
	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "correlate", Arguments: map[string]any{"id_a": "temp", "id_b": "nope"},
	})
	if res == nil || !res.IsError {
		t.Error("unknown id_b should be a tool error")
	}
}
