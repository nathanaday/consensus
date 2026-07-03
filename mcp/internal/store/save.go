package store

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// SaveRequest carries everything needed to persist one dataset. It is expressed
// in dataset-level terms so store stays independent of ingestion.
type SaveRequest struct {
	NameOverride    string
	SourcePath      string
	TimestampColumn string
	SeriesIDs       []string
	RowCount        int
	TimeRange       dataset.TimeRange
	Rows            []dataset.Row
}

// SaveDataset allocates an id, writes the Parquet file, and records the catalog
// entry. Each call produces a new immutable dataset.
func SaveDataset(cfg Config, req SaveRequest) (dataset.Entry, error) {
	cat, err := LoadCatalog(cfg.Dir)
	if err != nil {
		return dataset.Entry{}, err
	}

	base := req.NameOverride
	if base == "" {
		base = Slug(req.SourcePath)
	} else {
		base = Slug(base)
	}
	id := cat.AllocateID(base)

	if err := WriteRows(filepath.Join(cfg.Dir, id+".parquet"), req.Rows); err != nil {
		return dataset.Entry{}, err
	}

	entry := dataset.Entry{
		ID:              id,
		Kind:            "measurement",
		SourcePath:      req.SourcePath,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		TimestampColumn: req.TimestampColumn,
		SeriesIDs:       req.SeriesIDs,
		RowCount:        req.RowCount,
		TimeRange:       req.TimeRange,
	}
	if err := cat.Put(entry); err != nil {
		return dataset.Entry{}, fmt.Errorf("record catalog entry: %w", err)
	}
	return entry, nil
}
