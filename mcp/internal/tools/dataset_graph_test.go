package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/store"
)

func seedTwo(t *testing.T, dir string) {
	t.Helper()
	cfg := store.Config{Dir: dir}
	if _, err := store.SaveDataset(cfg, store.SaveRequest{NameOverride: "readings", Origin: "csv", SourceColumn: "temp_c"}); err != nil {
		t.Fatalf("seed root: %v", err)
	}
	if _, err := store.SaveDataset(cfg, store.SaveRequest{NameOverride: "readings-2", ParentID: "readings", Origin: "copy", SourceColumn: "temp_c"}); err != nil {
		t.Fatalf("seed child: %v", err)
	}
}

func TestDatasetGraphMermaidAndJSON(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seedTwo(t, dir)
	session := newConnectedSession(t)

	// Default format is mermaid.
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "dataset_graph"})
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	s := string(mustJSON(res))
	// encoding/json (used by the go-sdk to marshal tool output) HTML-escapes
	// the greater-than sign by default, so the wire form of the "-->" edge
	// arrow carries that escape sequence rather than the raw character.
	mermaidEdge := "readings --" + "\\u003e" + "|copy| readings_2"
	if !strings.Contains(s, `"format":"mermaid"`) || !strings.Contains(s, mermaidEdge) {
		t.Errorf("expected mermaid edge in %s", s)
	}

	// JSON format nests roots/nodes under graph.
	resJSON, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "dataset_graph", Arguments: map[string]any{"format": "json"}})
	if err != nil {
		t.Fatalf("graph json: %v", err)
	}
	sj := string(mustJSON(resJSON))
	if !strings.Contains(sj, `"roots":["readings"]`) {
		t.Errorf("expected roots in %s", sj)
	}
	// The go-sdk round-trips structured output through map[string]any to
	// apply schema defaults/validation, so object keys come back sorted
	// alphabetically rather than in GraphNode's declared field order.
	if !strings.Contains(sj, `"readings-2":{"children":[],"origin":"copy","parent_id":"readings"}`) {
		t.Errorf("expected child node in %s", sj)
	}
}

func TestDatasetGraphUnknownFormatErrors(t *testing.T) {
	ctx := context.Background()
	t.Setenv("CONSENSUS_STORE_DIR", t.TempDir())
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "dataset_graph", Arguments: map[string]any{"format": "svg"}})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for unknown format")
	}
}
