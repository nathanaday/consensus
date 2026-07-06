package lineage_test

import (
	"testing"

	"github.com/nathanaday/consensus/mcp/internal/lineage"
)

func TestMermaidRendersForest(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CONSENSUS_STORE_DIR", dir)
	seed(t, dir, "readings", "", "csv")
	seed(t, dir, "readings-2", "readings", "copy")

	g, err := lineage.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	want := "graph TD\n" +
		"  readings[\"readings\"]\n" +
		"  readings_2[\"readings-2\"]\n" +
		"  readings -->|copy| readings_2"
	if got := lineage.Mermaid(g); got != want {
		t.Errorf("Mermaid mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestMermaidEmptyStore(t *testing.T) {
	t.Setenv("CONSENSUS_STORE_DIR", t.TempDir())
	g, err := lineage.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := lineage.Mermaid(g); got != "graph TD" {
		t.Errorf("empty Mermaid = %q, want %q", got, "graph TD")
	}
}
