package lineage

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func TestResampleMeanReducesRows(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	// 6 points, 1 minute apart. Two 3-minute buckets.
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", SourceColumn: "temp_c", Unit: "celsius",
		RowCount: 6,
		Rows: []dataset.Row{
			{Timestamp: 0, Value: 2},
			{Timestamp: 60000, Value: 4},
			{Timestamp: 120000, Value: 6},
			{Timestamp: 180000, Value: 10},
			{Timestamp: 240000, Value: 20},
			{Timestamp: 300000, Value: 30},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	g, err := Open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	n, err := g.Node("readings")
	if err != nil {
		t.Fatalf("node: %v", err)
	}
	res, err := n.Resample(ResampleOptions{BucketMS: 180000, Agg: "mean"})
	if err != nil {
		t.Fatalf("resample: %v", err)
	}
	child := res.Child
	if child.Info().RowCount != 2 {
		t.Fatalf("want 2 rows, got %d", child.Info().RowCount)
	}
	if child.Info().Origin != "resample" {
		t.Errorf("want origin resample, got %q", child.Info().Origin)
	}
	if child.Info().Unit != "celsius" {
		t.Errorf("unit should carry over, got %q", child.Info().Unit)
	}
	rows, err := child.LoadData()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// bucket 0: mean(2,4,6)=4 ; bucket 1: mean(10,20,30)=20
	if rows[0].Value != 4 || rows[1].Value != 20 {
		t.Errorf("want values 4,20 got %v", rows)
	}
	if child.Parent() == nil || child.Parent().ID() != "readings" {
		t.Errorf("child parent should be readings")
	}
}

func TestResampleRejectsUpsampling(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: "readings", Origin: "csv", RowCount: 3,
		Rows: []dataset.Row{
			{Timestamp: 0, Value: 1}, {Timestamp: 60000, Value: 2}, {Timestamp: 120000, Value: 3},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	g, _ := Open()
	n, _ := g.Node("readings")
	// median interval 60000ms; a 1000ms bucket would upsample.
	if _, err := n.Resample(ResampleOptions{BucketMS: 1000, Agg: "mean"}); err == nil {
		t.Error("bucket smaller than median interval should error")
	}
}
