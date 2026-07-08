package store

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestWriteReadRowsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ds.parquet")
	rows := []dataset.Row{
		{Timestamp: 1767225600000, Value: 12.4},
		{Timestamp: 1767225600000, Value: 5.1},
	}
	if err := WriteRows(path, rows); err != nil {
		t.Fatalf("WriteRows: %v", err)
	}
	got, err := ReadRows(path)
	if err != nil {
		t.Fatalf("ReadRows: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("read %d rows, want 2", len(got))
	}
	if got[0] != rows[0] || got[1] != rows[1] {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// TestWriteRowsStampsMillisecondTimestampLogicalType verifies the stored
// timestamp column carries a TIMESTAMP(MILLIS, isAdjustedToUTC=true) logical
// type so external readers (DuckDB, Polars, pandas) see a real timestamp rather
// than a bare int64.
func TestWriteRowsStampsMillisecondTimestampLogicalType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ds.parquet")
	if err := WriteRows(path, []dataset.Row{{Timestamp: 1767225600000, Value: 12.4}}); err != nil {
		t.Fatalf("WriteRows: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	f, err := parquet.OpenFile(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	var lt string
	for _, field := range f.Schema().Fields() {
		if field.Name() == "timestamp" {
			lt = fmt.Sprintf("%v", field.Type().LogicalType())
		}
	}
	if !strings.Contains(lt, "MILLIS") || !strings.Contains(lt, "isAdjustedToUTC=true") {
		t.Errorf("timestamp logical type = %q, want TIMESTAMP with unit=MILLIS and isAdjustedToUTC=true", lt)
	}
}

func TestWriteRowsCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iot", "temp.parquet")
	rows := []dataset.Row{{Timestamp: 1, Value: 21.5}}
	if err := WriteRows(path, rows); err != nil {
		t.Fatalf("WriteRows into missing subdir: %v", err)
	}
	got, err := ReadRows(path)
	if err != nil {
		t.Fatalf("ReadRows: %v", err)
	}
	if len(got) != 1 || got[0].Value != 21.5 {
		t.Errorf("round-trip = %+v, want the written row", got)
	}
}
