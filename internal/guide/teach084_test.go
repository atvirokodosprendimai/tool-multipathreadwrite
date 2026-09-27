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
		"A write exits 1 when a hunk fails validation, and nothing is written; 2 on a usage or filesystem failure.",
		"a bare directory name prunes that whole subtree",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q", must)
		}
	}
}
