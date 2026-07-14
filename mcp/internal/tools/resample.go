package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type ResampleInput struct {
	ID     string `json:"id" jsonschema:"the dataset id to resample"`
	Bucket string `json:"bucket" jsonschema:"required Go duration bucket width like 1h or 15m; must be at least the source's median sampling interval"`
	Agg    string `json:"agg,omitempty" jsonschema:"aggregate per bucket: mean (default), min, max, or median"`
	Name   string `json:"name,omitempty" jsonschema:"optional id for the new dataset; defaults to a disambiguated form of the source id"`
	Start  string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; resample only rows at or after this instant"`
	End    string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; resample only rows at or before this instant"`
}

type ResampleOutput struct {
	DescribeDatasetOutput
	Bucket         string `json:"bucket"`
	Agg            string `json:"agg"`
	SourceRowCount int    `json:"source_row_count"`
}

// Resample writes a bucketed, downsampled copy of a dataset to a new
// immutable child and returns its description. It never returns row data.
func Resample(ctx context.Context, req *mcp.CallToolRequest, input ResampleInput) (*mcp.CallToolResult, ResampleOutput, error) {
	bucketMS, err := parseBucketMS(input.Bucket)
	if err != nil {
		return nil, ResampleOutput{}, err
	}
	agg := input.Agg
	if agg == "" {
		agg = "mean"
	}
	g, err := lineage.Open()
	if err != nil {
		return nil, ResampleOutput{}, err
	}
	n, err := g.Node(input.ID)
	if err != nil {
		return nil, ResampleOutput{}, err
	}
	res, err := n.Resample(lineage.ResampleOptions{
		Name:     input.Name,
		Start:    input.Start,
		End:      input.End,
		BucketMS: bucketMS,
		Agg:      agg,
	})
	if err != nil {
		return nil, ResampleOutput{}, err
	}
	return nil, ResampleOutput{
		DescribeDatasetOutput: describeNode(res.Child),
		Bucket:                input.Bucket,
		Agg:                   agg,
		SourceRowCount:        res.SourceRowCount,
	}, nil
}
