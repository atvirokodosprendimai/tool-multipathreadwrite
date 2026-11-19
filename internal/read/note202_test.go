package read

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-202. With NextLine off a numbered run of two or more lines that ends
// before the file does is followed by a note about a multi-line replace: it is
// about licensing a write, and a stream is never written, so the inverted read
// of standard input turns it off with NoReplaceNote. The zero value is today's
// behaviour and nothing else in the answer changes.
func TestARunCanLeaveOutTheReplaceNote(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec("f.txt:1-2")
	if err != nil {
		t.Fatal(err)
	}
	serve := func(opt Options) string {
		t.Helper()
		var out bytes.Buffer
		if _, problems := Run(&out, root, []Spec{sp}, opt); problems != 0 {
			t.Fatalf("%d problem(s):\n%s", problems, out.String())
		}
		return out.String()
	}
	const note = "-- note: a multi-line replace of 1-2 needs a served line after 2\n"
	with := serve(Options{Numbers: true})
	if !strings.Contains(with, note) {
		t.Fatalf("without the option the run has no note:\n%s", with)
	}
	without := serve(Options{Numbers: true, NoReplaceNote: true})
	if strings.Contains(without, "-- note:") {
		t.Errorf("NoReplaceNote left the note in:\n%s", without)
	}
	if without != strings.Replace(with, note, "", 1) {
		t.Errorf("the option changed more than the note:\nwith:\n%s\nwithout:\n%s", with, without)
	}
}
