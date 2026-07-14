package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestSeasonalProfileHourOfDay(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	// Three days of hourly samples whose value equals the hour: a perfect
	// daily cycle.
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := make([]dataset.Row, 0, 72)
	for d := 0; d < 3; d++ {
		for h := 0; h < 24; h++ {
			rows = append(rows, dataset.Row{Timestamp: start + int64(d*24+h)*3600000, Value: float64(h)})
		}
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", Unit: "celsius", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "seasonal_profile", Arguments: map[string]any{"id": "readings", "period": "hour_of_day"},
	})
	if err != nil {
		t.Fatalf("seasonal_profile: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"period":"hour_of_day"`, `"cycle_strength":1`, `"label":"05:00"`, `"row_count":72`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
	if strings.Count(s, `"label"`) != 24 {
		t.Errorf("want 24 positions, got %s", s)
	}
}

func TestSeasonalProfileShortWindowCaveat(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := make([]dataset.Row, 12)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: start + int64(i)*3600000, Value: float64(i)}
	}
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: len(rows), Rows: rows,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "seasonal_profile", Arguments: map[string]any{"id": "readings", "period": "hour_of_day"},
	})
	if err != nil {
		t.Fatalf("seasonal_profile: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "at least 2 full cycles") {
		t.Errorf("expected short-window caveat in %s", s)
	}
	if !strings.Contains(s, "positions have no samples") {
		t.Errorf("expected empty-positions caveat in %s", s)
	}
}

func TestSeasonalProfileRejectsUnknownPeriod(t *testing.T) {
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
		Name: "seasonal_profile", Arguments: map[string]any{"id": "readings", "period": "phase_of_moon"},
	})
	if err != nil {
		t.Fatalf("seasonal_profile: %v", err)
	}
	if !res.IsError {
		t.Fatalf("want error for unknown period, got %+v", res)
	}
}
