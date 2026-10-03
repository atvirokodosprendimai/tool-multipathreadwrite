package rooted

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkResolveADeepFile measures one Resolve of a file six directories
// below the root, with mrw's state directory present elsewhere (ADR-123).
func BenchmarkResolveADeepFile(b *testing.B) {
	root := b.TempDir()
	b.Setenv("XDG_STATE_HOME", filepath.Join(b.TempDir(), "state"))
	if err := os.MkdirAll(filepath.Join(os.Getenv("XDG_STATE_HOME"), "mrw"), 0o755); err != nil {
		b.Fatal(err)
	}
	rel := filepath.Join("a", "b", "c", "d", "e", "f", "x.go")
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte("package x\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Resolve(root, rel); err != nil {
			b.Fatal(err)
		}
	}
}
