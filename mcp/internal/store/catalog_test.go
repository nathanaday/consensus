package store

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"readings.csv":      "readings",
		"March Data.CSV":    "march_data",
		"/abs/path/foo.csv": "foo",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCatalogAllocateAndPersist(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	id1 := cat.AllocateID("readings")
	if id1 != "readings" {
		t.Errorf("first id = %q, want readings", id1)
	}
	if err := cat.Put(dataset.Entry{ID: id1, Kind: "measurement"}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	id2 := cat.AllocateID("readings")
	if id2 != "readings-2" {
		t.Errorf("second id = %q, want readings-2", id2)
	}

	// Reload from disk and confirm persistence.
	reloaded, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.Has("readings") {
		t.Errorf("reloaded catalog missing readings entry")
	}
}
