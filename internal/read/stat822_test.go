package read

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// statSample is 2 MiB of eight-byte lines: 262,144 of them, the shape #822
// measured, where building one string per line costs far more than the file.
func statSample() []byte { return bytes.Repeat([]byte("1234567\n"), 262144) }

// TestAStatReadDoesNotBuildTheFilesLines holds #822: a plain --stat prints a
// header (lines, bytes, sha) and licenses nothing, so it must not allocate a
// string per line. Splitting this sample allocates over 4 MiB for the slice of
// lines alone; the header-only path needs a few KiB.
func TestAStatReadDoesNotBuildTheFilesLines(t *testing.T) {
	data := statSample()
	specs := []Spec{{Path: "sample.txt", Data: data}}
	var out bytes.Buffer
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	observed, problems := Run(&out, ".", specs, Options{Stat: true})
	runtime.ReadMemStats(&after)
	// A nil span set is a whole-file licence; a stat records an empty, non-nil one.
	if o, ok := observed["sample.txt"]; problems != 0 || !ok || o.Spans == nil || len(o.Spans) != 0 {
		t.Fatalf("stat: problems %d, observed %+v (present %v); want 0 problems and an empty, non-nil span set", problems, o, ok)
	}
	if want := "==> sample.txt  262144L  2097152B  sha "; !bytes.HasPrefix(out.Bytes(), []byte(want)) {
		t.Fatalf("header %q, want it to begin %q", out.String(), want)
	}
	if got := after.TotalAlloc - before.TotalAlloc; got > 1<<20 {
		t.Errorf("a stat of %d bytes allocated %d bytes; the header-only path should stay under 1 MiB", len(data), got)
	}
}

// TestAStatThatNeedsTheLinesStillHasThem holds #822's guards in Run itself. The
// CLI and MCP refuse --count and --if-changed beside --stat today, so no surface
// reaches these; but Run takes the options as given, and a count or a guard
// answered from a header-only read would see no lines and report 0.
func TestAStatThatNeedsTheLinesStillHasThem(t *testing.T) {
	data := []byte("a\nb\na\n")
	var out bytes.Buffer
	specs := []Spec{{Path: "f.txt", Data: data, Ranges: []Range{{Re: regexp.MustCompile("a"), Text: "/a/"}}}}
	if _, problems := Run(&out, ".", specs, Options{Stat: true, Count: true}); problems != 0 || !strings.Contains(out.String(), "3L") || !strings.Contains(out.String(), "2 matching line(s)") {
		t.Errorf("stat with count: problems %d, output %q; want 3L and 2 matching line(s)", problems, out.String())
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	opt := Options{Stat: true, IfChanged: map[string]string{"f.txt": seen.SHA(data)[:8]}}
	if _, problems := Run(&out, root, []Spec{{Path: "f.txt"}}, opt); problems != 0 || !strings.Contains(out.String(), "unchanged since sha") || !strings.Contains(out.String(), "— 3 lines") {
		t.Errorf("stat with if_changed: problems %d, output %q; want the unchanged answer naming 3 lines", problems, out.String())
	}
}

func BenchmarkStatManyShortLines(b *testing.B) {
	data := statSample()
	specs := []Spec{{Path: "sample.txt", Data: data}}
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		observed, problems := Run(io.Discard, ".", specs, Options{Stat: true})
		if problems != 0 || observed["sample.txt"].Spans == nil || len(observed["sample.txt"].Spans) != 0 {
			b.Fatal("stat must report the file with no lines licensed")
		}
	}
}
