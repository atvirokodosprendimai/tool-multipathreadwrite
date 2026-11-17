package main

import (
	"strings"
	"testing"
)

// ADR-196 T7, from the Windows field test of v1.86.4: `\\.\C:\..` was refused,
// rightly, as `"" names a Windows object`; the `..` climbed above the volume
// and left nothing to name. The refusal says so.
func TestAPathThatNamesNothingPastItsPrefixSaysSo(t *testing.T) {
	for _, p := range []string{`\\.\C:\..`, `\\.\Volume{0d2a}\..`, `\\.\`, `//./x/..`} {
		err := keepsOn("windows", "the plan", p)
		if err == nil || !strings.Contains(err.Error(), "names nothing past its prefix") || strings.Contains(err.Error(), `""`) {
			t.Errorf("%q on windows: %v, want it to say it names nothing", p, err)
		}
	}
	if err := keepsOn("windows", "the plan", `\\.\GLOBALROOT\x`); err == nil || !strings.Contains(err.Error(), `"GLOBALROOT" names a Windows object`) {
		t.Errorf("an object is still named: %v", err)
	}
}
