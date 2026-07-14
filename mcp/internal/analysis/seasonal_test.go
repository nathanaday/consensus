package analysis

import (
	"math"
	"testing"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// seasonalFixture builds days*24 hourly rows whose value depends only on the
// hour of day (a perfect daily cycle), starting at midnight UTC.
func seasonalFixture(days int) []dataset.Row {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := make([]dataset.Row, 0, days*24)
	for d := 0; d < days; d++ {
		for h := 0; h < 24; h++ {
			ts := start + int64(d*24+h)*3600*1000
			rows = append(rows, dataset.Row{Timestamp: ts, Value: float64(h)})
		}
	}
	return rows
}

func TestSeasonalPerfectDailyCycle(t *testing.T) {
	rep, err := Seasonal(seasonalFixture(3), "hour_of_day")
	if err != nil {
		t.Fatalf("Seasonal: %v", err)
	}
	if len(rep.Positions) != 24 {
		t.Fatalf("want 24 positions, got %d", len(rep.Positions))
	}
	for _, p := range rep.Positions {
		if p.Count != 3 {
			t.Errorf("position %d: want 3 samples, got %d", p.Index, p.Count)
		}
		if p.Mean != float64(p.Index) || p.Min != p.Max {
			t.Errorf("position %d: want constant value %d, got %+v", p.Index, p.Index, p)
		}
	}
	if !rep.CycleStrengthOK || math.Abs(rep.CycleStrength-1) > 1e-9 {
		t.Errorf("perfect cycle should have strength 1, got %g (ok=%v)", rep.CycleStrength, rep.CycleStrengthOK)
	}
	if rep.Positions[5].Label != "05:00" {
		t.Errorf("want label 05:00, got %q", rep.Positions[5].Label)
	}
	// 3 days minus one hour of span.
	if rep.SpanPeriods < 2.9 || rep.SpanPeriods > 3.0 {
		t.Errorf("want about 3 periods, got %g", rep.SpanPeriods)
	}
}

func TestSeasonalNoCycleWeakStrength(t *testing.T) {
	// A pure linear ramp across 3 days: hour-of-day explains little variance.
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	rows := make([]dataset.Row, 72)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: start + int64(i)*3600*1000, Value: float64(i)}
	}
	rep, err := Seasonal(rows, "hour_of_day")
	if err != nil {
		t.Fatalf("Seasonal: %v", err)
	}
	if !rep.CycleStrengthOK || rep.CycleStrength > 0.2 {
		t.Errorf("ramp should have weak cycle strength, got %g", rep.CycleStrength)
	}
}

func TestSeasonalDayOfWeekLabels(t *testing.T) {
	// 2026-07-01 is a Wednesday.
	start := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	rows := []dataset.Row{{Timestamp: start, Value: 1}}
	rep, err := Seasonal(rows, "day_of_week")
	if err != nil {
		t.Fatalf("Seasonal: %v", err)
	}
	if len(rep.Positions) != 7 {
		t.Fatalf("want 7 positions, got %d", len(rep.Positions))
	}
	wed := rep.Positions[int(time.Wednesday)]
	if wed.Count != 1 || wed.Label != "Wednesday" {
		t.Errorf("want the single sample on Wednesday, got %+v", rep.Positions)
	}
}

func TestSeasonalConstantSeriesStrengthUndefined(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 5}, {Timestamp: 3600000, Value: 5},
	}
	rep, err := Seasonal(rows, "hour_of_day")
	if err != nil {
		t.Fatalf("Seasonal: %v", err)
	}
	if rep.CycleStrengthOK {
		t.Error("constant series should have undefined cycle strength")
	}
}

func TestSeasonalRejectsUnknownPeriod(t *testing.T) {
	if _, err := Seasonal([]dataset.Row{{Timestamp: 0, Value: 1}}, "phase_of_moon"); err == nil {
		t.Error("want error for unknown period")
	}
}
