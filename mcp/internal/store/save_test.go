package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// TestSaveDatasetRemovesParquetWhenCatalogWriteFails verifies that a catalog
// write failure does not leave an orphaned Parquet file with no catalog entry
// pointing at it.
func TestSaveDatasetRemovesParquetWhenCatalogWriteFails(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}

	orig := catalogPut
	catalogPut = func(*Catalog, dataset.Entry) error { return errors.New("boom") }
	defer func() { catalogPut = orig }()

	_, err := SaveDataset(cfg, SaveRequest{
		SourcePath: "/abs/readings.csv",
		Rows:       []dataset.Row{{Timestamp: 1, SeriesID: "s", Value: 1}},
	})
	if err == nil {
		t.Fatal("expected an error when the catalog write fails")
	}
	if _, statErr := os.Stat(filepath.Join(cfg.Dir, "readings.parquet")); !os.IsNotExist(statErr) {
		t.Errorf("orphaned parquet left behind after catalog failure: stat err = %v", statErr)
	}
}

func TestSaveDatasetWritesParquetAndCatalog(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	req := SaveRequest{
		SourcePath:      "/abs/readings.csv",
		TimestampColumn: "time",
		Series:          []dataset.Series{{ID: "temp_c", Unit: "celsius"}},
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
	if len(entry.Series) != 1 || entry.Series[0].ID != "temp_c" || entry.Series[0].Unit != "celsius" {
		t.Errorf("series = %+v, want [{temp_c celsius}]", entry.Series)
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

func TestSaveDatasetRecordsLineage(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}

	if _, err := SaveDataset(cfg, SaveRequest{NameOverride: "root", SourcePath: "/x/root.csv", Origin: "csv"}); err != nil {
		t.Fatalf("save root: %v", err)
	}
	child, err := SaveDataset(cfg, SaveRequest{NameOverride: "child", ParentID: "root", Origin: "copy"})
	if err != nil {
		t.Fatalf("save child: %v", err)
	}
	if child.ParentID != "root" || child.Origin != "copy" {
		t.Errorf("child lineage = {%q,%q}, want {root,copy}", child.ParentID, child.Origin)
	}

	cat, err := LoadCatalog(cfg.Dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	var got dataset.Entry
	for _, e := range cat.Entries() {
		if e.ID == "child" {
			got = e
		}
	}
	if got.ParentID != "root" || got.Origin != "copy" {
		t.Errorf("persisted child lineage = {%q,%q}, want {root,copy}", got.ParentID, got.Origin)
	}
}
