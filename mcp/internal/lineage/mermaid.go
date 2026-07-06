package lineage

import "strings"

// Mermaid renders the lineage forest as a Mermaid flowchart (graph TD): one node
// per dataset labeled with its real id, one edge per parent->child labeled with
// the child's origin. An empty store yields just "graph TD".
func Mermaid(g *Graph) string {
	var b strings.Builder
	b.WriteString("graph TD")
	for _, id := range g.ids {
		b.WriteString("\n  ")
		b.WriteString(sanitize(id))
		b.WriteString("[\"")
		b.WriteString(id)
		b.WriteString("\"]")
	}
	for _, id := range g.ids {
		e := g.nodes[id].entry
		if e.ParentID == "" {
			continue
		}
		b.WriteString("\n  ")
		b.WriteString(sanitize(e.ParentID))
		b.WriteString(" -->|")
		b.WriteString(e.Origin)
		b.WriteString("| ")
		b.WriteString(sanitize(e.ID))
	}
	return b.String()
}

// sanitize turns a dataset id into a Mermaid-safe node identifier. The human id
// is preserved in the node's bracket label; this only affects the identifier.
func sanitize(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
