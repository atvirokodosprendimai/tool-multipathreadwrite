package mcp

import (
	"runtime"
	"testing"
)

// ADR-187. mrw_write runs no check unless check: true: with no check argument,
// and with check: false, a code write in a tree whose check fails lands and
// carries no check; check: true runs it by ADR-054's rule and reports it failed.
func TestAnMCPWriteRunsNoCheckUnlessAskedFor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	for _, tc := range []struct {
		name string
		args map[string]any
		ran  bool
	}{
		{"no check argument", map[string]any{}, false},
		{"check false", map[string]any{"check": false}, false},
		{"check true", map[string]any{"check": true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := checkTree113(t, `{"check":"exit 3"}`)
			args := map[string]any{"plan": goEdit113}
			for k, v := range tc.args {
				args[k] = v
			}
			sc := structured(t, call(t, root, "mrw_write", args))
			if sc["applied"] != true {
				t.Fatalf("the write did not apply: %v", sc)
			}
			chk, has := sc["check"].(map[string]any)
			if has != tc.ran {
				t.Fatalf("check present %v, want %v: %v", has, tc.ran, sc)
			}
			if tc.ran && (chk["ran"] != true || chk["exit_code"] != float64(3)) {
				t.Errorf("check %v, want one that ran and exited 3", chk)
			}
		})
	}
}
