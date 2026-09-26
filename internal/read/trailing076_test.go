package read

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ADR-076 T1. `mrw read a.txt/` served a.txt: the separator names a directory,
// and cleaning dropped it. The spec is reported UNREADABLE, spelled relative or
// absolute; the file spelled plainly is served, and a directory keeps its
// separator.
func TestASpecEndingInASeparatorIsNotServedAsAFile(t *testing.T) {
	root, opt := fixture(t)
	sep := string(filepath.Separator)
	abs, err := filepath.Abs(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range []string{"a.go" + sep, abs + sep} {
		out, problems := run(t, root, opt, spec)
		if problems != 1 || !strings.Contains(out, "UNREADABLE") || !strings.Contains(out, "names a directory") {
			t.Errorf("read %q: problems=%d\n%s", spec, problems, out)
		}
		if strings.Contains(out, "| ") {
			t.Errorf("read %q served lines through a directory spelling:\n%s", spec, out)
		}
	}
	if out, problems := run(t, root, opt, "a.go"); problems != 0 || !strings.Contains(out, "| ") {
		t.Errorf("read a.go: problems=%d\n%s", problems, out)
	}
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "s.go"), []byte("package s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	specs, _, err := Walk(root, []string{"sub" + sep}, WalkOptions{Pattern: regexp.MustCompile("package")})
	if err != nil || len(specs) != 1 {
		t.Errorf("a grep under sub%s found %v, err %v; want sub/s.go", sep, specs, err)
	}
}

// Codex review of #237. A grep given a file spelled as a directory walked the
// file, relative or absolute: the absolute branch cleaned the separator away
// before the boundary could judge it. Both are reported, and nothing is served.
func TestAGrepThroughAFileSpelledAsADirectoryIsRefused(t *testing.T) {
	root, _ := fixture(t)
	sep := string(filepath.Separator)
	abs, err := filepath.Abs(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"a.go" + sep, abs + sep} {
		specs, probs, err := Walk(root, []string{p}, WalkOptions{Pattern: regexp.MustCompile(".")})
		if err != nil || len(specs) != 0 || len(probs) != 1 || !strings.Contains(probs[0].Reason, "names a directory") {
			t.Errorf("grep through %q: specs %v, problems %v, err %v", p, specs, probs, err)
		}
	}
}
