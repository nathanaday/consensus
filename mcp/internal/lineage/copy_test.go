package lineage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestNodeCopyCreatesChildEdge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride:    "readings",
		Origin:          "csv",
		TimestampColumn: "time",
		Series:          []dataset.Series{{ID: "temp_c", Unit: "celsius"}},
		RowCount:        2,
		TimeRange:       dataset.TimeRange{Start: "2026-01-01T00:00:00Z", End: "2026-01-01T00:05:00Z"},
		Rows: []dataset.Row{
			{Timestamp: 1, SeriesID: "temp_c", Value: 1.5},
			{Timestamp: 2, SeriesID: "temp_c", Value: 2.5},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	g, err := lineage.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	root, err := g.Node("readings")
	if err != nil {
		t.Fatalf("Node: %v", err)
	}

	child, err := root.Copy(lineage.CopyOptions{})
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if child.ID() != "readings-2" {
		t.Errorf("child id = %q, want readings-2", child.ID())
	}
	if p := child.Parent(); p == nil || p.ID() != "readings" {
		t.Errorf("child.Parent() = %v, want readings", p)
	}
	if child.Info().Origin != "copy" {
		t.Errorf("child origin = %q, want copy", child.Info().Origin)
	}
	if len(child.Info().Series) != 1 || child.Info().Series[0].Unit != "celsius" {
		t.Errorf("child series not carried over: %+v", child.Info().Series)
	}

	rows, err := child.LoadData()
	if err != nil {
		t.Fatalf("LoadData: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("copied rows = %d, want 2", len(rows))
	}
	if child.SizeBytes() <= 0 {
		t.Errorf("child SizeBytes = %d, want > 0", child.SizeBytes())
	}
	if _, err := os.Stat(filepath.Join(dir, "readings-2.parquet")); err != nil {
		t.Errorf("child parquet not written: %v", err)
	}

	// The graph reflects the new child immediately.
	kids := root.Children()
	if len(kids) != 1 || kids[0].ID() != "readings-2" {
		t.Errorf("root children after copy = %v, want [readings-2]", kids)
	}

	// A named copy uses the given id.
	named, err := root.Copy(lineage.CopyOptions{Name: "march"})
	if err != nil {
		t.Fatalf("named Copy: %v", err)
	}
	if named.ID() != "march" {
		t.Errorf("named copy id = %q, want march", named.ID())
	}
}
