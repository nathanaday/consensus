package analysis

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Quartiles are the 25th/50th/75th percentiles of a window's values.
type Quartiles struct {
	Q1 float64 `json:"q1"`
	Q2 float64 `json:"q2"`
	Q3 float64 `json:"q3"`
}

// Bounds is the inlier value range [Lower, Upper].
type Bounds struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

// OutlierPoint is one flagged row; Deviation is its distance beyond the
// violated bound.
type OutlierPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Deviation float64 `json:"deviation"`
}

// OutlierReport is the complete IQR outlier analysis of one window. Points
// holds every outlier, most extreme first; callers cap the list they expose.
type OutlierReport struct {
	RowCount  int
	Quartiles Quartiles
	Bounds    Bounds
	Points    []OutlierPoint
}

// Outliers flags rows outside [Q1 - k*IQR, Q3 + k*IQR]. k must be positive.
func Outliers(rows []dataset.Row, k float64) (OutlierReport, error) {
	if err := validateOutlierArgs(rows, k); err != nil {
		return OutlierReport{}, err
	}
	s := sortedByTime(rows)
	q, b := iqrBounds(s, k)
	points := make([]OutlierPoint, 0)
	for _, r := range s {
		if d := deviation(r.Value, b); d > 0 {
			points = append(points, OutlierPoint{Timestamp: formatTS(r.Timestamp), Value: r.Value, Deviation: d})
		}
	}
	sort.SliceStable(points, func(i, j int) bool { return points[i].Deviation > points[j].Deviation })
	return OutlierReport{RowCount: len(s), Quartiles: q, Bounds: b, Points: points}, nil
}

// RemoveOutliers returns the inliers in time order plus the bounds applied.
func RemoveOutliers(rows []dataset.Row, k float64) ([]dataset.Row, Bounds, error) {
	if err := validateOutlierArgs(rows, k); err != nil {
		return nil, Bounds{}, err
	}
	s := sortedByTime(rows)
	_, b := iqrBounds(s, k)
	out := make([]dataset.Row, 0, len(s))
	for _, r := range s {
		if deviation(r.Value, b) == 0 {
			out = append(out, r)
		}
	}
	return out, b, nil
}

func validateOutlierArgs(rows []dataset.Row, k float64) error {
	if k <= 0 {
		return fmt.Errorf("iqr multiplier must be positive, got %g", k)
	}
	if len(rows) == 0 {
		return fmt.Errorf("outlier detection needs at least 1 row, have 0")
	}
	return nil
}

func iqrBounds(sortedRows []dataset.Row, k float64) (Quartiles, Bounds) {
	values := make([]float64, len(sortedRows))
	for i, r := range sortedRows {
		values[i] = r.Value
	}
	sort.Float64s(values)
	q := Quartiles{Q1: quantile(values, 0.25), Q2: quantile(values, 0.5), Q3: quantile(values, 0.75)}
	iqr := q.Q3 - q.Q1
	return q, Bounds{Lower: q.Q1 - k*iqr, Upper: q.Q3 + k*iqr}
}

func deviation(v float64, b Bounds) float64 {
	if v < b.Lower {
		return b.Lower - v
	}
	if v > b.Upper {
		return v - b.Upper
	}
	return 0
}
