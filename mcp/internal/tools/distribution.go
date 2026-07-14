package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const distributionMaxBins = 40

type DistributionInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to analyze"`
	Start string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or after this instant"`
	End   string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or before this instant"`
	Bins  int    `json:"bins,omitempty" jsonschema:"optional histogram bin count, capped at 40; omitted picks one from the sample size"`
}

type HistBinOut struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

type DistributionOutput struct {
	ID            string               `json:"id"`
	RowCount      int                  `json:"row_count"`
	AnalyzedRange dataset.TimeRange    `json:"analyzed_range"`
	Min           float64              `json:"min"`
	Max           float64              `json:"max"`
	Mean          float64              `json:"mean"`
	Stddev        float64              `json:"stddev"`
	Percentiles   analysis.Percentiles `json:"percentiles"`
	BinWidth      float64              `json:"bin_width"`
	Histogram     []HistBinOut         `json:"histogram"`
	Unit          string               `json:"unit,omitempty"`
	Caveats       []string             `json:"caveats"`
}

// Distribution reports how a dataset's values are distributed: percentiles
// plus an equal-width histogram. It returns statistics only, never row data.
func Distribution(ctx context.Context, req *mcp.CallToolRequest, input DistributionInput) (*mcp.CallToolResult, DistributionOutput, error) {
	if input.Bins > distributionMaxBins {
		return nil, DistributionOutput{}, fmt.Errorf("bins %d is over the limit of %d; use fewer bins", input.Bins, distributionMaxBins)
	}
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, DistributionOutput{}, err
	}
	if len(rows) == 0 {
		return nil, DistributionOutput{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to analyze", input.ID)
	}
	rep, err := analysis.Distribution(rows, input.Bins)
	if err != nil {
		return nil, DistributionOutput{}, err
	}

	hist := make([]HistBinOut, len(rep.Bins))
	for i, b := range rep.Bins {
		hist[i] = HistBinOut{
			Lower: b.Lower, Upper: b.Upper, Count: b.Count,
			Pct: float64(b.Count) / float64(rep.RowCount) * 100,
		}
	}

	caveats := []string{}
	if rep.Min == rep.Max {
		caveats = append(caveats, "all values are identical; the histogram is a single bin")
	}

	return nil, DistributionOutput{
		ID:            n.ID(),
		RowCount:      rep.RowCount,
		AnalyzedRange: analysis.Span(rows),
		Min:           rep.Min,
		Max:           rep.Max,
		Mean:          rep.Mean,
		Stddev:        rep.Stddev,
		Percentiles:   rep.Percentiles,
		BinWidth:      rep.BinWidth,
		Histogram:     hist,
		Unit:          n.Info().Unit,
		Caveats:       caveats,
	}, nil
}
