// Package ingest turns CSV bytes into per-channel canonical rows. It has no
// knowledge of how or where datasets are stored.
package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Options overrides column detection. Empty fields mean "auto-detect".
type Options struct {
	TimestampCol string
	ValueCols    []string
}

// Channel is one value column's rows and stats. Blank cells are skipped, so
// row counts and time ranges are channel-specific.
type Channel struct {
	Column    string
	Rows      []dataset.Row
	RowCount  int
	TimeRange dataset.TimeRange
}

// Result is the per-channel split of one CSV plus the detected timestamp
// column.
type Result struct {
	TimestampColumn string
	Channels        []Channel
}

var timestampLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"01/02/2006 15:04:05",
	"01/02/2006",
	"1/2/2006 15:04:05",
	"1/2/2006",
}

// Epoch windows: each unit owns a disjoint plausible-date span (~1971 to
// ~2103), so magnitude alone determines the unit. Values in the gaps between
// windows are not timestamps.
const (
	epochMinSeconds = 3.0e7
	epochMaxSeconds = 4.2e9
)

// parseEpoch interprets v as a Unix epoch by magnitude: seconds,
// milliseconds, microseconds, or nanoseconds. Fractions survive down to the
// stored millisecond.
func parseEpoch(v float64) (time.Time, bool) {
	switch {
	case v >= epochMinSeconds && v < epochMaxSeconds:
		return time.UnixMilli(int64(v * 1e3)).UTC(), true
	case v >= epochMinSeconds*1e3 && v < epochMaxSeconds*1e3:
		return time.UnixMilli(int64(v)).UTC(), true
	case v >= epochMinSeconds*1e6 && v < epochMaxSeconds*1e6:
		return time.UnixMilli(int64(v / 1e3)).UTC(), true
	case v >= epochMinSeconds*1e9 && v < epochMaxSeconds*1e9:
		return time.UnixMilli(int64(v / 1e6)).UTC(), true
	}
	return time.Time{}, false
}

func parseTimestamp(s string) (time.Time, bool) {
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return parseEpoch(v)
	}
	return time.Time{}, false
}

// timestampError distinguishes a numeric value whose epoch unit cannot be
// inferred from a string that matches no known layout. row is 1-based and
// counts the header.
func timestampError(val, col string, row int) error {
	if _, err := strconv.ParseFloat(val, 64); err == nil {
		return fmt.Errorf("cannot infer epoch unit for value %q in column %q (row %d): numeric timestamps must be Unix seconds, milliseconds, microseconds, or nanoseconds between ~1971 and ~2103", val, col, row)
	}
	return fmt.Errorf("unparseable timestamp %q in column %q (row %d); accepted formats: RFC3339 and common date layouts, or Unix epoch seconds/milliseconds/microseconds/nanoseconds", val, col, row)
}

func indexOf(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

var timeLikeNames = map[string]bool{
	"ts": true, "time": true, "timestamp": true, "datetime": true,
	"date": true, "epoch": true,
}

// isTimeLikeName reports whether a column name suggests a timestamp: the
// whole name or any _-separated token matches a known time word,
// case-insensitively.
func isTimeLikeName(name string) bool {
	l := strings.ToLower(name)
	if timeLikeNames[l] {
		return true
	}
	for _, tok := range strings.Split(l, "_") {
		if timeLikeNames[tok] {
			return true
		}
	}
	return false
}

// FromCSV reads the CSV in r and splits it into one channel per value column.
func FromCSV(r io.Reader, opts Options) (Result, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return Result{}, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return Result{}, fmt.Errorf("csv has no data rows")
	}
	header, rows := records[0], records[1:]

	tsIdx, tsName, err := resolveTimestamp(header, rows[0], opts.TimestampCol)
	if err != nil {
		return Result{}, err
	}
	valIdx, valNames, err := resolveValueColumns(header, rows[0], tsIdx, opts.ValueCols)
	if err != nil {
		return Result{}, err
	}

	channels := make([]Channel, len(valIdx))
	for j, name := range valNames {
		channels[j].Column = name
	}
	minT := make([]time.Time, len(valIdx))
	maxT := make([]time.Time, len(valIdx))
	for i, rec := range rows {
		if tsIdx >= len(rec) {
			continue
		}
		ts, ok := parseTimestamp(rec[tsIdx])
		if !ok {
			return Result{}, timestampError(rec[tsIdx], tsName, i+2)
		}
		millis := ts.UnixMilli()
		for j, idx := range valIdx {
			if idx >= len(rec) || rec[idx] == "" {
				continue
			}
			v, err := strconv.ParseFloat(rec[idx], 64)
			if err != nil {
				return Result{}, fmt.Errorf("unparseable value %q in column %q", rec[idx], valNames[j])
			}
			channels[j].Rows = append(channels[j].Rows, dataset.Row{Timestamp: millis, Value: v})
			if minT[j].IsZero() || ts.Before(minT[j]) {
				minT[j] = ts
			}
			if maxT[j].IsZero() || ts.After(maxT[j]) {
				maxT[j] = ts
			}
		}
	}
	for j := range channels {
		channels[j].RowCount = len(channels[j].Rows)
		if !minT[j].IsZero() {
			channels[j].TimeRange = dataset.TimeRange{
				Start: minT[j].Format(time.RFC3339),
				End:   maxT[j].Format(time.RFC3339),
			}
		}
	}
	return Result{TimestampColumn: tsName, Channels: channels}, nil
}

func resolveTimestamp(header, firstRow []string, override string) (int, string, error) {
	if override != "" {
		idx := indexOf(header, override)
		if idx == -1 {
			return 0, "", fmt.Errorf("timestamp column %q not found; columns: %v", override, header)
		}
		return idx, override, nil
	}
	for _, hinted := range []bool{true, false} {
		for i, name := range header {
			if isTimeLikeName(name) != hinted || i >= len(firstRow) {
				continue
			}
			if _, ok := parseTimestamp(firstRow[i]); ok {
				return i, name, nil
			}
		}
	}
	return 0, "", fmt.Errorf("could not detect a timestamp column; pass timestamp_col (columns: %v)", header)
}

func resolveValueColumns(header, firstRow []string, tsIdx int, override []string) ([]int, []string, error) {
	if len(override) > 0 {
		var idx []int
		var names []string
		for _, name := range override {
			i := indexOf(header, name)
			if i == -1 {
				return nil, nil, fmt.Errorf("value column %q not found; columns: %v", name, header)
			}
			idx = append(idx, i)
			names = append(names, name)
		}
		return idx, names, nil
	}
	var idx []int
	var names []string
	for i, name := range header {
		if i == tsIdx || i >= len(firstRow) {
			continue
		}
		if _, err := strconv.ParseFloat(firstRow[i], 64); err == nil {
			idx = append(idx, i)
			names = append(names, name)
		}
	}
	if len(idx) == 0 {
		return nil, nil, fmt.Errorf("no numeric value columns detected; pass value_cols (columns: %v)", header)
	}
	return idx, names, nil
}
