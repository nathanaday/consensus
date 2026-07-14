package analysis

import (
	"fmt"
	"math"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// CorrelationReport is the aligned-grid correlation of two series. The OK
// flags are false when there are fewer than 2 aligned samples or a paired
// series is constant (correlation undefined).
type CorrelationReport struct {
	AlignedSamples int
	Pearson        float64
	PearsonOK      bool
	Spearman       float64
	SpearmanOK     bool
}

// bucketMeans returns bucketIndex -> mean value for one series.
func bucketMeans(rows []dataset.Row, originMS, widthMS int64) map[int64]float64 {
	sums := map[int64]float64{}
	counts := map[int64]int{}
	for _, r := range rows {
		idx := (r.Timestamp - originMS) / widthMS
		sums[idx] += r.Value
		counts[idx]++
	}
	means := make(map[int64]float64, len(sums))
	for idx, s := range sums {
		means[idx] = s / float64(counts[idx])
	}
	return means
}

// Correlate aligns a and b onto a common origin/width grid and correlates the
// bucket means where both have data.
func Correlate(a, b []dataset.Row, originMS, widthMS int64) (CorrelationReport, error) {
	if widthMS <= 0 {
		return CorrelationReport{}, fmt.Errorf("bucket width must be positive, got %dms", widthMS)
	}
	if len(a) == 0 || len(b) == 0 {
		return CorrelationReport{}, fmt.Errorf("correlation needs both series non-empty")
	}
	ma := bucketMeans(a, originMS, widthMS)
	mb := bucketMeans(b, originMS, widthMS)

	idxs := make([]int64, 0, len(ma))
	for idx := range ma {
		if _, ok := mb[idx]; ok {
			idxs = append(idxs, idx)
		}
	}
	sort.Slice(idxs, func(i, j int) bool { return idxs[i] < idxs[j] })

	xs := make([]float64, len(idxs))
	ys := make([]float64, len(idxs))
	for i, idx := range idxs {
		xs[i] = ma[idx]
		ys[i] = mb[idx]
	}

	rep := CorrelationReport{AlignedSamples: len(idxs)}
	if len(idxs) < 2 {
		return rep, nil
	}
	if r, ok := pearson(xs, ys); ok {
		rep.Pearson, rep.PearsonOK = r, true
	}
	if r, ok := pearson(rankOf(xs), rankOf(ys)); ok {
		rep.Spearman, rep.SpearmanOK = r, true
	}
	return rep, nil
}

// pearson returns the Pearson correlation of xs and ys and whether it is
// defined (both series must vary).
func pearson(xs, ys []float64) (float64, bool) {
	n := float64(len(xs))
	var sx, sy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
	}
	mx, my := sx/n, sy/n
	var num, dx2, dy2 float64
	for i := range xs {
		dx := xs[i] - mx
		dy := ys[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}
	if dx2 == 0 || dy2 == 0 {
		return 0, false
	}
	r := num / math.Sqrt(dx2*dy2)
	if r > 1 {
		r = 1
	} else if r < -1 {
		r = -1
	}
	return r, true
}

// rankOf returns fractional ranks (average ranks for ties) of vs.
func rankOf(vs []float64) []float64 {
	type iv struct {
		v float64
		i int
	}
	order := make([]iv, len(vs))
	for i, v := range vs {
		order[i] = iv{v, i}
	}
	sort.SliceStable(order, func(a, b int) bool { return order[a].v < order[b].v })

	ranks := make([]float64, len(vs))
	for i := 0; i < len(order); {
		j := i
		for j+1 < len(order) && order[j+1].v == order[i].v {
			j++
		}
		// average rank (1-based) for the tie group [i, j].
		avg := float64(i+j)/2 + 1
		for k := i; k <= j; k++ {
			ranks[order[k].i] = avg
		}
		i = j + 1
	}
	return ranks
}
