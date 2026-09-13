package apply

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Parser ops (plan.go `Op =` constants): replace, insert-after, insert-before,
// delete, create, unlink, rename — 7. The ADR-054 randomised test covers the
// five line-range ops (delimiter nets). This file covers the two path-level
// ops ADR-057 added. Pool vs parser: `rg 'Op [A-Z][a-zA-Z]+ +Op =' internal/plan/plan.go`.

func exists(t *testing.T, root, name string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func TestUnlinkThenRenameOntoFreedDest(t *testing.T) {
	root := t.TempDir()
	write(t, root, "old.txt", "payload\n")
	write(t, root, "dest.txt", "stale\n")
	res, err := Apply(root, []Input{
		{Path: "dest.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "old.txt", Op: "rename", Body: []string{"dest.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied {
		t.Fatalf("unlink dest + rename onto it refused: %+v", res.Hunks)
	}
	if exists(t, root, "old.txt") {
		t.Fatal("source still exists")
	}
	if got := read(t, root, "dest.txt"); got != "payload\n" {
		t.Fatalf("dest = %q, want payload", got)
	}
}

func TestPathOpMixedWithLineEditOnSamePathIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "stay\n")
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || res.Failed == 0 {
		t.Fatalf("mix applied: %+v", res)
	}
	if read(t, root, "a.txt") != "stay\n" {
		t.Fatal("mix wrote the tree")
	}
}

// TestRandomisedPathOpsFollowTheDecision is ADR-057 arm 2: random scenarios
// from the Decision, independent tree oracle, replay via MRW_SEED.
func TestRandomisedPathOpsFollowTheDecision(t *testing.T) {
	seeds := []int64{54, 1, 7, 13, 99}
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seeds = []int64{v}
	}
	const iterations = 80
	for _, seed := range seeds {
		r := rand.New(rand.NewSource(seed))
		for i := 0; i < iterations; i++ {
			randomPathOpCase(t, r, seed, i)
		}
	}
}

func randomPathOpCase(t *testing.T, r *rand.Rand, seed int64, i int) {
	t.Helper()
	root := t.TempDir()
	a, b := "a.txt", "b.txt"
	bodyA, bodyB := "aaa\n", "bbb\n"
	write(t, root, a, bodyA)
	write(t, root, b, bodyB)
	kind := r.Intn(7)
	label := fmt.Sprintf("seed=%d iter=%d kind=%d", seed, i, kind)

	switch kind {
	case 0: // unlink one path; the other stays
		res, err := Apply(root, []Input{{Path: a, Op: "unlink", Lines: -1}}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if !res.Applied {
			t.Fatalf("%s: unlink refused: %+v", label, res.Hunks)
		}
		if exists(t, root, a) {
			t.Fatalf("%s: unlinked path still exists", label)
		}
		if read(t, root, b) != bodyB {
			t.Fatalf("%s: sibling mutated", label)
		}
		if res.Hunks[0].Balance != "" {
			t.Fatalf("%s: unlink carried Balance %q", label, res.Hunks[0].Balance)
		}
	case 1: // rename onto a fresh dest, maybe nested
		dest := "d.txt"
		if r.Intn(2) == 0 {
			dest = "sub/d.txt"
		}
		res, err := Apply(root, []Input{{Path: a, Op: "rename", Body: []string{dest}, Lines: -1}}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if !res.Applied {
			t.Fatalf("%s: rename refused: %+v", label, res.Hunks)
		}
		if exists(t, root, a) {
			t.Fatalf("%s: source still exists", label)
		}
		if read(t, root, dest) != bodyA {
			t.Fatalf("%s: dest content", label)
		}
		if res.Hunks[0].Balance != "" {
			t.Fatalf("%s: rename carried Balance %q", label, res.Hunks[0].Balance)
		}
	case 2: // rename onto an existing dest, dest not unlinked → refuse, tree unchanged
		res, err := Apply(root, []Input{{Path: a, Op: "rename", Body: []string{b}, Lines: -1}}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if res.Applied {
			t.Fatalf("%s: rename onto existing dest applied", label)
		}
		if read(t, root, a) != bodyA || read(t, root, b) != bodyB {
			t.Fatalf("%s: refused rename wrote", label)
		}
	case 3: // unlink + unread sibling → restore
		res, err := Apply(root, []Input{
			{Path: a, Op: "unlink", Lines: -1, Index: 0},
			{Path: b, Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 1},
		}, Options{Seen: map[string]Seen{a: {SHA: shaOfFile(t, root, a)}}})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if res.Applied {
			t.Fatalf("%s: sibling fail applied", label)
		}
		if read(t, root, a) != bodyA || read(t, root, b) != bodyB {
			t.Fatalf("%s: sibling fail wrote", label)
		}
	case 4: // mix unlink + replace on the same path → refuse
		res, err := Apply(root, []Input{
			{Path: a, Op: "unlink", Lines: -1, Index: 0},
			{Path: a, Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 1},
		}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if res.Applied {
			t.Fatalf("%s: mix applied", label)
		}
		if read(t, root, a) != bodyA {
			t.Fatalf("%s: mix wrote", label)
		}
	case 5: // unlink dest then rename source onto it
		res, err := Apply(root, []Input{
			{Path: b, Op: "unlink", Lines: -1, Index: 0},
			{Path: a, Op: "rename", Body: []string{b}, Lines: -1, Index: 1},
		}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if !res.Applied {
			t.Fatalf("%s: unlink-then-rename refused: %+v", label, res.Hunks)
		}
		if exists(t, root, a) {
			t.Fatalf("%s: source still exists", label)
		}
		if read(t, root, b) != bodyA {
			t.Fatalf("%s: dest after swap-in", label)
		}
	default: // two unlinks, different paths
		res, err := Apply(root, []Input{
			{Path: a, Op: "unlink", Lines: -1, Index: 0},
			{Path: b, Op: "unlink", Lines: -1, Index: 1},
		}, Options{})
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if !res.Applied {
			t.Fatalf("%s: two unlinks refused: %+v", label, res.Hunks)
		}
		if exists(t, root, a) || exists(t, root, b) {
			t.Fatalf("%s: a path survived two unlinks", label)
		}
	}
}

func FuzzUnlinkRemovesOrRefuses(f *testing.F) {
	f.Add("aaa\n")
	f.Add("")
	f.Add("a\nb\nc\n")
	f.Fuzz(func(t *testing.T, body string) {
		root := t.TempDir()
		write(t, root, "f.txt", body)
		res, err := Apply(root, []Input{{Path: "f.txt", Op: "unlink", Lines: -1}}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		gone := !exists(t, root, "f.txt")
		if res.Applied != gone {
			t.Fatalf("applied=%v gone=%v hunks=%+v", res.Applied, gone, res.Hunks)
		}
		if !res.Applied && gone {
			t.Fatal("a refused unlink removed the path")
		}
	})
}
