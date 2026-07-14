package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// BaselineDist is a baseline window reduced to robust distribution stats.
type BaselineDist struct {
	Median float64 `json:"median"`
	Q1     float64 `json:"q1"`
	Q3     float64 `json:"q3"`
	P05    float64 `json:"p05"`
	P95    float64 `json:"p95"`
	Mean   float64 `json:"mean"`
	Stddev float64 `json:"stddev"`
	Count  int     `json:"count"`
}

// SubjectSummary is the subject window's headline stats.
type SubjectSummary struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
}

// Episode is a run of consecutive out-of-bounds subject points.
type Episode struct {
	StartMS       int64
	EndMS         int64
	Direction     string // "above" or "below"
	PointCount    int
	PeakValue     float64
	PeakMS        int64
	PeakDeviation float64
}

// BaselineReport is the full comparison of a subject window to a baseline.
type BaselineReport struct {
	Baseline      BaselineDist
	Bounds        Bounds
	Subject       SubjectSummary
	PointsOutside int
	Episodes      []Episode
}

func baselineDist(sorted []dataset.Row, k float64) (BaselineDist, Bounds) {
	q, b := iqrBounds(sorted, k)
	values := make([]float64, len(sorted))
	var sum float64
	for i, r := range sorted {
		values[i] = r.Value
		sum += r.Value
	}
	mean := sum / float64(len(sorted))
	var sq float64
	for _, v := range values {
		d := v - mean
		sq += d * d
	}
	sort.Float64s(values)
	return BaselineDist{
		Median: q.Q2, Q1: q.Q1, Q3: q.Q3,
		P05:    quantile(values, 0.05),
		P95:    quantile(values, 0.95),
		Mean:   mean,
		Stddev: math.Sqrt(sq / float64(len(values))),
		Count:  len(sorted),
	}, b
}

func subjectSummary(sorted []dataset.Row) SubjectSummary {
	values := make([]float64, len(sorted))
	minV, maxV := sorted[0].Value, sorted[0].Value
	var sum float64
	for i, r := range sorted {
		values[i] = r.Value
		if r.Value < minV {
			minV = r.Value
		}
		if r.Value > maxV {
			maxV = r.Value
		}
		sum += r.Value
	}
	sort.Float64s(values)
	return SubjectSummary{Min: minV, Max: maxV, Mean: sum / float64(len(sorted)), Median: quantile(values, 0.5)}
}

// Baseline scores a subject window against a baseline distribution and groups
// out-of-bounds points into episodes. Both windows must be non-empty and k
// positive.
func Baseline(subject, baseline []dataset.Row, k float64) (BaselineReport, error) {
	if k <= 0 {
		return BaselineReport{}, fmt.Errorf("iqr multiplier must be positive, got %g", k)
	}
	if len(subject) == 0 {
		return BaselineReport{}, fmt.Errorf("baseline comparison needs at least 1 subject row, have 0")
	}
	if len(baseline) == 0 {
		return BaselineReport{}, fmt.Errorf("baseline window is empty; pass baseline_start/baseline_end or baseline_id explicitly")
	}
	subj := sortedByTime(subject)
	base := sortedByTime(baseline)

	dist, bounds := baselineDist(base, k)
	tol := 3 * MedianIntervalMS(subj)

	var episodes []Episode
	var cur *Episode
	var lastMS int64
	points := 0
	for _, r := range subj {
		d := deviation(r.Value, bounds)
		if d <= 0 {
			continue
		}
		points++
		dir := "above"
		if r.Value < bounds.Lower {
			dir = "below"
		}
		newEpisode := cur == nil || cur.Direction != dir || (tol > 0 && r.Timestamp-lastMS > tol)
		if newEpisode {
			episodes = append(episodes, Episode{
				StartMS: r.Timestamp, EndMS: r.Timestamp, Direction: dir,
				PointCount: 1, PeakValue: r.Value, PeakMS: r.Timestamp, PeakDeviation: d,
			})
			cur = &episodes[len(episodes)-1]
		} else {
			cur.EndMS = r.Timestamp
			cur.PointCount++
			if d > cur.PeakDeviation {
				cur.PeakDeviation = d
				cur.PeakValue = r.Value
				cur.PeakMS = r.Timestamp
			}
		}
		lastMS = r.Timestamp
	}

	sort.SliceStable(episodes, func(i, j int) bool { return episodes[i].PeakDeviation > episodes[j].PeakDeviation })
	return BaselineReport{
		Baseline:      dist,
		Bounds:        bounds,
		Subject:       subjectSummary(subj),
		PointsOutside: points,
		Episodes:      episodes,
	}, nil
}
