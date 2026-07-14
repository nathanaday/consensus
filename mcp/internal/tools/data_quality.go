package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const qualityListCap = 10

type DataQualityInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to check"`
	Start string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; check only rows at or after this instant"`
	End   string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; check only rows at or before this instant"`
}

type GapOut struct {
	Start           string `json:"start"`
	End             string `json:"end"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type FlatlineOut struct {
	Start      string  `json:"start"`
	End        string  `json:"end"`
	Value      float64 `json:"value"`
	PointCount int     `json:"point_count"`
}

type DataQualityOutput struct {
	ID                    string            `json:"id"`
	RowCount              int               `json:"row_count"`
	AnalyzedRange         dataset.TimeRange `json:"analyzed_range"`
	MedianIntervalSeconds float64           `json:"median_interval_seconds"`
	IntervalP95Seconds    float64           `json:"interval_p95_seconds"`
	MinIntervalSeconds    float64           `json:"min_interval_seconds"`
	MaxIntervalSeconds    float64           `json:"max_interval_seconds"`
	TotalGaps             int               `json:"total_gaps"`
	Gaps                  []GapOut          `json:"gaps"`
	TotalFlatlines        int               `json:"total_flatlines"`
	Flatlines             []FlatlineOut     `json:"flatlines"`
	DuplicateTimestamps   int               `json:"duplicate_timestamps"`
	Unit                  string            `json:"unit,omitempty"`
	Caveats               []string          `json:"caveats"`
}

// DataQuality reports sampling regularity, gaps, flatlines, and duplicate
// timestamps for one dataset. It returns statistics only, never row data.
func DataQuality(ctx context.Context, req *mcp.CallToolRequest, input DataQualityInput) (*mcp.CallToolResult, DataQualityOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, DataQualityOutput{}, err
	}
	if len(rows) == 0 {
		return nil, DataQualityOutput{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to check", input.ID)
	}
	rep, err := analysis.Quality(rows)
	if err != nil {
		return nil, DataQualityOutput{}, err
	}

	gaps := rep.Gaps
	if len(gaps) > qualityListCap {
		gaps = gaps[:qualityListCap]
	}
	gapsOut := make([]GapOut, len(gaps))
	for i, g := range gaps {
		gapsOut[i] = GapOut{Start: renderMS(g.StartMS), End: renderMS(g.EndMS), DurationSeconds: g.DurationMS / 1000}
	}

	flat := rep.Flatlines
	if len(flat) > qualityListCap {
		flat = flat[:qualityListCap]
	}
	flatOut := make([]FlatlineOut, len(flat))
	for i, f := range flat {
		flatOut[i] = FlatlineOut{Start: renderMS(f.StartMS), End: renderMS(f.EndMS), Value: f.Value, PointCount: f.PointCount}
	}

	caveats := []string{}
	if rep.RowCount < 3 {
		caveats = append(caveats, "fewer than 3 points; interval statistics are unreliable")
	}

	return nil, DataQualityOutput{
		ID:                    n.ID(),
		RowCount:              rep.RowCount,
		AnalyzedRange:         analysis.Span(rows),
		MedianIntervalSeconds: float64(rep.MedianIntervalMS) / 1000,
		IntervalP95Seconds:    float64(rep.IntervalP95MS) / 1000,
		MinIntervalSeconds:    float64(rep.MinIntervalMS) / 1000,
		MaxIntervalSeconds:    float64(rep.MaxIntervalMS) / 1000,
		TotalGaps:             rep.TotalGaps,
		Gaps:                  gapsOut,
		TotalFlatlines:        rep.TotalFlatlines,
		Flatlines:             flatOut,
		DuplicateTimestamps:   rep.DuplicateTimestamps,
		Unit:                  n.Info().Unit,
		Caveats:               caveats,
	}, nil
}
