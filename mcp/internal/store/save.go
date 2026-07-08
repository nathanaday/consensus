package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// catalogPutAll records entries. It is a package var so the catalog-write
// failure path (which the save functions clean up after) can be exercised in
// tests; the directory-sharing between the Parquet writes and the catalog
// makes that failure impractical to trigger through the filesystem alone.
var catalogPutAll = (*Catalog).PutAll

// ChannelData carries one channel of a group save.
type ChannelData struct {
	Column    string
	Unit      string
	Rows      []dataset.Row
	RowCount  int
	TimeRange dataset.TimeRange
}

// GroupRequest persists one ingested source as a group of channel datasets
// named <group>/<column>.
type GroupRequest struct {
	NameOverride    string // group id base; defaults to the source filename
	SourcePath      string
	TimestampColumn string
	Channels        []ChannelData
	ParentID        string // "" for roots loaded from a file
	Origin          string
}

// SaveRequest persists a single dataset. NameOverride is a full id base (flat
// or group/channel); copies use this path.
type SaveRequest struct {
	NameOverride    string
	SourcePath      string
	SourceColumn    string
	Unit            string
	TimestampColumn string
	RowCount        int
	TimeRange       dataset.TimeRange
	Rows            []dataset.Row
	ParentID        string
	Origin          string
}

// defaultBase derives an id base from the override, falling back to the
// source filename without its extension.
func defaultBase(override, sourcePath string) string {
	if override != "" {
		return override
	}
	b := filepath.Base(sourcePath)
	return strings.TrimSuffix(b, filepath.Ext(b))
}

// SaveGroup allocates a group id, writes one Parquet file per channel, and
// commits every catalog entry in a single write. The catalog write is the
// commit point: on any failure the files written so far are removed and no
// entry appears.
func SaveGroup(cfg Config, req GroupRequest) ([]dataset.Entry, error) {
	if len(req.Channels) == 0 {
		return nil, fmt.Errorf("save group: no channels")
	}
	cat, err := LoadCatalog(cfg.Dir)
	if err != nil {
		return nil, err
	}
	group := cat.AllocateGroupID(Slug(defaultBase(req.NameOverride, req.SourcePath)))

	seen := make(map[string]string, len(req.Channels))
	for _, ch := range req.Channels {
		id := Slug(ch.Column)
		if prev, ok := seen[id]; ok {
			return nil, fmt.Errorf("columns %q and %q collide on channel id %q; rename one or narrow value_cols", prev, ch.Column, id)
		}
		seen[id] = ch.Column
	}

	now := time.Now().UTC().Format(time.RFC3339)
	entries := make([]dataset.Entry, 0, len(req.Channels))
	written := make([]string, 0, len(req.Channels))
	cleanup := func() {
		for _, p := range written {
			os.Remove(p)
		}
		os.Remove(filepath.Join(cfg.Dir, group)) // group dir, if now empty
	}
	for _, ch := range req.Channels {
		id := group + "/" + Slug(ch.Column)
		path := filepath.Join(cfg.Dir, id+".parquet")
		if err := WriteRows(path, ch.Rows); err != nil {
			cleanup()
			return nil, fmt.Errorf("write channel %q: %w", ch.Column, err)
		}
		written = append(written, path)
		entries = append(entries, dataset.Entry{
			ID:              id,
			Kind:            "measurement",
			SourcePath:      req.SourcePath,
			SourceColumn:    ch.Column,
			Unit:            ch.Unit,
			CreatedAt:       now,
			TimestampColumn: req.TimestampColumn,
			RowCount:        ch.RowCount,
			TimeRange:       ch.TimeRange,
			ParentID:        req.ParentID,
			Origin:          req.Origin,
		})
	}
	if err := catalogPutAll(cat, entries); err != nil {
		cleanup()
		return nil, fmt.Errorf("record catalog entries: %w", err)
	}
	return entries, nil
}

// SaveDataset allocates an id, writes the Parquet file, and records the
// catalog entry. Each call produces a new immutable dataset.
func SaveDataset(cfg Config, req SaveRequest) (dataset.Entry, error) {
	cat, err := LoadCatalog(cfg.Dir)
	if err != nil {
		return dataset.Entry{}, err
	}
	id := cat.AllocateID(Slug(defaultBase(req.NameOverride, req.SourcePath)))

	path := filepath.Join(cfg.Dir, id+".parquet")
	if err := WriteRows(path, req.Rows); err != nil {
		return dataset.Entry{}, err
	}

	entry := dataset.Entry{
		ID:              id,
		Kind:            "measurement",
		SourcePath:      req.SourcePath,
		SourceColumn:    req.SourceColumn,
		Unit:            req.Unit,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		TimestampColumn: req.TimestampColumn,
		RowCount:        req.RowCount,
		TimeRange:       req.TimeRange,
		ParentID:        req.ParentID,
		Origin:          req.Origin,
	}
	if err := catalogPutAll(cat, []dataset.Entry{entry}); err != nil {
		// The catalog entry is the commit point; without it the Parquet file
		// is unreachable, so remove it rather than leave an orphan behind.
		os.Remove(path)
		return dataset.Entry{}, fmt.Errorf("record catalog entry: %w", err)
	}
	return entry, nil
}
