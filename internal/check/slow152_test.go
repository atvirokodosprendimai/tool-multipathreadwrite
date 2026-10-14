package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ADR-152. A check that ran the whole project's command for named paths, and
// took SlowAfter or longer, says how to scope it; a scoped run, a run for no
// paths and a fast run say nothing.
func TestASlowUnscopedCheckSaysHowToScopeIt(t *testing.T) {
	needSh(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := SlowAfter
	t.Cleanup(func() { SlowAfter = old })
	SlowAfter = 0

	hint := func(cfg Config, paths []string) string {
		t.Helper()
		res, err := Run(context.Background(), root, cfg, paths)
		if err != nil || !res.Ran {
			t.Fatalf("the check did not run: %+v %v", res, err)
		}
		return res.Hint
	}

	whole := Config{Check: "true", declared: true}
	if h := hint(whole, []string{"a.txt"}); !strings.Contains(h, `declare "scoped_check"`) || !strings.Contains(h, "{dirs}") {
		t.Errorf("a slow whole-project run for a named path did not name scoped_check: %q", h)
	}
	if h := hint(Config{Check: "true", ScopedCheck: "true {files}", declared: true}, []string{"a.txt"}); h != "" {
		t.Errorf("a scoped run carried a hint: %q", h)
	}
	if h := hint(Config{Check: "true", ScopedCheck: "go test {packages}", declared: true}, []string{"a.txt"}); !strings.Contains(h, `scoped_check "go test {packages}" could not name these paths`) {
		t.Errorf("a scoped_check that fell back did not name its template: %q", h)
	}
	if h := hint(whole, nil); h != "" {
		t.Errorf("a run for no paths (--full) carried a hint: %q", h)
	}
	SlowAfter = time.Hour
	if h := hint(whole, []string{"a.txt"}); h != "" {
		t.Errorf("a fast run carried a hint: %q", h)
	}
}
