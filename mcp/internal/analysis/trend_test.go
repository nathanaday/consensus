package analysis

import (
	"math"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestTrendPerfectRamp(t *testing.T) {
	// +1 unit per minute for 10 minutes -> slope 60/hour, 1440/day, R²=1.
	rows := make([]dataset.Row, 11)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: float64(i)}
	}
	got, err := Trend(rows)
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}
	if math.Abs(got.SlopePerHour-60) > 1e-6 {
		t.Errorf("want 60/hour, got %g", got.SlopePerHour)
	}
	if math.Abs(got.SlopePerDay-1440) > 1e-3 {
		t.Errorf("want 1440/day, got %g", got.SlopePerDay)
	}
	if math.Abs(got.RSquared-1) > 1e-9 {
		t.Errorf("want R²=1, got %g", got.RSquared)
	}
	if got.Direction != "increasing" {
		t.Errorf("want increasing, got %q", got.Direction)
	}
}

func TestTrendFlatWhenNoisyAroundConstant(t *testing.T) {
	// alternating +/-1 around a constant -> slope near zero, CI includes 0.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 10}, {Timestamp: 60000, Value: 11},
		{Timestamp: 120000, Value: 9}, {Timestamp: 180000, Value: 11},
		{Timestamp: 240000, Value: 9}, {Timestamp: 300000, Value: 10},
	}
	got, err := Trend(rows)
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}
	if got.Direction != "flat" {
		t.Errorf("want flat, got %q (slope/hour %g)", got.Direction, got.SlopePerHour)
	}
}

func TestTrendErrorsOnSingleTimestamp(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 0, Value: 2}}
	if _, err := Trend(rows); err == nil {
		t.Error("all-equal timestamps should error")
	}
}
