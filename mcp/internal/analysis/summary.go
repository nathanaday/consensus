package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Extreme is a value paired with the timestamp it occurred at.
type Extreme struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

// SummaryStats are descriptive statistics for one window of rows.
type SummaryStats struct {
	RowCount int
	Min      Extreme
	Max      Extreme
	Mean     float64
	Median   float64
	Stddev   float64 // population standard deviation
}

// Summary computes descriptive statistics. Ties on min/max keep the
// earliest occurrence.
func Summary(rows []dataset.Row) (SummaryStats, error) {
	if len(rows) == 0 {
		return SummaryStats{}, fmt.Errorf("summary needs at least 1 row, have 0")
	}
	s := sortedByTime(rows)
	minRow, maxRow := s[0], s[0]
	var sum float64
	for _, r := range s {
		if r.Value < minRow.Value {
			minRow = r
		}
		if r.Value > maxRow.Value {
			maxRow = r
		}
		sum += r.Value
	}
	mean := sum / float64(len(s))
	values := make([]float64, len(s))
	var sq float64
	for i, r := range s {
		values[i] = r.Value
		d := r.Value - mean
		sq += d * d
	}
	sort.Float64s(values)
	return SummaryStats{
		RowCount: len(s),
		Min:      Extreme{Value: minRow.Value, Timestamp: formatTS(minRow.Timestamp)},
		Max:      Extreme{Value: maxRow.Value, Timestamp: formatTS(maxRow.Timestamp)},
		Mean:     mean,
		Median:   quantile(values, 0.5),
		Stddev:   math.Sqrt(sq / float64(len(s))),
	}, nil
}
