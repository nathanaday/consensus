package analysis

// SummaryDelta expresses side B relative to side A. The OK flags are false
// when the corresponding side-A denominator is zero.
type SummaryDelta struct {
	MeanDifference  float64
	MeanPctChange   float64
	MeanPctChangeOK bool
	StddevRatio     float64
	StddevRatioOK   bool
}

// CompareSummaries computes B-relative-to-A deltas between two summaries.
func CompareSummaries(a, b SummaryStats) SummaryDelta {
	d := SummaryDelta{MeanDifference: b.Mean - a.Mean}
	if a.Mean != 0 {
		d.MeanPctChange = 100 * (b.Mean - a.Mean) / a.Mean
		d.MeanPctChangeOK = true
	}
	if a.Stddev != 0 {
		d.StddevRatio = b.Stddev / a.Stddev
		d.StddevRatioOK = true
	}
	return d
}
