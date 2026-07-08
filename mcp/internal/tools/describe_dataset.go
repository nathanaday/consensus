package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type DescribeDatasetInput struct {
	ID string `json:"id" jsonschema:"the dataset id to describe"`
}

// DatasetRef is a lineage reference to another dataset.
type DatasetRef struct {
	ID     string `json:"id"`
	Origin string `json:"origin,omitempty"`
}

type DescribeDatasetOutput struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	SourceColumn string            `json:"source_column"`
	Unit         string            `json:"unit,omitempty"`
	RowCount     int               `json:"row_count"`
	TimeRange    dataset.TimeRange `json:"time_range"`
	SizeBytes    int64             `json:"size_bytes"`
	SourcePath   string            `json:"source_path"`
	CreatedAt    string            `json:"created_at"`
	Origin       string            `json:"origin,omitempty"`
	Parent       *DatasetRef       `json:"parent"`
	Children     []DatasetRef      `json:"children"`
}

// describeNode builds the describe view for a node. Shared with copy_dataset.
func describeNode(n *lineage.Node) DescribeDatasetOutput {
	e := n.Info()

	var parent *DatasetRef
	if p := n.Parent(); p != nil {
		pe := p.Info()
		parent = &DatasetRef{ID: pe.ID, Origin: pe.Origin}
	}

	children := make([]DatasetRef, 0)
	for _, c := range n.Children() {
		ce := c.Info()
		children = append(children, DatasetRef{ID: ce.ID, Origin: ce.Origin})
	}

	return DescribeDatasetOutput{
		ID:           e.ID,
		Kind:         e.Kind,
		SourceColumn: e.SourceColumn,
		Unit:         e.Unit,
		RowCount:     e.RowCount,
		TimeRange:    e.TimeRange,
		SizeBytes:    n.SizeBytes(),
		SourcePath:   e.SourcePath,
		CreatedAt:    e.CreatedAt,
		Origin:       e.Origin,
		Parent:       parent,
		Children:     children,
	}
}

// DescribeDataset returns one dataset's full metadata plus its lineage (parent
// and children). It returns no row data.
func DescribeDataset(ctx context.Context, req *mcp.CallToolRequest, input DescribeDatasetInput) (*mcp.CallToolResult, DescribeDatasetOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, DescribeDatasetOutput{}, err
	}
	n, err := g.Node(input.ID)
	if err != nil {
		return nil, DescribeDatasetOutput{}, err
	}
	return nil, describeNode(n), nil
}
