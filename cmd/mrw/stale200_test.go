package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// ADR-200 T1, as ADR-203 changed it. `mrw stats` prints what a stale-number
// guard counted; the count is taken only from a plan the caller typed (a plan
// mrw computed from the file as it is names the lines it saw), and since
// ADR-203 an unshown candidate is refused, so numbered_would_refuse grows only
// by landings an older mrw made: the test seeds one.
func TestStatsPrintsTheStaleNumberPricing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"f.txt": "a\nb\nc\nd\ne\n", "g.go": "package g\n\nfunc g() { foo(1, 2) }\n"})
	type numbering struct {
		Landed      int `json:"landed_writes"`
		Candidates  int `json:"numbered_candidates"`
		WouldRefuse int `json:"numbered_would_refuse"`
	}
	stats := func(t *testing.T) numbering {
		t.Helper()
		js, code := runIn(t, root, "stats", "--json")
		var s struct {
			Numbering *numbering `json:"numbering"`
		}
		if err := json.Unmarshal([]byte(js), &s); code != 0 || err != nil || s.Numbering == nil {
			t.Fatalf("stats --json: exit %d, %v, numbering %v:\n%s", code, err, s.Numbering, js)
		}
		return *s.Numbering
	}
	write := func(t *testing.T, args ...string) {
		t.Helper()
		if out, code := writeIn(t, root, append([]string{"--no-check"}, args...)...); code != 0 {
			t.Fatalf("write %v: exit %d:\n%s", args, code, out)
		}
	}
	native := func(t *testing.T, doc string) { t.Helper(); write(t, planFile(t, doc)) }
	read := func(t *testing.T, specs ...string) {
		t.Helper()
		if out, code := runIn(t, root, append([]string{"read"}, specs...)...); code != 0 {
			t.Fatalf("read %v: exit %d:\n%s", specs, code, out)
		}
	}

	if z := stats(t); z != (numbering{}) {
		t.Fatalf("a fresh root: %+v, want zeros, the key present", z)
	}
	// One landing an older mrw counted as would-refuse, seeded, then three landings
	// of this one: two without a candidate, one with a candidate whose line was shown.
	authoring.RecordNumbering(root, 1, 1)
	if z := stats(t); z != (numbering{Landed: 1, Candidates: 1, WouldRefuse: 1}) {
		t.Fatalf("a seeded root: %+v, want the one landing, its key present", z)
	}
	read(t, "f.txt")
	native(t, "@@ f.txt 1 insert-after\nNEW\n")               // the file was read whole: no candidate
	native(t, "@@ f.txt 5 replace anchor=\"d\"\nD-CHANGED\n") // anchor=: no candidate
	read(t, "f.txt:3")
	native(t, "@@ f.txt 3 replace\nC-NEW\n") // numbered, line 3 shown since the write
	if got, want := stats(t), (numbering{Landed: 4, Candidates: 2, WouldRefuse: 1}); got != want {
		t.Fatalf("after three native writes: %+v, want %+v", got, want)
	}
	txt, code := runIn(t, root, "stats")
	if code != 0 {
		t.Fatalf("stats: exit %d:\n%s", code, txt)
	}
	for _, must := range []string{
		"stale-number pricing:", "4 landed writes", "2 with a numbered hunk", "1 where a line it addresses was not shown",
		"25.0%", "counts landings made before a number alone was refused (ADR-203) or by an older mrw",
	} {
		if !strings.Contains(txt, must) {
			t.Errorf("stats does not say %q:\n%s", must, txt)
		}
	}
	if strings.Contains(txt, "2026-12-18") {
		t.Errorf("stats still names the date ADR-203 set aside:\n%s", txt)
	}
	if i, j := strings.Index(txt, "strict-balance pricing:"), strings.Index(txt, "stale-number pricing:"); i < 0 || j < i {
		t.Errorf("the block does not follow the strict-balance one (%d, %d)", i, j)
	}

	// A computed plan adds a landing and no candidate, on a file that is
	// written-shaped, which is where a typed plan of the same hunk is counted.
	owned := func(t *testing.T) { read(t, "f.txt"); native(t, "@@ f.txt $ insert-after\nz\n") } // a write that is no candidate, leaving the record written whole
	computed := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"--sub", func(t *testing.T) { write(t, "--grep", "^e$", "--sub", "E", "f.txt") }},
		{"--format=apply_patch", func(t *testing.T) {
			write(t, "--format=apply_patch", planFile(t, "*** Begin Patch\n*** Update File: f.txt\n@@\n-NEW\n+new\n*** End Patch\n"))
		}},
		{"--format=search_replace", func(t *testing.T) {
			write(t, "--format=search_replace", planFile(t, "f.txt\n<<<<<<< SEARCH\nD-CHANGED\n=======\nd-changed\n>>>>>>> REPLACE\n"))
		}},
	}
	for _, c := range computed {
		t.Run(c.name, func(t *testing.T) {
			owned(t)
			before := stats(t)
			c.run(t)
			after := stats(t)
			if after.Landed != before.Landed+1 || after.Candidates != before.Candidates || after.WouldRefuse != before.WouldRefuse {
				t.Errorf("%+v then %+v, want one landing and no candidate", before, after)
			}
		})
	}
	t.Run("--rewrite", func(t *testing.T) {
		if _, err := exec.LookPath("ast-grep"); err != nil {
			t.Skip("ast-grep is not on PATH")
		}
		read(t, "g.go")
		native(t, "@@ g.go 1 insert-after\n// top\n")
		before := stats(t)
		write(t, "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "g.go")
		after := stats(t)
		if after.Landed != before.Landed+1 || after.Candidates != before.Candidates {
			t.Errorf("%+v then %+v, want one landing and no candidate", before, after)
		}
	})
	t.Run("a native plan given as a file is counted when its line was shown", func(t *testing.T) {
		owned(t)
		read(t, "f.txt:4")
		before := stats(t)
		native(t, "@@ f.txt 4 replace\nD2\n")
		after := stats(t)
		if after.Landed != before.Landed+1 || after.Candidates != before.Candidates+1 || after.WouldRefuse != before.WouldRefuse {
			t.Errorf("%+v then %+v, want a landing and a candidate, and no would-refuse", before, after)
		}
	})
	t.Run("a refused number counts nothing, and an anchored landing is no candidate", func(t *testing.T) {
		owned(t)
		before := stats(t)
		if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.txt 4 replace\nD3\n")); code != 1 || !strings.Contains(out, "has no recorded showing in this version") {
			t.Fatalf("the unshown number: exit %d, want 1 and the reason:\n%s", code, out)
		}
		if after := stats(t); after != before {
			t.Errorf("%+v then %+v, want no change after a refusal", before, after)
		}
		native(t, "@@ f.txt 4 replace anchor=\"D2\"\nD4\n")
		if after := stats(t); after.Landed != before.Landed+1 || after.Candidates != before.Candidates || after.WouldRefuse != before.WouldRefuse {
			t.Errorf("%+v then %+v, want one landing and no candidate", before, after)
		}
	})
	t.Run("a dry run counts nothing", func(t *testing.T) {
		read(t, "f.txt:4")
		before := stats(t)
		if out, code := writeIn(t, root, "--dry-run", planFile(t, "@@ f.txt 4 replace\nD5\n")); code != 0 {
			t.Fatalf("dry run: exit %d:\n%s", code, out)
		}
		if after := stats(t); after != before {
			t.Errorf("%+v then %+v, want no change", before, after)
		}
	})
	if _, err := os.Stat(filepath.Join(root, "f.txt")); err != nil {
		t.Fatal(err)
	}
}
