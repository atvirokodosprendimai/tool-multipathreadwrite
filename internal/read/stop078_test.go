package read

import (
	"strings"
	"testing"
)

// ADR-078 T3. A caller that has already decided to refuse the answer asks the
// read to stop; Stop is asked before each spec, and true ends the read there.
func TestRunStopsBetweenSpecsWhenAskedTo(t *testing.T) {
	root, opt := fixture(t)
	asked := 0
	opt.Stop = func() bool { asked++; return asked > 1 }
	var specs []Spec
	for i := 0; i < 3; i++ {
		sp, err := ParseSpec("a.go")
		if err != nil {
			t.Fatal(err)
		}
		specs = append(specs, sp)
	}
	var sb strings.Builder
	Run(&sb, root, specs, opt)
	if n := strings.Count(sb.String(), "==> "); n != 1 {
		t.Errorf("served %d specs, want 1 before Stop:\n%s", n, sb.String())
	}
}

// ADR-078 T2. A glob that can never match is refused: one path.Match rejects,
// and one that starts with /, since globs match root-relative paths and base
// names.
func TestCheckExcludeRefusesWhatCanNeverMatch(t *testing.T) {
	for _, g := range []string{"[", "/vendor"} {
		if err := CheckExclude([]string{"ok", g}); err == nil {
			t.Errorf("CheckExclude accepted %q", g)
		}
	}
	if err := CheckExclude([]string{"vendor", "*_test.go", "sub/*.go"}); err != nil {
		t.Errorf("CheckExclude refused well-formed globs: %v", err)
	}
}
