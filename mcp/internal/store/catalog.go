package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathanaday/consensus/mcp/internal/dataset"
)

const catalogFile = "catalog.json"

// Catalog is the in-memory view of catalog.json for one store directory.
type Catalog struct {
	dir     string
	entries map[string]dataset.Entry
}

// Slug derives a filesystem- and reference-friendly id base from a filename.
func Slug(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		return "dataset"
	}
	return s
}

// LoadCatalog reads catalog.json from dir, treating a missing file as empty.
func LoadCatalog(dir string) (*Catalog, error) {
	c := &Catalog{dir: dir, entries: map[string]dataset.Entry{}}
	b, err := os.ReadFile(filepath.Join(dir, catalogFile))
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	if err := json.Unmarshal(b, &c.entries); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return c, nil
}

// Has reports whether id is already present.
func (c *Catalog) Has(id string) bool {
	_, ok := c.entries[id]
	return ok
}

// AllocateID returns base if free, otherwise base-2, base-3, ... Uncommitted
// until a matching Put, so call Put before the next AllocateID for the same base.
func (c *Catalog) AllocateID(base string) string {
	if base == "" {
		base = "dataset"
	}
	if !c.Has(base) {
		return base
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s-%d", base, i)
		if !c.Has(cand) {
			return cand
		}
	}
}

// Put adds e and rewrites catalog.json.
func (c *Catalog) Put(e dataset.Entry) error {
	c.entries[e.ID] = e
	b, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog: %w", err)
	}
	if err := os.WriteFile(filepath.Join(c.dir, catalogFile), b, 0o644); err != nil {
		return fmt.Errorf("write catalog: %w", err)
	}
	return nil
}
