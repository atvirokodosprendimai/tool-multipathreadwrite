package guide

import (
	"strings"
	"testing"
)

// ADR-084 (blind reading 03). An agent learned a write's exit 1 and 2 from the
// output, and three runs avoided --exclude because nothing said a bare
// directory name prunes the subtree. `mrw instructions` says both.
func TestCLITeachesAWritesExitCodesAndExcludePruning(t *testing.T) {
	got := CLI()
	for _, must := range []string{
		"A write exits 1 when a hunk fails validation, or is refused during staging, before the commit loop, for a cause in its target (held, a permission, a name refused, changed since mrw read it), and nothing is written; 2 on a usage or filesystem failure, or a failure in the commit loop.",
		"a bare directory name met below where the walk starts prunes that whole subtree",
		"a path you name is walked even if it matches",
		"--ast-grep drops excluded hits after the binary runs",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q", must)
		}
	}
}
