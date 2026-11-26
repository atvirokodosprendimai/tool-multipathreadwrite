package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-214 Decision 7. The two foreign formats' misses on a file whose line 1
// begins with the UTF-8 byte order mark, when the caller's old side equals it
// without the mark, say so and say how to retry. M is the mark, as bytes.
func TestAMarkedFileRefusalTeachesTheMark(t *testing.T) {
	const (
		m      = "\xEF\xBB\xBF"
		taught = "line 1 begins with a UTF-8 byte order mark (U+FEFF); an exact match must include it"
	)
	root := t.TempDir()
	for name, body := range map[string]string{"m.txt": m + "one\ntwo\n", "u.txt": "one\ntwo\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sr := func(path, old string) error {
		_, err := CompileSearchReplace(root, []byte(path+"\n<<<<<<< SEARCH\n"+old+"\n=======\nONE\n>>>>>>> REPLACE\n"))
		return err
	}
	ap := func(path, old string) error {
		_, err := CompileApplyPatch(root, []byte("*** Begin Patch\n*** Update File: "+path+"\n@@\n-"+old+"\n+ONE\n*** End Patch\n"))
		return err
	}
	for name, miss := range map[string]func(string, string) error{"search_replace": sr, "apply_patch": ap} {
		if err := miss("m.txt", "one"); err == nil || !strings.Contains(err.Error(), taught) {
			t.Errorf("%s: a miss on a marked line 1 is %v, want the mark taught", name, err)
		}
		if err := miss("m.txt", "nothing"); err == nil || strings.Contains(err.Error(), "byte order mark") {
			t.Errorf("%s: a miss that is not the line 1 without its mark is %v, want no mention of a mark", name, err)
		}
		if err := miss("u.txt", "nothing"); err == nil || strings.Contains(err.Error(), "byte order mark") {
			t.Errorf("%s: an unmarked miss is %v, want no mention of a mark", name, err)
		}
		if err := miss("m.txt", m+"one"); err != nil {
			t.Errorf("%s: an old side with the mark: %v", name, err)
		}
	}
}
