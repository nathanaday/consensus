package analysis

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// AtReport locates a dataset's value at (or nearest to) one instant.
// OffsetMS is Nearest.Timestamp minus the requested instant (negative when
// the nearest sample is earlier). Interpolated is the linear estimate between
// the bracketing samples; InterpolatedOK is false when the instant falls
// outside the data's span.
type AtReport struct {
	Nearest        dataset.Row
	OffsetMS       int64
	Interpolated   float64
	InterpolatedOK bool
}

// At finds the sample nearest to atMS (ties prefer the earlier sample) and,
// when atMS lies within the data's span, the linearly interpolated value.
func At(rows []dataset.Row, atMS int64) (AtReport, error) {
	if len(rows) == 0 {
		return AtReport{}, fmt.Errorf("value lookup needs at least 1 row, have 0")
	}
	s := sortedByTime(rows)

	// i is the first index at or after atMS.
	i := sort.Search(len(s), func(j int) bool { return s[j].Timestamp >= atMS })

	rep := AtReport{}
	switch {
	case i == 0:
		rep.Nearest = s[0]
	case i == len(s):
		rep.Nearest = s[len(s)-1]
	default:
		before, after := s[i-1], s[i]
		if atMS-before.Timestamp <= after.Timestamp-atMS {
			rep.Nearest = before
		} else {
			rep.Nearest = after
		}
	}
	rep.OffsetMS = rep.Nearest.Timestamp - atMS

	if atMS >= s[0].Timestamp && atMS <= s[len(s)-1].Timestamp {
		rep.InterpolatedOK = true
		if i < len(s) && s[i].Timestamp == atMS {
			rep.Interpolated = s[i].Value
		} else {
			before, after := s[i-1], s[i]
			frac := float64(atMS-before.Timestamp) / float64(after.Timestamp-before.Timestamp)
			rep.Interpolated = before.Value + frac*(after.Value-before.Value)
		}
	}
	return rep, nil
}
