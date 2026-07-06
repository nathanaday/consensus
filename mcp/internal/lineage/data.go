package lineage

import (
	"os"
	"path/filepath"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// LoadData reads the dataset's canonical long-format rows from Parquet.
func (n *Node) LoadData() ([]dataset.Row, error) {
	return store.ReadRows(n.parquetPath())
}

// SizeBytes is the on-disk size of the dataset's Parquet file, or 0 if missing.
func (n *Node) SizeBytes() int64 {
	fi, err := os.Stat(n.parquetPath())
	if err != nil {
		return 0
	}
	return fi.Size()
}

func (n *Node) parquetPath() string {
	return filepath.Join(n.g.cfg.Dir, n.entry.ID+".parquet")
}
