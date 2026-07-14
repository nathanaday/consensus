package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFitTrendReportsDirection(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t) // {1,2,3,4,100} -> increasing overall
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "fit_trend", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("fit_trend: %v", err)
	}
	if res.IsError {
		t.Fatalf("fit_trend returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"direction":`, `"slope_per_hour":`, `"slope_per_day":`, `"r_squared":`, `"caveats":`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
	// 5 points over 4 minutes -> both small-sample and short-window caveats.
	if !strings.Contains(s, "fewer than 10") {
		t.Errorf("expected small-sample caveat in %s", s)
	}
}

func TestFitTrendShortWindowCaveat(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t) // 5 points over 4 minutes -> short window
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "fit_trend", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("fit_trend: %v", err)
	}
	if res.IsError {
		t.Fatalf("returned error: %+v", res)
	}
	s := string(mustJSON(res))
	if !strings.Contains(s, "less than 48 hours") {
		t.Errorf("expected short-window caveat in %s", s)
	}
}
