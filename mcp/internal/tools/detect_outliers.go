package tools

import (
	"context"
	"fmt"
	"math"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const (
	defaultIQRMultiplier = 1.5
	outlierDefaultLimit  = 20
	outlierMaxLimit      = 100
)

type DetectOutliersInput struct {
	ID            string  `json:"id" jsonschema:"the dataset id to analyze"`
	Start         string  `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or after this instant"`
	End           string  `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or before this instant"`
	IQRMultiplier float64 `json:"iqr_multiplier,omitempty" jsonschema:"IQR fence multiplier k; a point outside [Q1-k*IQR, Q3+k*IQR] is an outlier (default 1.5, must be positive)"`
	Limit         int     `json:"limit,omitempty" jsonschema:"max outlier points to return (default 20, max 100); total_outliers always reports the full count"`
}

type DetectOutliersOutput struct {
	ID            string                  `json:"id"`
	RowCount      int                     `json:"row_count"`
	AnalyzedRange dataset.TimeRange       `json:"analyzed_range"`
	IQRMultiplier float64                 `json:"iqr_multiplier"`
	Quartiles     analysis.Quartiles      `json:"quartiles"`
	Bounds        analysis.Bounds         `json:"bounds"`
	TotalOutliers int                     `json:"total_outliers"`
	Percent       float64                 `json:"percent"`
	Outliers      []analysis.OutlierPoint `json:"outliers"`
}

// DetectOutliers reports IQR outliers in a dataset: the quartiles and bounds
// used, the total count, and the most extreme points up to limit. It never
// returns bulk row data.
func DetectOutliers(ctx context.Context, req *mcp.CallToolRequest, input DetectOutliersInput) (*mcp.CallToolResult, DetectOutliersOutput, error) {
	k := input.IQRMultiplier
	if k == 0 {
		k = defaultIQRMultiplier
	}
	if k < 0 {
		return nil, DetectOutliersOutput{}, fmt.Errorf("iqr_multiplier must be positive, got %g", k)
	}
	limit := input.Limit
	if limit < 0 {
		return nil, DetectOutliersOutput{}, fmt.Errorf("limit must be at least 0, got %d", limit)
	}
	if limit == 0 {
		limit = outlierDefaultLimit
	}
	if limit > outlierMaxLimit {
		limit = outlierMaxLimit
	}

	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, DetectOutliersOutput{}, err
	}
	report, err := analysis.Outliers(rows, k)
	if err != nil {
		return nil, DetectOutliersOutput{}, fmt.Errorf("dataset %q in the requested window: %w", input.ID, err)
	}

	points := report.Points
	if len(points) > limit {
		points = points[:limit]
	}
	percent := math.Round(10000*float64(len(report.Points))/float64(report.RowCount)) / 100
	return nil, DetectOutliersOutput{
		ID:            n.ID(),
		RowCount:      report.RowCount,
		AnalyzedRange: analysis.Span(rows),
		IQRMultiplier: k,
		Quartiles:     report.Quartiles,
		Bounds:        report.Bounds,
		TotalOutliers: len(report.Points),
		Percent:       percent,
		Outliers:      points,
	}, nil
}
