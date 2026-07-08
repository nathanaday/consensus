package store

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// StorageFormat is the on-disk format the store writes datasets in.
const StorageFormat = "parquet"

// SupportedIngestFormats lists the source formats the server can ingest today.
// It grows as ingest tools are added.
func SupportedIngestFormats() []string {
	return []string{"csv"}
}

// Capabilities is a short, curated description of what the server can do today.
// It must stay truthful as the tool set grows.
func Capabilities() []string {
	return []string{
		"CSV ingest into a Parquet store",
		"Dataset catalog with schema, units, and time-range metadata",
	}
}

// ListStoreFiles returns the regular files under dir (subdirectories
// included) as slash-separated paths relative to dir, sorted by name.
// In-flight temp files (those a partial catalog write leaves behind, named
// with a ".tmp-" segment) are excluded so a concurrent ingest never leaks a
// transient name. The result is always non-nil.
func ListStoreFiles(dir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if strings.HasPrefix(d.Name(), catalogFile+".tmp-") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
