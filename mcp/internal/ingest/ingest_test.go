package ingest

import (
	"strings"
	"testing"
)

const wideCSV = `time,temp_c,humidity
2026-01-01T00:00:00Z,12.4,5.1
2026-01-01T00:05:00Z,12.6,5.0
`

func TestFromCSVMeltsWideToLong(t *testing.T) {
	res, err := FromCSV(strings.NewReader(wideCSV), Options{})
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}
	if res.TimestampColumn != "time" {
		t.Errorf("timestamp column = %q, want time", res.TimestampColumn)
	}
	if got := len(res.Rows); got != 4 {
		t.Fatalf("rows = %d, want 4 (2 timestamps x 2 series)", got)
	}
	if res.RowCount != 4 {
		t.Errorf("RowCount = %d, want 4", res.RowCount)
	}
	want := map[string]bool{"temp_c": true, "humidity": true}
	for _, s := range res.SeriesIDs {
		if !want[s] {
			t.Errorf("unexpected series %q", s)
		}
	}
	if len(res.SeriesIDs) != 2 {
		t.Errorf("series count = %d, want 2", len(res.SeriesIDs))
	}
	// First row is the earliest timestamp for a series, normalized to UTC millis.
	if res.Rows[0].Timestamp != 1767225600000 {
		t.Errorf("first timestamp = %d, want 1767225600000", res.Rows[0].Timestamp)
	}
	if res.TimeRange.Start != "2026-01-01T00:00:00Z" {
		t.Errorf("range start = %q", res.TimeRange.Start)
	}
	if res.TimeRange.End != "2026-01-01T00:05:00Z" {
		t.Errorf("range end = %q", res.TimeRange.End)
	}
}

func TestFromCSVTimestampOverride(t *testing.T) {
	csv := `id,reading_at,val
a,2026-01-01T00:00:00Z,3.5
`
	res, err := FromCSV(strings.NewReader(csv), Options{TimestampCol: "reading_at", ValueCols: []string{"val"}})
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}
	if res.TimestampColumn != "reading_at" {
		t.Errorf("timestamp column = %q, want reading_at", res.TimestampColumn)
	}
	if len(res.Rows) != 1 || res.Rows[0].SeriesID != "val" {
		t.Fatalf("rows = %+v", res.Rows)
	}
}

func TestFromCSVErrorsWhenNoTimestamp(t *testing.T) {
	csv := "a,b\n1,2\n"
	_, err := FromCSV(strings.NewReader(csv), Options{})
	if err == nil || !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("expected timestamp detection error, got %v", err)
	}
}

func TestFromCSVErrorsOnUnparseableValue(t *testing.T) {
	csv := "time,temp_c\n2026-01-01T00:00:00Z,not_a_number\n"
	_, err := FromCSV(strings.NewReader(csv), Options{ValueCols: []string{"temp_c"}})
	if err == nil || !strings.Contains(err.Error(), "not_a_number") {
		t.Fatalf("expected unparseable value error naming the value, got %v", err)
	}
}

func TestParseTimestampEpochs(t *testing.T) {
	cases := []struct {
		in     string
		wantMS int64
	}{
		{"1594512094", 1594512094000},          // integer seconds
		{"1594512094.3859746", 1594512094385},  // float seconds, fraction kept to the ms
		{"1594512094385", 1594512094385},       // milliseconds
		{"1594512094385974", 1594512094385},    // microseconds, truncated to ms
		{"1594512094385974600", 1594512094385}, // nanoseconds, truncated to ms
		{"30000000", 30000000000},              // lower window edge (~1970-12-13), seconds
	}
	for _, c := range cases {
		ts, ok := parseTimestamp(c.in)
		if !ok {
			t.Errorf("parseTimestamp(%q) not recognized", c.in)
			continue
		}
		if got := ts.UnixMilli(); got != c.wantMS {
			t.Errorf("parseTimestamp(%q) = %d ms, want %d", c.in, got, c.wantMS)
		}
	}

	for _, reject := range []string{"0", "1000", "29999999", "5000000000", "-1594512094"} {
		if _, ok := parseTimestamp(reject); ok {
			t.Errorf("parseTimestamp(%q) accepted, want rejected (outside every epoch window)", reject)
		}
	}
}

func TestFromCSVParsesEpochSecondsCSV(t *testing.T) {
	csv := "ts,humidity\n1594512094.3859746,51.0\n1594512112.798518,50.9\n"
	res, err := FromCSV(strings.NewReader(csv), Options{})
	if err != nil {
		t.Fatalf("FromCSV: %v", err)
	}
	if res.TimestampColumn != "ts" {
		t.Errorf("timestamp column = %q, want ts", res.TimestampColumn)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(res.Rows))
	}
	if res.Rows[0].Timestamp != 1594512094385 {
		t.Errorf("first timestamp = %d, want 1594512094385", res.Rows[0].Timestamp)
	}
}

func TestFromCSVCannotInferEpochUnit(t *testing.T) {
	// 5000000000 falls in the gap between the seconds and milliseconds windows.
	csv := "ts,v\n5000000000,1.5\n"
	_, err := FromCSV(strings.NewReader(csv), Options{TimestampCol: "ts", ValueCols: []string{"v"}})
	if err == nil || !strings.Contains(err.Error(), "cannot infer epoch unit") {
		t.Fatalf("expected epoch-unit error, got %v", err)
	}
	if !strings.Contains(err.Error(), "row 2") {
		t.Errorf("expected error to name the row, got %v", err)
	}
}

func TestFromCSVTimestampErrorNamesRowAndFormats(t *testing.T) {
	csv := "time,v\n2026-01-01T00:00:00Z,1\nnot-a-time,2\n"
	_, err := FromCSV(strings.NewReader(csv), Options{ValueCols: []string{"v"}})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{`"not-a-time"`, `"time"`, "row 3", "epoch"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}
