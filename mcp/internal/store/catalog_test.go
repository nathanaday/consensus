package store

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestPutWritesAtomicallyNoLeftoverTemp(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	if err := cat.Put(dataset.Entry{ID: "readings", Kind: "measurement"}); err != nil {
		t.Fatalf("Put 1: %v", err)
	}
	if err := cat.Put(dataset.Entry{ID: "readings-2", Kind: "measurement"}); err != nil {
		t.Fatalf("Put 2: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != catalogFile {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("dir contains %v, want only %q", names, catalogFile)
	}

	b, err := os.ReadFile(filepath.Join(dir, catalogFile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var parsed map[string]dataset.Entry
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("catalog.json is not valid JSON: %v", err)
	}
	if _, ok := parsed["readings"]; !ok {
		t.Errorf("catalog.json missing %q entry", "readings")
	}
	if _, ok := parsed["readings-2"]; !ok {
		t.Errorf("catalog.json missing %q entry", "readings-2")
	}
}

func TestCatalogEntriesSortedByID(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	for _, id := range []string{"zulu", "alpha", "mike"} {
		if err := cat.Put(dataset.Entry{ID: id, Kind: "measurement"}); err != nil {
			t.Fatalf("Put %q: %v", id, err)
		}
	}

	entries := cat.Entries()
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	got := []string{entries[0].ID, entries[1].ID, entries[2].ID}
	want := []string{"alpha", "mike", "zulu"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entries order = %v, want %v", got, want)
		}
	}
}
