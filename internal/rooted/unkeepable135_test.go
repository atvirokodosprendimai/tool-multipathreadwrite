package rooted

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-135. UnkeepableName separates Resolve's refusal of a name from its other
// refusals: only the first may be counted by a walk. The name refusals are built
// here by the functions that make them, since only a Windows build refuses a
// name in Resolve.
func TestOnlyANameRefusalIsUnkeepable(t *testing.T) {
	root := t.TempDir()
	if err := deviceName("aux.txt", root, filepath.Join(root, "aux.txt")); !UnkeepableName(err) {
		t.Errorf("deviceName(aux.txt) = %v, want a name refusal", err)
	}
	if !UnkeepableName(aliasError("trail.: Windows does not keep it")) {
		t.Error("an alias refusal is not a name refusal")
	}
	if _, err := Resolve(root, "../outside.txt"); err == nil || UnkeepableName(err) {
		t.Errorf("Resolve(../outside.txt) = %v, want a refusal that is not about a name", err)
	}
	st := t.TempDir()
	t.Setenv("XDG_STATE_HOME", st)
	if err := os.MkdirAll(filepath.Join(st, "mrw"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(st, "mrw/seen"); err == nil || UnkeepableName(err) {
		t.Errorf("Resolve of mrw's state = %v, want a refusal that is not about a name", err)
	}
	if UnkeepableName(nil) {
		t.Error("UnkeepableName(nil) = true")
	}
}
