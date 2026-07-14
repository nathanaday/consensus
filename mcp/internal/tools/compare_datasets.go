package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type CompareDatasetsInput struct {
	IDA    string `json:"id_a" jsonschema:"first dataset id"`
	StartA string `json:"start_a,omitempty" jsonschema:"optional RFC3339 UTC start of side A's window"`
	EndA   string `json:"end_a,omitempty" jsonschema:"optional RFC3339 UTC end of side A's window"`
	IDB    string `json:"id_b" jsonschema:"second dataset id (use the same id as id_a to compare two periods of one channel)"`
	StartB string `json:"start_b,omitempty" jsonschema:"optional RFC3339 UTC start of side B's window"`
	EndB   string `json:"end_b,omitempty" jsonschema:"optional RFC3339 UTC end of side B's window"`
}

type CompareSide struct {
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

type CompareDatasetsOutput struct {
	A              CompareSide `json:"a"`
	B              CompareSide `json:"b"`
	MeanDifference float64     `json:"mean_difference"`
	MeanPctChange  *float64    `json:"mean_pct_change,omitempty"`
	StddevRatio    *float64    `json:"stddev_ratio,omitempty"`
	Caveats        []string    `json:"caveats"`
}

func summarizeSide(g *lineage.Graph, id, start, end string) (CompareSide, analysis.SummaryStats, error) {
	n, err := g.Node(id)
	if err != nil {
		return CompareSide{}, analysis.SummaryStats{}, err
	}
	all, err := n.LoadData()
	if err != nil {
		return CompareSide{}, analysis.SummaryStats{}, err
	}
	rows, err := analysis.Window(all, start, end)
	if err != nil {
		return CompareSide{}, analysis.SummaryStats{}, err
	}
	if len(rows) == 0 {
		return CompareSide{}, analysis.SummaryStats{}, fmt.Errorf("dataset %q has no rows in the requested window", id)
	}
	sum, err := analysis.Summary(rows)
	if err != nil {
		return CompareSide{}, analysis.SummaryStats{}, err
	}
	side := CompareSide{
		ID:            n.ID(),
		RowCount:      sum.RowCount,
		AnalyzedRange: analysis.Span(rows),
		Min:           sum.Min,
		Max:           sum.Max,
		Mean:          sum.Mean,
		Median:        sum.Median,
		Stddev:        sum.Stddev,
		Unit:          n.Info().Unit,
	}
	return side, sum, nil
}

// CompareDatasets reports side-by-side statistics for two dataset windows and
// the deltas between them. It returns statistics only, never row data.
func CompareDatasets(ctx context.Context, req *mcp.CallToolRequest, input CompareDatasetsInput) (*mcp.CallToolResult, CompareDatasetsOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, CompareDatasetsOutput{}, err
	}
	sideA, sumA, err := summarizeSide(g, input.IDA, input.StartA, input.EndA)
	if err != nil {
		return nil, CompareDatasetsOutput{}, err
	}
	sideB, sumB, err := summarizeSide(g, input.IDB, input.StartB, input.EndB)
	if err != nil {
		return nil, CompareDatasetsOutput{}, err
	}

	d := analysis.CompareSummaries(sumA, sumB)
	caveats := []string{}
	if sideA.Unit != sideB.Unit {
		caveats = append(caveats, fmt.Sprintf("units differ (%q vs %q); comparison may not be meaningful", sideA.Unit, sideB.Unit))
	}
	if !d.MeanPctChangeOK {
		caveats = append(caveats, "mean_pct_change omitted because side A's mean is 0")
	}
	if !d.StddevRatioOK {
		caveats = append(caveats, "stddev_ratio omitted because side A's stddev is 0")
	}

	out := CompareDatasetsOutput{
		A:              sideA,
		B:              sideB,
		MeanDifference: d.MeanDifference,
		Caveats:        caveats,
	}
	if d.MeanPctChangeOK {
		v := d.MeanPctChange
		out.MeanPctChange = &v
	}
	if d.StddevRatioOK {
		v := d.StddevRatio
		out.StddevRatio = &v
	}
	return nil, out, nil
}
