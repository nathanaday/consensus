package analysis

import (
	"fmt"
	"math"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// TrendStats describe an ordinary-least-squares fit of value against time.
type TrendStats struct {
	RowCount        int
	SlopePerHour    float64
	SlopePerDay     float64
	RSquared        float64
	Direction       string // "increasing", "decreasing", or "flat"
	DurationSeconds float64
}

// Trend fits value against seconds-since-start by OLS. Direction is "flat"
// when the slope's 95% confidence interval includes zero. Needs at least 2
// distinct timestamps.
func Trend(rows []dataset.Row) (TrendStats, error) {
	if len(rows) < 2 {
		return TrendStats{}, fmt.Errorf("trend needs at least 2 rows, have %d", len(rows))
	}
	s := sortedByTime(rows)
	n := float64(len(s))
	origin := s[0].Timestamp

	xs := make([]float64, len(s))
	ys := make([]float64, len(s))
	var sx, sy float64
	for i, r := range s {
		xs[i] = float64(r.Timestamp-origin) / 1000
		ys[i] = r.Value
		sx += xs[i]
		sy += ys[i]
	}
	xbar, ybar := sx/n, sy/n

	var sxx, sxy, syy float64
	for i := range s {
		dx := xs[i] - xbar
		dy := ys[i] - ybar
		sxx += dx * dx
		sxy += dx * dy
		syy += dy * dy
	}
	if sxx == 0 {
		return TrendStats{}, fmt.Errorf("all %d rows share one timestamp; trend needs distinct timestamps", len(s))
	}

	slope := sxy / sxx
	r2 := 0.0
	if syy > 0 {
		r := sxy / math.Sqrt(sxx*syy)
		r2 = r * r
	}

	direction := "flat"
	if len(s) >= 3 {
		var resid float64
		for i := range s {
			pred := ybar + slope*(xs[i]-xbar)
			e := ys[i] - pred
			resid += e * e
		}
		se := math.Sqrt((resid / (n - 2)) / sxx)
		lo := slope - 1.96*se
		hi := slope + 1.96*se
		if lo > 0 {
			direction = "increasing"
		} else if hi < 0 {
			direction = "decreasing"
		}
	}

	return TrendStats{
		RowCount:        len(s),
		SlopePerHour:    slope * 3600,
		SlopePerDay:     slope * 86400,
		RSquared:        r2,
		Direction:       direction,
		DurationSeconds: xs[len(xs)-1],
	}, nil
}
