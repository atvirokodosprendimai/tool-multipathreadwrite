package check

import (
	"slices"
	"strings"
	"testing"
)

// #754. check.outside_scope names the edited files a {packages}-only scoped
// check ran without. They come from the engine, which spells a path with the
// platform's separator, and a receipt spells a root-relative path with "/"
// (ADR-091, ADR-132). Driven with `\` so it runs on every platform.
func TestOutsideScopeIsSpelledWithSlashesInAReceipt(t *testing.T) {
	in := &Result{Ran: true, OutsideScope: []string{`d\x.go`, `y.rs`}, outsideTemplate: "go test {packages}", Command: `go test .\d`}
	got := in.Slashed('\\')
	if want := []string{"d/x.go", "y.rs"}; !slices.Equal(got.OutsideScope, want) {
		t.Errorf("outside_scope = %q, want %q", got.OutsideScope, want)
	}
	if line := got.OutsideLine(); !strings.Contains(line, "d/x.go") || strings.Contains(line, `d\x.go`) {
		t.Errorf("the human line spells the path as the engine does: %s", line)
	}
	if got.Command != in.Command {
		t.Errorf("the command that ran was rewritten: %q", got.Command)
	}
	if in.OutsideScope[0] != `d\x.go` {
		t.Errorf("Slashed changed the result it was handed: %q", in.OutsideScope)
	}
	if (*Result)(nil).Slashed('\\') != nil {
		t.Error("a nil result did not stay nil")
	}
}
