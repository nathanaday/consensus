package store

import (
	"path/filepath"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestWriteReadRowsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ds.parquet")
	rows := []dataset.Row{
		{Timestamp: 1767225600000, SeriesID: "temp_c", Value: 12.4},
		{Timestamp: 1767225600000, SeriesID: "humidity", Value: 5.1},
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
