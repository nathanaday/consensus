package analysis

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestSpanReportsFirstAndLastTimestamp(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 120000, Value: 3},
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 2},
	}
	got := Span(rows)
	want := dataset.TimeRange{Start: "1970-01-01T00:00:00Z", End: "1970-01-01T00:02:00Z"}
	if got != want {
		t.Errorf("Span = %+v, want %+v", got, want)
	}
}

func TestFormatTSKeepsMilliseconds(t *testing.T) {
	if got := formatTS(1500); got != "1970-01-01T00:00:01.5Z" {
		t.Errorf("formatTS(1500) = %q, want 1970-01-01T00:00:01.5Z", got)
	}
	if got := formatTS(2000); got != "1970-01-01T00:00:02Z" {
		t.Errorf("formatTS(2000) = %q, want 1970-01-01T00:00:02Z", got)
	}
}

func TestQuantileInterpolatesLinearly(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 100}
	cases := []struct {
		q    float64
		want float64
	}{
		{0, 1}, {0.25, 2}, {0.5, 3}, {0.75, 4}, {1, 100},
	}
	for _, c := range cases {
		if got := quantile(sorted, c.q); got != c.want {
			t.Errorf("quantile(%v) = %v, want %v", c.q, got, c.want)
		}
	}
	// interpolation between order statistics: n=4, q=0.5 -> pos 1.5
	if got := quantile([]float64{1, 2, 3, 4}, 0.5); got != 2.5 {
		t.Errorf("quantile(0.5 of 4) = %v, want 2.5", got)
	}
}

func TestSortedByTimeDoesNotMutateInput(t *testing.T) {
	rows := []dataset.Row{{Timestamp: 2, Value: 2}, {Timestamp: 1, Value: 1}}
	sorted := sortedByTime(rows)
	if rows[0].Timestamp != 2 {
		t.Error("input slice was mutated")
	}
	if sorted[0].Timestamp != 1 || sorted[1].Timestamp != 2 {
		t.Errorf("not sorted: %+v", sorted)
	}
}
