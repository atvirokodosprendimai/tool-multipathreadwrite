package main

import (
	"strings"
	"testing"
)

// A name Windows does not keep as written is refused for its spelling, and the
// closing words that fit an escape from the root ("read it with --root pointed
// where you mean", "a plan may only change files under the directory mrw was
// pointed at") are wrong for it: no --root makes "trail." a name. The refusal
// says what is wrong with the name and stops.
func TestANameRefusalOffersNoAdviceAboutTheRoot(t *testing.T) {
	root := t.TempDir()
	out, code := runIn(t, root, "read", "trail.")
	if code != 1 || !strings.Contains(out, "REFUSED") || !strings.Contains(out, "Windows does not keep") || strings.Contains(out, "--root") {
		t.Errorf("read trail.: exit %d\n%s\nwant a refusal of the name and no advice about --root", code, out)
	}
	withStdin(t, "x\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "trail.") })
	if code != 1 || !strings.Contains(out, "Windows does not keep") || strings.Contains(out, "under the directory mrw was pointed at") {
		t.Errorf("write --create trail.: exit %d\n%s\nwant a refusal of the name and no advice about the root", code, out)
	}
}
