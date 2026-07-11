package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// RatePoint is a rate of change paired with the timestamp of the interval's
// end point.
type RatePoint struct {
	Rate      float64 `json:"rate"`
	Timestamp string  `json:"timestamp"`
}

// RateStats describe the pairwise derivative of one window of rows.
type RateStats struct {
	RowCount              int
	MaxRise               RatePoint
	MaxFall               RatePoint
	MeanAbsRate           float64
	MedianIntervalSeconds float64
}

// Rates computes value-per-second between consecutive points. Pairs with a
// zero time delta have no defined rate and are skipped.
func Rates(rows []dataset.Row) (RateStats, error) {
	if len(rows) < 2 {
		return RateStats{}, fmt.Errorf("rate of change needs at least 2 rows, have %d", len(rows))
	}
	s := sortedByTime(rows)
	var intervals []float64
	var sumAbs float64
	count := 0
	maxRise := RatePoint{Rate: math.Inf(-1)}
	maxFall := RatePoint{Rate: math.Inf(1)}
	for i := 1; i < len(s); i++ {
		dtMS := s[i].Timestamp - s[i-1].Timestamp
		if dtMS == 0 {
			continue
		}
		dt := float64(dtMS) / 1000
		rate := (s[i].Value - s[i-1].Value) / dt
		count++
		intervals = append(intervals, dt)
		sumAbs += math.Abs(rate)
		if rate > maxRise.Rate {
			maxRise = RatePoint{Rate: rate, Timestamp: formatTS(s[i].Timestamp)}
		}
		if rate < maxFall.Rate {
			maxFall = RatePoint{Rate: rate, Timestamp: formatTS(s[i].Timestamp)}
		}
	}
	if count == 0 {
		return RateStats{}, fmt.Errorf("all %d rows share one timestamp; rate of change needs distinct timestamps", len(s))
	}
	sort.Float64s(intervals)
	return RateStats{
		RowCount:              len(s),
		MaxRise:               maxRise,
		MaxFall:               maxFall,
		MeanAbsRate:           sumAbs / float64(count),
		MedianIntervalSeconds: quantile(intervals, 0.5),
	}, nil
}
