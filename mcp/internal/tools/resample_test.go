package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResampleToolCreatesChild(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t) // 5 rows one minute apart
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "resample", Arguments: map[string]any{"id": "readings", "bucket": "2m", "agg": "mean"},
	})
	if err != nil {
		t.Fatalf("resample: %v", err)
	}
	if res.IsError {
		t.Fatalf("resample returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"origin":"resample"`, `"bucket":"2m"`, `"agg":"mean"`, `"source_row_count":5`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestResampleToolRejectsBadBucket(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "resample", Arguments: map[string]any{"id": "readings"},
	})
	if res == nil || !res.IsError {
		t.Error("missing bucket should be a tool error")
	}
}
