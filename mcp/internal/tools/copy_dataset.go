package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type CopyDatasetInput struct {
	ID   string `json:"id" jsonschema:"the dataset id to copy"`
	Name string `json:"name,omitempty" jsonschema:"optional id for the new copy; defaults to a disambiguated form of the source id"`
}

// CopyDataset creates an immutable copy of a dataset as a child in the lineage
// graph and returns the new dataset's description (including its parent edge).
func CopyDataset(ctx context.Context, req *mcp.CallToolRequest, input CopyDatasetInput) (*mcp.CallToolResult, DescribeDatasetOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, DescribeDatasetOutput{}, err
	}
	n, err := g.Node(input.ID)
	if err != nil {
		return nil, DescribeDatasetOutput{}, err
	}
	child, err := n.Copy(lineage.CopyOptions{Name: input.Name})
	if err != nil {
		return nil, DescribeDatasetOutput{}, err
	}
	return nil, describeNode(child), nil
}
