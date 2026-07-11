package analysis

import (
	"math"
	"strings"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestSummaryComputesDescriptiveStats(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 2},
		{Timestamp: 120000, Value: 3},
		{Timestamp: 180000, Value: 4},
		{Timestamp: 240000, Value: 100},
	}
	got, err := Summary(rows)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.RowCount != 5 {
		t.Errorf("RowCount = %d, want 5", got.RowCount)
	}
	if got.Min != (Extreme{Value: 1, Timestamp: "1970-01-01T00:00:00Z"}) {
		t.Errorf("Min = %+v", got.Min)
	}
	if got.Max != (Extreme{Value: 100, Timestamp: "1970-01-01T00:04:00Z"}) {
		t.Errorf("Max = %+v", got.Max)
	}
	if got.Mean != 22 {
		t.Errorf("Mean = %v, want 22", got.Mean)
	}
	if got.Median != 3 {
		t.Errorf("Median = %v, want 3", got.Median)
	}
	// population stddev of {1,2,3,4,100}: sqrt(7610/5)
	if want := math.Sqrt(1522); math.Abs(got.Stddev-want) > 1e-9 {
		t.Errorf("Stddev = %v, want %v", got.Stddev, want)
	}
}

func TestSummaryMedianOnEvenCount(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 1, Value: 2},
		{Timestamp: 2, Value: 3},
		{Timestamp: 3, Value: 4},
	}
	got, err := Summary(rows)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Median != 2.5 {
		t.Errorf("Median = %v, want 2.5", got.Median)
	}
}

func TestSummaryTiesKeepEarliestTimestamp(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 2000, Value: 7},
		{Timestamp: 1000, Value: 7},
	}
	got, err := Summary(rows)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got.Min.Timestamp != "1970-01-01T00:00:01Z" || got.Max.Timestamp != "1970-01-01T00:00:01Z" {
		t.Errorf("tie should keep earliest: min %s max %s", got.Min.Timestamp, got.Max.Timestamp)
	}
}

func TestSummaryRejectsEmptyInput(t *testing.T) {
	_, err := Summary(nil)
	if err == nil || !strings.Contains(err.Error(), "at least 1") {
		t.Errorf("err = %v, want 'at least 1'", err)
	}
}
