package rooted

import (
	"errors"
	"path/filepath"
	"testing"
)

// A refusal of a name's own spelling (a device name, a name Windows does not
// keep as written) is not a refusal of where a path leads, so the advice that
// fits an escape ("point --root where you mean") is wrong for it. RefusedByName
// is what callers ask before appending it.
func TestARefusalOfASpellingIsRefusedByName(t *testing.T) {
	root := t.TempDir()
	for name, err := range map[string]error{
		"a device name":     deviceName("aux.txt", root, filepath.Join(root, "aux.txt")),
		"an alias":          aliasError("trail.: Windows does not keep it"),
		"a device via link": viaLinkError{deviceName("aux.txt", root, filepath.Join(root, "aux.txt"))},
	} {
		if !RefusedByName(err) {
			t.Errorf("%s: %v is not refused by name", name, err)
		}
	}
	if RefusedByName(nil) || RefusedByName(errors.New("a plain error")) {
		t.Error("an error that names no spelling was refused by name")
	}
	if _, err := Resolve(root, "../outside.txt"); err == nil || RefusedByName(err) {
		t.Errorf("Resolve(../outside.txt) = %v, want a refusal of where it leads, not of a name", err)
	}
}
