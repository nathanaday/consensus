package analysis

import (
	"fmt"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// IntegralStats describe the area under one window's curve.
type IntegralStats struct {
	RowCount             int
	IntegralValueSeconds float64
	DurationSeconds      float64
	TimeWeightedMean     float64
	LargeGapCount        int
	LargeGapSeconds      float64
}

// Integrate computes the trapezoidal integral of value over time. The result
// is in value-seconds; TimeWeightedMean is the integral divided by the window
// duration (the average that weights each reading by how long it held).
// Intervals longer than gapFactor times the median interval are still
// integrated as straight lines but counted so callers can flag them. Needs at
// least 2 distinct timestamps.
func Integrate(rows []dataset.Row) (IntegralStats, error) {
	if len(rows) < 2 {
		return IntegralStats{}, fmt.Errorf("integration needs at least 2 rows, have %d", len(rows))
	}
	s := sortedByTime(rows)
	durationMS := s[len(s)-1].Timestamp - s[0].Timestamp
	if durationMS == 0 {
		return IntegralStats{}, fmt.Errorf("all %d rows share one timestamp; integration needs distinct timestamps", len(s))
	}
	median := MedianIntervalMS(s)
	threshold := median * gapFactor

	var integral float64
	var gapCount int
	var gapMS int64
	for i := 1; i < len(s); i++ {
		dtMS := s[i].Timestamp - s[i-1].Timestamp
		if dtMS == 0 {
			continue
		}
		integral += (s[i].Value + s[i-1].Value) / 2 * float64(dtMS) / 1000
		if median > 0 && dtMS > threshold {
			gapCount++
			gapMS += dtMS
		}
	}
	duration := float64(durationMS) / 1000
	return IntegralStats{
		RowCount:             len(s),
		IntegralValueSeconds: integral,
		DurationSeconds:      duration,
		TimeWeightedMean:     integral / duration,
		LargeGapCount:        gapCount,
		LargeGapSeconds:      float64(gapMS) / 1000,
	}, nil
}
