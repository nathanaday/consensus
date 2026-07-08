package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// catalogPut records an entry. It is a package var so the catalog-write failure
// path (which SaveDataset cleans up after) can be exercised in tests; the
// directory-sharing between the Parquet write and the catalog makes that
// failure impractical to trigger through the filesystem alone.
var catalogPut = (*Catalog).Put

// SaveRequest carries everything needed to persist one dataset. It is expressed
// in dataset-level terms so store stays independent of ingestion.
type SaveRequest struct {
	NameOverride    string
	SourcePath      string
	TimestampColumn string
	Series          []dataset.Series
	RowCount        int
	TimeRange       dataset.TimeRange
	Rows            []dataset.Row
	ParentID        string
	Origin          string
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
		b := filepath.Base(req.SourcePath)
		base = strings.TrimSuffix(b, filepath.Ext(b))
	}
	id := cat.AllocateID(Slug(base))

	parquetPath := filepath.Join(cfg.Dir, id+".parquet")
	if err := WriteRows(parquetPath, req.Rows); err != nil {
		return dataset.Entry{}, err
	}

	entry := dataset.Entry{
		ID:              id,
		Kind:            "measurement",
		SourcePath:      req.SourcePath,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		TimestampColumn: req.TimestampColumn,
		Series:          req.Series,
		RowCount:        req.RowCount,
		TimeRange:       req.TimeRange,
		ParentID:        req.ParentID,
		Origin:          req.Origin,
	}
	if err := catalogPut(cat, entry); err != nil {
		// The catalog entry is the commit point; without it the Parquet file is
		// unreachable, so remove it rather than leave an orphan behind.
		_ = os.Remove(parquetPath)
		return dataset.Entry{}, fmt.Errorf("record catalog entry: %w", err)
	}
	return entry, nil
}
