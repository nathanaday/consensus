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

	orig := catalogPutAll
	catalogPutAll = func(*Catalog, []dataset.Entry) error { return errors.New("boom") }
	defer func() { catalogPutAll = orig }()

	_, err := SaveDataset(cfg, SaveRequest{
		SourcePath: "/abs/readings.csv",
		Rows:       []dataset.Row{{Timestamp: 1, Value: 1}},
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
		SourceColumn:    "temp_c",
		Unit:            "celsius",
		RowCount:        1,
		TimeRange:       dataset.TimeRange{Start: "2026-01-01T00:00:00Z", End: "2026-01-01T00:00:00Z"},
		Rows:            []dataset.Row{{Timestamp: 1767225600000, Value: 12.4}},
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
	if entry.SourceColumn != "temp_c" || entry.Unit != "celsius" {
		t.Errorf("channel = {%q %q}, want {temp_c celsius}", entry.SourceColumn, entry.Unit)
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

func groupReq() GroupRequest {
	return GroupRequest{
		SourcePath:      "/abs/readings.csv",
		TimestampColumn: "time",
		Origin:          "csv",
		Channels: []ChannelData{
			{
				Column: "temp_c", Unit: "celsius", RowCount: 1,
				TimeRange: dataset.TimeRange{Start: "2026-01-01T00:00:00Z", End: "2026-01-01T00:00:00Z"},
				Rows:      []dataset.Row{{Timestamp: 1767225600000, Value: 12.4}},
			},
			{
				Column: "humidity", RowCount: 1,
				TimeRange: dataset.TimeRange{Start: "2026-01-01T00:00:00Z", End: "2026-01-01T00:00:00Z"},
				Rows:      []dataset.Row{{Timestamp: 1767225600000, Value: 5.1}},
			},
		},
	}
}

func TestSaveGroupCreatesOneDatasetPerChannel(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}

	entries, err := SaveGroup(cfg, groupReq())
	if err != nil {
		t.Fatalf("SaveGroup: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].ID != "readings/temp_c" || entries[1].ID != "readings/humidity" {
		t.Errorf("ids = %q, %q; want readings/temp_c, readings/humidity", entries[0].ID, entries[1].ID)
	}
	if entries[0].SourceColumn != "temp_c" || entries[0].Unit != "celsius" {
		t.Errorf("temp_c entry channel = {%q %q}", entries[0].SourceColumn, entries[0].Unit)
	}
	if entries[1].Unit != "" {
		t.Errorf("humidity unit = %q, want empty (not recorded)", entries[1].Unit)
	}
	for _, id := range []string{"readings/temp_c", "readings/humidity"} {
		if _, err := os.Stat(filepath.Join(cfg.Dir, filepath.FromSlash(id)+".parquet")); err != nil {
			t.Errorf("parquet for %s not written: %v", id, err)
		}
	}

	cat, err := LoadCatalog(cfg.Dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !cat.Has("readings/temp_c") || !cat.Has("readings/humidity") {
		t.Error("catalog missing group entries after reload")
	}
}

func TestSaveGroupDisambiguatesTheGroup(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	if _, err := SaveGroup(cfg, groupReq()); err != nil {
		t.Fatalf("first SaveGroup: %v", err)
	}
	entries, err := SaveGroup(cfg, groupReq())
	if err != nil {
		t.Fatalf("second SaveGroup: %v", err)
	}
	if entries[0].ID != "readings-2/temp_c" || entries[1].ID != "readings-2/humidity" {
		t.Errorf("re-ingest ids = %q, %q; want readings-2/temp_c, readings-2/humidity (group suffixed, channels together)", entries[0].ID, entries[1].ID)
	}
}

func TestSaveGroupCleansUpWhenCatalogWriteFails(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}

	orig := catalogPutAll
	catalogPutAll = func(*Catalog, []dataset.Entry) error { return errors.New("boom") }
	defer func() { catalogPutAll = orig }()

	if _, err := SaveGroup(cfg, groupReq()); err == nil {
		t.Fatal("expected an error when the catalog write fails")
	}
	for _, id := range []string{"readings/temp_c", "readings/humidity"} {
		if _, statErr := os.Stat(filepath.Join(cfg.Dir, filepath.FromSlash(id)+".parquet")); !os.IsNotExist(statErr) {
			t.Errorf("orphaned parquet %s left behind: stat err = %v", id, statErr)
		}
	}
	cat, err := LoadCatalog(cfg.Dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cat.Entries()) != 0 {
		t.Errorf("catalog entries = %d, want 0", len(cat.Entries()))
	}
}

func TestSaveGroupRejectsCollidingChannelIDs(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	req := groupReq()
	req.Channels[1].Column = "temp c" // slugs to temp_c, same as channel 0
	if _, err := SaveGroup(cfg, req); err == nil {
		t.Fatal("expected an error for colliding channel ids")
	}
}

func TestSaveGroupRejectsEmptyChannels(t *testing.T) {
	cfg := Config{Dir: t.TempDir()}
	if _, err := SaveGroup(cfg, GroupRequest{SourcePath: "/x/y.csv"}); err == nil {
		t.Fatal("expected an error for a group with no channels")
	}
}
