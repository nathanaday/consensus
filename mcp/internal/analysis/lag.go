package analysis

import (
	"fmt"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// LagReport is the result of scanning bucket shifts between two aligned
// series. BestShift is in buckets; positive means b's pattern follows a's
// (b lags a). The OK flags are false when no shift produced a defined
// correlation (or the zero-shift correlation is undefined).
type LagReport struct {
	BestShift     int
	BestPearson   float64
	BestOK        bool
	AlignedAtBest int
	ZeroPearson   float64
	ZeroOK        bool
	AlignedAtZero int
	ShiftsScanned int
}

// minLagSamples is the fewest aligned pairs a shift needs for its
// correlation to enter the scan; fewer is numerically legal but meaningless.
const minLagSamples = 3

// LagScan buckets both series onto a shared origin/width grid and computes
// the Pearson correlation of a's bucket means against b's shifted by every
// whole-bucket offset in [-maxShift, maxShift], reporting the shift with the
// strongest absolute correlation. maxShift must be at least 1.
func LagScan(a, b []dataset.Row, originMS, widthMS int64, maxShift int) (LagReport, error) {
	if widthMS <= 0 {
		return LagReport{}, fmt.Errorf("bucket width must be positive, got %dms", widthMS)
	}
	if maxShift < 1 {
		return LagReport{}, fmt.Errorf("max shift must be at least 1 bucket, got %d", maxShift)
	}
	if len(a) == 0 || len(b) == 0 {
		return LagReport{}, fmt.Errorf("lag scan needs both series non-empty")
	}
	ma := bucketMeans(sortedByTime(a), originMS, widthMS)
	mb := bucketMeans(sortedByTime(b), originMS, widthMS)

	rep := LagReport{ShiftsScanned: 2*maxShift + 1}
	for k := -maxShift; k <= maxShift; k++ {
		xs := make([]float64, 0, len(ma))
		ys := make([]float64, 0, len(ma))
		for idx, va := range ma {
			if vb, ok := mb[idx+int64(k)]; ok {
				xs = append(xs, va)
				ys = append(ys, vb)
			}
		}
		if k == 0 {
			rep.AlignedAtZero = len(xs)
		}
		if len(xs) < minLagSamples {
			continue
		}
		r, ok := pearson(xs, ys)
		if !ok {
			continue
		}
		if k == 0 {
			rep.ZeroPearson, rep.ZeroOK = r, true
		}
		if !rep.BestOK || abs(r) > abs(rep.BestPearson) {
			rep.BestShift, rep.BestPearson, rep.BestOK = k, r, true
			rep.AlignedAtBest = len(xs)
		}
	}
	return rep, nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
