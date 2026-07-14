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

const (
	findLagTargetBuckets  = 200
	findLagDefaultShare   = 4 // default max shift is 1/4 of the window's buckets
	findLagMaxShiftBucket = 500
)

type FindLagInput struct {
	IDA    string `json:"id_a" jsonschema:"first dataset id (the candidate leader)"`
	IDB    string `json:"id_b" jsonschema:"second dataset id (the candidate follower)"`
	Start  string `json:"start,omitempty" jsonschema:"optional RFC3339 UTC start; defaults to the overlap of the two datasets"`
	End    string `json:"end,omitempty" jsonschema:"optional RFC3339 UTC end; defaults to the overlap of the two datasets"`
	Bucket string `json:"bucket,omitempty" jsonschema:"optional Go duration bucket width for alignment; omitted auto-picks a round width. The lag estimate resolves to whole buckets"`
	MaxLag string `json:"max_lag,omitempty" jsonschema:"optional Go duration bounding the scan in each direction; omitted scans a quarter of the window"`
}

type FindLagOutput struct {
	IDA                     string            `json:"id_a"`
	IDB                     string            `json:"id_b"`
	AnalyzedRange           dataset.TimeRange `json:"analyzed_range"`
	Bucket                  string            `json:"bucket"`
	MaxLagScannedSeconds    float64           `json:"max_lag_scanned_seconds"`
	BestLagSeconds          *float64          `json:"best_lag_seconds,omitempty"`
	PearsonAtBestLag        *float64          `json:"pearson_at_best_lag,omitempty"`
	AlignedSamplesAtBestLag int               `json:"aligned_samples_at_best_lag"`
	PearsonAtZeroLag        *float64          `json:"pearson_at_zero_lag,omitempty"`
	AlignedSamplesAtZeroLag int               `json:"aligned_samples_at_zero_lag"`
	UnitA                   string            `json:"unit_a,omitempty"`
	UnitB                   string            `json:"unit_b,omitempty"`
	Caveats                 []string          `json:"caveats"`
}

// FindLag scans time offsets between two datasets and reports the lag where
// they correlate most strongly. A positive best lag means id_b follows id_a
// by that many seconds. It returns statistics only, never row data.
func FindLag(ctx context.Context, req *mcp.CallToolRequest, input FindLagInput) (*mcp.CallToolResult, FindLagOutput, error) {
	wa, wb, loMS, hiMS, na, nb, err := overlappingRows(input.IDA, input.IDB, input.Start, input.End)
	if err != nil {
		return nil, FindLagOutput{}, err
	}

	var widthMS int64
	if input.Bucket == "" {
		union := append(append([]dataset.Row{}, wa...), wb...)
		widthMS, err = analysis.AutoWidthMS(union, findLagTargetBuckets)
		if err != nil {
			return nil, FindLagOutput{}, err
		}
	} else {
		widthMS, err = parseBucketMS(input.Bucket)
		if err != nil {
			return nil, FindLagOutput{}, err
		}
	}

	windowBuckets := (hiMS-loMS)/widthMS + 1
	maxShift := int(windowBuckets / findLagDefaultShare)
	if input.MaxLag != "" {
		lagMS, perr := parseBucketMS(input.MaxLag)
		if perr != nil {
			return nil, FindLagOutput{}, fmt.Errorf("invalid max_lag: %w", perr)
		}
		maxShift = int((lagMS + widthMS - 1) / widthMS)
	}
	if maxShift < 1 {
		maxShift = 1
	}
	if maxShift > findLagMaxShiftBucket {
		maxShift = findLagMaxShiftBucket
	}

	rep, err := analysis.LagScan(wa, wb, loMS, widthMS, maxShift)
	if err != nil {
		return nil, FindLagOutput{}, err
	}

	caveats := []string{}
	out := FindLagOutput{
		IDA:                     na.ID(),
		IDB:                     nb.ID(),
		AnalyzedRange:           dataset.TimeRange{Start: renderMS(loMS), End: renderMS(hiMS)},
		Bucket:                  (time.Duration(widthMS) * time.Millisecond).String(),
		MaxLagScannedSeconds:    float64(int64(maxShift)*widthMS) / 1000,
		AlignedSamplesAtBestLag: rep.AlignedAtBest,
		AlignedSamplesAtZeroLag: rep.AlignedAtZero,
		UnitA:                   na.Info().Unit,
		UnitB:                   nb.Info().Unit,
	}
	if rep.BestOK {
		lag := float64(int64(rep.BestShift)*widthMS) / 1000
		p := round6(rep.BestPearson)
		out.BestLagSeconds = &lag
		out.PearsonAtBestLag = &p
		if math.Abs(rep.BestPearson) < 0.3 {
			caveats = append(caveats, "even the best lag correlates weakly; the series may simply not track each other")
		}
		if rep.BestShift == maxShift || rep.BestShift == -maxShift {
			caveats = append(caveats, "the best lag sits at the edge of the scanned range; the true lag may be larger — retry with a bigger max_lag")
		}
		if rep.AlignedAtBest < 10 {
			caveats = append(caveats, fmt.Sprintf("only %d aligned samples at the best lag; the estimate may be unreliable", rep.AlignedAtBest))
		}
	} else {
		caveats = append(caveats, "no scanned lag produced a defined correlation (too few aligned samples or a constant series)")
	}
	if rep.ZeroOK {
		z := round6(rep.ZeroPearson)
		out.PearsonAtZeroLag = &z
	}
	out.Caveats = caveats
	return nil, out, nil
}

