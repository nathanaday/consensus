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
		"readings":       "readings",
		"March Data":     "march_data",
		"iot/temp":       "iot/temp",
		"IoT Rig/Temp C": "iot_rig/temp_c",
		"///":            "dataset",
		"":               "dataset",
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

func TestAllocateGroupID(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}

	if got := cat.AllocateGroupID("iot"); got != "iot" {
		t.Errorf("empty catalog group = %q, want iot", got)
	}

	// A flat dataset named exactly "iot" does not collide with group "iot".
	if err := cat.Put(dataset.Entry{ID: "iot", Kind: "measurement"}); err != nil {
		t.Fatalf("Put flat: %v", err)
	}
	if got := cat.AllocateGroupID("iot"); got != "iot" {
		t.Errorf("group after flat id = %q, want iot", got)
	}

	// Any dataset under iot/ makes the group collide.
	if err := cat.Put(dataset.Entry{ID: "iot/temp", Kind: "measurement"}); err != nil {
		t.Fatalf("Put channel: %v", err)
	}
	if got := cat.AllocateGroupID("iot"); got != "iot-2" {
		t.Errorf("group after channel = %q, want iot-2", got)
	}
}

func TestPutAllCommitsEveryEntryAtomically(t *testing.T) {
	dir := t.TempDir()
	cat, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	entries := []dataset.Entry{
		{ID: "iot/humidity", Kind: "measurement"},
		{ID: "iot/smoke", Kind: "measurement"},
		{ID: "iot/temp", Kind: "measurement"},
	}
	if err := cat.PutAll(entries); err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	reloaded, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, e := range entries {
		if !reloaded.Has(e.ID) {
			t.Errorf("reloaded catalog missing %q", e.ID)
		}
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 1 || files[0].Name() != catalogFile {
		t.Errorf("store dir should contain only %q after PutAll, got %d entries", catalogFile, len(files))
	}
}
