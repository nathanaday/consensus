package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

type DatasetGraphInput struct {
	Format string `json:"format,omitempty" jsonschema:"output format: \"mermaid\" (default) or \"json\""`
}

// GraphNode is one dataset's adjacency in the JSON graph rendering.
type GraphNode struct {
	ParentID string   `json:"parent_id"`
	Origin   string   `json:"origin,omitempty"`
	Children []string `json:"children"`
}

// GraphJSON is the structured adjacency view of the whole forest.
type GraphJSON struct {
	Roots []string             `json:"roots"`
	Nodes map[string]GraphNode `json:"nodes"`
}

type DatasetGraphOutput struct {
	Format  string     `json:"format"`
	Mermaid string     `json:"mermaid,omitempty"`
	Graph   *GraphJSON `json:"graph,omitempty"`
}

// DatasetGraph renders the whole lineage forest as Mermaid (default) or as a
// structured JSON adjacency graph.
func DatasetGraph(ctx context.Context, req *mcp.CallToolRequest, input DatasetGraphInput) (*mcp.CallToolResult, DatasetGraphOutput, error) {
	g, err := lineage.Open()
	if err != nil {
		return nil, DatasetGraphOutput{}, err
	}

	format := input.Format
	if format == "" {
		format = "mermaid"
	}

	switch format {
	case "mermaid":
		return nil, DatasetGraphOutput{Format: "mermaid", Mermaid: lineage.Mermaid(g)}, nil
	case "json":
		roots := make([]string, 0)
		for _, r := range g.Roots() {
			roots = append(roots, r.ID())
		}
		nodes := make(map[string]GraphNode)
		for _, n := range g.Nodes() {
			e := n.Info()
			childIDs := make([]string, 0)
			for _, c := range n.Children() {
				childIDs = append(childIDs, c.ID())
			}
			nodes[e.ID] = GraphNode{ParentID: e.ParentID, Origin: e.Origin, Children: childIDs}
		}
		return nil, DatasetGraphOutput{Format: "json", Graph: &GraphJSON{Roots: roots, Nodes: nodes}}, nil
	default:
		return nil, DatasetGraphOutput{}, fmt.Errorf("unknown format %q; valid formats: mermaid, json", format)
	}
}
