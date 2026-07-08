package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListStoreFilesSortsAndHidesTempFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"readings.parquet", "catalog.json", "catalog.json.tmp-123"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	files, err := ListStoreFiles(dir)
	if err != nil {
		t.Fatalf("ListStoreFiles: %v", err)
	}
	want := []string{"catalog.json", "readings.parquet"}
	if len(files) != len(want) {
		t.Fatalf("files = %v, want %v", files, want)
	}
	for i := range want {
		if files[i] != want[i] {
			t.Fatalf("files = %v, want %v", files, want)
		}
	}
}

func TestListStoreFilesEmptyDirReturnsNonNil(t *testing.T) {
	files, err := ListStoreFiles(t.TempDir())
	if err != nil {
		t.Fatalf("ListStoreFiles: %v", err)
	}
	if files == nil {
		t.Fatal("files is nil, want non-nil empty slice")
	}
	if len(files) != 0 {
		t.Fatalf("files = %v, want empty", files)
	}
}

func TestStoreFactConstants(t *testing.T) {
	if StorageFormat != "parquet" {
		t.Errorf("StorageFormat = %q, want parquet", StorageFormat)
	}
	formats := SupportedIngestFormats()
	if len(formats) != 1 || formats[0] != "csv" {
		t.Errorf("SupportedIngestFormats() = %v, want [csv]", formats)
	}
	if len(Capabilities()) == 0 {
		t.Error("Capabilities() is empty, want at least one entry")
	}
}

func TestListStoreFilesWalksSubdirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "iot"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, p := range []string{"catalog.json", "iot/temp.parquet"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(p)), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	files, err := ListStoreFiles(dir)
	if err != nil {
		t.Fatalf("ListStoreFiles: %v", err)
	}
	want := []string{"catalog.json", "iot/temp.parquet"}
	if len(files) != len(want) || files[0] != want[0] || files[1] != want[1] {
		t.Errorf("files = %v, want %v", files, want)
	}
}
