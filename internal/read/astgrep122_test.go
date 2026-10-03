package read

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// tree122 is a checkout with an ignored file, an ignored directory, a nested
// checkout with its own rule, a binary, a hidden file and a .git object, each
// holding one hit the fake ast-grep reports.
func tree122(t *testing.T) (root string, hits string) {
	t.Helper()
	root = t.TempDir()
	files := map[string]string{
		".gitignore":          "*.log\ngen/\n",
		"b.go":                "package b\n",
		".hidden.go":          "package h\n",
		"x.log":               "package x\n",
		"gen/a.go":            "package gen\n",
		"nested/.gitignore":   "inner.go\n",
		"nested/inner.go":     "package n\n",
		"nested/outer.go":     "package n\n",
		"nested/keep.log":     "package n\n",
		"bin.go":              "package\x00bin\n",
		".git/objects/obj.go": "package o\n",
	}
	for p, body := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "nested", ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	var names []string
	for p := range files {
		if !strings.HasSuffix(p, ".gitignore") {
			names = append(names, p)
		}
	}
	sort.Strings(names)
	return root, hit096(names...)
}

func servedPaths(specs []Spec) []string {
	var out []string
	for _, s := range specs {
		out = append(out, s.Path)
	}
	sort.Strings(out)
	return out
}

// ADR-122. ast-grep walked with its own rules and served what --grep skips;
// every hit now passes the walk's judgement and what is dropped is counted.
func TestAnAstGrepHitTheWalkWouldSkipIsDroppedAndCounted(t *testing.T) {
	root, hits := tree122(t)
	installRecordingAstGrep(t, hits)
	var sk WalkSkipped
	specs, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := servedPaths(specs), []string{".hidden.go", "b.go", "nested/keep.log", "nested/outer.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("served %v, want %v", got, want)
	}
	if want := (WalkSkipped{Ignored: 2, IgnoredDirs: 1, Binary: 1}); sk != want {
		t.Fatalf("skipped %+v, want %+v (x.log and nested/inner.go, gen/, bin.go)", sk, want)
	}
}

func TestAstGrepUnderNoIgnoreServesEveryHit(t *testing.T) {
	root, hits := tree122(t)
	installRecordingAstGrep(t, hits)
	var sk WalkSkipped
	specs, _, err := AstGrep(root, nil, "package $A", nil, AstGrepOptions{NoIgnore: true, Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"x.log", "gen/a.go", "nested/inner.go", "bin.go"} {
		found := false
		for _, p := range servedPaths(specs) {
			found = found || p == want
		}
		if !found {
			t.Errorf("under NoIgnore %s was not served: %v", want, servedPaths(specs))
		}
	}
	if sk != (WalkSkipped{}) {
		t.Errorf("skipped %+v under NoIgnore", sk)
	}
}

func TestAstGrepIsAskedToIgnoreOnlyWhatMrwIgnores(t *testing.T) {
	for _, tc := range []struct {
		noIgnore bool
		want     []string
	}{
		{false, []string{"hidden", "dot", "global"}},
		{true, []string{"hidden", "dot", "global", "exclude", "parent", "vcs"}},
	} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		argv := installRecordingAstGrep(t, "[]")
		if _, _, err := AstGrep(root, nil, "x", nil, AstGrepOptions{NoIgnore: tc.noIgnore}); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(argv)
		if err != nil {
			t.Fatal(err)
		}
		all := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		var off []string
		for i, a := range all {
			if a == "--json" {
				break
			}
			if a == "--no-ignore" && i+1 < len(all) {
				off = append(off, all[i+1])
			}
		}
		if !reflect.DeepEqual(off, tc.want) {
			t.Errorf("noIgnore %v: --no-ignore %v before --json, want %v", tc.noIgnore, off, tc.want)
		}
	}
}

func TestANamedFileTheRulesIgnoreIsServedByAstGrep(t *testing.T) {
	root, _ := tree122(t)
	installRecordingAstGrep(t, hit096("x.log"))
	specs, _, err := AstGrep(root, []string{"x.log"}, "package $A", nil, AstGrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := servedPaths(specs); !reflect.DeepEqual(got, []string{"x.log"}) {
		t.Fatalf("served %v, want the named x.log", got)
	}
}

// The counts agree with --grep's where mrw sees the hit (the review of #329):
// an excluded hit is excluded, not counted as ignored, and an ignored
// directory a named start is below is entered, not counted.
func TestAstGrepCountsExcludeAndNamedStartsAsTheWalkDoes(t *testing.T) {
	root, hits := tree122(t)
	installRecordingAstGrep(t, hits)
	var sk WalkSkipped
	if _, _, err := AstGrep(root, nil, "package $A", []string{"x.log", "gen"}, AstGrepOptions{Skipped: &sk}); err != nil {
		t.Fatal(err)
	}
	if sk.IgnoredDirs != 0 || sk.Ignored != 1 {
		t.Fatalf("skipped %+v: an excluded hit was counted as ignored (want only nested/inner.go)", sk)
	}
	if err := os.MkdirAll(filepath.Join(root, "gen", "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "gen", "keep", "k.go"), []byte("package k\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The only hit is under gen/ from the root walk; the named gen/keep serves
	// nothing, so only the start can say gen/ was entered.
	installRecordingAstGrep(t, hit096("gen/a.go"))
	sk = WalkSkipped{}
	specs, _, err := AstGrep(root, []string{".", "gen/keep"}, "package $A", nil, AstGrepOptions{Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if got := servedPaths(specs); len(got) != 0 || sk.IgnoredDirs != 0 {
		t.Fatalf("served %v, skipped %+v: a named start below an ignored directory is entered, not counted", got, sk)
	}
}
