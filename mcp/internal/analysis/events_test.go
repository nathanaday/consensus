package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestEventsGroupsConsecutiveMatches(t *testing.T) {
	// 1-minute spacing; values exceed 10 at indexes 2-4 and again at 8.
	values := []float64{5, 6, 12, 15, 11, 6, 5, 6, 20, 5}
	rows := make([]dataset.Row, len(values))
	for i, v := range values {
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: v}
	}
	rep, err := Events(rows, Condition{Kind: "above", Threshold: 10})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(rep.Events) != 2 {
		t.Fatalf("want 2 events, got %d: %+v", len(rep.Events), rep.Events)
	}
	if rep.PointsMatching != 4 {
		t.Errorf("want 4 matching points, got %d", rep.PointsMatching)
	}
	// Longest first: the 3-point run (2 minutes) before the single point.
	e := rep.Events[0]
	if e.StartMS != 2*60000 || e.EndMS != 4*60000 || e.PointCount != 3 {
		t.Errorf("unexpected first event %+v", e)
	}
	if e.PeakValue != 15 || e.PeakDeviation != 5 {
		t.Errorf("want peak 15 (deviation 5), got %+v", e)
	}
	if rep.TimeInEventsMS != 2*60000 {
		t.Errorf("want 120000ms in events, got %d", rep.TimeInEventsMS)
	}
}

func TestEventsSplitsAcrossLongGaps(t *testing.T) {
	// Two matching points separated by far more than 3x the median interval
	// stay separate events even though nothing in between fails the condition.
	rows := []dataset.Row{
		{Timestamp: 0, Value: 20},
		{Timestamp: 60000, Value: 20},
		{Timestamp: 60 * 60000, Value: 20},
		{Timestamp: 61 * 60000, Value: 20},
	}
	rep, err := Events(rows, Condition{Kind: "above", Threshold: 10})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(rep.Events) != 2 {
		t.Fatalf("want 2 events split by the gap, got %d: %+v", len(rep.Events), rep.Events)
	}
}

func TestEventsBelowAndOutsideDirections(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: -5},
		{Timestamp: 60000, Value: 50},
		{Timestamp: 120000, Value: 5},
	}
	rep, err := Events(rows, Condition{Kind: "outside", Lower: 0, Upper: 10})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(rep.Events) != 2 {
		t.Fatalf("want 2 events, got %d", len(rep.Events))
	}
	dirs := map[string]bool{}
	for _, e := range rep.Events {
		dirs[e.Direction] = true
	}
	if !dirs["above"] || !dirs["below"] {
		t.Errorf("want one above and one below event, got %+v", rep.Events)
	}
}

func TestEventsBetweenUsesMembership(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 5},
		{Timestamp: 120000, Value: 7},
		{Timestamp: 180000, Value: 20},
	}
	rep, err := Events(rows, Condition{Kind: "between", Lower: 4, Upper: 10})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(rep.Events) != 1 || rep.Events[0].PointCount != 2 {
		t.Fatalf("want one 2-point event, got %+v", rep.Events)
	}
	if rep.Events[0].PeakValue != 7 {
		t.Errorf("between peak should be the highest value, got %g", rep.Events[0].PeakValue)
	}
}

func TestEventsValidatesCondition(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 0, Value: 1}}
	if _, err := Events(rows, Condition{Kind: "sideways"}); err == nil {
		t.Error("want error for unknown condition kind")
	}
	if _, err := Events(rows, Condition{Kind: "between", Lower: 10, Upper: 5}); err == nil {
		t.Error("want error for lower > upper")
	}
	if _, err := Events(nil, Condition{Kind: "above"}); err == nil {
		t.Error("want error for empty rows")
	}
}

func TestEventsNoMatches(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}}
	rep, err := Events(rows, Condition{Kind: "above", Threshold: 100})
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(rep.Events) != 0 || rep.PointsMatching != 0 || rep.TimeInEventsMS != 0 {
		t.Errorf("want empty report, got %+v", rep)
	}
}
