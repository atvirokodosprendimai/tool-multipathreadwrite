package read

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-164 Decision 8. A --sub replacement that names a group the pattern lacks
// expanded to nothing and wrote the line without it; it is a usage error, found
// by the scan Go's regexp.Expand makes, so exactly the references Go would
// expand to nothing are refused and every other spelling plans as before.
func TestASubNamingAMissingGroupIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "s.sh"), []byte("x | head -1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	build := func(pattern, repl string) error {
		t.Helper()
		_, err := BuildSub(root, nil, pattern, repl, nil, false, io.Discard)
		return err
	}
	refused := []struct{ pattern, repl, names string }{
		{`head -1`, `^0?$1 x`, `$1`},
		{`(head) -1`, `$1x`, `$1x`},
		{`(head) -1`, `${2}`, `${2}`},
		{`(head) -1`, `${name}`, `${name}`},
		{`(head) -1`, `${100000000}`, `${100000000}`},
		{`(head) -1`, `${1000000000}`, `${1000000000}`},
	}
	for _, c := range refused {
		err := build(c.pattern, c.repl)
		var be *BuildError
		if !errors.As(err, &be) || !be.Usage {
			t.Errorf("--grep %q --sub %q: want a usage BuildError, got %v", c.pattern, c.repl, err)
			continue
		}
		if !strings.Contains(be.Msg, c.names) || !strings.Contains(be.Msg, "$$") {
			t.Errorf("--grep %q --sub %q: the refusal must name %s and $$, got %q", c.pattern, c.repl, c.names, be.Msg)
		}
	}
	planned := []struct{ pattern, repl string }{
		{`(head) -1`, `${1}x`},
		{`head -1`, `$0 x`},
		{`(?P<word>head) -1`, `${word} x`},
		{`head -1`, `$$1 x`},
		{`head -1`, `x $`},
		{`head -1`, `x $-`},
		{`(?P<1000000000>head) -1`, `${1000000000}x`},
		{`(?P<1000000000>head) -1`, `$1000000000 x`},
	}
	for _, c := range planned {
		if err := build(c.pattern, c.repl); err != nil {
			t.Errorf("--grep %q --sub %q: want a plan, got %v", c.pattern, c.repl, err)
		}
	}
}
