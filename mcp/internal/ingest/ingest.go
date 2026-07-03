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
}

func parseTimestamp(s string) (time.Time, bool) {
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
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
	for _, rec := range rows {
		if tsIdx >= len(rec) {
			continue
		}
		ts, ok := parseTimestamp(rec[tsIdx])
		if !ok {
			return Result{}, fmt.Errorf("unparseable timestamp %q in column %q", rec[tsIdx], tsName)
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
