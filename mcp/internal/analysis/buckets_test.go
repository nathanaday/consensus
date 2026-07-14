package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestBucketsAssignsBoundaryToLaterBucket(t *testing.T) {
	// width 100ms. Points at 0, 50, 100, 150. 100 must land in the second bucket.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 50, Value: 3},
		{Timestamp: 100, Value: 10},
		{Timestamp: 150, Value: 20},
	}
	got, err := Buckets(rows, 100)
	if err != nil {
		t.Fatalf("Buckets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 buckets, got %d: %+v", len(got), got)
	}
	if got[0].Count != 2 || got[0].Mean != 2 {
		t.Errorf("bucket 0: want count 2 mean 2, got count %d mean %g", got[0].Count, got[0].Mean)
	}
	if got[1].Count != 2 || got[1].Min != 10 || got[1].Max != 20 {
		t.Errorf("bucket 1: want count 2 min 10 max 20, got %+v", got[1])
	}
}

func TestBucketsIncludesEmptyBuckets(t *testing.T) {
	// width 100ms, points at 0 and 250 -> 3 buckets, middle one empty.
	rows := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 250, Value: 2}}
	got, err := Buckets(rows, 100)
	if err != nil {
		t.Fatalf("Buckets: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 buckets, got %d", len(got))
	}
	if got[1].Count != 0 {
		t.Errorf("middle bucket should be empty, got count %d", got[1].Count)
	}
}

func TestBucketsErrors(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 0, Value: 1}}
	if _, err := Buckets(rows, 0); err == nil {
		t.Error("zero width should error")
	}
	if _, err := Buckets(nil, 100); err == nil {
		t.Error("empty rows should error")
	}
}

func TestAutoWidthMS(t *testing.T) {
	// 23h span, max 48 buckets. 15m -> 92+1 buckets (too many); 30m -> 46+1=47
	// buckets (fits). Smallest rung that fits is 30m.
	span := int64(23 * 60 * 60 * 1000)
	rows := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: span, Value: 2}}
	w, err := AutoWidthMS(rows, 48)
	if err != nil {
		t.Fatalf("AutoWidthMS: %v", err)
	}
	if w != 30*60*1000 {
		t.Errorf("want 30m width, got %dms", w)
	}
}

func TestMedianIntervalMS(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0}, {Timestamp: 100}, {Timestamp: 200}, {Timestamp: 500},
	}
	// gaps: 100,100,300 -> median 100
	if got := MedianIntervalMS(rows); got != 100 {
		t.Errorf("want 100, got %d", got)
	}
	if got := MedianIntervalMS(rows[:1]); got != 0 {
		t.Errorf("single row should give 0, got %d", got)
	}
}
