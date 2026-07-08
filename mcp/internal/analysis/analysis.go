// Package analysis provides pure statistical functions over dataset rows.
// Nothing here touches storage or MCP; callers load rows and shape output.
// Row order on disk is not guaranteed, so every exported function works on
// a timestamp-sorted copy of its input.
package analysis

import (
	"math"
	"sort"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// sortedByTime returns a copy of rows ordered by timestamp ascending.
func sortedByTime(rows []dataset.Row) []dataset.Row {
	out := make([]dataset.Row, len(rows))
	copy(out, rows)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })
	return out
}

// formatTS renders an epoch-millisecond timestamp as RFC3339 UTC, keeping
// sub-second precision when present.
func formatTS(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339Nano)
}

// quantile returns the q-th quantile (0..1) of ascending values using linear
// interpolation between order statistics. values must be non-empty.
func quantile(sorted []float64, q float64) float64 {
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

// Span reports the first and last timestamps of rows, which must be non-empty.
func Span(rows []dataset.Row) dataset.TimeRange {
	s := sortedByTime(rows)
	return dataset.TimeRange{
		Start: formatTS(s[0].Timestamp),
		End:   formatTS(s[len(s)-1].Timestamp),
	}
}
