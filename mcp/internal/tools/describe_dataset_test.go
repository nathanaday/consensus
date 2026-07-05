package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func seedLineage(t *testing.T, dir string) {
	t.Helper()
	cfg := store.Config{Dir: dir}
	if _, err := store.SaveDataset(cfg, store.SaveRequest{
		NameOverride: "readings", Origin: "csv",
		Series:   []dataset.Series{{ID: "temp_c", Unit: "celsius"}},
		RowCount: 1,
		Rows:     []dataset.Row{{Timestamp: 1, SeriesID: "temp_c", Value: 1.5}},
	}); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	if _, err := store.SaveDataset(cfg, store.SaveRequest{
		NameOverride: "readings-2", ParentID: "readings", Origin: "copy",
		Series:   []dataset.Series{{ID: "temp_c", Unit: "celsius"}},
		RowCount: 1,
		Rows:     []dataset.Row{{Timestamp: 1, SeriesID: "temp_c", Value: 1.5}},
	}); err != nil {
		t.Fatalf("seed child: %v", err)
	}
}

func TestDescribeDatasetShowsLineage(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedLineage(t, dir)

	session := newConnectedSession(t)

	childRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe_dataset", Arguments: map[string]any{"id": "readings-2"}})
	if err != nil {
		t.Fatalf("describe child: %v", err)
	}
	childData, _ := json.Marshal(childRes)
	cs := string(childData)
	if !strings.Contains(cs, `"origin":"copy"`) {
		t.Errorf("child origin missing in %s", cs)
	}
	if !strings.Contains(cs, `"parent":{"id":"readings","origin":"csv"}`) {
		t.Errorf("child parent edge missing in %s", cs)
	}

	rootRes, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe_dataset", Arguments: map[string]any{"id": "readings"}})
	if err != nil {
		t.Fatalf("describe root: %v", err)
	}
	rootData, _ := json.Marshal(rootRes)
	rs := string(rootData)
	if !strings.Contains(rs, `"parent":null`) {
		t.Errorf("root parent should be null in %s", rs)
	}
	if !strings.Contains(rs, `"id":"readings-2","origin":"copy"`) {
		t.Errorf("root children should list readings-2 in %s", rs)
	}
	if !strings.Contains(rs, `"origin":"csv"`) {
		t.Errorf("root origin should be csv in %s", rs)
	}
}

func TestDescribeDatasetUnknownIDErrors(t *testing.T) {
	ctx := context.Background()
	t.Setenv("CONSENSUS_STORE_DIR", t.TempDir())
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "describe_dataset", Arguments: map[string]any{"id": "nope"}})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for unknown id")
	}
}
