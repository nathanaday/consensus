package lineage

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/store"
)

// CopyOptions configures a copy. An empty Name derives the new id from the
// source id (disambiguated by the catalog).
type CopyOptions struct {
	Name string
}

// Copy duplicates the dataset into a new immutable node whose parent is the
// receiver. It reads the source rows, writes them to a new Parquet file via the
// shared store save path (recording parent_id and origin "copy"), inserts the
// child into the graph, and returns it.
func (n *Node) Copy(opts CopyOptions) (*Node, error) {
	src := n.entry
	rows, err := store.ReadRows(n.parquetPath())
	if err != nil {
		return nil, fmt.Errorf("read source dataset %q: %w", src.ID, err)
	}

	name := opts.Name
	if name == "" {
		name = src.ID
	}

	entry, err := store.SaveDataset(n.g.cfg, store.SaveRequest{
		NameOverride:    name,
		SourcePath:      src.SourcePath,
		TimestampColumn: src.TimestampColumn,
		Series:          src.Series,
		RowCount:        src.RowCount,
		TimeRange:       src.TimeRange,
		Rows:            rows,
		ParentID:        src.ID,
		Origin:          "copy",
	})
	if err != nil {
		return nil, err
	}

	child := &Node{g: n.g, entry: entry}
	n.g.nodes[entry.ID] = child
	n.g.ids = append(n.g.ids, entry.ID)
	sort.Strings(n.g.ids)
	return child, nil
}
