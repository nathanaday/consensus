package lineage_test

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/lineage"
	"github.com/nathanaday/consensus/mcp/internal/store"
)

func seed(t *testing.T, dir, name, parent, origin string) {
	t.Helper()
	if _, err := store.SaveDataset(store.Config{Dir: dir}, store.SaveRequest{
		NameOverride: name,
		ParentID:     parent,
		Origin:       origin,
	}); err != nil {
		t.Fatalf("seed %q: %v", name, err)
	}
}

func TestGraphTraversal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seed(t, dir, "root", "", "csv")
	seed(t, dir, "child_a", "root", "copy")
	seed(t, dir, "child_b", "root", "copy")
	seed(t, dir, "grand", "child_a", "copy")

	g, err := lineage.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	roots := g.Roots()
	if len(roots) != 1 || roots[0].ID() != "root" {
		t.Fatalf("roots = %v, want [root]", roots)
	}

	root, err := g.Node("root")
	if err != nil {
		t.Fatalf("Node(root): %v", err)
	}
	if root.Parent() != nil {
		t.Errorf("root.Parent() = %v, want nil", root.Parent())
	}
	kids := root.Children()
	if len(kids) != 2 || kids[0].ID() != "child_a" || kids[1].ID() != "child_b" {
		t.Errorf("root children = %v, want [child_a child_b] sorted", kids)
	}

	grand, err := g.Node("grand")
	if err != nil {
		t.Fatalf("Node(grand): %v", err)
	}
	if p := grand.Parent(); p == nil || p.ID() != "child_a" {
		t.Errorf("grand.Parent() = %v, want child_a", p)
	}
	if grand.Info().Origin != "copy" {
		t.Errorf("grand origin = %q, want copy", grand.Info().Origin)
	}

	if _, err := g.Node("nope"); err == nil {
		t.Error("Node(nope): expected error for unknown id")
	}

	if got := len(g.Nodes()); got != 4 {
		t.Errorf("Nodes() count = %d, want 4", got)
	}
}
