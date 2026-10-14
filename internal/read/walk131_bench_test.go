package read

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// BenchmarkWalkDeep131 walks 3,000 files, 300 in each of ten directories ten
// levels deep, matching none of them (ADR-131). On Windows a peer measured it at
// 9.39 s a walk before ADR-131, rooted.Resolve 85% of the CPU.
func BenchmarkWalkDeep131(b *testing.B) {
	root := b.TempDir()
	for br := 0; br < 10; br++ {
		d := root
		for lvl := 0; lvl < 10; lvl++ {
			d = filepath.Join(d, fmt.Sprintf("l%d_%d", lvl, br))
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			b.Fatal(err)
		}
		for f := 0; f < 300; f++ {
			if err := os.WriteFile(filepath.Join(d, fmt.Sprintf("f%d.txt", f)), []byte("line one\nnot here\n"), 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	re := regexp.MustCompile("needle")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Walk(root, nil, WalkOptions{Pattern: re}); err != nil {
			b.Fatal(err)
		}
	}
}
