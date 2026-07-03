package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestSaveDatasetWritesParquetAndCatalog(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	req := SaveRequest{
		SourcePath:      "/abs/readings.csv",
		TimestampColumn: "time",
		SeriesIDs:       []string{"temp_c"},
		RowCount:        1,
		TimeRange:       dataset.TimeRange{Start: "2026-01-01T00:00:00Z", End: "2026-01-01T00:00:00Z"},
		Rows:            []dataset.Row{{Timestamp: 1767225600000, SeriesID: "temp_c", Value: 12.4}},
	}

	entry, err := SaveDataset(cfg, req)
	if err != nil {
		t.Fatalf("SaveDataset: %v", err)
	}
	if entry.ID != "readings" {
		t.Errorf("id = %q, want readings", entry.ID)
	}
	if entry.Kind != "measurement" {
		t.Errorf("kind = %q, want measurement", entry.Kind)
	}
	if _, err := time.Parse(time.RFC3339, entry.CreatedAt); err != nil {
		t.Errorf("created_at not RFC3339: %q", entry.CreatedAt)
	}
	if _, err := os.Stat(filepath.Join(cfg.Dir, "readings.parquet")); err != nil {
		t.Errorf("parquet not written: %v", err)
	}

	// A second save of the same source disambiguates.
	entry2, err := SaveDataset(cfg, req)
	if err != nil {
		t.Fatalf("second SaveDataset: %v", err)
	}
	if entry2.ID != "readings-2" {
		t.Errorf("second id = %q, want readings-2", entry2.ID)
	}
}

func TestSaveDatasetHonorsNameOverride(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	entry, err := SaveDataset(cfg, SaveRequest{NameOverride: "march", SourcePath: "/x/readings.csv"})
	if err != nil {
		t.Fatalf("SaveDataset: %v", err)
	}
	if entry.ID != "march" {
		t.Errorf("id = %q, want march", entry.ID)
	}
}
