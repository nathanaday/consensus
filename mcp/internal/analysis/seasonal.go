package analysis

import (
	"fmt"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// SeasonalPosition is one cycle position's aggregates (e.g. one hour of the
// day). An empty position has Count 0 and zero stats.
type SeasonalPosition struct {
	Index int
	Label string
	Count int
	Mean  float64
	Min   float64
	Max   float64
}

// SeasonalReport summarizes how values vary across a repeating cycle.
// CycleStrength is the share of total variance explained by the cycle
// position (eta squared, 0..1); CycleStrengthOK is false when the series is
// constant and the share is undefined.
type SeasonalReport struct {
	Period          string
	Positions       []SeasonalPosition
	CycleStrength   float64
	CycleStrengthOK bool
	SpanPeriods     float64
}

// seasonalPeriods maps a period name to its position count, label renderer,
// and cycle length in milliseconds. Timestamps are interpreted as UTC.
func seasonalPeriod(period string) (int, func(t time.Time) int, func(i int) string, int64, error) {
	switch period {
	case "hour_of_day":
		return 24,
			func(t time.Time) int { return t.Hour() },
			func(i int) string { return fmt.Sprintf("%02d:00", i) },
			24 * 3600 * 1000, nil
	case "day_of_week":
		return 7,
			func(t time.Time) int { return int(t.Weekday()) },
			func(i int) string { return time.Weekday(i).String() },
			7 * 24 * 3600 * 1000, nil
	default:
		return 0, nil, nil, 0, fmt.Errorf("unknown period %q; use hour_of_day or day_of_week", period)
	}
}

// Seasonal groups values by their position in a repeating cycle (hour of day
// or day of week, UTC) and reports per-position statistics plus how much of
// the total variance the cycle explains.
func Seasonal(rows []dataset.Row, period string) (SeasonalReport, error) {
	nPos, posOf, labelOf, periodMS, err := seasonalPeriod(period)
	if err != nil {
		return SeasonalReport{}, err
	}
	if len(rows) == 0 {
		return SeasonalReport{}, fmt.Errorf("seasonal profile needs at least 1 row, have 0")
	}
	s := sortedByTime(rows)

	sums := make([]float64, nPos)
	counts := make([]int, nPos)
	mins := make([]float64, nPos)
	maxs := make([]float64, nPos)
	var grand float64
	for _, r := range s {
		p := posOf(time.UnixMilli(r.Timestamp).UTC())
		if counts[p] == 0 {
			mins[p], maxs[p] = r.Value, r.Value
		} else {
			if r.Value < mins[p] {
				mins[p] = r.Value
			}
			if r.Value > maxs[p] {
				maxs[p] = r.Value
			}
		}
		sums[p] += r.Value
		counts[p]++
		grand += r.Value
	}
	grandMean := grand / float64(len(s))

	positions := make([]SeasonalPosition, nPos)
	var ssBetween float64
	for i := range positions {
		positions[i] = SeasonalPosition{Index: i, Label: labelOf(i), Count: counts[i]}
		if counts[i] > 0 {
			mean := sums[i] / float64(counts[i])
			positions[i].Mean = mean
			positions[i].Min = mins[i]
			positions[i].Max = maxs[i]
			d := mean - grandMean
			ssBetween += float64(counts[i]) * d * d
		}
	}

	var ssTotal float64
	for _, r := range s {
		d := r.Value - grandMean
		ssTotal += d * d
	}

	rep := SeasonalReport{
		Period:      period,
		Positions:   positions,
		SpanPeriods: float64(s[len(s)-1].Timestamp-s[0].Timestamp) / float64(periodMS),
	}
	if ssTotal > 0 {
		rep.CycleStrength = ssBetween / ssTotal
		rep.CycleStrengthOK = true
	}
	return rep, nil
}
