package check

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// oldTail is lastLines as it was before ADR-104: the whole file, split.
func oldTail(b []byte, n int) ([]string, int) {
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, 0
	}
	if len(lines) <= n {
		return lines, 0
	}
	return lines[len(lines)-n:], len(lines) - n
}

// ADR-104 T1. lastLines read the whole check log into memory — the file, a
// string copy and the split — to keep 30 lines. It streams now, with a ring of
// the last n lines and at most maxTailLineBytes of each, and answers exactly as
// the split did for every line under that cap.
func TestTheCheckTailReadsALogInBoundedMemory(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, b []byte) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, in := range []string{"", "\n", "a", "a\n", "a\nb", "a\nb\n", "a\n\n", "\n\na\n", "a\r\nb\r\n", "x\ny\nz\n"} {
		for _, n := range []int{1, 2, 30} {
			got, gotT := lastLines(write("e.log", []byte(in)), n)
			want, wantT := oldTail([]byte(in), n)
			if !reflect.DeepEqual(got, want) || gotT != wantT {
				t.Errorf("lastLines(%q, %d) = %q, %d; the split gave %q, %d", in, n, got, gotT, want, wantT)
			}
		}
	}

	var b bytes.Buffer
	for i := 0; i < 1_000_000; i++ {
		fmt.Fprintf(&b, "line %07d of a noisy check ........................\n", i)
	}
	big := write("big.log", b.Bytes())
	b = bytes.Buffer{}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	got, trunc := lastLines(big, 30)
	runtime.ReadMemStats(&after)
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 16<<20 {
		t.Errorf("reading a 50 MB log for 30 lines allocated %d bytes; the tail must not hold the log", alloc)
	}
	if want := fmt.Sprintf("line %07d of a noisy check ........................", 999999); len(got) != 30 || trunc != 1_000_000-30 || got[29] != want {
		t.Errorf("tail of the 50 MB log: %d lines, %d earlier, last %q", len(got), trunc, got[len(got)-1])
	}

	got, _ = lastLines(write("long.log", []byte("first\n"+strings.Repeat("x", 1<<20)+"\n")), 30)
	if len(got) != 2 || got[0] != "first" || len(got[1]) > maxTailLineBytes+64 || !strings.Contains(got[1], "more bytes]") {
		t.Errorf("a 1 MB line was not capped with its marker: %d lines, last %d bytes", len(got), len(got[len(got)-1]))
	}
}
