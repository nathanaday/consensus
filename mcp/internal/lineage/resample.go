package lineage

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/analysis"
	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// ResampleOptions configures a resample transform. Agg is one of "mean",
// "min", "max", "median". BucketMS must be at least the source's median
// sampling interval (resampling must reduce, not upsample).
type ResampleOptions struct {
	Name       string
	Start, End string
	BucketMS   int64
	Agg        string
}

// ResampleResult is the created child plus the source row count consumed.
type ResampleResult struct {
	Child          *Node
	SourceRowCount int
}

func bucketValue(b analysis.Bucket, agg string) (float64, error) {
	switch agg {
	case "mean", "":
		return b.Mean, nil
	case "min":
		return b.Min, nil
	case "max":
		return b.Max, nil
	case "median":
		return b.Median, nil
	default:
		return 0, fmt.Errorf("unknown agg %q; use mean, min, max, or median", agg)
	}
}

// Resample buckets the window and writes one row per non-empty bucket (its
// aggregate) to a new immutable child dataset (origin "resample").
func (n *Node) Resample(opts ResampleOptions) (ResampleResult, error) {
	src := n.entry
	rows, err := store.ReadRows(n.parquetPath())
	if err != nil {
		return ResampleResult{}, fmt.Errorf("read source dataset %q: %w", src.ID, err)
	}
	windowed, err := analysis.Window(rows, opts.Start, opts.End)
	if err != nil {
		return ResampleResult{}, err
	}
	if len(windowed) == 0 {
		return ResampleResult{}, fmt.Errorf("dataset %q has no rows in the requested window; nothing to resample", src.ID)
	}
	if med := analysis.MedianIntervalMS(windowed); opts.BucketMS < med {
		return ResampleResult{}, fmt.Errorf("bucket %dms is smaller than the median sampling interval %dms; resampling must reduce, not upsample", opts.BucketMS, med)
	}

	buckets, err := analysis.Buckets(windowed, opts.BucketMS)
	if err != nil {
		return ResampleResult{}, err
	}
	out := make([]dataset.Row, 0, len(buckets))
	for _, b := range buckets {
		if b.Count == 0 {
			continue
		}
		v, err := bucketValue(b, opts.Agg)
		if err != nil {
			return ResampleResult{}, err
		}
		out = append(out, dataset.Row{Timestamp: b.StartMS, Value: v})
	}
	if len(out) == 0 {
		return ResampleResult{}, fmt.Errorf("resampling dataset %q produced no non-empty buckets", src.ID)
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
		RowCount:        len(out),
		TimeRange:       analysis.Span(out),
		Rows:            out,
		ParentID:        src.ID,
		Origin:          "resample",
	})
	if err != nil {
		return ResampleResult{}, err
	}
	child := &Node{g: n.g, entry: entry}
	n.g.nodes[entry.ID] = child
	n.g.ids = append(n.g.ids, entry.ID)
	sort.Strings(n.g.ids)
	return ResampleResult{Child: child, SourceRowCount: len(windowed)}, nil
}
