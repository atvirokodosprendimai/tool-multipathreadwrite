package main

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// BenchmarkR1 is the pre-registration's R1 in process: one 40-line range at the
// middle of a 10,000-line file, for its allocations (-benchmem).
func BenchmarkR1(b *testing.B) {
	root := b.TempDir()
	if err := writeLines(filepath.Join(root, "f.txt"), lines(10_000, rand.New(rand.NewPCG(seed, seed)))); err != nil {
		b.Fatal(err)
	}
	spec, err := read.ParseSpec("f.txt:4981-5020")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, problems := read.Run(io.Discard, root, []read.Spec{spec}, read.Options{Numbers: true}); problems != 0 {
			b.Fatalf("%d problem(s)", problems)
		}
	}
}

// BenchmarkW1 is W1 at 50 hunks over 10 files in process, with no ledger: the
// engine's apply, each file put back before every iteration.
func BenchmarkW1(b *testing.B) {
	root := b.TempDir()
	rng := rand.New(rand.NewPCG(seed, seed))
	orig := map[string][]string{}
	var in []apply.Input
	for f := 0; f < 10; f++ {
		name := fmt.Sprintf("f%03d.txt", f)
		orig[name] = lines(1000, rng)
		for h := 0; h < 5; h++ {
			line := 1 + h*200
			in = append(in, apply.Input{Path: name, Start: line, End: line, Op: "replace", Body: []string{fmt.Sprintf("edited %d", h)}, Lines: -1})
		}
	}
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for name, ls := range orig {
			if err := writeLines(filepath.Join(root, name), ls); err != nil {
				b.Fatal(err)
			}
		}
		b.StartTimer()
		res, err := apply.Apply(root, in, apply.Options{})
		if err != nil {
			b.Fatal(err)
		}
		if res.Failed != 0 || !res.Applied {
			b.Fatalf("%d hunk(s) failed, applied %v", res.Failed, res.Applied)
		}
	}
}

// BenchmarkR3Repo is R3 "repo" in process, for its profile: the walk --grep
// makes and the --count read over it, on the tree NATIVEBENCH_TREE names (this
// repository at the tagged tree). It is skipped without one.
func BenchmarkR3Repo(b *testing.B) {
	root := os.Getenv("NATIVEBENCH_TREE")
	if root == "" {
		b.Skip("NATIVEBENCH_TREE names no tree")
	}
	re := regexp.MustCompile("func Run")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		specs, _, err := read.Walk(root, nil, read.WalkOptions{Pattern: re})
		if err != nil {
			b.Fatal(err)
		}
		read.Run(io.Discard, root, specs, read.Options{Count: true, Counts: map[string]int{}, Numbers: true, NextLine: true})
	}
}
