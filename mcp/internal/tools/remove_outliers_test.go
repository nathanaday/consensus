package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRemoveOutliersCreatesCleanedChild(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t) // values {1,2,3,4,100}: bounds [-1,7], one outlier
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remove_outliers", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("remove_outliers: %v", err)
	}
	if res.IsError {
		t.Fatalf("remove_outliers returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"id":"readings-2"`,
		`"origin":"remove_outliers"`,
		`"parent":{"id":"readings","origin":"csv"}`,
		`"row_count":4`,
		`"rows_removed":1`,
		`"bounds":{"lower":-1,"upper":7}`,
		`"iqr_multiplier":1.5`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestRemoveOutliersHonorsNameAndMultiplier(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remove_outliers", Arguments: map[string]any{
			"id": "readings", "name": "readings-clean", "iqr_multiplier": 0.1,
		},
	})
	if err != nil {
		t.Fatalf("remove_outliers: %v", err)
	}
	if res.IsError {
		t.Fatalf("remove_outliers returned error: %+v", res)
	}
	s := string(mustJSON(res))
	// k=0.1: bounds [1.8, 4.2] -> removes 1 and 100, keeps {2,3,4}
	for _, want := range []string{
		`"id":"readings-clean"`,
		`"row_count":3`,
		`"rows_removed":2`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestRemoveOutliersValidation(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)

	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remove_outliers", Arguments: map[string]any{"id": "readings", "iqr_multiplier": -2},
	})
	if res == nil || !res.IsError {
		t.Error("negative iqr_multiplier should be a tool error")
	}

	res, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remove_outliers", Arguments: map[string]any{"id": "readings", "start": "2030-01-01T00:00:00Z"},
	})
	if res == nil || !res.IsError {
		t.Error("empty window should be a tool error")
	}
}
