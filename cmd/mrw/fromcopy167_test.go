package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-167 T2 through the command: a from= copy of lines the caller read applies
// and says what it copied; without the read it exits 1 and writes nothing.
func TestAFromCopyThroughTheCLINamesWhatItCopied(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const src = "// one\n// two\n// three\n// four\n// five\n// six\n"
	mkTree := func(t *testing.T) string {
		return grepTree(t, map[string]string{"d/a.go": src, "b.txt": "x\ny\n"})
	}
	read := func(t *testing.T, root string, specs ...string) {
		t.Helper()
		if out, err := readIn(t, root, specs...); err != nil {
			t.Fatalf("read %v: %v\n%s", specs, err, out)
		}
	}
	slurp := func(t *testing.T, root, name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	const plan = "@@ b.txt 1 insert-after from=d/a.go:3-5\n"

	t.Run("after the read it applies and the receipt names the copy", func(t *testing.T) {
		root := mkTree(t)
		read(t, root, "d/a.go:3-5", "b.txt")
		out, code := writeIn(t, root, "--no-check", planFile(t, plan))
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		if got := slurp(t, root, "b.txt"); got != "x\n// three\n// four\n// five\ny\n" {
			t.Errorf("b.txt = %q", got)
		}
		if !strings.Contains(out, `copied d/a.go:3-5 from "// three" to "// five"`) {
			t.Errorf("the human receipt does not name the copy:\n%s", out)
		}
	})

	t.Run("--json carries the three keys, spelled with slashes", func(t *testing.T) {
		root := mkTree(t)
		read(t, root, "d/a.go:3-5", "b.txt")
		out, code := writeIn(t, root, "--no-check", "--json", planFile(t, plan))
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		var rec struct {
			Hunks []map[string]any `json:"hunks"`
		}
		if err := json.Unmarshal([]byte(out), &rec); err != nil || len(rec.Hunks) != 1 {
			t.Fatalf("receipt: %v\n%s", err, out)
		}
		h := rec.Hunks[0]
		if h["body_from"] != "d/a.go:3-5" || h["copied_first"] != "// three" || h["copied_last"] != "// five" {
			t.Errorf("hunk keys: %v", h)
		}
	})

	t.Run("--dry-run names from= in parsed:", func(t *testing.T) {
		root := mkTree(t)
		read(t, root, "d/a.go:3-5", "b.txt")
		out, code := writeIn(t, root, "--no-check", "--dry-run", planFile(t, plan))
		if code != 0 || !strings.Contains(out, "from=d/a.go:3-5") || strings.Contains(out, "body=0") {
			t.Errorf("dry run (exit %d) does not name the source in parsed:\n%s", code, out)
		}
		if slurp(t, root, "b.txt") != "x\ny\n" {
			t.Error("a dry run wrote")
		}
	})

	t.Run("a create, $ and A,+N each land", func(t *testing.T) {
		for _, c := range []struct{ plan, want string }{
			{"@@ c.txt - create from=d/a.go:3-5\n", "// three\n// four\n// five\n"},
			{"@@ c.txt - create from=d/a.go:$\n", "// six\n"},
			{"@@ c.txt - create from=d/a.go:2,+2\n", "// two\n// three\n// four\n"},
		} {
			root := mkTree(t)
			read(t, root, "d/a.go")
			out, code := writeIn(t, root, "--no-check", planFile(t, c.plan))
			if code != 0 {
				t.Errorf("%q: exit %d:\n%s", c.plan, code, out)
				continue
			}
			if got := slurp(t, root, "c.txt"); got != c.want {
				t.Errorf("%q: c.txt = %q, want %q", c.plan, got, c.want)
			}
		}
	})

	t.Run("without the read it exits 1 and writes nothing, Force or not", func(t *testing.T) {
		for _, args := range [][]string{{"--no-check"}, {"--no-check", "--force"}} {
			root := mkTree(t)
			read(t, root, "b.txt")
			out, code := writeIn(t, root, append(args, planFile(t, plan))...)
			if code != 1 {
				t.Errorf("%v: exit %d, want 1:\n%s", args, code, out)
			}
			if slurp(t, root, "b.txt") != "x\ny\n" {
				t.Errorf("%v: a refused copy wrote", args)
			}
			if strings.Contains(out, "// three") {
				t.Errorf("%v: the refusal discloses a copied line:\n%s", args, out)
			}
		}
	})

	t.Run("a --json copy from a source that is not valid UTF-8 is refused before anything lands", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows names are not bytes")
		}
		root := mkTree(t)
		bad := "bad\xffname.txt"
		if err := os.WriteFile(filepath.Join(root, bad), []byte("one\n"), 0o644); err != nil {
			t.Skipf("this filesystem keeps no such name: %v", err)
		}
		read(t, root, "b.txt", bad)
		out, code := writeIn(t, root, "--no-check", "--json", planFile(t, "@@ b.txt 1 insert-after from="+bad+":1\n"))
		if code == 0 {
			t.Errorf("exit 0, want a refusal:\n%s", out)
		}
		if slurp(t, root, "b.txt") != "x\ny\n" {
			t.Error("the refused plan wrote")
		}
	})
}
