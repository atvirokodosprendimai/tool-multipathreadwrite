package read

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ADR-153 Decision 2, from the Codex review of #406: a file the walk found
// whose lines no longer match when it is read — another writer replaced it in
// between — is a missed range, as a plain grep reports it, not a count of 0
// that exits 0. Run is handed the spec the walk made, over a file that no
// longer matches, which is the state that race leaves.
func TestACountOfAFileThatNoLongerMatchesIsAProblem(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"a.txt": "NEEDLE\n", "b.txt": "changed since the walk\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	re := regexp.MustCompile("NEEDLE")
	specs := []Spec{
		{Path: "a.txt", Raw: "a.txt", Ranges: []Range{{Re: re, Text: "/NEEDLE/"}}},
		{Path: "b.txt", Raw: "b.txt", Ranges: []Range{{Re: re, Text: "/NEEDLE/"}}},
	}
	var out bytes.Buffer
	counts := map[string]int{}
	_, problems := Run(&out, root, specs, Options{Count: true, Counts: counts})
	if problems != 1 {
		t.Errorf("a count of a file that no longer matches: %d problem(s), want 1:\n%s", problems, out.String())
	}
	if !strings.Contains(out.String(), "!! no match for /NEEDLE/") {
		t.Errorf("the miss is not reported as a plain grep reports it:\n%s", out.String())
	}
	if n, ok := counts["b.txt"]; !ok || n != 0 || counts["a.txt"] != 1 {
		t.Errorf("counts = %v, want a.txt 1 and b.txt 0", counts)
	}
	if !strings.Contains(out.String(), "-- 1 matching line(s) in 2 file(s)") {
		t.Errorf("the total does not count both files read:\n%s", out.String())
	}
}
