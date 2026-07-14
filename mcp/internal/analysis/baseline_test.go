package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// baselineRows: 20 points hovering around 30 (29..31), 1 minute apart.
func baselineRows() []dataset.Row {
	rows := make([]dataset.Row, 20)
	for i := range rows {
		v := 30.0
		if i%2 == 0 {
			v = 29
		} else {
			v = 31
		}
		rows[i] = dataset.Row{Timestamp: int64(i) * 60000, Value: v}
	}
	return rows
}

func TestBaselineGroupsSpikesIntoOneEpisode(t *testing.T) {
	base := baselineRows()
	// subject: mostly ~30, then three consecutive spikes ~110 one minute apart.
	subj := []dataset.Row{
		{Timestamp: 0, Value: 30},
		{Timestamp: 60000, Value: 30},
		{Timestamp: 120000, Value: 108},
		{Timestamp: 180000, Value: 109},
		{Timestamp: 240000, Value: 112},
		{Timestamp: 300000, Value: 30},
	}
	rep, err := Baseline(subj, base, 1.5)
	if err != nil {
		t.Fatalf("Baseline: %v", err)
	}
	if rep.PointsOutside != 3 {
		t.Fatalf("want 3 points outside, got %d", rep.PointsOutside)
	}
	if len(rep.Episodes) != 1 {
		t.Fatalf("want 1 episode, got %d: %+v", len(rep.Episodes), rep.Episodes)
	}
	ep := rep.Episodes[0]
	if ep.Direction != "above" {
		t.Errorf("want direction above, got %q", ep.Direction)
	}
	if ep.PeakValue != 112 {
		t.Errorf("want peak value 112, got %g", ep.PeakValue)
	}
	if ep.PointCount != 3 {
		t.Errorf("want 3 points in episode, got %d", ep.PointCount)
	}
}

func TestBaselineNoAnomalies(t *testing.T) {
	base := baselineRows()
	subj := []dataset.Row{
		{Timestamp: 0, Value: 30}, {Timestamp: 60000, Value: 31}, {Timestamp: 120000, Value: 29},
	}
	rep, err := Baseline(subj, base, 1.5)
	if err != nil {
		t.Fatalf("Baseline: %v", err)
	}
	if rep.PointsOutside != 0 || len(rep.Episodes) != 0 {
		t.Errorf("want no anomalies, got %d points / %d episodes", rep.PointsOutside, len(rep.Episodes))
	}
}

func TestBaselineErrors(t *testing.T) {
	base := baselineRows()
	if _, err := Baseline(nil, base, 1.5); err == nil {
		t.Error("empty subject should error")
	}
	if _, err := Baseline(base, nil, 1.5); err == nil {
		t.Error("empty baseline should error")
	}
	if _, err := Baseline(base, base, 0); err == nil {
		t.Error("non-positive k should error")
	}
}
