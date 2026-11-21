package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #764. The MCP plan makers, mrw_read with grep and sub and mrw_write with grep
// and sub, say UNREADABLE of a named path that is not there, as a direct read
// does; a path outside the root stays REFUSED.
func TestMrwPlanMakersSayUnreadableOfANamedPathThatIsNotThere(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".state"))
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\nvar foo = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := "../outside-" + filepath.Base(root)
	for _, c := range []struct {
		name, tool string
		args       map[string]any
	}{
		{"mrw_read grep sub", "mrw_read", map[string]any{"grep": "foo", "sub": "bar", "specs": []any{"a.go", "absent.go", outside}}},
		{"mrw_write grep sub", "mrw_write", map[string]any{"grep": "foo", "sub": "bar", "paths": []any{"a.go", "absent.go", outside}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			text := firstText(t, call(t, root, c.tool, c.args))
			if !strings.Contains(text, "==> absent.go  UNREADABLE  ") || strings.Contains(text, "absent.go  REFUSED") {
				t.Errorf("absent.go is not reported UNREADABLE:\n%s", text)
			}
			if !strings.Contains(text, "==> "+outside+"  REFUSED  ") {
				t.Errorf("%s is not reported REFUSED:\n%s", outside, text)
			}
		})
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); strings.Contains(string(b), "bar") {
		t.Errorf("a.go was written beside a path that could not be read: %s", b)
	}
	// The control: the same write without the bad paths does rewrite a.go, so
	// the check above can fail.
	call(t, root, "mrw_write", map[string]any{"grep": "foo", "sub": "bar", "paths": []any{"a.go"}})
	if b, _ := os.ReadFile(filepath.Join(root, "a.go")); !strings.Contains(string(b), "var bar = 1") {
		t.Errorf("the control did not write a.go: %s", b)
	}
}
