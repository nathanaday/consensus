package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDetectOutliersFlagsExtremePoint(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t) // values {1,2,3,4,100}: bounds [-1,7], one outlier
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "detect_outliers", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("detect_outliers: %v", err)
	}
	if res.IsError {
		t.Fatalf("detect_outliers returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{
		`"row_count":5`,
		`"iqr_multiplier":1.5`,
		`"quartiles":{"q1":2,"q2":3,"q3":4}`,
		`"bounds":{"lower":-1,"upper":7}`,
		`"total_outliers":1`,
		`"percent":20`,
		`"outliers":[{"deviation":93,"timestamp":"1970-01-01T00:04:00Z","value":100}]`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestDetectOutliersRespectsLimit(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "detect_outliers", Arguments: map[string]any{
			"id": "readings", "iqr_multiplier": 0.1, "limit": 1,
		},
	})
	if err != nil {
		t.Fatalf("detect_outliers: %v", err)
	}
	if res.IsError {
		t.Fatalf("detect_outliers returned error: %+v", res)
	}
	s := string(mustJSON(res))
	// k=0.1: bounds [1.8, 4.2] -> outliers are 1 and 100; limit keeps only
	// the most extreme (100), but total_outliers reports both.
	if !strings.Contains(s, `"total_outliers":2`) {
		t.Errorf("expected total_outliers 2 in %s", s)
	}
	if strings.Count(s, `"deviation":`) != 1 {
		t.Errorf("expected exactly 1 returned outlier in %s", s)
	}
	if !strings.Contains(s, `"value":100`) {
		t.Errorf("expected the most extreme outlier (100) to survive the cap in %s", s)
	}
}

func TestDetectOutliersValidation(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)

	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "detect_outliers", Arguments: map[string]any{"id": "readings", "iqr_multiplier": -1},
	})
	if res == nil || !res.IsError {
		t.Error("negative iqr_multiplier should be a tool error")
	}

	res, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "detect_outliers", Arguments: map[string]any{"id": "readings", "limit": -5},
	})
	if res == nil || !res.IsError {
		t.Error("negative limit should be a tool error")
	}
}
