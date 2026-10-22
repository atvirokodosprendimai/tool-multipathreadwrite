package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
)

// ADR-167 T2. The source of a from= copy has its own gate: the ledger must
// show the caller was served the lines, in the version on disk, and neither
// Force nor a missing observation waives it.
const srcText167 = "secret-1\nsecret-2\nsecret-3\nsecret-4\nsecret-5\nsecret-6\n"

func copyInput167(path, op string, at int, src string, from, to int) Input {
	return Input{Path: path, Op: op, Start: at, End: at, Lines: -1, From: src, FromStart: from, FromEnd: to}
}

func ledgerOf167(t *testing.T, root string, spans map[string][][2]int) map[string]Seen {
	t.Helper()
	l := map[string]Seen{}
	for name, sp := range spans {
		l[name] = Seen{SHA: shaOfFile(t, root, name), Spans: sp}
	}
	return l
}

func TestAFromCopyCopiesOnlyLinesTheCallerWasServed(t *testing.T) {
	newTree := func(t *testing.T) string {
		root := t.TempDir()
		write(t, root, "a.txt", srcText167)
		write(t, root, "b.txt", "x\ny\n")
		return root
	}
	const want = "x\nsecret-3\nsecret-4\nsecret-5\ny\n"
	copyIn := func() []Input { return []Input{copyInput167("b.txt", "insert-after", 1, "a.txt", 3, 5)} }

	t.Run("served lines land byte for byte and the receipt names them", func(t *testing.T) {
		root := newTree(t)
		res, err := Apply(root, copyIn(), Options{Seen: ledgerOf167(t, root, map[string][][2]int{"a.txt": {{3, 5}}, "b.txt": nil})})
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
		if got := read(t, root, "b.txt"); got != want {
			t.Errorf("b.txt = %q, want %q", got, want)
		}
		h := res.Hunks[0]
		if h.BodyFrom != "a.txt:3-5" || h.CopiedFirst != "secret-3" || h.CopiedLast != "secret-5" {
			t.Errorf("receipt body_from=%q first=%q last=%q", h.BodyFrom, h.CopiedFirst, h.CopiedLast)
		}
		b, _ := json.Marshal(h)
		if !strings.Contains(string(b), `"body_from":"a.txt:3-5"`) || !strings.Contains(string(b), `"copied_first":"secret-3"`) {
			t.Errorf("--json hunk lacks the keys: %s", b)
		}
	})
	// ADR-091: the receipt spells the source's path with "/" as it does the
	// target's, in body_from and in a reason that names it.
	sl := Slashed(Result{Hunks: []HunkResult{{
		Path: `d\b.txt`, BodyFrom: `d\a.txt:3-5`, srcPath: `d\a.txt`, Reason: `d\a.txt has not been read`,
	}}}, '\\').Hunks[0]
	if sl.BodyFrom != "d/a.txt:3-5" || !strings.HasPrefix(sl.Reason, "d/a.txt has") {
		t.Errorf("Slashed left the source's path as written: %+v", sl)
	}

	t.Run("an inline-body hunk carries none of the three keys", func(t *testing.T) {
		root := newTree(t)
		res, err := Apply(root, []Input{{Path: "b.txt", Op: "insert-after", Start: 1, End: 1, Lines: -1, Body: []string{"z"}}},
			Options{Seen: ledgerOf167(t, root, map[string][][2]int{"b.txt": nil})})
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		b, _ := json.Marshal(res.Hunks[0])
		for _, k := range []string{"body_from", "copied_first", "copied_last"} {
			if strings.Contains(string(b), k) {
				t.Errorf("an inline hunk carries %s: %s", k, b)
			}
		}
	})

	// Every way the licence can be missing, with and without Force: refused,
	// nothing written, the sibling skips, and no copied line in the answer.
	for _, c := range []struct {
		name   string
		ledger func(t *testing.T, root string) map[string]Seen
	}{
		{"line 5 unserved", func(t *testing.T, root string) map[string]Seen {
			return ledgerOf167(t, root, map[string][][2]int{"a.txt": {{3, 4}}, "b.txt": nil})
		}},
		{"a line cut by --max-cols is in no span", func(t *testing.T, root string) map[string]Seen {
			return ledgerOf167(t, root, map[string][][2]int{"a.txt": {{1, 2}, {4, 6}}, "b.txt": nil})
		}},
		{"no observation of the source", func(t *testing.T, root string) map[string]Seen {
			return ledgerOf167(t, root, map[string][][2]int{"b.txt": nil})
		}},
		{"the source moved since it was read", func(t *testing.T, root string) map[string]Seen {
			l := ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "b.txt": nil})
			l["a.txt"] = Seen{SHA: "0000000000000000"}
			return l
		}},
		{"stat-only read licenses nothing", func(t *testing.T, root string) map[string]Seen {
			return ledgerOf167(t, root, map[string][][2]int{"a.txt": {}, "b.txt": nil})
		}},
	} {
		for _, force := range []bool{false, true} {
			root := newTree(t)
			res, err := Apply(root, append(copyIn(), Input{Path: "c.txt", Op: "create", Lines: -1, Body: []string{"c"}}),
				Options{Seen: c.ledger(t, root), Force: force})
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			h := res.Hunks[0]
			if res.Failed != 1 || h.Status != StatusFailed || h.Kind != refusal.SourceNotRead {
				t.Errorf("%s force=%v: failed=%d status=%s kind=%q reason=%q, want a SourceNotRead refusal", c.name, force, res.Failed, h.Status, h.Kind, h.Reason)
			}
			if res.Hunks[1].Status != StatusSkipped {
				t.Errorf("%s force=%v: sibling is %s, want skipped", c.name, force, res.Hunks[1].Status)
			}
			if got := read(t, root, "b.txt"); got != "x\ny\n" {
				t.Errorf("%s force=%v: a refused copy changed b.txt: %q", c.name, force, got)
			}
			if _, err := os.Stat(filepath.Join(root, "c.txt")); err == nil {
				t.Errorf("%s force=%v: a refused plan created c.txt", c.name, force)
			}
			if strings.Contains(h.Reason, "--force") || strings.Contains(h.Reason, "secret-") || h.CopiedFirst != "" || h.CopiedLast != "" {
				t.Errorf("%s force=%v: the refusal advises --force or discloses a copied line: %+v", c.name, force, h)
			}
			if !strings.Contains(h.Reason, "a.txt") {
				t.Errorf("%s force=%v: the refusal does not name the source: %q", c.name, force, h.Reason)
			}
		}
	}

	t.Run("a nil ledger cannot license a copy", func(t *testing.T) {
		root := newTree(t)
		res, err := Apply(root, copyIn(), Options{})
		if err != nil || res.Failed != 1 || res.Hunks[0].Kind != refusal.SourceNotRead {
			t.Errorf("nil ledger: %v %+v", err, res.Hunks)
		}
	})

	t.Run("a file mrw just wrote, and read since, copies what it showed", func(t *testing.T) {
		// Zy, 2026-10-22: only lines SHOWN count. A written file is licensed
		// whole but shown nothing until a read serves lines of it (ADR-118).
		root := newTree(t)
		l := ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "b.txt": nil})
		a := l["a.txt"]
		a.Written, a.Shown = true, [][2]int{{3, 5}}
		l["a.txt"] = a
		res, err := Apply(root, copyIn(), Options{Seen: l})
		if err != nil || res.Failed != 0 || read(t, root, "b.txt") != want {
			t.Errorf("a written source, shown 3-5: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
	})

	t.Run("a source served under another spelling copies", func(t *testing.T) {
		root := newTree(t)
		if err := os.Symlink("a.txt", filepath.Join(root, "ln.txt")); err != nil {
			t.Skipf("no symlinks here: %v", err)
		}
		l := ledgerOf167(t, root, map[string][][2]int{"a.txt": {{3, 5}}, "b.txt": nil})
		res, err := Apply(root, []Input{copyInput167("b.txt", "insert-after", 1, "ln.txt", 3, 5)}, Options{Seen: l})
		if err != nil || res.Failed != 0 || read(t, root, "b.txt") != want {
			t.Errorf("an alias-spelled source: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
	})

	t.Run("the copy is of the original, whatever the plan does to the source", func(t *testing.T) {
		whole := func(root string) map[string]Seen {
			return ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "b.txt": nil})
		}
		// The same plan replaces the source lines.
		root := newTree(t)
		res, err := Apply(root, []Input{
			copyInput167("b.txt", "insert-after", 1, "a.txt", 3, 5),
			{Path: "a.txt", Op: "replace", Start: 3, End: 5, Lines: -1, Anchor: "secret-3", Body: []string{"REPLACED"}},
		}, Options{Seen: whole(root)})
		if err != nil || res.Failed != 0 || read(t, root, "b.txt") != want {
			t.Fatalf("copy beside a replace of its source: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
		if !strings.Contains(read(t, root, "a.txt"), "REPLACED") {
			t.Error("the replace of the source did not land")
		}
		// The plan unlinks the source.
		root = newTree(t)
		res, err = Apply(root, []Input{
			copyInput167("c.txt", "create", 0, "a.txt", 3, 5),
			{Path: "a.txt", Op: "unlink", Lines: -1},
		}, Options{Seen: whole(root)})
		if err != nil || res.Failed != 0 || read(t, root, "c.txt") != "secret-3\nsecret-4\nsecret-5\n" {
			t.Fatalf("copy beside an unlink of its source: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
		// The plan renames the source.
		root = newTree(t)
		res, err = Apply(root, []Input{
			copyInput167("c.txt", "create", 0, "a.txt", 3, 5),
			{Path: "a.txt", Op: "rename", Lines: -1, Body: []string{"d.txt"}},
		}, Options{Seen: whole(root)})
		if err != nil || res.Failed != 0 || read(t, root, "c.txt") != "secret-3\nsecret-4\nsecret-5\n" || read(t, root, "d.txt") != srcText167 {
			t.Fatalf("copy beside a rename of its source: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
	})

	t.Run("a copy and a delete of the range move the block", func(t *testing.T) {
		root := newTree(t)
		res, err := Apply(root, []Input{
			copyInput167("b.txt", "insert-after", 1, "a.txt", 3, 5),
			{Path: "a.txt", Op: "delete", Start: 3, End: 5, Lines: -1},
		}, Options{Seen: ledgerOf167(t, root, map[string][][2]int{"a.txt": {{3, 5}}, "b.txt": nil})})
		if err != nil || res.Failed != 0 {
			t.Fatalf("move: %v %+v", err, res.Hunks)
		}
		if read(t, root, "b.txt") != want || read(t, root, "a.txt") != "secret-1\nsecret-2\nsecret-6\n" {
			t.Errorf("move: a=%q b=%q", read(t, root, "a.txt"), read(t, root, "b.txt"))
		}
		if res.Hunks[1].RemovedFirst != "secret-3" || res.Hunks[0].CopiedFirst != "secret-3" {
			t.Errorf("the delete and the copy do not name the same first line: %+v %+v", res.Hunks[1], res.Hunks[0])
		}
	})

	t.Run("a create, $ and A,+N each copy", func(t *testing.T) {
		for _, c := range []struct {
			name           string
			start, end, re int
			want           string
			span           string
		}{
			{"create span", 3, 5, 0, "secret-3\nsecret-4\nsecret-5\n", "a.txt:3-5"},
			{"dollar", EOF, EOF, 0, "secret-6\n", "a.txt:6"},
			{"relative end", 2, 2, 2, "secret-2\nsecret-3\nsecret-4\n", "a.txt:2-4"},
			{"start to the end", 5, EOF, 0, "secret-5\nsecret-6\n", "a.txt:5-6"},
		} {
			root := newTree(t)
			in := Input{Path: "c.txt", Op: "create", Lines: -1, From: "a.txt", FromStart: c.start, FromEnd: c.end, FromRelEnd: c.re}
			res, err := Apply(root, []Input{in}, Options{Seen: ledgerOf167(t, root, map[string][][2]int{"a.txt": nil})})
			if err != nil || res.Failed != 0 {
				t.Errorf("%s: %v %+v", c.name, err, res.Hunks)
				continue
			}
			if got := read(t, root, "c.txt"); got != c.want {
				t.Errorf("%s: c.txt = %q, want %q", c.name, got, c.want)
			}
			if res.Hunks[0].BodyFrom != c.span {
				t.Errorf("%s: body_from = %q, want %q", c.name, res.Hunks[0].BodyFrom, c.span)
			}
		}
	})

	t.Run("a source that cannot be copied is refused", func(t *testing.T) {
		for name, mk := range map[string]func(root string) Input{
			"missing":      func(string) Input { return copyInput167("b.txt", "insert-after", 1, "nope.txt", 1, 1) },
			"past the end": func(string) Input { return copyInput167("b.txt", "insert-after", 1, "a.txt", 5, 9) },
			"ends before it starts": func(string) Input {
				return copyInput167("b.txt", "insert-after", 1, "a.txt", 4, 2)
			},
			"utf-16": func(root string) Input {
				write(t, root, "u.txt", "\xff\xfea\x00b\x00\n\x00")
				return copyInput167("b.txt", "insert-after", 1, "u.txt", 1, 1)
			},
			"a directory": func(root string) Input {
				if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
					t.Fatal(err)
				}
				return copyInput167("b.txt", "insert-after", 1, "dir", 1, 1)
			},
		} {
			root := newTree(t)
			in := mk(root)
			l := ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "b.txt": nil})
			if _, err := os.Stat(filepath.Join(root, "u.txt")); err == nil {
				l["u.txt"] = Seen{SHA: shaOfFile(t, root, "u.txt")}
			}
			res, err := Apply(root, []Input{in}, Options{Seen: l, Force: true})
			if err != nil || res.Failed != 1 {
				t.Errorf("%s: err=%v failed=%d %+v", name, err, res.Failed, res.Hunks)
				continue
			}
			if got := read(t, root, "b.txt"); got != "x\ny\n" {
				t.Errorf("%s: b.txt = %q", name, got)
			}
		}
	})

	t.Run("a source under .git may be copied", func(t *testing.T) {
		root := newTree(t)
		write(t, root, ".git/note.txt", "g1\ng2\n")
		l := ledgerOf167(t, root, map[string][][2]int{"b.txt": nil})
		l[".git/note.txt"] = Seen{SHA: shaOfFile(t, root, ".git/note.txt")}
		res, err := Apply(root, []Input{copyInput167("b.txt", "insert-after", 1, ".git/note.txt", 1, 2)}, Options{Seen: l})
		if err != nil || res.Failed != 0 || read(t, root, "b.txt") != "x\ng1\ng2\ny\n" {
			t.Errorf("a .git source: %v failed=%d %+v", err, res.Failed, res.Hunks)
		}
	})

	t.Run("a CRLF source lands with the target's endings", func(t *testing.T) {
		root := newTree(t)
		write(t, root, "crlf.txt", "c1\r\nc2\r\n")
		l := ledgerOf167(t, root, map[string][][2]int{"b.txt": nil, "crlf.txt": nil})
		res, err := Apply(root, []Input{copyInput167("b.txt", "insert-after", 1, "crlf.txt", 1, 2)}, Options{Seen: l})
		if err != nil || res.Failed != 0 || read(t, root, "b.txt") != "x\nc1\nc2\ny\n" {
			t.Errorf("a CRLF source: %v failed=%d %q", err, res.Failed, read(t, root, "b.txt"))
		}
	})

	t.Run("a source line that is not UTF-8 encodes as U+FFFD, as removed_first does", func(t *testing.T) {
		root := newTree(t)
		write(t, root, "bad.txt", "ok\n\xffbyte\n")
		l := ledgerOf167(t, root, map[string][][2]int{"b.txt": nil, "bad.txt": nil})
		res, err := Apply(root, []Input{copyInput167("b.txt", "insert-after", 1, "bad.txt", 2, 2)}, Options{Seen: l})
		if err != nil || res.Failed != 0 {
			t.Fatalf("%v %+v", err, res.Hunks)
		}
		b, _ := json.Marshal(res.Hunks[0])
		if !strings.Contains(string(b), `"copied_first":"�byte"`) {
			t.Errorf("copied_first = %s", b)
		}
		if read(t, root, "b.txt") != "x\n\xffbyte\ny\n" {
			t.Errorf("the bytes did not land as read: %q", read(t, root, "b.txt"))
		}
	})
}

// ADR-030: the engine refuses what the parser refuses, for a caller that
// builds Inputs by hand.
func TestTheEngineRefusesAMalformedFromInput(t *testing.T) {
	good := func() Input { return copyInput167("b.txt", "insert-after", 1, "a.txt", 3, 5) }
	for name, c := range map[string]struct {
		in   Input
		opts func(root string) Options
	}{
		"from on delete": {Input{Path: "b.txt", Op: "delete", Start: 1, End: 1, Lines: -1, From: "a.txt", FromStart: 1, FromEnd: 1}, nil},
		"from on unlink": {Input{Path: "b.txt", Op: "unlink", Lines: -1, From: "a.txt", FromStart: 1, FromEnd: 1}, nil},
		"from on rename": {Input{Path: "b.txt", Op: "rename", Lines: -1, Body: []string{"n.txt"}, From: "a.txt", FromStart: 1, FromEnd: 1}, nil},
		"from beside a body": {func() Input {
			in := good()
			in.Body = []string{"inline"}
			return in
		}(), nil},
		"a span that is not numeric": {copyInput167("b.txt", "insert-after", 1, "a.txt", 0, 0), nil},
		"a negative start":           {copyInput167("b.txt", "insert-after", 1, "a.txt", -5, 2), nil},
		"a nil ledger":               {good(), func(string) Options { return Options{} }},
	} {
		root := t.TempDir()
		write(t, root, "a.txt", srcText167)
		write(t, root, "b.txt", "x\ny\n")
		opts := Options{Seen: ledgerOf167(t, root, map[string][][2]int{"a.txt": nil, "b.txt": nil})}
		if c.opts != nil {
			opts = c.opts(root)
		}
		res, err := Apply(root, []Input{c.in, {Path: "n.txt", Op: "create", Lines: -1, Body: []string{"n"}}}, opts)
		if err != nil || res.Failed != 1 || res.Hunks[0].Status != StatusFailed {
			t.Errorf("%s: err=%v failed=%d %+v", name, err, res.Failed, res.Hunks)
			continue
		}
		if read(t, root, "b.txt") != "x\ny\n" || read(t, root, "a.txt") != srcText167 {
			t.Errorf("%s: a refused plan changed a file", name)
		}
		if _, err := os.Stat(filepath.Join(root, "n.txt")); err == nil {
			t.Errorf("%s: a refused plan wrote its sibling", name)
		}
	}
}
