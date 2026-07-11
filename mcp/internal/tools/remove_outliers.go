package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type RemoveOutliersInput struct {
	ID            string  `json:"id" jsonschema:"the dataset id to clean"`
	Name          string  `json:"name,omitempty" jsonschema:"optional id for the new dataset; defaults to a disambiguated form of the source id"`
	Start         string  `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; the new dataset keeps only rows at or after this instant"`
	End           string  `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; the new dataset keeps only rows at or before this instant"`
	IQRMultiplier float64 `json:"iqr_multiplier,omitempty" jsonschema:"IQR fence multiplier k; a point outside [Q1-k*IQR, Q3+k*IQR] is removed (default 1.5, must be positive)"`
}

type RemoveOutliersOutput struct {
	DescribeDatasetOutput
	RowsRemoved   int             `json:"rows_removed"`
	Bounds        analysis.Bounds `json:"bounds"`
	IQRMultiplier float64         `json:"iqr_multiplier"`
}

// RemoveOutliers writes a dataset's IQR inliers to a new immutable child
// dataset and returns its description plus what was removed.
func RemoveOutliers(ctx context.Context, req *mcp.CallToolRequest, input RemoveOutliersInput) (*mcp.CallToolResult, RemoveOutliersOutput, error) {
	k := input.IQRMultiplier
	if k == 0 {
		k = defaultIQRMultiplier
	}
	if k < 0 {
		return nil, RemoveOutliersOutput{}, fmt.Errorf("iqr_multiplier must be positive, got %g", k)
	}
	g, err := lineage.Open()
	if err != nil {
		return nil, RemoveOutliersOutput{}, err
	}
	n, err := g.Node(input.ID)
	if err != nil {
		return nil, RemoveOutliersOutput{}, err
	}
	res, err := n.RemoveOutliers(lineage.RemoveOutliersOptions{
		Name:          input.Name,
		Start:         input.Start,
		End:           input.End,
		IQRMultiplier: k,
	})
	if err != nil {
		return nil, RemoveOutliersOutput{}, err
	}
	return nil, RemoveOutliersOutput{
		DescribeDatasetOutput: describeNode(res.Child),
		RowsRemoved:           res.RowsRemoved,
		Bounds:                res.Bounds,
		IQRMultiplier:         k,
	}, nil
}
