//go:build !race

package read

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// TestAnUnchangedGuardSplitsTheFileOnce holds the --if-changed answer for a file
// that still has the quoted sha: it counts what it did not serve, and that needs
// the lines once. Measured on this 2 MiB sample on darwin: 17,312,680 bytes
// allocated when the answer split the file a second time to learn its
// terminator, 11,021,224 now. Not under -race: the race detector adds about
// 4.7 MB to both, which leaves no bound between them worth trusting.
func TestAnUnchangedGuardSplitsTheFileOnce(t *testing.T) {
	data := statSample()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "s.txt"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	opt := Options{IfChanged: map[string]string{"s.txt": seen.SHA(data)[:8]}}
	var out bytes.Buffer
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, problems := Run(&out, root, []Spec{{Path: "s.txt"}}, opt)
	runtime.ReadMemStats(&after)
	if want := fmt.Sprintf("==> s.txt  unchanged since sha %s — 262144 lines, 2097152 bytes not served\n", seen.SHA(data)[:8]); problems != 0 || out.String() != want {
		t.Fatalf("problems %d, answer %q; want %q", problems, out.String(), want)
	}
	if got := after.TotalAlloc - before.TotalAlloc; got > 14<<20 {
		t.Errorf("an unchanged answer for %d bytes allocated %d bytes; one split of it stays under 14 MiB, two do not", len(data), got)
	}
}
