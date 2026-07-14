package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestOverviewReportsAllChannels(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	temp := []dataset.Row{
		{Timestamp: 0, Value: 20}, {Timestamp: 60000, Value: 30}, {Timestamp: 120000, Value: 25},
	}
	hum := []dataset.Row{
		{Timestamp: 0, Value: 50}, {Timestamp: 60000, Value: 60},
	}
	for name, rows := range map[string][]dataset.Row{"iot/temp": temp, "iot/humidity": hum} {
		if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
			NameOverride: name, Origin: "csv", Unit: "u", RowCount: len(rows), Rows: rows,
		}); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "overview", Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"dataset_count":2`, `"id":"iot/temp"`, `"id":"iot/humidity"`,
		`"last_value":25`, `"last_value":60`, `"mean":25`, `"mean":55`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestOverviewPrefixFilter(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	rows := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}}
	for _, name := range []string{"plant-a/temp", "plant-b/temp"} {
		if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
			NameOverride: name, Origin: "csv", RowCount: len(rows), Rows: rows,
		}); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "overview", Arguments: map[string]any{"prefix": "plant-a"},
	})
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, `"dataset_count":1`) || strings.Contains(s, "plant-b") {
		t.Errorf("prefix filter failed: %s", s)
	}
}

func TestOverviewEmptyStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	session := newConnectedSession(t)
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "overview", Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, `"dataset_count":0`) {
		t.Errorf("want an empty overview, got %s", s)
	}
}
