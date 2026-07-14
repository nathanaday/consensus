package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestQualityDetectsGap(t *testing.T) {
	// 1-minute spacing, then a 30-minute jump (gap), then 1-minute again.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 2},
		{Timestamp: 120000, Value: 3},
		{Timestamp: 120000 + 30*60000, Value: 4},
		{Timestamp: 120000 + 31*60000, Value: 5},
	}
	rep, err := Quality(rows)
	if err != nil {
		t.Fatalf("Quality: %v", err)
	}
	if rep.TotalGaps != 1 {
		t.Fatalf("want 1 gap, got %d", rep.TotalGaps)
	}
	if rep.Gaps[0].DurationMS != 30*60000 {
		t.Errorf("want 30m gap, got %dms", rep.Gaps[0].DurationMS)
	}
}

func TestQualityDetectsFlatline(t *testing.T) {
	rows := make([]dataset.Row, 12)
	for i := range rows {
		v := float64(i)
		if i >= 1 && i <= 10 { // 10 identical values
			v = 5
		}
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: v}
	}
	rep, err := Quality(rows)
	if err != nil {
		t.Fatalf("Quality: %v", err)
	}
	if rep.TotalFlatlines != 1 {
		t.Fatalf("want 1 flatline, got %d", rep.TotalFlatlines)
	}
	if rep.Flatlines[0].PointCount != 10 || rep.Flatlines[0].Value != 5 {
		t.Errorf("want 10 points at value 5, got %+v", rep.Flatlines[0])
	}
}

func TestQualityCountsDuplicateTimestamps(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1}, {Timestamp: 0, Value: 2}, {Timestamp: 60000, Value: 3},
	}
	rep, err := Quality(rows)
	if err != nil {
		t.Fatalf("Quality: %v", err)
	}
	if rep.DuplicateTimestamps != 1 {
		t.Errorf("want 1 duplicate, got %d", rep.DuplicateTimestamps)
	}
}
