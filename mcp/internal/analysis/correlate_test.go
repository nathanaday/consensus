package analysis

import (
	"math"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestCorrelatePerfectPositive(t *testing.T) {
	// b = 2a + 5, sampled at identical timestamps -> Pearson and Spearman = 1.
	var a, b []dataset.Row
	for i := 0; i < 10; i++ {
		ts := int64(i) * 60000
		av := float64(i)
		a = append(a, dataset.Row{Timestamp: ts, Value: av})
		b = append(b, dataset.Row{Timestamp: ts, Value: 2*av + 5})
	}
	rep, err := Correlate(a, b, 0, 60000)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if rep.AlignedSamples != 10 {
		t.Fatalf("want 10 aligned, got %d", rep.AlignedSamples)
	}
	if !rep.PearsonOK || math.Abs(rep.Pearson-1) > 1e-9 {
		t.Errorf("want Pearson 1, got %g (ok=%v)", rep.Pearson, rep.PearsonOK)
	}
	if !rep.SpearmanOK || math.Abs(rep.Spearman-1) > 1e-9 {
		t.Errorf("want Spearman 1, got %g (ok=%v)", rep.Spearman, rep.SpearmanOK)
	}
}

func TestCorrelatePerfectNegative(t *testing.T) {
	var a, b []dataset.Row
	for i := 0; i < 10; i++ {
		ts := int64(i) * 60000
		a = append(a, dataset.Row{Timestamp: ts, Value: float64(i)})
		b = append(b, dataset.Row{Timestamp: ts, Value: float64(-i)})
	}
	rep, err := Correlate(a, b, 0, 60000)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if math.Abs(rep.Pearson+1) > 1e-9 {
		t.Errorf("want Pearson -1, got %g", rep.Pearson)
	}
}

func TestCorrelateMonotonicNonlinearSpearmanOne(t *testing.T) {
	// b = a^3 (strictly monotonic, nonlinear): Spearman = 1, Pearson < 1.
	var a, b []dataset.Row
	for i := 0; i < 10; i++ {
		ts := int64(i) * 60000
		av := float64(i)
		a = append(a, dataset.Row{Timestamp: ts, Value: av})
		b = append(b, dataset.Row{Timestamp: ts, Value: av * av * av})
	}
	rep, err := Correlate(a, b, 0, 60000)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if math.Abs(rep.Spearman-1) > 1e-9 {
		t.Errorf("want Spearman 1, got %g", rep.Spearman)
	}
	if rep.Pearson >= 1 {
		t.Errorf("want Pearson < 1 for a cubic relation, got %g", rep.Pearson)
	}
}

func TestCorrelateConstantSeriesNotOK(t *testing.T) {
	var a, b []dataset.Row
	for i := 0; i < 10; i++ {
		ts := int64(i) * 60000
		a = append(a, dataset.Row{Timestamp: ts, Value: float64(i)})
		b = append(b, dataset.Row{Timestamp: ts, Value: 7}) // constant
	}
	rep, err := Correlate(a, b, 0, 60000)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if rep.PearsonOK {
		t.Errorf("Pearson should be not-OK for a constant series")
	}
}

func TestCorrelateErrors(t *testing.T) {
	a := []dataset.Row{{Timestamp: 0, Value: 1}}
	if _, err := Correlate(a, a, 0, 0); err == nil {
		t.Error("zero width should error")
	}
	if _, err := Correlate(nil, a, 0, 60000); err == nil {
		t.Error("empty series should error")
	}
}
