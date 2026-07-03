package store

import (
	"bytes"
	"fmt"
	"os"

	"github.com/parquet-go/parquet-go"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

// WriteRows writes rows to a Parquet file at path.
func WriteRows(path string, rows []dataset.Row) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	if err := parquet.Write(f, rows); err != nil {
		f.Close()
		return fmt.Errorf("write parquet: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %q: %w", path, err)
	}
	return nil
}

// ReadRows reads all rows from a Parquet file at path.
func ReadRows(path string) ([]dataset.Row, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	rows, err := parquet.Read[dataset.Row](bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, fmt.Errorf("read parquet: %w", err)
	}
	return rows, nil
}
