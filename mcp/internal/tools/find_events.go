package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const (
	findEventsDefaultLimit = 20
	findEventsMaxLimit     = 100
)

type FindEventsInput struct {
	ID        string   `json:"id" jsonschema:"the dataset id to scan"`
	Condition string   `json:"condition" jsonschema:"the value condition: above, below, between, or outside"`
	Threshold *float64 `json:"threshold,omitempty" jsonschema:"the threshold value for above/below conditions"`
	Lower     *float64 `json:"lower,omitempty" jsonschema:"the lower bound for between/outside conditions"`
	Upper     *float64 `json:"upper,omitempty" jsonschema:"the upper bound for between/outside conditions"`
	Start     string   `json:"start,omitempty" jsonschema:"optional RFC3339 UTC timestamp; scan only rows at or after this instant"`
	End       string   `json:"end,omitempty" jsonschema:"optional RFC3339 UTC timestamp; scan only rows at or before this instant"`
	Limit     int      `json:"limit,omitempty" jsonschema:"maximum events to return, default 20, capped at 100"`
}

type EventOut struct {
	Start           string   `json:"start"`
	End             string   `json:"end"`
	DurationSeconds float64  `json:"duration_seconds"`
	Direction       string   `json:"direction"`
	PointCount      int      `json:"point_count"`
	PeakValue       float64  `json:"peak_value"`
	PeakTimestamp   string   `json:"peak_timestamp"`
	PeakDeviation   *float64 `json:"peak_deviation,omitempty"`
}

type FindEventsOutput struct {
	ID                  string            `json:"id"`
	RowCount            int               `json:"row_count"`
	AnalyzedRange       dataset.TimeRange `json:"analyzed_range"`
	Condition           string            `json:"condition"`
	Threshold           *float64          `json:"threshold,omitempty"`
	Lower               *float64          `json:"lower,omitempty"`
	Upper               *float64          `json:"upper,omitempty"`
	PointsMatching      int               `json:"points_matching"`
	PctPoints           float64           `json:"pct_points"`
	TimeInEventsSeconds float64           `json:"time_in_events_seconds"`
	PctTime             float64           `json:"pct_time"`
	TotalEvents         int               `json:"total_events"`
	Events              []EventOut        `json:"events"`
	Unit                string            `json:"unit,omitempty"`
	Caveats             []string          `json:"caveats"`
}

// FindEvents reports when a dataset satisfied a value condition, grouped into
// events with durations. It returns statistics only, never row data.
func FindEvents(ctx context.Context, req *mcp.CallToolRequest, input FindEventsInput) (*mcp.CallToolResult, FindEventsOutput, error) {
	cond := analysis.Condition{Kind: input.Condition}
	switch input.Condition {
	case "above", "below":
		if input.Threshold == nil {
			return nil, FindEventsOutput{}, fmt.Errorf("condition %q needs threshold", input.Condition)
		}
		cond.Threshold = *input.Threshold
	case "between", "outside":
		if input.Lower == nil || input.Upper == nil {
			return nil, FindEventsOutput{}, fmt.Errorf("condition %q needs both lower and upper", input.Condition)
		}
		cond.Lower, cond.Upper = *input.Lower, *input.Upper
	}

	limit := input.Limit
	if limit <= 0 {
		limit = findEventsDefaultLimit
	}
	if limit > findEventsMaxLimit {
		limit = findEventsMaxLimit
	}

	n, rows, err := analyzedRows(input.ID, input.Start, input.End)
	if err != nil {
		return nil, FindEventsOutput{}, err
	}
	if len(rows) == 0 {
		return nil, FindEventsOutput{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to scan", input.ID)
	}
	rep, err := analysis.Events(rows, cond)
	if err != nil {
		return nil, FindEventsOutput{}, err
	}

	span := analysis.Span(rows)
	spanMS := rows[len(rows)-1].Timestamp - rows[0].Timestamp

	events := rep.Events
	if len(events) > limit {
		events = events[:limit]
	}
	eventsOut := make([]EventOut, len(events))
	singlePoint := 0
	for i, e := range events {
		out := EventOut{
			Start:           renderMS(e.StartMS),
			End:             renderMS(e.EndMS),
			DurationSeconds: float64(e.EndMS-e.StartMS) / 1000,
			Direction:       e.Direction,
			PointCount:      e.PointCount,
			PeakValue:       e.PeakValue,
			PeakTimestamp:   renderMS(e.PeakMS),
		}
		if input.Condition != "between" {
			d := e.PeakDeviation
			out.PeakDeviation = &d
		}
		if e.PointCount == 1 {
			singlePoint++
		}
		eventsOut[i] = out
	}

	caveats := []string{}
	if singlePoint > 0 {
		caveats = append(caveats, fmt.Sprintf("%d returned events are single points; their durations read as zero because duration spans first to last matching sample", singlePoint))
	}

	out := FindEventsOutput{
		ID:                  n.ID(),
		RowCount:            len(rows),
		AnalyzedRange:       span,
		Condition:           input.Condition,
		Threshold:           input.Threshold,
		Lower:               input.Lower,
		Upper:               input.Upper,
		PointsMatching:      rep.PointsMatching,
		PctPoints:           float64(rep.PointsMatching) / float64(len(rows)) * 100,
		TimeInEventsSeconds: float64(rep.TimeInEventsMS) / 1000,
		TotalEvents:         len(rep.Events),
		Events:              eventsOut,
		Unit:                n.Info().Unit,
		Caveats:             caveats,
	}
	if spanMS > 0 {
		out.PctTime = float64(rep.TimeInEventsMS) / float64(spanMS) * 100
	}
	return nil, out, nil
}
