package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProfileAutoBuckets(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t) // values {1,2,3,4,100} one minute apart
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "profile", Arguments: map[string]any{"id": "readings"},
	})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if res.IsError {
		t.Fatalf("profile returned error: %+v", res)
	}
	s := string(mustJSON(res))
	for _, want := range []string{`"row_count":5`, `"bucket":`, `"buckets":[`, `"unit":"celsius"`} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s in %s", want, s)
		}
	}
}

func TestProfileExplicitBucket(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "profile", Arguments: map[string]any{"id": "readings", "bucket": "1m"},
	})
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	if res.IsError {
		t.Fatalf("profile returned error: %+v", res)
	}
	// span is 4 minutes -> 5 one-minute buckets, each with a single point.
	s := string(mustJSON(res))
	if !strings.Contains(s, `"bucket_count":5`) {
		t.Errorf("want bucket_count 5 in %s", s)
	}
}

func TestProfileTooManyBucketsErrors(t *testing.T) {
	ctx := context.Background()
	seedAnalysisDataset(t)
	session := newConnectedSession(t)
	// 4-minute span at 1ms buckets would be 240000 buckets -> over cap.
	res, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "profile", Arguments: map[string]any{"id": "readings", "bucket": "1ms"},
	})
	if res == nil || !res.IsError {
		t.Error("explicit bucket over the cap should be a tool error")
	}
}
