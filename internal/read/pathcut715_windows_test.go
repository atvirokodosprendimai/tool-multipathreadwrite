//go:build windows

package read

import (
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/iter"
)

// Review of #722, P3-6. iter.Path skips the drive prefix as ParseSpec does, so
// `C:\repo\doc.json:$["a:b"]` names the file and not the drive.
func TestIterPathSkipsTheVolumePrefix(t *testing.T) {
	const file = `C:\repo\doc.json`
	if filepath.VolumeName(file) != "C:" {
		t.Skip("no drive volume on this host")
	}
	for _, spec := range []string{
		file,
		file + `:$["a:b"]`,
		file + ":1-3",
		"C:/repo/doc.json",
		"C:/repo/doc.json:1-3",
	} {
		want := file
		if spec[2] == '/' {
			want = "C:/repo/doc.json"
		}
		if got := iter.Path(spec); got != want {
			t.Errorf("iter.Path(%q) = %q, want %q", spec, got, want)
		}
	}
}
