package analysis

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Condition is a value predicate for event detection. Kind is "above" or
// "below" (using Threshold), or "between" or "outside" (using Lower/Upper).
type Condition struct {
	Kind         string
	Threshold    float64
	Lower, Upper float64
}

// Validate checks the condition's kind and bounds.
func (c Condition) Validate() error {
	switch c.Kind {
	case "above", "below":
		return nil
	case "between", "outside":
		if c.Lower > c.Upper {
			return fmt.Errorf("condition lower %g is greater than upper %g", c.Lower, c.Upper)
		}
		return nil
	default:
		return fmt.Errorf("unknown condition %q; use above, below, between, or outside", c.Kind)
	}
}

// match reports whether v satisfies the condition and the direction of the
// match: "above" or "below" the relevant bound, or "within" for between.
func (c Condition) match(v float64) (bool, string) {
	switch c.Kind {
	case "above":
		return v > c.Threshold, "above"
	case "below":
		return v < c.Threshold, "below"
	case "between":
		return v >= c.Lower && v <= c.Upper, "within"
	case "outside":
		if v < c.Lower {
			return true, "below"
		}
		if v > c.Upper {
			return true, "above"
		}
		return false, ""
	}
	return false, ""
}

// deviation is how far v sits beyond the condition's violated bound; 0 for
// "between", where matching is membership rather than exceedance.
func (c Condition) deviation(v float64) float64 {
	switch c.Kind {
	case "above":
		return v - c.Threshold
	case "below":
		return c.Threshold - v
	case "outside":
		if v < c.Lower {
			return c.Lower - v
		}
		return v - c.Upper
	}
	return 0
}

// Event is a run of consecutive points matching a condition.
type Event struct {
	StartMS       int64
	EndMS         int64
	Direction     string
	PointCount    int
	PeakValue     float64
	PeakMS        int64
	PeakDeviation float64
}

// EventsReport is the full event scan of one window.
type EventsReport struct {
	PointsMatching int
	TimeInEventsMS int64
	Events         []Event // sorted by duration descending
}

// Events groups the points matching cond into events. Consecutive matching
// points merge while they share a direction and are separated by no more than
// 3x the window's median sampling interval (the same tolerance Baseline uses
// for episodes). Events are returned longest first.
func Events(rows []dataset.Row, cond Condition) (EventsReport, error) {
	if err := cond.Validate(); err != nil {
		return EventsReport{}, err
	}
	if len(rows) == 0 {
		return EventsReport{}, fmt.Errorf("event detection needs at least 1 row, have 0")
	}
	s := sortedByTime(rows)
	tol := 3 * MedianIntervalMS(s)

	var events []Event
	var cur *Event
	var lastMS int64
	points := 0
	for _, r := range s {
		ok, dir := cond.match(r.Value)
		if !ok {
			cur = nil
			continue
		}
		points++
		d := cond.deviation(r.Value)
		newEvent := cur == nil || cur.Direction != dir || (tol > 0 && r.Timestamp-lastMS > tol)
		if newEvent {
			events = append(events, Event{
				StartMS: r.Timestamp, EndMS: r.Timestamp, Direction: dir,
				PointCount: 1, PeakValue: r.Value, PeakMS: r.Timestamp, PeakDeviation: d,
			})
			cur = &events[len(events)-1]
		} else {
			cur.EndMS = r.Timestamp
			cur.PointCount++
			// The peak is the most extreme matching point: largest exceedance
			// for above/below/outside, highest value for between.
			better := d > cur.PeakDeviation
			if cond.Kind == "between" {
				better = r.Value > cur.PeakValue
			}
			if better {
				cur.PeakValue = r.Value
				cur.PeakMS = r.Timestamp
				cur.PeakDeviation = d
			}
		}
		lastMS = r.Timestamp
	}

	var total int64
	for _, e := range events {
		total += e.EndMS - e.StartMS
	}
	sort.SliceStable(events, func(i, j int) bool {
		di, dj := events[i].EndMS-events[i].StartMS, events[j].EndMS-events[j].StartMS
		if di != dj {
			return di > dj
		}
		if events[i].PointCount != events[j].PointCount {
			return events[i].PointCount > events[j].PointCount
		}
		return events[i].StartMS < events[j].StartMS
	})
	return EventsReport{PointsMatching: points, TimeInEventsMS: total, Events: events}, nil
}
