package tools

import (
	"context"
	"fmt"
	"math"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

const baselineMaxEpisodes = 10

type CompareToBaselineInput struct {
	ID            string  `json:"id" jsonschema:"the dataset id to examine (the subject)"`
	Start         string  `json:"start,omitempty" jsonschema:"optional RFC3339 UTC start of the subject window"`
	End           string  `json:"end,omitempty" jsonschema:"optional RFC3339 UTC end of the subject window"`
	BaselineID    string  `json:"baseline_id,omitempty" jsonschema:"dataset id for the baseline; defaults to the subject dataset"`
	BaselineStart string  `json:"baseline_start,omitempty" jsonschema:"optional RFC3339 UTC start of the baseline window"`
	BaselineEnd   string  `json:"baseline_end,omitempty" jsonschema:"optional RFC3339 UTC end of the baseline window; defaults to all history before the subject window"`
	IQRMultiplier float64 `json:"iqr_multiplier,omitempty" jsonschema:"IQR fence multiplier k for the baseline distribution (default 1.5, must be positive)"`
}

type EpisodeOut struct {
	Start         string  `json:"start"`
	End           string  `json:"end"`
	Direction     string  `json:"direction"`
	PointCount    int     `json:"point_count"`
	PeakValue     float64 `json:"peak_value"`
	PeakTimestamp string  `json:"peak_timestamp"`
	PeakDeviation float64 `json:"peak_deviation"`
}

type CompareToBaselineOutput struct {
	ID               string                  `json:"id"`
	RowCount         int                     `json:"row_count"`
	AnalyzedRange    dataset.TimeRange       `json:"analyzed_range"`
	BaselineID       string                  `json:"baseline_id"`
	BaselineRange    dataset.TimeRange       `json:"baseline_range"`
	BaselineRowCount int                     `json:"baseline_row_count"`
	Baseline         analysis.BaselineDist   `json:"baseline"`
	Bounds           analysis.Bounds         `json:"bounds"`
	Subject          analysis.SubjectSummary `json:"subject"`
	PointsOutside    int                     `json:"points_outside"`
	PctOutside       float64                 `json:"pct_outside"`
	TotalEpisodes    int                     `json:"total_episodes"`
	Episodes         []EpisodeOut            `json:"episodes"`
	Unit             string                  `json:"unit,omitempty"`
	Caveats          []string                `json:"caveats"`
}

// CompareToBaseline scores a subject window against a baseline distribution
// and reports anomalous episodes. It returns statistics only, never rows.
func CompareToBaseline(ctx context.Context, req *mcp.CallToolRequest, input CompareToBaselineInput) (*mcp.CallToolResult, CompareToBaselineOutput, error) {
	k := input.IQRMultiplier
	if k == 0 {
		k = defaultIQRMultiplier
	}
	if k < 0 {
		return nil, CompareToBaselineOutput{}, fmt.Errorf("iqr_multiplier must be positive, got %g", k)
	}
	g, err := lineage.Open()
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}

	subjNode, err := g.Node(input.ID)
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}
	subjAll, err := subjNode.LoadData()
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}
	subject, err := analysis.Window(subjAll, input.Start, input.End)
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}
	if len(subject) == 0 {
		return nil, CompareToBaselineOutput{}, fmt.Errorf("dataset %q has no rows in the subject window; nothing to analyze", input.ID)
	}
	subjRange := analysis.Span(subject)

	baselineID := input.BaselineID
	if baselineID == "" {
		baselineID = input.ID
	}
	baseNode, err := g.Node(baselineID)
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}
	baseAll, err := baseNode.LoadData()
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}

	var baseline []dataset.Row
	usedDefault := input.BaselineStart == "" && input.BaselineEnd == ""
	if usedDefault {
		// default: all baseline rows strictly before the subject window start.
		firstMS := subjectFirstMS(subject)
		for _, r := range baseAll {
			if r.Timestamp < firstMS {
				baseline = append(baseline, r)
			}
		}
		if len(baseline) == 0 {
			return nil, CompareToBaselineOutput{}, fmt.Errorf("no rows in %q before the subject window start %s to use as a baseline; pass baseline_start/baseline_end or a different baseline_id", baselineID, subjRange.Start)
		}
	} else {
		baseline, err = analysis.Window(baseAll, input.BaselineStart, input.BaselineEnd)
		if err != nil {
			return nil, CompareToBaselineOutput{}, err
		}
		if len(baseline) == 0 {
			return nil, CompareToBaselineOutput{}, fmt.Errorf("baseline window of %q is empty", baselineID)
		}
	}

	rep, err := analysis.Baseline(subject, baseline, k)
	if err != nil {
		return nil, CompareToBaselineOutput{}, err
	}

	episodes := rep.Episodes
	total := len(episodes)
	if len(episodes) > baselineMaxEpisodes {
		episodes = episodes[:baselineMaxEpisodes]
	}
	out := make([]EpisodeOut, len(episodes))
	for i, e := range episodes {
		out[i] = EpisodeOut{
			Start:         renderMS(e.StartMS),
			End:           renderMS(e.EndMS),
			Direction:     e.Direction,
			PointCount:    e.PointCount,
			PeakValue:     e.PeakValue,
			PeakTimestamp: renderMS(e.PeakMS),
			PeakDeviation: e.PeakDeviation,
		}
	}

	caveats := []string{}
	if rep.Baseline.Count < 30 {
		caveats = append(caveats, fmt.Sprintf("baseline has only %d rows; the typical range may be unreliable", rep.Baseline.Count))
	}
	if len(subject) < 4 {
		caveats = append(caveats, fmt.Sprintf("subject window has only %d points", len(subject)))
	}

	pct := math.Round(10000*float64(rep.PointsOutside)/float64(len(subject))) / 100
	e := subjNode.Info()
	return nil, CompareToBaselineOutput{
		ID:               subjNode.ID(),
		RowCount:         len(subject),
		AnalyzedRange:    subjRange,
		BaselineID:       baselineID,
		BaselineRange:    analysis.Span(baseline),
		BaselineRowCount: len(baseline),
		Baseline:         rep.Baseline,
		Bounds:           rep.Bounds,
		Subject:          rep.Subject,
		PointsOutside:    rep.PointsOutside,
		PctOutside:       pct,
		TotalEpisodes:    total,
		Episodes:         out,
		Unit:             e.Unit,
		Caveats:          caveats,
	}, nil
}

// subjectFirstMS returns the earliest timestamp of a non-empty sorted-or-
// unsorted row set.
func subjectFirstMS(rows []dataset.Row) int64 {
	first := rows[0].Timestamp
	for _, r := range rows {
		if r.Timestamp < first {
			first = r.Timestamp
		}
	}
	return first
}
