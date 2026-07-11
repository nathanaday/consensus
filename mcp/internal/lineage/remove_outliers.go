package lineage

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// RemoveOutliersOptions configures an outlier-removal transform. An empty
// Name derives the new id from the source id (disambiguated by the catalog).
// Start/End optionally window the source first, so the child is that slice
// minus its outliers. IQRMultiplier must be set by the caller.
type RemoveOutliersOptions struct {
	Name          string
	Start, End    string
	IQRMultiplier float64
}

// RemoveOutliersResult is the created child plus what the transform did.
type RemoveOutliersResult struct {
	Child       *Node
	Bounds      analysis.Bounds
	RowsRemoved int
}

// RemoveOutliers writes the window's IQR inliers to a new immutable child
// dataset (origin "remove_outliers") and returns it. Bounds are computed
// over the window, not the whole source.
func (n *Node) RemoveOutliers(opts RemoveOutliersOptions) (RemoveOutliersResult, error) {
	src := n.entry
	rows, err := store.ReadRows(n.parquetPath())
	if err != nil {
		return RemoveOutliersResult{}, fmt.Errorf("read source dataset %q: %w", src.ID, err)
	}
	windowed, err := analysis.Window(rows, opts.Start, opts.End)
	if err != nil {
		return RemoveOutliersResult{}, err
	}
	if len(windowed) == 0 {
		return RemoveOutliersResult{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to analyze", src.ID)
	}
	inliers, bounds, err := analysis.RemoveOutliers(windowed, opts.IQRMultiplier)
	if err != nil {
		return RemoveOutliersResult{}, err
	}
	if len(inliers) == 0 {
		return RemoveOutliersResult{}, fmt.Errorf("every row of dataset %q in the window is an outlier at multiplier %g; refusing to create an empty dataset", src.ID, opts.IQRMultiplier)
	}

	name := opts.Name
	if name == "" {
		name = src.ID
	}
	entry, err := store.SaveDataset(n.g.cfg, store.SaveRequest{
		NameOverride:    name,
		SourcePath:      src.SourcePath,
		SourceColumn:    src.SourceColumn,
		Unit:            src.Unit,
		TimestampColumn: src.TimestampColumn,
		RowCount:        len(inliers),
		TimeRange:       analysis.Span(inliers),
		Rows:            inliers,
		ParentID:        src.ID,
		Origin:          "remove_outliers",
	})
	if err != nil {
		return RemoveOutliersResult{}, err
	}
	child := &Node{g: n.g, entry: entry}
	n.g.nodes[entry.ID] = child
	n.g.ids = append(n.g.ids, entry.ID)
	sort.Strings(n.g.ids)
	return RemoveOutliersResult{Child: child, Bounds: bounds, RowsRemoved: len(windowed) - len(inliers)}, nil
}
