package tools

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestFindLagFindsKnownShift(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	// b is a copy of a shifted 4 minutes later, both at 1-minute spacing.
	value := func(i int) float64 { return math.Sin(float64(i) / 5) }
	var a, b []dataset.Row
	for i := 0; i < 100; i++ {
		ts := int64(i) * 60000
		a = append(a, dataset.Row{Timestamp: ts, Value: value(i)})
		b = append(b, dataset.Row{Timestamp: ts, Value: value(i - 4)})
	}
	for name, rows := range map[string][]dataset.Row{"leader": a, "follower": b} {
		if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
			NameOverride: name, Origin: "csv", RowCount: len(rows), Rows: rows,
		}); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "find_lag", Arguments: map[string]any{
			"id_a": "leader", "id_b": "follower", "bucket": "1m", "max_lag": "10m",
		},
	})
	if err != nil {
		t.Fatalf("find_lag: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"best_lag_seconds":240`, `"pearson_at_best_lag":1`, `"max_lag_scanned_seconds":600`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestFindLagAutoBucketAndDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	value := func(i int) float64 { return math.Sin(float64(i) / 5) }
	var a []dataset.Row
	for i := 0; i < 100; i++ {
		a = append(a, dataset.Row{Timestamp: int64(i) * 60000, Value: value(i)})
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: len(a), Rows: a,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "find_lag", Arguments: map[string]any{"id_a": "readings", "id_b": "readings"},
	})
	if err != nil {
		t.Fatalf("find_lag: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, `"best_lag_seconds":0`) {
		t.Errorf("a series against itself should peak at lag 0, got %s", s)
	}
}

func TestFindLagRejectsNonOverlapping(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	early := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}}
	late := []dataset.Row{{Timestamp: 3600000000, Value: 1}, {Timestamp: 3600060000, Value: 2}}
	for name, rows := range map[string][]dataset.Row{"early": early, "late": late} {
		if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
			NameOverride: name, Origin: "csv", RowCount: len(rows), Rows: rows,
		}); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "find_lag", Arguments: map[string]any{"id_a": "early", "id_b": "late"},
	})
	if err != nil {
		t.Fatalf("find_lag: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want error for non-overlapping datasets, got %+v", res)
	}
}
