package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const profileMaxBuckets = 48

type ProfileInput struct {
	ID     string `json:"id" jsonschema:"the dataset id to profile"`
	Start  string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; profile only rows at or after this instant"`
	End    string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; profile only rows at or before this instant"`
	Bucket string `json:"bucket,omitempty" jsonschema:"optional Go duration bucket width like 1h or 15m; omitted picks a round width yielding at most 48 buckets"`
}

// ProfileBucket is one bucket in the profile. Empty buckets omit statistics.
type ProfileBucket struct {
	Start string   `json:"start"`
	Mean  *float64 `json:"mean,omitempty"`
	Min   *float64 `json:"min,omitempty"`
	Max   *float64 `json:"max,omitempty"`
	Count int      `json:"count"`
}

type ProfileOutput struct {
	ID            string            `json:"id"`
	RowCount      int               `json:"row_count"`
	AnalyzedRange dataset.TimeRange `json:"analyzed_range"`
	Bucket        string            `json:"bucket"`
	BucketCount   int               `json:"bucket_count"`
	Buckets       []ProfileBucket   `json:"buckets"`
	Unit          string            `json:"unit,omitempty"`
	Caveats       []string          `json:"caveats"`
}

func renderMS(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339Nano)
}

// Profile reports a bounded set of time buckets (mean/min/max/count) so the
// shape of a window can be read without row data.
func Profile(ctx context.Context, req *mcp.CallToolRequest, input ProfileInput) (*mcp.CallToolResult, ProfileOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, ProfileOutput{}, err
	}
	if len(rows) == 0 {
		return nil, ProfileOutput{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to profile", input.ID)
	}

	var widthMS int64
	if input.Bucket == "" {
		widthMS, err = analysis.AutoWidthMS(rows, profileMaxBuckets)
		if err != nil {
			return nil, ProfileOutput{}, err
		}
	} else {
		widthMS, err = parseBucketMS(input.Bucket)
		if err != nil {
			return nil, ProfileOutput{}, err
		}
		// rows is sorted (analyzedRows windows via analysis.Window); project the
		// bucket count and reject an over-cap explicit bucket before allocating.
		span := rows[len(rows)-1].Timestamp - rows[0].Timestamp
		if projected := span/widthMS + 1; projected > int64(profileMaxBuckets) {
			return nil, ProfileOutput{}, fmt.Errorf("bucket %s produces %d buckets, over the limit of %d; use a wider bucket or a narrower window", input.Bucket, projected, profileMaxBuckets)
		}
	}

	buckets, err := analysis.Buckets(rows, widthMS)
	if err != nil {
		return nil, ProfileOutput{}, err
	}

	out := make([]ProfileBucket, len(buckets))
	for i, b := range buckets {
		pb := ProfileBucket{Start: renderMS(b.StartMS), Count: b.Count}
		if b.Count > 0 {
			mean, min, max := b.Mean, b.Min, b.Max
			pb.Mean, pb.Min, pb.Max = &mean, &min, &max
		}
		out[i] = pb
	}

	e := n.Info()
	return nil, ProfileOutput{
		ID:            n.ID(),
		RowCount:      len(rows),
		AnalyzedRange: analysis.Span(rows),
		Bucket:        (time.Duration(widthMS) * time.Millisecond).String(),
		BucketCount:   len(buckets),
		Buckets:       out,
		Unit:          e.Unit,
		Caveats:       []string{},
	}, nil
}
