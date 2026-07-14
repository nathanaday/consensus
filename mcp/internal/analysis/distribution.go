package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Percentiles are standard order statistics of one window's values.
type Percentiles struct {
	P05 float64 `json:"p05"`
	P10 float64 `json:"p10"`
	P25 float64 `json:"p25"`
	P50 float64 `json:"p50"`
	P75 float64 `json:"p75"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// HistBin is one histogram bin. Bins are half-open [Lower, Upper) except the
// last, which includes the maximum value.
type HistBin struct {
	Lower float64
	Upper float64
	Count int
}

// DistributionReport describes how one window's values are distributed.
type DistributionReport struct {
	RowCount    int
	Min         float64
	Max         float64
	Mean        float64
	Stddev      float64
	Percentiles Percentiles
	BinWidth    float64
	Bins        []HistBin
}

// autoBins picks a bin count from the sample size: roughly sqrt(n), clamped
// to [3, 12] so small and huge windows both stay readable.
func autoBins(n int) int {
	b := int(math.Round(math.Sqrt(float64(n))))
	if b < 3 {
		b = 3
	}
	if b > 12 {
		b = 12
	}
	return b
}

// Distribution computes percentiles and an equal-width histogram. bins <= 0
// auto-picks from the sample size. A constant series collapses to one bin.
func Distribution(rows []dataset.Row, bins int) (DistributionReport, error) {
	if len(rows) == 0 {
		return DistributionReport{}, fmt.Errorf("distribution needs at least 1 row, have 0")
	}
	if bins <= 0 {
		bins = autoBins(len(rows))
	}
	values := make([]float64, len(rows))
	var sum float64
	for i, r := range rows {
		values[i] = r.Value
		sum += r.Value
	}
	sort.Float64s(values)
	mean := sum / float64(len(values))
	var sq float64
	for _, v := range values {
		d := v - mean
		sq += d * d
	}
	minV, maxV := values[0], values[len(values)-1]

	rep := DistributionReport{
		RowCount: len(values),
		Min:      minV,
		Max:      maxV,
		Mean:     mean,
		Stddev:   math.Sqrt(sq / float64(len(values))),
		Percentiles: Percentiles{
			P05: quantile(values, 0.05), P10: quantile(values, 0.10),
			P25: quantile(values, 0.25), P50: quantile(values, 0.50),
			P75: quantile(values, 0.75), P90: quantile(values, 0.90),
			P95: quantile(values, 0.95), P99: quantile(values, 0.99),
		},
	}

	if minV == maxV {
		rep.Bins = []HistBin{{Lower: minV, Upper: maxV, Count: len(values)}}
		return rep, nil
	}
	width := (maxV - minV) / float64(bins)
	rep.BinWidth = width
	rep.Bins = make([]HistBin, bins)
	for i := range rep.Bins {
		rep.Bins[i] = HistBin{Lower: minV + float64(i)*width, Upper: minV + float64(i+1)*width}
	}
	rep.Bins[bins-1].Upper = maxV
	for _, v := range values {
		idx := int((v - minV) / width)
		if idx >= bins {
			idx = bins - 1
		}
		rep.Bins[idx].Count++
	}
	return rep, nil
}
