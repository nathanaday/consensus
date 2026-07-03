// Package store persists datasets as Parquet files and maintains a JSON
// catalog. It has no knowledge of CSV or ingestion.
package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config locates the on-disk store.
type Config struct {
	Dir string
}

// Resolve determines the store directory from CONSENSUS_STORE_DIR (falling back
// to ~/.consensus/store) and ensures it exists.
func Resolve() (Config, error) {
	dir := os.Getenv("CONSENSUS_STORE_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("resolve home dir: %w", err)
		}
		dir = filepath.Join(home, ".consensus", "store")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Config{}, fmt.Errorf("create store dir %q: %w", dir, err)
	}
	return Config{Dir: dir}, nil
}
