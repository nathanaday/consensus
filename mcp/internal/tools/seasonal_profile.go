package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

type SeasonalProfileInput struct {
	ID     string `json:"id" jsonschema:"the dataset id to profile"`
	Period string `json:"period" jsonschema:"the cycle to profile against: hour_of_day or day_of_week"`
	Start  string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; profile only rows at or after this instant"`
	End    string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; profile only rows at or before this instant"`
}

type SeasonalPositionOut struct {
	Position int      `json:"position"`
	Label    string   `json:"label"`
	Count    int      `json:"count"`
	Mean     *float64 `json:"mean,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

type SeasonalProfileOutput struct {
	ID            string                `json:"id"`
	RowCount      int                   `json:"row_count"`
	AnalyzedRange dataset.TimeRange     `json:"analyzed_range"`
	Period        string                `json:"period"`
	Positions     []SeasonalPositionOut `json:"positions"`
	CycleStrength *float64              `json:"cycle_strength,omitempty"`
	SpanPeriods   float64               `json:"span_periods"`
	Unit          string                `json:"unit,omitempty"`
	Caveats       []string              `json:"caveats"`
}

// SeasonalProfile reports how a dataset's values vary across a repeating
// cycle (hour of day or day of week, UTC). It returns statistics only, never
// row data.
func SeasonalProfile(ctx context.Context, req *mcp.CallToolRequest, input SeasonalProfileInput) (*mcp.CallToolResult, SeasonalProfileOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, SeasonalProfileOutput{}, err
	}
	if len(rows) == 0 {
		return nil, SeasonalProfileOutput{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to profile", input.ID)
	}
	rep, err := analysis.Seasonal(rows, input.Period)
	if err != nil {
		return nil, SeasonalProfileOutput{}, err
	}

	positions := make([]SeasonalPositionOut, len(rep.Positions))
	empty := 0
	for i, p := range rep.Positions {
		out := SeasonalPositionOut{Position: p.Index, Label: p.Label, Count: p.Count}
		if p.Count > 0 {
			mean, min, max := p.Mean, p.Min, p.Max
			out.Mean, out.Min, out.Max = &mean, &min, &max
		} else {
			empty++
		}
		positions[i] = out
	}

	caveats := []string{"positions use UTC; a pattern anchored to local clock time may appear shifted"}
	if rep.SpanPeriods < 2 {
		caveats = append(caveats, fmt.Sprintf("the window spans %.1f %s cycles; a profile needs at least 2 full cycles to show a repeating pattern", rep.SpanPeriods, input.Period))
	}
	if empty > 0 {
		caveats = append(caveats, fmt.Sprintf("%d positions have no samples", empty))
	}
	if !rep.CycleStrengthOK {
		caveats = append(caveats, "cycle strength is undefined for a constant series")
	}

	out := SeasonalProfileOutput{
		ID:            n.ID(),
		RowCount:      len(rows),
		AnalyzedRange: analysis.Span(rows),
		Period:        rep.Period,
		Positions:     positions,
		SpanPeriods:   rep.SpanPeriods,
		Unit:          n.Info().Unit,
		Caveats:       caveats,
	}
	if rep.CycleStrengthOK {
		cs := round6(rep.CycleStrength)
		out.CycleStrength = &cs
	}
	return nil, out, nil
}
