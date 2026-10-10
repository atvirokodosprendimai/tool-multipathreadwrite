//go:build windows

package rooted

import (
	"strings"
	"testing"
)

// The v1.61.0 Windows retest: a rename destination reaches GitDir spelled with
// the OS separator, and the refusal shows the path with "/" as every other does.
func TestADotGitRefusalSpellsThePathWithSlashes(t *testing.T) {
	err := GitDir(t.TempDir(), `.git\hooks\pre-push`)
	if err == nil || strings.Contains(err.Error(), `\`) || !strings.Contains(err.Error(), ".git/hooks/pre-push") {
		t.Errorf("GitDir with a backslash path = %v, want a refusal spelling it with slashes", err)
	}
}
