package rooted

import (
	"errors"
	"path/filepath"
	"testing"
)

// The Codex review of #363. A normally named link whose target holds a device
// name is refused with ErrDeviceName too, and counting it would say where the
// link leads, outside the root included. It stays an ErrDeviceName, with its
// words, and is not a refusal of the discovered file's own name.
func TestADeviceNameReachedThroughALinkIsNotUnkeepable(t *testing.T) {
	root := t.TempDir()
	err := deviceName("view.txt", root, filepath.Join(root, "sub", "aux.txt"))
	if err == nil || !errors.Is(err, ErrDeviceName) {
		t.Fatalf("deviceName through a link = %v, want an ErrDeviceName", err)
	}
	if UnkeepableName(err) {
		t.Errorf("a device name reached through a link was counted as the file's own name: %v", err)
	}
	if err := deviceName("aux.txt", root, filepath.Join(root, "aux.txt")); !UnkeepableName(err) {
		t.Errorf("a direct device name was not counted: %v", err)
	}
}
