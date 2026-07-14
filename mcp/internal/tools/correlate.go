package tools

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

const correlateTargetBuckets = 100

type CorrelateInput struct {
	IDA    string `json:"id_a" jsonschema:"first dataset id"`
	IDB    string `json:"id_b" jsonschema:"second dataset id"`
	Start  string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC start; defaults to the overlap of the two datasets"`
	End    string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC end; defaults to the overlap of the two datasets"`
	Bucket string `json:"bucket,omitempty" jsonschema:"optional Go duration bucket width for alignment; omitted auto-picks a round width"`
}

type CorrelateOutput struct {
	IDA            string            `json:"id_a"`
	IDB            string            `json:"id_b"`
	RowCountA      int               `json:"row_count_a"`
	RowCountB      int               `json:"row_count_b"`
	AnalyzedRange  dataset.TimeRange `json:"analyzed_range"`
	Bucket         string            `json:"bucket"`
	AlignedSamples int               `json:"aligned_samples"`
	Pearson        *float64          `json:"pearson,omitempty"`
	Spearman       *float64          `json:"spearman,omitempty"`
	UnitA          string            `json:"unit_a,omitempty"`
	UnitB          string            `json:"unit_b,omitempty"`
	Caveats        []string          `json:"caveats"`
}

func firstLastMS(rows []dataset.Row) (int64, int64) {
	first, last := rows[0].Timestamp, rows[0].Timestamp
	for _, r := range rows {
		if r.Timestamp < first {
			first = r.Timestamp
		}
		if r.Timestamp > last {
			last = r.Timestamp
		}
	}
	return first, last
}

func windowMS(rows []dataset.Row, loMS, hiMS int64) []dataset.Row {
	out := make([]dataset.Row, 0, len(rows))
	for _, r := range rows {
		if r.Timestamp >= loMS && r.Timestamp <= hiMS {
			out = append(out, r)
		}
	}
	return out
}

// Correlate reports how two datasets move together over an aligned time grid.
// It returns statistics only, never row data.
func Correlate(ctx context.Context, req *mcp.CallToolRequest, input CorrelateInput) (*mcp.CallToolResult, CorrelateOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, CorrelateOutput{}, err
	}
	na, err := g.Node(input.IDA)
	if err != nil {
		return nil, CorrelateOutput{}, err
	}
	nb, err := g.Node(input.IDB)
	if err != nil {
		return nil, CorrelateOutput{}, err
	}
	rowsA, err := na.LoadData()
	if err != nil {
		return nil, CorrelateOutput{}, err
	}
	rowsB, err := nb.LoadData()
	if err != nil {
		return nil, CorrelateOutput{}, err
	}
	if len(rowsA) == 0 || len(rowsB) == 0 {
		return nil, CorrelateOutput{}, fmt.Errorf("both datasets must have rows; %q has %d, %q has %d", input.IDA, len(rowsA), input.IDB, len(rowsB))
	}

	firstA, lastA := firstLastMS(rowsA)
	firstB, lastB := firstLastMS(rowsB)

	loMS, hiMS := max64(firstA, firstB), min64(lastA, lastB)
	if input.Start != "" {
		t, perr := time.Parse(time.RFC3339, input.Start)
		if perr != nil {
			return nil, CorrelateOutput{}, fmt.Errorf("invalid start %q; expected an RFC3339 UTC timestamp", input.Start)
		}
		loMS = t.UnixMilli()
	}
	if input.End != "" {
		t, perr := time.Parse(time.RFC3339, input.End)
		if perr != nil {
			return nil, CorrelateOutput{}, fmt.Errorf("invalid end %q; expected an RFC3339 UTC timestamp", input.End)
		}
		hiMS = t.UnixMilli()
	}
	if loMS > hiMS {
		return nil, CorrelateOutput{}, fmt.Errorf("datasets do not overlap in time: %q spans %s..%s, %q spans %s..%s", input.IDA, renderMS(firstA), renderMS(lastA), input.IDB, renderMS(firstB), renderMS(lastB))
	}

	wa := windowMS(rowsA, loMS, hiMS)
	wb := windowMS(rowsB, loMS, hiMS)
	if len(wa) == 0 || len(wb) == 0 {
		return nil, CorrelateOutput{}, fmt.Errorf("no overlapping rows in the analyzed window for %q and %q", input.IDA, input.IDB)
	}

	var widthMS int64
	if input.Bucket == "" {
		union := append(append([]dataset.Row{}, wa...), wb...)
		widthMS, err = analysis.AutoWidthMS(union, correlateTargetBuckets)
		if err != nil {
			return nil, CorrelateOutput{}, err
		}
	} else {
		widthMS, err = parseBucketMS(input.Bucket)
		if err != nil {
			return nil, CorrelateOutput{}, err
		}
	}

	rep, err := analysis.Correlate(wa, wb, loMS, widthMS)
	if err != nil {
		return nil, CorrelateOutput{}, err
	}

	caveats := []string{}
	if rep.AlignedSamples < 10 {
		caveats = append(caveats, fmt.Sprintf("only %d aligned samples; correlation may be unreliable", rep.AlignedSamples))
	}
	if rep.AlignedSamples >= 2 && (!rep.PearsonOK || !rep.SpearmanOK) {
		caveats = append(caveats, "correlation is undefined for a constant series; a coefficient is omitted")
	}

	out := CorrelateOutput{
		IDA:            na.ID(),
		IDB:            nb.ID(),
		RowCountA:      len(wa),
		RowCountB:      len(wb),
		AnalyzedRange:  dataset.TimeRange{Start: renderMS(loMS), End: renderMS(hiMS)},
		Bucket:         (time.Duration(widthMS) * time.Millisecond).String(),
		AlignedSamples: rep.AlignedSamples,
		UnitA:          na.Info().Unit,
		UnitB:          nb.Info().Unit,
		Caveats:        caveats,
	}
	if rep.PearsonOK {
		p := round6(rep.Pearson)
		out.Pearson = &p
	}
	if rep.SpearmanOK {
		s := round6(rep.Spearman)
		out.Spearman = &s
	}
	return nil, out, nil
}

// round6 rounds a correlation to 6 decimals so a perfect fit that lands an ULP
// short of 1 (float rounding) reports cleanly as 1.
func round6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
