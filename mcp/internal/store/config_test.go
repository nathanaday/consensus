package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesEnvAndCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	t.Setenv("CONSENSUS_STORE_DIR", dir)

	cfg, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Dir != dir {
		t.Errorf("Dir = %q, want %q", cfg.Dir, dir)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("store dir not created: %v", err)
	}
}
