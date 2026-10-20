package writer

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-164 Decision 4. A write that took a Go or module file away runs the whole
// check, because the check is handed only the removed file's directory and the
// packages that import it are not among the paths; a write that only edits
// places what it can and names the rest.
func TestAWriteThatRemovesCodeRunsTheWholeCheckThroughVerify(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	edit := apply.Input{Path: "a/a.go", Start: 2, End: 2, Op: "replace", Body: []string{"func A() { _ = 1 }"}, Lines: -1, SrcLine: 1}
	script := apply.Input{Path: "run.sh", Start: 1, End: 1, Op: "replace", Body: []string{"false"}, Lines: -1, SrcLine: 2}
	for name, tc := range map[string]struct {
		in      []apply.Input
		command string
		outside []string
	}{
		"an edit beside a script is scoped and names it": {[]apply.Input{edit, script}, "echo ./a", []string{"run.sh"}},
		"the unlink of go.sum runs everything":           {[]apply.Input{edit, {Path: "go.sum", Op: "unlink", Lines: -1, SrcLine: 2}}, "echo WHOLE", nil},
		"the unlink of a Go file runs everything":        {[]apply.Input{edit, {Path: "b/b.go", Op: "unlink", Lines: -1, SrcLine: 2}}, "echo WHOLE", nil},
		"a rename away from a Go file runs everything":   {[]apply.Input{edit, {Path: "b/b.go", Op: "rename", Body: []string{"b/c.go"}, Lines: -1, SrcLine: 2}}, "echo WHOLE", nil},
	} {
		root := checkout(t)
		obs := map[string]seen.Observation{}
		for path, body := range map[string]string{
			".quality-harness.json": `{"check":"echo WHOLE","scoped_check":"echo {packages}"}`,
			"go.mod":                "module x\n",
			"go.sum":                "\n",
			"a/a.go":                "package a\nfunc A() {}\n",
			"b/b.go":                "package b\n",
			"b/b2.go":               "package b\n",
			"run.sh":                "true\n",
		} {
			full := filepath.Join(root, path)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			obs[path] = seen.Observation{SHA: seen.SHA([]byte(body))}
		}
		if err := seen.Record(root, obs); err != nil {
			t.Fatal(err)
		}
		p, err := Prepare(Request{Root: root, In: tc.in})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		l, err := p.Land()
		if err != nil || l.Err != nil || l.LedgerErr != nil || !l.Res.Applied {
			t.Fatalf("%s: landing: %v %v %v %+v", name, err, l.Err, l.LedgerErr, l.Res.Hunks)
		}
		v := l.Verify(context.Background())
		if v.Check == nil || !v.Check.Ran {
			t.Fatalf("%s: no check ran: %+v %v", name, v.Check, v.CheckErr)
		}
		if v.Check.Command != tc.command || !reflect.DeepEqual(v.Check.OutsideScope, tc.outside) {
			t.Errorf("%s: ran %q outside %v, want %q outside %v", name, v.Check.Command, v.Check.OutsideScope, tc.command, tc.outside)
		}
	}
}
