package lineage_test

import (
	"strings"
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// seedOutlierStore stores values {1,2,3,4,100} one minute apart as "readings"
// and returns its lineage node. Bounds at k=1.5 are [-1, 7].
func seedOutlierStore(t *testing.T) *lineage.Node {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv",
		SourceColumn: "temp_c", Unit: "celsius", TimestampColumn: "ts",
		RowCount: 5,
		Rows: []dataset.Row{
			{Timestamp: 0, Value: 1},
			{Timestamp: 60000, Value: 2},
			{Timestamp: 120000, Value: 3},
			{Timestamp: 180000, Value: 4},
			{Timestamp: 240000, Value: 100},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	g, err := lineage.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	n, err := g.Node("readings")
	if err != nil {
		t.Fatalf("Node: %v", err)
	}
	return n
}

func TestRemoveOutliersCreatesCleanedChild(t *testing.T) {
	root := seedOutlierStore(t)
	res, err := root.RemoveOutliers(lineage.RemoveOutliersOptions{IQRMultiplier: 1.5})
	if err != nil {
		t.Fatalf("RemoveOutliers: %v", err)
	}
	if res.RowsRemoved != 1 {
		t.Errorf("RowsRemoved = %d, want 1", res.RowsRemoved)
	}
	if res.Bounds.Lower != -1 || res.Bounds.Upper != 7 {
		t.Errorf("Bounds = %+v, want [-1, 7]", res.Bounds)
	}
	child := res.Child
	if child.ID() != "readings-2" {
		t.Errorf("child id = %q, want readings-2", child.ID())
	}
	info := child.Info()
	if info.Origin != "remove_outliers" {
		t.Errorf("origin = %q, want remove_outliers", info.Origin)
	}
	if info.ParentID != "readings" {
		t.Errorf("parent = %q, want readings", info.ParentID)
	}
	if info.RowCount != 4 {
		t.Errorf("row count = %d, want 4", info.RowCount)
	}
	if info.TimeRange.End != "1970-01-01T00:03:00Z" {
		t.Errorf("time range end = %q, want the last inlier's timestamp", info.TimeRange.End)
	}
	if info.SourceColumn != "temp_c" || info.Unit != "celsius" || info.TimestampColumn != "ts" {
		t.Errorf("source metadata not carried over: %+v", info)
	}
	rows, err := child.LoadData()
	if err != nil {
		t.Fatalf("LoadData: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("stored rows = %d, want 4", len(rows))
	}
	for _, r := range rows {
		if r.Value == 100 {
			t.Error("outlier stored in child dataset")
		}
	}
}

func TestRemoveOutliersWindowDoublesAsSlice(t *testing.T) {
	root := seedOutlierStore(t)
	res, err := root.RemoveOutliers(lineage.RemoveOutliersOptions{
		IQRMultiplier: 1.5,
		Start:         "1970-01-01T00:01:00Z",
		End:           "1970-01-01T00:03:00Z",
	})
	if err != nil {
		t.Fatalf("RemoveOutliers: %v", err)
	}
	// window {2,3,4} has no outliers; the child is just the slice
	if res.RowsRemoved != 0 {
		t.Errorf("RowsRemoved = %d, want 0", res.RowsRemoved)
	}
	if res.Child.Info().RowCount != 3 {
		t.Errorf("row count = %d, want 3 (window only)", res.Child.Info().RowCount)
	}
}

func TestRemoveOutliersNamedChild(t *testing.T) {
	root := seedOutlierStore(t)
	res, err := root.RemoveOutliers(lineage.RemoveOutliersOptions{IQRMultiplier: 1.5, Name: "clean"})
	if err != nil {
		t.Fatalf("RemoveOutliers: %v", err)
	}
	if res.Child.ID() != "clean" {
		t.Errorf("child id = %q, want clean", res.Child.ID())
	}
}

func TestRemoveOutliersEmptyWindowErrors(t *testing.T) {
	root := seedOutlierStore(t)
	_, err := root.RemoveOutliers(lineage.RemoveOutliersOptions{
		IQRMultiplier: 1.5,
		Start:         "1980-01-01T00:00:00Z",
	})
	if err == nil || !strings.Contains(err.Error(), "no rows") {
		t.Errorf("err = %v, want 'no rows'", err)
	}
}
