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

// #752. A device name is told from the other names Windows will not keep, so a walk
// can count the two apart; a device name reached through a link is neither.
func TestADeviceNameIsToldFromAnAlias(t *testing.T) {
	root := t.TempDir()
	if err := deviceName("aux.txt", root, filepath.Join(root, "aux.txt")); !DeviceName(err) {
		t.Errorf("deviceName(aux.txt) = %v, want a device name", err)
	}
	if DeviceName(aliasError("trail.: Windows does not keep it")) {
		t.Error("an alias refusal is a device name")
	}
	// A device name reached through a link is the link's, not the file's own: not
	// one for DeviceName either, whatever the caller (#762 review).
	if err := deviceName("view.txt", root, filepath.Join(root, "sub", "aux.txt")); DeviceName(err) {
		t.Errorf("a device name reached through a link was counted as a device name: %v", err)
	}
	if DeviceName(nil) {
		t.Error("DeviceName(nil) = true")
	}
}