// overlappingRows loads two datasets and windows both to their shared span
// (optionally narrowed by start/end). Shared by correlate and find_lag.
func overlappingRows(idA, idB, start, end string) (wa, wb []dataset.Row, loMS, hiMS int64, na, nb *lineage.Node, err error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, nil, 0, 0, nil, nil, err
	}
	na, err = g.Node(idA)
	if err != nil {
		return nil, nil, 0, 0, nil, nil, err
	}
	nb, err = g.Node(idB)
	if err != nil {
		return nil, nil, 0, 0, nil, nil, err
	}
	rowsA, err := na.LoadData()
	if err != nil {
		return nil, nil, 0, 0, nil, nil, err
	}
	rowsB, err := nb.LoadData()
	if err != nil {
		return nil, nil, 0, 0, nil, nil, err
	}
	if len(rowsA) == 0 || len(rowsB) == 0 {
		return nil, nil, 0, 0, nil, nil, fmt.Errorf("both datasets must have rows; %q has %d, %q has %d", idA, len(rowsA), idB, len(rowsB))
	}

	firstA, lastA := firstLastMS(rowsA)
	firstB, lastB := firstLastMS(rowsB)
	loMS, hiMS = max64(firstA, firstB), min64(lastA, lastB)
	if start != "" {
		t, perr := time.Parse(time.RFC3339, start)
		if perr != nil {
			return nil, nil, 0, 0, nil, nil, fmt.Errorf("invalid start %q; expected an RFC3339 UTC timestamp", start)
		}
		loMS = t.UnixMilli()
	}
	if end != "" {
		t, perr := time.Parse(time.RFC3339, end)
		if perr != nil {
			return nil, nil, 0, 0, nil, nil, fmt.Errorf("invalid end %q; expected an RFC3339 UTC timestamp", end)
		}
		hiMS = t.UnixMilli()
	}
	if loMS > hiMS {
		return nil, nil, 0, 0, nil, nil, fmt.Errorf("datasets do not overlap in time: %q spans %s..%s, %q spans %s..%s", idA, renderMS(firstA), renderMS(lastA), idB, renderMS(firstB), renderMS(lastB))
	}

	wa = windowMS(rowsA, loMS, hiMS)
	wb = windowMS(rowsB, loMS, hiMS)
	if len(wa) == 0 || len(wb) == 0 {
		return nil, nil, 0, 0, nil, nil, fmt.Errorf("no overlapping rows in the analyzed window for %q and %q", idA, idB)
	}
	return wa, wb, loMS, hiMS, na, nb, nil
}
