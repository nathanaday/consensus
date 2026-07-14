package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

type FitTrendInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to fit a trend on"`
	Start string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; fit only rows at or after this instant"`
	End   string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; fit only rows at or before this instant"`
}

type FitTrendOutput struct {
	ID                    string            `json:"id"`
	RowCount              int               `json:"row_count"`
	AnalyzedRange         dataset.TimeRange `json:"analyzed_range"`
	Direction             string            `json:"direction"`
	SlopePerHour          float64           `json:"slope_per_hour"`
	SlopePerDay           float64           `json:"slope_per_day"`
	RSquared              float64           `json:"r_squared"`
	WindowDurationSeconds float64           `json:"window_duration_seconds"`
	Unit                  string            `json:"unit,omitempty"`
	Caveats               []string          `json:"caveats"`
}

// FitTrend fits a linear trend over a window and reports slope per hour and
// per day, direction, and goodness of fit. It returns statistics only.
func FitTrend(ctx context.Context, req *mcp.CallToolRequest, input FitTrendInput) (*mcp.CallToolResult, FitTrendOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, FitTrendOutput{}, err
	}
	t, err := analysis.Trend(rows)
	if err != nil {
		return nil, FitTrendOutput{}, fmt.Errorf("dataset %q in the requested window: %w", input.ID, err)
	}

	caveats := []string{}
	if t.RowCount < 10 {
		caveats = append(caveats, fmt.Sprintf("only %d points; a fit over fewer than 10 points may not be meaningful", t.RowCount))
	}
	if t.DurationSeconds < 48*3600 {
		caveats = append(caveats, "window spans less than 48 hours; short-window trends may reflect daily cycles rather than drift")
	}
	if t.RSquared < 0.3 {
		caveats = append(caveats, "weak fit (r_squared below 0.3); the data may not be linear")
	}

	e := n.Info()
	return nil, FitTrendOutput{
		ID:                    n.ID(),
		RowCount:              t.RowCount,
		AnalyzedRange:         analysis.Span(rows),
		Direction:             t.Direction,
		SlopePerHour:          t.SlopePerHour,
		SlopePerDay:           t.SlopePerDay,
		RSquared:              t.RSquared,
		WindowDurationSeconds: t.DurationSeconds,
		Unit:                  e.Unit,
		Caveats:               caveats,
	}, nil
}
