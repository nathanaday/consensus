package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestIntegrateConstantChannel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	// A constant 2.0 kW for one hour: 7200 kW-seconds, 2 kWh.
	rows := make([]dataset.Row, 61)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: 2}
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "power", Origin: "csv", Unit: "kW", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "integrate", Arguments: map[string]any{"id": "power"},
	})
	if err != nil {
		t.Fatalf("integrate: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"integral_value_seconds":7200`, `"integral_value_hours":2`,
		`"time_weighted_mean":2`, `"duration_seconds":3600`, `"unit":"kW"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestIntegrateGapCaveat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 1},
		{Timestamp: 120000, Value: 1},
		{Timestamp: 120000 + 30*60000, Value: 1},
		{Timestamp: 120000 + 31*60000, Value: 1},
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "integrate", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("integrate: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "integrated as straight lines") {
		t.Errorf("expected gap caveat in %s", s)
	}
}

func TestIntegrateRejectsSingleRow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := []dataset.Row{{Timestamp: 0, Value: 1}}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "integrate", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("integrate: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want error for a single row, got %+v", res)
	}
}
