package analysis

import (
	"fmt"
	"math"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Window returns the rows inside the inclusive [start, end] span, sorted by
// time. An empty start or end leaves that side unbounded.
func Window(rows []dataset.Row, start, end string) ([]dataset.Row, error) {
	lo := int64(math.MinInt64)
	hi := int64(math.MaxInt64)
	if start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("invalid start %q; expected an RFC3339 UTC timestamp like 2026-07-07T00:00:00Z", start)
		}
		lo = t.UnixMilli()
	}
	if end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, fmt.Errorf("invalid end %q; expected an RFC3339 UTC timestamp like 2026-07-07T00:00:00Z", end)
		}
		hi = t.UnixMilli()
	}
	if lo > hi {
		return nil, fmt.Errorf("start %s is after end %s", start, end)
	}
	out := make([]dataset.Row, 0)
	for _, r := range sortedByTime(rows) {
		if r.Timestamp >= lo && r.Timestamp <= hi {
			out = append(out, r)
		}
	}
	return out, nil
}
