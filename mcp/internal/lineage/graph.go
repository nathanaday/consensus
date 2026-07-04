// Package lineage is the object+pointer SDK over the dataset catalog. It loads
// the flat catalog into a graph of nodes so tools can traverse parentage,
// copy datasets, and render the lineage without touching storage internals.
package lineage

import (
	"fmt"
	"sort"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

// Graph is an in-memory view of one store's catalog as a lineage forest.
type Graph struct {
	cfg   store.Config
	nodes map[string]*Node
	ids   []string // all ids, sorted
}

// Node wraps one catalog entry and links back to its graph for traversal.
type Node struct {
	g     *Graph
	entry dataset.Entry
}

// Open resolves the store directory, loads its catalog, and builds the graph.
func Open() (*Graph, error) {
	cfg, err := store.Resolve()
	if err != nil {
		return nil, err
	}
	cat, err := store.LoadCatalog(cfg.Dir)
	if err != nil {
		return nil, err
	}
	return newGraph(cfg, cat.Entries()), nil
}

func newGraph(cfg store.Config, entries []dataset.Entry) *Graph {
	g := &Graph{
		cfg:   cfg,
		nodes: make(map[string]*Node, len(entries)),
		ids:   make([]string, 0, len(entries)),
	}
	for _, e := range entries {
		g.nodes[e.ID] = &Node{g: g, entry: e}
		g.ids = append(g.ids, e.ID)
	}
	sort.Strings(g.ids)
	return g
}

// Node returns the node with the given id, or an error naming the known ids.
func (g *Graph) Node(id string) (*Node, error) {
	if n, ok := g.nodes[id]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("unknown dataset %q; known datasets: %v", id, g.ids)
}

// Roots returns the datasets with no parent (loaded from a source file), sorted.
func (g *Graph) Roots() []*Node {
	out := make([]*Node, 0)
	for _, id := range g.ids {
		if g.nodes[id].entry.ParentID == "" {
			out = append(out, g.nodes[id])
		}
	}
	return out
}

// Nodes returns every node, sorted by id.
func (g *Graph) Nodes() []*Node {
	out := make([]*Node, 0, len(g.ids))
	for _, id := range g.ids {
		out = append(out, g.nodes[id])
	}
	return out
}

// ID is the dataset's id.
func (n *Node) ID() string { return n.entry.ID }

// Info returns the underlying catalog entry.
func (n *Node) Info() dataset.Entry { return n.entry }

// Parent returns the dataset this one was copied from, or nil for a root.
func (n *Node) Parent() *Node {
	if n.entry.ParentID == "" {
		return nil
	}
	return n.g.nodes[n.entry.ParentID]
}

// Children returns the datasets copied directly from this one, sorted by id.
func (n *Node) Children() []*Node {
	out := make([]*Node, 0)
	for _, id := range n.g.ids {
		if n.g.nodes[id].entry.ParentID == n.entry.ID {
			out = append(out, n.g.nodes[id])
		}
	}
	return out
}
