package analysis

import (
	"strings"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestRatesComputesPairwiseDerivative(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 0},
		{Timestamp: 1000, Value: 10},
		{Timestamp: 3000, Value: 5},
	}
	got, err := Rates(rows)
	if err != nil {
		t.Fatalf("Rates: %v", err)
	}
	if got.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", got.RowCount)
	}
	// rises 10/s over [0,1s]; falls 2.5/s over [1s,3s]
	if got.MaxRise != (RatePoint{Rate: 10, Timestamp: "1970-01-01T00:00:01Z"}) {
		t.Errorf("MaxRise = %+v", got.MaxRise)
	}
	if got.MaxFall != (RatePoint{Rate: -2.5, Timestamp: "1970-01-01T00:00:03Z"}) {
		t.Errorf("MaxFall = %+v", got.MaxFall)
	}
	if got.MeanAbsRate != 6.25 {
		t.Errorf("MeanAbsRate = %v, want 6.25", got.MeanAbsRate)
	}
	if got.MedianIntervalSeconds != 1.5 {
		t.Errorf("MedianIntervalSeconds = %v, want 1.5", got.MedianIntervalSeconds)
	}
}

func TestRatesSortsOutOfOrderInput(t *testing.T) {
	shuffled := []dataset.Row{
		{Timestamp: 3000, Value: 5},
		{Timestamp: 0, Value: 0},
		{Timestamp: 1000, Value: 10},
	}
	got, err := Rates(shuffled)
	if err != nil {
		t.Fatalf("Rates: %v", err)
	}
	if got.MaxRise.Rate != 10 || got.MaxFall.Rate != -2.5 {
		t.Errorf("out-of-order input gave %+v", got)
	}
}

func TestRatesSkipsZeroTimeDeltas(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 0},
		{Timestamp: 0, Value: 100}, // duplicate timestamp: no defined rate
		{Timestamp: 1000, Value: 101},
	}
	got, err := Rates(rows)
	if err != nil {
		t.Fatalf("Rates: %v", err)
	}
	if got.MaxRise.Rate != 1 {
		t.Errorf("MaxRise.Rate = %v, want 1 (duplicate-timestamp pair skipped)", got.MaxRise.Rate)
	}
}

func TestRatesRejectsTooFewRows(t *testing.T) {
	_, err := Rates([]dataset.Row{{Timestamp: 0, Value: 1}})
	if err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Errorf("err = %v, want 'at least 2'", err)
	}
}

func TestRatesRejectsAllDuplicateTimestamps(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 5, Value: 1}, {Timestamp: 5, Value: 2}}
	_, err := Rates(rows)
	if err == nil || !strings.Contains(err.Error(), "distinct") {
		t.Errorf("err = %v, want 'distinct'", err)
	}
}
