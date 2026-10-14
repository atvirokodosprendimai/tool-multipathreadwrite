package apply

import (
	"slices"
	"strings"
	"testing"
)

// ADR-129 amendment (the Windows chaos round of 2026-10-09). A plan that spells
// the source otherwise than its directory does, and names as the destination
// the very spelling the directory holds, was refused "dest already exists":
// the dest IS the source's own entry. The cause is the source's spelling, and
// the refusal now says so.
func TestARenameWhoseDestIsTheDirectorysOwnSpellingNamesTheSourcesSpelling(t *testing.T) {
	root := t.TempDir()
	if !caseInsensitiveFS(t, root) {
		t.Skip("this filesystem does not fold case")
	}
	write(t, root, "plain.txt", "x\n")
	res, err := Apply(root, []Input{{Path: "PLAIN.TXT", Op: "rename", Body: []string{"plain.txt"}, Lines: -1}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || !strings.Contains(failedReason(res), "not spelled that way") {
		t.Fatalf("a source spelled otherwise than its directory, renamed to the directory's spelling, was not refused as the source: applied=%v %q", res.Applied, failedReason(res))
	}
	if got := names(t, root); !slices.Equal(got, []string{"plain.txt"}) {
		t.Fatalf("the directory holds %v, want [plain.txt]", got)
	}
}
