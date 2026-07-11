package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type SummaryStatsInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to summarize"`
	Start string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or after this instant"`
	End   string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; analyze only rows at or before this instant"`
}

type SummaryStatsOutput struct {
	ID            string            `json:"id"`
	RowCount      int               `json:"row_count"`
	AnalyzedRange dataset.TimeRange `json:"analyzed_range"`
	Min           analysis.Extreme  `json:"min"`
	Max           analysis.Extreme  `json:"max"`
	Mean          float64           `json:"mean"`
	Median        float64           `json:"median"`
	Stddev        float64           `json:"stddev"`
	Unit          string            `json:"unit,omitempty"`
}

// analyzedRows loads a dataset's rows windowed to [start, end]. Shared by
// the analysis tools.
func analyzedRows(id, start, end string) (*lineage.Node, []dataset.Row, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, nil, err
	}
	n, err := g.Node(id)
	if err != nil {
		return nil, nil, err
	}
	rows, err := n.LoadData()
	if err != nil {
		return nil, nil, err
	}
	windowed, err := analysis.Window(rows, start, end)
	if err != nil {
		return nil, nil, err
	}
	return n, windowed, nil
}

// SummaryStats reports descriptive statistics for a dataset (optionally a
// time window of it). It returns statistics only, never row data.
func SummaryStats(ctx context.Context, req *mcp.CallToolRequest, input SummaryStatsInput) (*mcp.CallToolResult, SummaryStatsOutput, error) {
	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, SummaryStatsOutput{}, err
	}
	stats, err := analysis.Summary(rows)
	if err != nil {
		return nil, SummaryStatsOutput{}, fmt.Errorf("dataset %q in the requested window: %w", input.ID, err)
	}
	return nil, SummaryStatsOutput{
		ID:            n.ID(),
		RowCount:      stats.RowCount,
		AnalyzedRange: analysis.Span(rows),
		Min:           stats.Min,
		Max:           stats.Max,
		Mean:          stats.Mean,
		Median:        stats.Median,
		Stddev:        stats.Stddev,
		Unit:          n.Info().Unit,
	}, nil
}
