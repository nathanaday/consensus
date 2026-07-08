// Package ingest turns CSV bytes into canonical long-format rows. It has no
// knowledge of how or where datasets are stored.
package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// Options overrides column detection. Empty fields mean "auto-detect".
type Options struct {
	TimestampCol string
	ValueCols    []string
}

// Result is the canonical dataset plus the metadata needed for a catalog entry
// and a schema summary.
type Result struct {
	Rows            []dataset.Row
	TimestampColumn string
	ValueColumns    []string
	SeriesIDs       []string
	RowCount        int
	TimeRange       dataset.TimeRange
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

// FromCSV reads the CSV in r and normalizes it to long-format rows.
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

	var out []dataset.Row
	var minT, maxT time.Time
	for i, rec := range rows {
		if tsIdx >= len(rec) {
			continue
		}
		ts, ok := parseTimestamp(rec[tsIdx])
		if !ok {
			return Result{}, timestampError(rec[tsIdx], tsName, i+2)
		}
		if minT.IsZero() || ts.Before(minT) {
			minT = ts
		}
		if maxT.IsZero() || ts.After(maxT) {
			maxT = ts
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
			out = append(out, dataset.Row{Timestamp: millis, SeriesID: valNames[j], Value: v})
		}
	}

	return Result{
		Rows:            out,
		TimestampColumn: tsName,
		ValueColumns:    valNames,
		SeriesIDs:       valNames,
		RowCount:        len(out),
		TimeRange: dataset.TimeRange{
			Start: minT.Format(time.RFC3339),
			End:   maxT.Format(time.RFC3339),
		},
	}, nil
}

func resolveTimestamp(header, firstRow []string, override string) (int, string, error) {
	if override != "" {
		idx := indexOf(header, override)
		if idx == -1 {
			return 0, "", fmt.Errorf("timestamp column %q not found; columns: %v", override, header)
		}
		return idx, override, nil
	}
	for i, name := range header {
		if i < len(firstRow) {
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
