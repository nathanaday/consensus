package analysis

import (
	"math"
	"testing"
)

func TestCompareSummaries(t *testing.T) {
	a := SummaryStats{Mean: 10, Stddev: 2}
	b := SummaryStats{Mean: 12, Stddev: 4}
	d := CompareSummaries(a, b)
	if d.MeanDifference != 2 {
		t.Errorf("want mean diff 2, got %g", d.MeanDifference)
	}
	if !d.MeanPctChangeOK || math.Abs(d.MeanPctChange-20) > 1e-9 {
		t.Errorf("want +20%%, got %g (ok=%v)", d.MeanPctChange, d.MeanPctChangeOK)
	}
	if !d.StddevRatioOK || math.Abs(d.StddevRatio-2) > 1e-9 {
		t.Errorf("want ratio 2, got %g (ok=%v)", d.StddevRatio, d.StddevRatioOK)
	}
}

func TestCompareSummariesZeroDenominators(t *testing.T) {
	a := SummaryStats{Mean: 0, Stddev: 0}
	b := SummaryStats{Mean: 5, Stddev: 3}
	d := CompareSummaries(a, b)
	if d.MeanDifference != 5 {
		t.Errorf("want mean diff 5, got %g", d.MeanDifference)
	}
	if d.MeanPctChangeOK {
		t.Error("pct change should be not-OK when a.Mean is 0")
	}
	if d.StddevRatioOK {
		t.Error("stddev ratio should be not-OK when a.Stddev is 0")
	}
}
