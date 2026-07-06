package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

const (
	previewDefaultLimit = 20
	previewMaxLimit     = 200
)

type PreviewDatasetInput struct {
	ID    string `json:"id" jsonschema:"the dataset id to preview"`
	Limit int    `json:"limit,omitempty" jsonschema:"number of rows to return (default 20, max 200)"`
}

type PreviewDatasetOutput struct {
	DatasetID string        `json:"dataset_id"`
	Returned  int           `json:"returned"`
	RowCount  int           `json:"row_count"`
	Rows      []dataset.Row `json:"rows"`
}

// PreviewDataset returns a bounded sample of a dataset's canonical long-format
// rows so a caller can eyeball the data. It is a preview, never an export:
// the limit defaults to 20 and is capped at 200.
func PreviewDataset(ctx context.Context, req *mcp.CallToolRequest, input PreviewDatasetInput) (*mcp.CallToolResult, PreviewDatasetOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, PreviewDatasetOutput{}, err
	}
	n, err := g.Node(input.ID)
	if err != nil {
		return nil, PreviewDatasetOutput{}, err
	}
	rows, err := n.LoadData()
	if err != nil {
		return nil, PreviewDatasetOutput{}, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = previewDefaultLimit
	}
	if limit > previewMaxLimit {
		limit = previewMaxLimit
	}
	if limit > len(rows) {
		limit = len(rows)
	}

	out := make([]dataset.Row, 0, limit)
	out = append(out, rows[:limit]...)

	return nil, PreviewDatasetOutput{
		DatasetID: n.ID(),
		Returned:  limit,
		RowCount:  len(rows),
		Rows:      out,
	}, nil
}
