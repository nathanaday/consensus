package tools

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
)

type ValueAtInput struct {
	ID          string `json:"id" jsonschema:"the dataset id to look up"`
	At          string `json:"at" jsonschema:"the RFC3339 UTC instant to look up"`
	MaxDistance string `json:"max_distance,omitempty" jsonschema:"optional Go duration like 5m; error if the nearest sample is further from the instant than this"`
}

type SampleOut struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type ValueAtOutput struct {
	ID            string    `json:"id"`
	At            string    `json:"at"`
	Nearest       SampleOut `json:"nearest"`
	OffsetSeconds float64   `json:"offset_seconds"`
	Interpolated  *float64  `json:"interpolated,omitempty"`
	Unit          string    `json:"unit,omitempty"`
	Caveats       []string  `json:"caveats"`
}

// ValueAt reports the sample nearest to one instant plus the linearly
// interpolated value there. It returns a single point, never bulk row data.
func ValueAt(ctx context.Context, req *mcp.CallToolRequest, input ValueAtInput) (*mcp.CallToolResult, ValueAtOutput, error) {
	at, err := time.Parse(time.RFC3339, input.At)
	if err != nil {
		return nil, ValueAtOutput{}, fmt.Errorf("invalid at %q; expected an RFC3339 UTC timestamp like 2026-07-07T00:00:00Z", input.At)
	}
	n, rows, err := analyzedRows(input.ID, "", "")
	if err != nil {
		return nil, ValueAtOutput{}, err
	}
	if len(rows) == 0 {
		return nil, ValueAtOutput{}, fmt.Errorf("dataset %q has no rows; nothing to look up", input.ID)
	}
	rep, err := analysis.At(rows, at.UnixMilli())
	if err != nil {
		return nil, ValueAtOutput{}, err
	}

	offsetSec := float64(rep.OffsetMS) / 1000
	if input.MaxDistance != "" {
		maxMS, perr := parseBucketMS(input.MaxDistance)
		if perr != nil {
			return nil, ValueAtOutput{}, fmt.Errorf("invalid max_distance: %w", perr)
		}
		absMS := rep.OffsetMS
		if absMS < 0 {
			absMS = -absMS
		}
		if absMS > maxMS {
			return nil, ValueAtOutput{}, fmt.Errorf("the nearest sample is %.0fs away from %s, further than max_distance %s", math.Abs(offsetSec), input.At, input.MaxDistance)
		}
	}

	caveats := []string{}
	if !rep.InterpolatedOK {
		caveats = append(caveats, "the requested instant is outside the dataset's span; no interpolated value")
	}
	if med := analysis.MedianIntervalMS(rows); med > 0 && math.Abs(float64(rep.OffsetMS)) > float64(med) {
		caveats = append(caveats, fmt.Sprintf("the nearest sample is %.0fs away, more than the median sampling interval of %.0fs", math.Abs(offsetSec), float64(med)/1000))
	}

	out := ValueAtOutput{
		ID:            n.ID(),
		At:            at.UTC().Format(time.RFC3339Nano),
		Nearest:       SampleOut{Timestamp: renderMS(rep.Nearest.Timestamp), Value: rep.Nearest.Value},
		OffsetSeconds: offsetSec,
		Unit:          n.Info().Unit,
		Caveats:       caveats,
	}
	if rep.InterpolatedOK {
		v := rep.Interpolated
		out.Interpolated = &v
	}
	return nil, out, nil
}
