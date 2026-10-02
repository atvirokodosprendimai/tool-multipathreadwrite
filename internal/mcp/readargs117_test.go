package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// ADR-117 T1. mrw_read routed a cap, a stat and a list of specs to the CLI. It
// takes them now, each as the CLI's flag does, and each licenses exactly what
// it served: a capped read the lines it showed, a stat nothing.
func TestMrwReadTakesMaxLinesStatAndFilesFrom(t *testing.T) {
	const body = "one\ntwo\nthree\n"
	writeWith := func(t *testing.T, root string, acks []any, plan string) map[string]any {
		t.Helper()
		return call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks})
	}
	t.Run("max_lines serves N lines and licenses only those", func(t *testing.T) {
		root, path := checkout(t, "a.txt", body)
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_lines": 1})
		txt := served0(t, res)
		if res["isError"] == true || !strings.Contains(txt, "1| one") || strings.Contains(txt, "2| two") || !strings.Contains(strings.ToLower(txt), "withheld") {
			t.Fatalf("max_lines 1 served: %q", txt)
		}
		// The cut is named as this caller sent it, not as the CLI's flag.
		if !strings.Contains(txt, "max_lines reached") || strings.Contains(txt, "--max-lines") {
			t.Errorf("the withheld line does not name max_lines: %q", txt)
		}
		acks := checkpointsIn(txt)
		if r := writeWith(t, root, acks, "@@ a.txt 2 replace\nTWO\n"); r["isError"] != true {
			t.Errorf("a write to a WITHHELD line applied: %v", r)
		}
		if r := writeWith(t, root, acks, "@@ a.txt 1 replace\nONE\n"); r["isError"] == true {
			t.Errorf("a write to the served line was refused: %v", firstText(t, r))
		}
	})
	t.Run("max_lines 0 serves headers only", func(t *testing.T) {
		root, path := checkout(t, "a.txt", body)
		txt := served0(t, call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_lines": 0}))
		if !strings.Contains(txt, "==> a.txt") || strings.Contains(txt, "1| one") || !strings.Contains(strings.ToLower(txt), "withheld") {
			t.Errorf("max_lines 0 served: %q", txt)
		}
	})
	t.Run("a negative max_lines is refused", func(t *testing.T) {
		root, path := checkout(t, "a.txt", body)
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_lines": -1})
		if res["isError"] != true || !strings.Contains(firstText(t, res), "negative") {
			t.Errorf("max_lines -1: %v", res)
		}
	})
	t.Run("stat serves no line and licenses nothing", func(t *testing.T) {
		root, path := checkout(t, "a.txt", body)
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "stat": true})
		txt := served0(t, res)
		if res["isError"] == true || !strings.Contains(txt, "==> a.txt") || strings.Contains(txt, "| one") {
			t.Fatalf("stat served: %q", txt)
		}
		if acks := checkpointsIn(txt); len(acks) != 0 {
			t.Errorf("a stat carries checkpoints: %v", acks)
		}
		if r := writeWith(t, root, nil, "@@ a.txt 1 replace\nONE\n"); r["isError"] != true {
			t.Errorf("a write after a stat applied: %v", r)
		}
	})
	t.Run("files_from reads a list of specs", func(t *testing.T) {
		root, _ := checkout(t, "a.txt", body)
		if err := os.WriteFile(filepath.Join(root, "list.txt"), []byte("# a note\n\na.txt:2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		txt := served0(t, call(t, root, "mrw_read", map[string]any{"files_from": "list.txt"}))
		if !strings.Contains(txt, "2| two") || strings.Contains(txt, "1| one") {
			t.Errorf("files_from served: %q", txt)
		}
	})
	t.Run("files_from is refused where it cannot be honoured", func(t *testing.T) {
		root, _ := checkout(t, "a.txt", body)
		outside := filepath.Join(t.TempDir(), "list")
		if err := os.WriteFile(outside, []byte("a.txt\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		// Creating a symlink needs a privilege a Windows desktop without
		// Developer Mode lacks (a peer's run, 2026-10-02): drop that one row
		// rather than the whole subtest, whose other refusals hold anywhere.
		linked := true
		if err := os.Symlink(outside, filepath.Join(root, "out.lnk")); err != nil {
			t.Logf("symlinks unavailable, the link row is not run: %v", err)
			linked = false
		}
		if err := os.WriteFile(filepath.Join(root, "empty.txt"), []byte("# only\n\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
			t.Fatal(err)
		}
		state := filepath.Join(root, "st")
		t.Setenv("XDG_STATE_HOME", state)
		if err := os.MkdirAll(filepath.Join(state, "mrw"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(state, "mrw", "list"), []byte("a.txt\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cases := map[string]map[string]any{
			"beside specs":    {"files_from": "empty.txt", "specs": []any{"a.txt"}},
			"beside grep":     {"files_from": "empty.txt", "grep": "one"},
			"empty":           {"files_from": ""},
			"stdin":           {"files_from": "-"},
			"outside":         {"files_from": "../list"},
			"a link out":      {"files_from": "out.lnk"},
			"the state dir":   {"files_from": "st/mrw/list"},
			"missing":         {"files_from": "nope.txt"},
			"no spec":         {"files_from": "empty.txt"},
			"a directory":     {"files_from": "d"},
			"beside ast_grep": {"files_from": "empty.txt", "ast_grep": "X"},
		}
		if !linked {
			delete(cases, "a link out")
		}
		for name, args := range cases {
			if res := call(t, root, "mrw_read", args); res["isError"] != true {
				t.Errorf("%s: files_from was not refused: %v", name, res)
			}
		}
	})
	t.Run("a list naming one file twice licenses both ranges", func(t *testing.T) {
		root, _ := checkout(t, "a.txt", "one\ntwo\nthree\nfour\n")
		if err := os.WriteFile(filepath.Join(root, "list.txt"), []byte("a.txt:1\na.txt:3-4\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		acks := checkpointsIn(served0(t, call(t, root, "mrw_read", map[string]any{"files_from": "list.txt", "max_lines": 1})))
		// Dry runs: a write that lands licenses the whole file it wrote
		// (ADR-005), which would hide what the acks alone license.
		dry := func(plan string) map[string]any {
			return call(t, root, "mrw_write", map[string]any{"plan": plan, "ack": acks, "dry_run": true})
		}
		for _, plan := range []string{"@@ a.txt 1 replace\nONE\n", "@@ a.txt 3 replace\nTHREE\n"} {
			if r := dry(plan); r["isError"] == true {
				t.Errorf("%q: a line its own spec served and its ack covers was refused: %v", plan, firstText(t, r))
			}
		}
		if r := dry("@@ a.txt 4 replace\nFOUR\n"); r["isError"] != true {
			t.Errorf("a line the cap withheld was licensed: %v", r)
		}
	})
	t.Run("a capped grep too large to serve answers with its index", func(t *testing.T) {
		root := grepTree(t, 60, 400)
		old := MaxResultChars
		MaxResultChars = 4000
		t.Cleanup(func() { MaxResultChars = old })
		res := call(t, root, "mrw_read", map[string]any{"grep": "NEEDLE", "max_lines": 5})
		if _, ok := receipt(t, res)["index"]; !ok || res["isError"] == true {
			t.Errorf("a capped grep too large did not answer with its index: %q", served0(t, res))
		}
	})
	t.Run("a stat too large names fewer specs, and a lower bound", func(t *testing.T) {
		// Enough headers to pass the reader's 4 KiB buffer, so the read
		// stops before it has rendered them all and states a lower bound.
		root := grepTree(t, 300, 1)
		var specs []any
		for i := 0; i < 300; i++ {
			specs = append(specs, fmt.Sprintf("document%05d.csv", i))
		}
		old := MaxResultChars
		MaxResultChars = 1500
		t.Cleanup(func() { MaxResultChars = old })
		res := call(t, root, "mrw_read", map[string]any{"specs": specs, "stat": true})
		txt := firstText(t, res)
		if res["isError"] != true || !strings.Contains(txt, "send fewer specs") || strings.Contains(txt, "max_lines") || !strings.Contains(txt, "at least") {
			t.Errorf("a stat too large: %q", txt)
		}
	})
	t.Run("a capped read too large for one answer is refused, not paged past the cap", func(t *testing.T) {
		root, path := bigCheckout(t, 12000)
		old := MaxResultChars
		MaxResultChars = 4000
		t.Cleanup(func() { MaxResultChars = old })
		res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_lines": 9000})
		if res["isError"] != true || strings.Contains(served0(t, res), "next_read") {
			t.Errorf("a capped read paged past its cap: %q", served0(t, res))
		}
		if strings.Contains(firstText(t, res), "max_lines") == false {
			t.Errorf("the refusal does not name max_lines: %q", firstText(t, res))
		}
		// The size it names is the one that went over the limit, never less.
		if m := regexp.MustCompile(`(\d+) characters`).FindStringSubmatch(firstText(t, res)); m == nil {
			t.Errorf("the refusal names no size: %q", firstText(t, res))
		} else if n, _ := strconv.Atoi(m[1]); n <= 4000 {
			t.Errorf("the refusal says %d characters against a limit of 4000: %q", n, firstText(t, res))
		}
	})
	// Two ways a capped answer's encoding overflows while its text fits: the
	// checkpoint markers push it over (plain lines), or JSON escaping does
	// before any marker is added — "<" costs six characters encoded. Each
	// refusal names the size that went over, never the smaller raw one.
	// 300 such lines are about 4,800 characters raw and over 9,000 encoded.
	for name, line := range map[string]string{"by its markers": "line 1 ", "by its escaping": "<<<< "} {
		t.Run("a capped read whose answer, encoded, is too large "+name+" names that size", func(t *testing.T) {
			var b strings.Builder
			for i := 1; i <= 400; i++ {
				b.WriteString(line + strconv.Itoa(i) + "\n")
			}
			root, path := checkout(t, "f.txt", b.String())
			old := MaxResultChars
			MaxResultChars = 6000
			t.Cleanup(func() { MaxResultChars = old })
			res := call(t, root, "mrw_read", map[string]any{"specs": []any{path}, "max_lines": 300})
			m := regexp.MustCompile(`(\d+) characters`).FindStringSubmatch(firstText(t, res))
			if res["isError"] != true || m == nil {
				t.Fatalf("want a refusal naming a size: %q", firstText(t, res))
			}
			if n, _ := strconv.Atoi(m[1]); n <= 6000 {
				t.Errorf("the refusal says %d characters against a limit of 6000", n)
			}
		})
	}
}
