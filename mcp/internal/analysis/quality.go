package analysis

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const (
	gapFactor   = 5  // a gap is an interval > gapFactor * median interval
	flatlineRun = 10 // a flatline is >= flatlineRun identical consecutive values
)

// Gap is an unusually long interval between consecutive points.
type Gap struct {
	StartMS    int64
	EndMS      int64
	DurationMS int64
}

// Flatline is a run of identical consecutive values.
type Flatline struct {
	StartMS    int64
	EndMS      int64
	Value      float64
	PointCount int
}

// QualityReport summarizes sampling regularity and data-health signals.
type QualityReport struct {
	RowCount            int
	MedianIntervalMS    int64
	IntervalP95MS       int64
	MinIntervalMS       int64
	MaxIntervalMS       int64
	Gaps                []Gap
	TotalGaps           int
	Flatlines           []Flatline
	TotalFlatlines      int
	DuplicateTimestamps int
}

// Quality reports interval statistics, gaps, flatlines, and duplicate
// timestamps. Needs at least 1 row.
func Quality(rows []dataset.Row) (QualityReport, error) {
	if len(rows) == 0 {
		return QualityReport{}, fmt.Errorf("data quality needs at least 1 row, have 0")
	}
	s := sortedByTime(rows)
	rep := QualityReport{RowCount: len(s)}

	if len(s) < 2 {
		return rep, nil
	}

	intervals := make([]float64, 0, len(s)-1)
	dups := 0
	for i := 1; i < len(s); i++ {
		dt := s[i].Timestamp - s[i-1].Timestamp
		if dt == 0 {
			dups++
			continue
		}
		intervals = append(intervals, float64(dt))
	}
	rep.DuplicateTimestamps = dups

	median := MedianIntervalMS(s)
	rep.MedianIntervalMS = median
	if len(intervals) > 0 {
		sorted := append([]float64(nil), intervals...)
		sort.Float64s(sorted)
		rep.MinIntervalMS = int64(sorted[0])
		rep.MaxIntervalMS = int64(sorted[len(sorted)-1])
		rep.IntervalP95MS = int64(quantile(sorted, 0.95))
	}

	// gaps
	threshold := median * gapFactor
	for i := 1; i < len(s); i++ {
		dt := s[i].Timestamp - s[i-1].Timestamp
		if median > 0 && dt > threshold {
			rep.Gaps = append(rep.Gaps, Gap{StartMS: s[i-1].Timestamp, EndMS: s[i].Timestamp, DurationMS: dt})
		}
	}
	rep.TotalGaps = len(rep.Gaps)
	sort.SliceStable(rep.Gaps, func(i, j int) bool { return rep.Gaps[i].DurationMS > rep.Gaps[j].DurationMS })

	// flatlines
	runStart := 0
	flush := func(end int) {
		count := end - runStart + 1
		if count >= flatlineRun {
			rep.Flatlines = append(rep.Flatlines, Flatline{
				StartMS: s[runStart].Timestamp, EndMS: s[end].Timestamp,
				Value: s[runStart].Value, PointCount: count,
			})
		}
	}
	for i := 1; i < len(s); i++ {
		if s[i].Value != s[i-1].Value {
			flush(i - 1)
			runStart = i
		}
	}
	flush(len(s) - 1)
	rep.TotalFlatlines = len(rep.Flatlines)
	sort.SliceStable(rep.Flatlines, func(i, j int) bool { return rep.Flatlines[i].PointCount > rep.Flatlines[j].PointCount })

	return rep, nil
}
