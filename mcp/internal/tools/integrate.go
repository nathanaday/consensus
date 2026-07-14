package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

type IntegrateInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to integrate"`
	Start string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; integrate only rows at or after this instant"`
	End   string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; integrate only rows at or before this instant"`
}

type IntegrateOutput struct {
	ID                   string            `json:"id"`
	RowCount             int               `json:"row_count"`
	AnalyzedRange        dataset.TimeRange `json:"analyzed_range"`
	IntegralValueSeconds float64           `json:"integral_value_seconds"`
	IntegralValueHours   float64           `json:"integral_value_hours"`
	TimeWeightedMean     float64           `json:"time_weighted_mean"`
	DurationSeconds      float64           `json:"duration_seconds"`
	Unit                 string            `json:"unit,omitempty"`
	Caveats              []string          `json:"caveats"`
}

// Integrate reports the area under a dataset's curve (trapezoidal rule) and
// the time-weighted mean. It returns statistics only, never row data.
func Integrate(ctx context.Context, req *mcp.CallToolRequest, input IntegrateInput) (*mcp.CallToolResult, IntegrateOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, IntegrateOutput{}, err
	}
	if len(rows) == 0 {
		return nil, IntegrateOutput{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to integrate", input.ID)
	}
	stats, err := analysis.Integrate(rows)
	if err != nil {
		return nil, IntegrateOutput{}, fmt.Errorf("dataset %q in the requested window: %w", input.ID, err)
	}

	caveats := []string{}
	if stats.LargeGapCount > 0 {
		caveats = append(caveats, fmt.Sprintf(
			"%d sampling gaps totalling %.0fs (%.1f%% of the window) were integrated as straight lines between their endpoints",
			stats.LargeGapCount, stats.LargeGapSeconds, stats.LargeGapSeconds/stats.DurationSeconds*100))
	}

	return nil, IntegrateOutput{
		ID:                   n.ID(),
		RowCount:             stats.RowCount,
		AnalyzedRange:        analysis.Span(rows),
		IntegralValueSeconds: stats.IntegralValueSeconds,
		IntegralValueHours:   stats.IntegralValueSeconds / 3600,
		TimeWeightedMean:     stats.TimeWeightedMean,
		DurationSeconds:      stats.DurationSeconds,
		Unit:                 n.Info().Unit,
		Caveats:              caveats,
	}, nil
}
