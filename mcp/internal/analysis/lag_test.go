package analysis

import (
	"math"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// lagFixture builds a sine-like series and a copy shifted later by
// shiftBuckets buckets of widthMS, both sampled once per bucket.
func lagFixture(n int, widthMS int64, shiftBuckets int) (a, b []dataset.Row) {
	value := func(i int) float64 { return math.Sin(float64(i) / 5) }
	for i := 0; i < n; i++ {
		ts := int64(i) * widthMS
		a = append(a, dataset.Row{Timestamp: ts, Value: value(i)})
		b = append(b, dataset.Row{Timestamp: ts, Value: value(i - shiftBuckets)})
	}
	return a, b
}

func TestLagScanFindsKnownShift(t *testing.T) {
	a, b := lagFixture(100, 60000, 4)
	rep, err := LagScan(a, b, 0, 60000, 10)
	if err != nil {
		t.Fatalf("LagScan: %v", err)
	}
	if !rep.BestOK {
		t.Fatal("want a defined best correlation")
	}
	if rep.BestShift != 4 {
		t.Errorf("want best shift 4 (b follows a), got %d", rep.BestShift)
	}
	if rep.BestPearson < 0.999 {
		t.Errorf("want near-perfect correlation at the true lag, got %g", rep.BestPearson)
	}
	if !rep.ZeroOK || rep.ZeroPearson >= rep.BestPearson {
		t.Errorf("zero-lag correlation %g should be below best %g", rep.ZeroPearson, rep.BestPearson)
	}
	if rep.ShiftsScanned != 21 {
		t.Errorf("want 21 shifts scanned, got %d", rep.ShiftsScanned)
	}
}

func TestLagScanZeroShiftForAlignedSeries(t *testing.T) {
	a, _ := lagFixture(100, 60000, 0)
	rep, err := LagScan(a, a, 0, 60000, 10)
	if err != nil {
		t.Fatalf("LagScan: %v", err)
	}
	if rep.BestShift != 0 || !rep.BestOK {
		t.Errorf("identical series should peak at shift 0, got %+v", rep)
	}
}

func TestLagScanValidatesInput(t *testing.T) {
	a, b := lagFixture(10, 60000, 0)
	if _, err := LagScan(a, b, 0, 0, 5); err == nil {
		t.Error("want error for zero bucket width")
	}
	if _, err := LagScan(a, b, 0, 60000, 0); err == nil {
		t.Error("want error for zero max shift")
	}
	if _, err := LagScan(nil, b, 0, 60000, 5); err == nil {
		t.Error("want error for empty series")
	}
}

func TestLagScanConstantSeriesUndefined(t *testing.T) {
	a := []dataset.Row{}
	b := []dataset.Row{}
	for i := 0; i < 20; i++ {
		a = append(a, dataset.Row{Timestamp: int64(i) * 60000, Value: 5})
		b = append(b, dataset.Row{Timestamp: int64(i) * 60000, Value: float64(i)})
	}
	rep, err := LagScan(a, b, 0, 60000, 5)
	if err != nil {
		t.Fatalf("LagScan: %v", err)
	}
	if rep.BestOK || rep.ZeroOK {
		t.Errorf("constant series should leave correlations undefined, got %+v", rep)
	}
}
