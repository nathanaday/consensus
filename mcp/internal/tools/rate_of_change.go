package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

type RateOfChangeInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to analyze"`
	Start string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or after this instant"`
	End   string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or before this instant"`
}

type RateOfChangeOutput struct {
	ID                          string             `json:"id"`
	RowCount                    int                `json:"row_count"`
	AnalyzedRange               dataset.TimeRange  `json:"analyzed_range"`
	MaxRise                     analysis.RatePoint `json:"max_rise"`
	MaxFall                     analysis.RatePoint `json:"max_fall"`
	MeanAbsRate                 float64            `json:"mean_abs_rate"`
	Unit                        string             `json:"unit"`
	MedianSampleIntervalSeconds float64            `json:"median_sample_interval_seconds"`
}

// rateUnit derives the derivative's unit label from the dataset's unit.
func rateUnit(unit string) string {
	if unit == "" {
		return "per second"
	}
	return unit + "/second"
}

// RateOfChange reports the pairwise derivative (value per second) of a
// dataset: steepest rise, steepest fall, mean absolute rate, and the median
// sampling interval. It returns statistics only, never row data.
func RateOfChange(ctx context.Context, req *mcp.CallToolRequest, input RateOfChangeInput) (*mcp.CallToolResult, RateOfChangeOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, RateOfChangeOutput{}, err
	}
	stats, err := analysis.Rates(rows)
	if err != nil {
		return nil, RateOfChangeOutput{}, fmt.Errorf("dataset %q in the requested window: %w", input.ID, err)
	}
	return nil, RateOfChangeOutput{
		ID:                          n.ID(),
		RowCount:                    stats.RowCount,
		AnalyzedRange:               analysis.Span(rows),
		MaxRise:                     stats.MaxRise,
		MaxFall:                     stats.MaxFall,
		MeanAbsRate:                 stats.MeanAbsRate,
		Unit:                        rateUnit(n.Info().Unit),
		MedianSampleIntervalSeconds: stats.MedianIntervalSeconds,
	}, nil
}
