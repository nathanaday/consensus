package analysis

import (
	"math"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestIntegrateConstantPower(t *testing.T) {
	// A constant 2.0 held for one hour integrates to 7200 value-seconds.
	rows := []dataset.Row{}
	for i := 0; i <= 60; i++ {
		rows = append(rows, dataset.Row{Timestamp: int64(i) * 60000, Value: 2})
	}
	stats, err := Integrate(rows)
	if err != nil {
		t.Fatalf("Integrate: %v", err)
	}
	if math.Abs(stats.IntegralValueSeconds-7200) > 1e-9 {
		t.Errorf("want 7200 value-seconds, got %g", stats.IntegralValueSeconds)
	}
	if math.Abs(stats.TimeWeightedMean-2) > 1e-9 {
		t.Errorf("want time-weighted mean 2, got %g", stats.TimeWeightedMean)
	}
	if stats.DurationSeconds != 3600 {
		t.Errorf("want duration 3600s, got %g", stats.DurationSeconds)
	}
}

func TestIntegrateTrapezoidRamp(t *testing.T) {
	// A ramp 0 -> 10 over 10 seconds: area is 10*10/2 = 50.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 0},
		{Timestamp: 10000, Value: 10},
	}
	stats, err := Integrate(rows)
	if err != nil {
		t.Fatalf("Integrate: %v", err)
	}
	if math.Abs(stats.IntegralValueSeconds-50) > 1e-9 {
		t.Errorf("want 50 value-seconds, got %g", stats.IntegralValueSeconds)
	}
	if math.Abs(stats.TimeWeightedMean-5) > 1e-9 {
		t.Errorf("want time-weighted mean 5, got %g", stats.TimeWeightedMean)
	}
}

func TestIntegrateTimeWeightedMeanDiffersFromPointMean(t *testing.T) {
	// A value held at 10 for an hour then briefly at 0: the point mean is 5
	// but the time-weighted mean stays near 10.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 10},
		{Timestamp: 3600000, Value: 10},
		{Timestamp: 3601000, Value: 0},
		{Timestamp: 3602000, Value: 0},
	}
	stats, err := Integrate(rows)
	if err != nil {
		t.Fatalf("Integrate: %v", err)
	}
	if stats.TimeWeightedMean < 9.9 {
		t.Errorf("want time-weighted mean near 10, got %g", stats.TimeWeightedMean)
	}
}

func TestIntegrateCountsLargeGaps(t *testing.T) {
	// 1-minute spacing with one 30-minute hole.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 1},
		{Timestamp: 120000, Value: 1},
		{Timestamp: 120000 + 30*60000, Value: 1},
		{Timestamp: 120000 + 31*60000, Value: 1},
	}
	stats, err := Integrate(rows)
	if err != nil {
		t.Fatalf("Integrate: %v", err)
	}
	if stats.LargeGapCount != 1 {
		t.Errorf("want 1 large gap, got %d", stats.LargeGapCount)
	}
	if stats.LargeGapSeconds != 1800 {
		t.Errorf("want 1800s of gap, got %g", stats.LargeGapSeconds)
	}
}

func TestIntegrateRejectsDegenerateInput(t *testing.T) {
	if _, err := Integrate([]dataset.Row{{Timestamp: 0, Value: 1}}); err == nil {
		t.Error("want error for a single row")
	}
	dup := []dataset.Row{{Timestamp: 5, Value: 1}, {Timestamp: 5, Value: 2}}
	if _, err := Integrate(dup); err == nil {
		t.Error("want error when all rows share one timestamp")
	}
}
