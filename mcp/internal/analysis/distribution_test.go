package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestDistributionBinsAndPercentiles(t *testing.T) {
	// Values 0..99, one per minute: uniform, so quartiles are easy to check.
	rows := make([]dataset.Row, 100)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: float64(i)}
	}
	rep, err := Distribution(rows, 10)
	if err != nil {
		t.Fatalf("Distribution: %v", err)
	}
	if len(rep.Bins) != 10 {
		t.Fatalf("want 10 bins, got %d", len(rep.Bins))
	}
	total := 0
	for _, b := range rep.Bins {
		total += b.Count
	}
	if total != 100 {
		t.Errorf("bin counts sum to %d, want 100", total)
	}
	// The last bin must include the max value.
	if rep.Bins[9].Count != 10 {
		t.Errorf("last bin should hold 10 values, got %d", rep.Bins[9].Count)
	}
	if rep.Percentiles.P50 != 49.5 {
		t.Errorf("want median 49.5, got %g", rep.Percentiles.P50)
	}
	if rep.Percentiles.P25 != 24.75 || rep.Percentiles.P75 != 74.25 {
		t.Errorf("unexpected quartiles %+v", rep.Percentiles)
	}
	if rep.Min != 0 || rep.Max != 99 {
		t.Errorf("want min 0 max 99, got %g %g", rep.Min, rep.Max)
	}
}

func TestDistributionConstantSeriesSingleBin(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 7}, {Timestamp: 60000, Value: 7}, {Timestamp: 120000, Value: 7},
	}
	rep, err := Distribution(rows, 10)
	if err != nil {
		t.Fatalf("Distribution: %v", err)
	}
	if len(rep.Bins) != 1 || rep.Bins[0].Count != 3 {
		t.Fatalf("want a single bin of 3, got %+v", rep.Bins)
	}
	if rep.Stddev != 0 {
		t.Errorf("want stddev 0, got %g", rep.Stddev)
	}
}

func TestDistributionAutoBins(t *testing.T) {
	rows := make([]dataset.Row, 100)
	for i := range rows {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: float64(i)}
	}
	rep, err := Distribution(rows, 0)
	if err != nil {
		t.Fatalf("Distribution: %v", err)
	}
	// sqrt(100) = 10, within the [3, 12] clamp.
	if len(rep.Bins) != 10 {
		t.Errorf("want 10 auto bins, got %d", len(rep.Bins))
	}
	small := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 1, Value: 2}}
	rep, err = Distribution(small, 0)
	if err != nil {
		t.Fatalf("Distribution: %v", err)
	}
	if len(rep.Bins) != 3 {
		t.Errorf("want the 3-bin floor for tiny samples, got %d", len(rep.Bins))
	}
}

func TestDistributionEmptyRows(t *testing.T) {
	if _, err := Distribution(nil, 5); err == nil {
		t.Error("want error for empty rows")
	}
}
