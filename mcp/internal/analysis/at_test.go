package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestAtExactMatch(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}, {Timestamp: 120000, Value: 3},
	}
	rep, err := At(rows, 60000)
	if err != nil {
		t.Fatalf("At: %v", err)
	}
	if rep.Nearest.Timestamp != 60000 || rep.OffsetMS != 0 {
		t.Errorf("want the exact sample, got %+v", rep)
	}
	if !rep.InterpolatedOK || rep.Interpolated != 2 {
		t.Errorf("exact match should interpolate to itself, got %+v", rep)
	}
}

func TestAtInterpolatesBetweenPoints(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 10}, {Timestamp: 100000, Value: 20},
	}
	rep, err := At(rows, 25000)
	if err != nil {
		t.Fatalf("At: %v", err)
	}
	if rep.Nearest.Timestamp != 0 || rep.OffsetMS != -25000 {
		t.Errorf("want nearest at 0 offset -25000, got %+v", rep)
	}
	if !rep.InterpolatedOK || rep.Interpolated != 12.5 {
		t.Errorf("want interpolated 12.5, got %+v", rep)
	}
}

func TestAtOutsideSpan(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 100000, Value: 1}, {Timestamp: 200000, Value: 2},
	}
	rep, err := At(rows, 0)
	if err != nil {
		t.Fatalf("At: %v", err)
	}
	if rep.Nearest.Timestamp != 100000 || rep.OffsetMS != 100000 {
		t.Errorf("want the first sample, got %+v", rep)
	}
	if rep.InterpolatedOK {
		t.Error("interpolation should be unavailable before the first sample")
	}
	rep, err = At(rows, 500000)
	if err != nil {
		t.Fatalf("At: %v", err)
	}
	if rep.Nearest.Timestamp != 200000 || rep.InterpolatedOK {
		t.Errorf("want the last sample without interpolation, got %+v", rep)
	}
}

func TestAtTiePrefersEarlier(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1}, {Timestamp: 100, Value: 2},
	}
	rep, err := At(rows, 50)
	if err != nil {
		t.Fatalf("At: %v", err)
	}
	if rep.Nearest.Timestamp != 0 {
		t.Errorf("equidistant lookup should prefer the earlier sample, got %+v", rep)
	}
}

func TestAtEmptyRows(t *testing.T) {
	if _, err := At(nil, 0); err == nil {
		t.Error("want error for empty rows")
	}
}
