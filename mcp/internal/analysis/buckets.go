package analysis

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Bucket is one fixed-width time bucket's aggregates. StartMS is the bucket's
// inclusive start (epoch ms). An empty bucket has Count 0 and zero stats.
type Bucket struct {
	StartMS int64
	Count   int
	Mean    float64
	Min     float64
	Max     float64
	Median  float64
}

// ladderMS is the round bucket-width ladder AutoWidthMS chooses from.
var ladderMS = []int64{
	1000, 5000, 15000, 30000,
	60000, 300000, 900000, 1800000,
	3600000, 3 * 3600000, 6 * 3600000, 12 * 3600000,
	24 * 3600000, 7 * 24 * 3600000,
}

// Buckets splits rows into start-aligned half-open [start, start+width)
// buckets spanning the first through last timestamp. Empty buckets are
// included with Count 0. widthMS must be positive and rows non-empty.
func Buckets(rows []dataset.Row, widthMS int64) ([]Bucket, error) {
	if widthMS <= 0 {
		return nil, fmt.Errorf("bucket width must be positive, got %dms", widthMS)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("bucketing needs at least 1 row, have 0")
	}
	s := sortedByTime(rows)
	origin := s[0].Timestamp
	last := s[len(s)-1].Timestamp
	n := int((last-origin)/widthMS) + 1

	values := make([][]float64, n)
	for _, r := range s {
		idx := int((r.Timestamp - origin) / widthMS)
		values[idx] = append(values[idx], r.Value)
	}

	out := make([]Bucket, n)
	for i := range out {
		b := Bucket{StartMS: origin + int64(i)*widthMS}
		vs := values[i]
		if len(vs) > 0 {
			sort.Float64s(vs)
			var sum float64
			b.Min, b.Max = vs[0], vs[len(vs)-1]
			for _, v := range vs {
				sum += v
			}
			b.Count = len(vs)
			b.Mean = sum / float64(len(vs))
			b.Median = quantile(vs, 0.5)
		}
		out[i] = b
	}
	return out, nil
}

// AutoWidthMS picks the smallest ladder rung keeping the window within
// maxBuckets buckets, falling back to the largest rung.
func AutoWidthMS(rows []dataset.Row, maxBuckets int) (int64, error) {
	if len(rows) == 0 {
		return 0, fmt.Errorf("auto width needs at least 1 row, have 0")
	}
	if maxBuckets <= 0 {
		return 0, fmt.Errorf("maxBuckets must be positive, got %d", maxBuckets)
	}
	s := sortedByTime(rows)
	span := s[len(s)-1].Timestamp - s[0].Timestamp
	for _, w := range ladderMS {
		if span/w+1 <= int64(maxBuckets) {
			return w, nil
		}
	}
	return ladderMS[len(ladderMS)-1], nil
}

// MedianIntervalMS is the median gap between consecutive sorted timestamps,
// or 0 when there are fewer than 2 rows.
func MedianIntervalMS(rows []dataset.Row) int64 {
	if len(rows) < 2 {
		return 0
	}
	s := sortedByTime(rows)
	gaps := make([]float64, 0, len(s)-1)
	for i := 1; i < len(s); i++ {
		gaps = append(gaps, float64(s[i].Timestamp-s[i-1].Timestamp))
	}
	sort.Float64s(gaps)
	return int64(quantile(gaps, 0.5))
}
