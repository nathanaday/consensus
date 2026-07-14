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

func seedCorrelatePair(t *testing.T, aVals, bVals []float64, bUnit string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	mk := func(id, unit string, vals []float64) {
		rows := make([]dataset.Row, len(vals))
		for i, v := range vals {
			rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: v}
		}
		if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
			NameOverride: id, Origin: "csv", Unit: unit, RowCount: len(rows), Rows: rows,
		}); err != nil {
			t.Fatalf("seed %s: %v", id, err)
		}
	}
	mk("temp", "celsius", aVals)
	mk("pressure", bUnit, bVals)
}

func TestCorrelateLowSampleCaveatNotConstant(t *testing.T) {
	ctx := context.Background()
	// 5 aligned, non-constant pair: expect the low-sample caveat but NOT the
	// constant-series caveat (which previously fired incorrectly for <2, and
	// must not fire here regardless).
	seedCorrelatePair(t, []float64{1, 2, 3, 4, 5}, []float64{2, 4, 6, 8, 10}, "kpa")
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
	if !strings.Contains(s, "aligned samples") {
		t.Errorf("expected low-sample caveat in %s", s)
	}
	if strings.Contains(s, "constant series") {
		t.Errorf("constant-series caveat must not appear for a non-constant pair: %s", s)
	}
}

func TestCorrelateConstantSeriesCaveat(t *testing.T) {
	ctx := context.Background()
	// 12 aligned, b constant: expect the constant-series caveat and pearson omitted.
	seedCorrelatePair(t,
		[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		[]float64{7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7}, "kpa")
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
	if !strings.Contains(s, "constant series") {
		t.Errorf("expected constant-series caveat in %s", s)
	}
	if strings.Contains(s, `"pearson":`) {
		t.Errorf("pearson must be omitted for a constant series: %s", s)
	}
}
