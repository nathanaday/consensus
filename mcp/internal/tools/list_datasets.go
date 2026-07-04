package tools

import (
	"context"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

type ListDatasetsInput struct{}

type DatasetSummary struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Series     []dataset.Series  `json:"series"`
	RowCount   int               `json:"row_count"`
	TimeRange  dataset.TimeRange `json:"time_range"`
	SizeBytes  int64             `json:"size_bytes"`
	SourcePath string            `json:"source_path"`
	CreatedAt  string            `json:"created_at"`
}

type ListDatasetsOutput struct {
	Datasets []DatasetSummary `json:"datasets"`
}

// ListDatasets returns catalog metadata for every stored dataset. It is a pure
// read: schema, stats, and file size only, never row values.
func ListDatasets(ctx context.Context, req *mcp.CallToolRequest, input ListDatasetsInput) (*mcp.CallToolResult, ListDatasetsOutput, error) {
	cfg, err := store.Resolve()
	if err != nil {
		return nil, ListDatasetsOutput{}, err
	}
	cat, err := store.LoadCatalog(cfg.Dir)
	if err != nil {
		return nil, ListDatasetsOutput{}, err
	}

	entries := cat.Entries()
	summaries := make([]DatasetSummary, 0, len(entries))
	for _, e := range entries {
		var size int64
		if fi, err := os.Stat(filepath.Join(cfg.Dir, e.ID+".parquet")); err == nil {
			size = fi.Size()
		}
		summaries = append(summaries, DatasetSummary{
			ID:         e.ID,
			Kind:       e.Kind,
			Series:     e.Series,
			RowCount:   e.RowCount,
			TimeRange:  e.TimeRange,
			SizeBytes:  size,
			SourcePath: e.SourcePath,
			CreatedAt:  e.CreatedAt,
		})
	}

	return nil, ListDatasetsOutput{Datasets: summaries}, nil
}
