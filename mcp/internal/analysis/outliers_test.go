package analysis

import (
	"strings"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// values {1,2,3,4,100}: Q1=2, Q2=3, Q3=4, IQR=2, k=1.5 -> bounds [-1, 7]
func outlierRows() []dataset.Row {
	return []dataset.Row{
		{Timestamp: 0, Value: 1},
		{Timestamp: 60000, Value: 2},
		{Timestamp: 120000, Value: 3},
		{Timestamp: 180000, Value: 4},
		{Timestamp: 240000, Value: 100},
	}
}

func TestOutliersFlagsPointsOutsideIQRBounds(t *testing.T) {
	got, err := Outliers(outlierRows(), 1.5)
	if err != nil {
		t.Fatalf("Outliers: %v", err)
	}
	if got.RowCount != 5 {
		t.Errorf("RowCount = %d, want 5", got.RowCount)
	}
	if got.Quartiles != (Quartiles{Q1: 2, Q2: 3, Q3: 4}) {
		t.Errorf("Quartiles = %+v", got.Quartiles)
	}
	if got.Bounds != (Bounds{Lower: -1, Upper: 7}) {
		t.Errorf("Bounds = %+v", got.Bounds)
	}
	if len(got.Points) != 1 {
		t.Fatalf("Points = %+v, want exactly one outlier", got.Points)
	}
	want := OutlierPoint{Timestamp: "1970-01-01T00:04:00Z", Value: 100, Deviation: 93}
	if got.Points[0] != want {
		t.Errorf("Points[0] = %+v, want %+v", got.Points[0], want)
	}
}

func TestOutliersSortsMostExtremeFirst(t *testing.T) {
	rows := append(outlierRows(),
		dataset.Row{Timestamp: 300000, Value: -50}, // deviation 49 below lower bound
	)
	// values {-50,1,2,3,4,100}: Q1=1.25, Q3=3.75, IQR=2.5 -> bounds [-2.5, 7.5]
	got, err := Outliers(rows, 1.5)
	if err != nil {
		t.Fatalf("Outliers: %v", err)
	}
	if len(got.Points) != 2 {
		t.Fatalf("Points = %+v, want 2 outliers", got.Points)
	}
	if got.Points[0].Value != 100 || got.Points[1].Value != -50 {
		t.Errorf("not sorted by deviation desc: %+v", got.Points)
	}
	if got.Points[1].Deviation != 47.5 {
		t.Errorf("low-side deviation = %v, want 47.5", got.Points[1].Deviation)
	}
}

func TestOutliersIdenticalValuesHaveNoOutliers(t *testing.T) {
	rows := []dataset.Row{
		{Timestamp: 0, Value: 5}, {Timestamp: 1, Value: 5}, {Timestamp: 2, Value: 5},
	}
	got, err := Outliers(rows, 1.5)
	if err != nil {
		t.Fatalf("Outliers: %v", err)
	}
	if len(got.Points) != 0 {
		t.Errorf("identical values produced outliers: %+v", got.Points)
	}
}

func TestOutliersValidation(t *testing.T) {
	if _, err := Outliers(outlierRows(), 0); err == nil || !strings.Contains(err.Error(), "positive") {
		t.Errorf("k=0 err = %v, want 'positive'", err)
	}
	if _, err := Outliers(nil, 1.5); err == nil || !strings.Contains(err.Error(), "at least 1") {
		t.Errorf("empty err = %v, want 'at least 1'", err)
	}
}

func TestRemoveOutliersReturnsInliersInTimeOrder(t *testing.T) {
	shuffled := []dataset.Row{
		{Timestamp: 240000, Value: 100},
		{Timestamp: 60000, Value: 2},
		{Timestamp: 0, Value: 1},
		{Timestamp: 180000, Value: 4},
		{Timestamp: 120000, Value: 3},
	}
	inliers, bounds, err := RemoveOutliers(shuffled, 1.5)
	if err != nil {
		t.Fatalf("RemoveOutliers: %v", err)
	}
	if bounds != (Bounds{Lower: -1, Upper: 7}) {
		t.Errorf("bounds = %+v", bounds)
	}
	if len(inliers) != 4 {
		t.Fatalf("inliers = %+v, want 4 rows", inliers)
	}
	for i := 1; i < len(inliers); i++ {
		if inliers[i].Timestamp < inliers[i-1].Timestamp {
			t.Errorf("inliers not time-ordered: %+v", inliers)
		}
	}
	for _, r := range inliers {
		if r.Value == 100 {
			t.Error("outlier survived removal")
		}
	}
}
