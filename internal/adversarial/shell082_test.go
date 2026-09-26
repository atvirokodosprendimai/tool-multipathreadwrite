package adversarial

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// needShell skips a test whose check must start when this machine has no POSIX
// shell: no sh on PATH and, on Windows, no Git for Windows (ADR-082). The test
// is about the check's verdict, and without a shell there is no check to judge.
func needShell(t *testing.T) {
	t.Helper()
	if _, ok := check.Shell(); !ok {
		t.Skip("no POSIX shell: no sh on PATH and no Git for Windows (ADR-082)")
	}
}
