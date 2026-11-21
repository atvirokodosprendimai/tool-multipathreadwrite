package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Final pass of #745. The ledger's spellings are found by a stat of each of its
// keys; a batch of files each asking must pay for that once, not once per file.
func TestTheLedgerIsStattedOncePerApplyNotOncePerFile(t *testing.T) {
	const body = "func A() {\n\tif x {\n\t}\n}\n"
	root := t.TempDir()
	sha := seen.SHA([]byte(body))
	ledger := map[string]Seen{}
	var in []Input
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("f%d.go", i)
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		ledger[name] = Seen{SHA: sha, Written: true}
		in = append(in, Input{Path: name, Op: "replace", Start: 1, End: 3, Anchor: "func A", Lines: -1, Body: []string{"func A() {", "}"}})
	}
	for i := 0; i < 10; i++ {
		ledger[fmt.Sprintf("other%d.go", i)] = Seen{SHA: sha, Written: true}
	}
	var stats int
	restore := aliasStatFn
	t.Cleanup(func() { aliasStatFn = restore })
	aliasStatFn = func(p string) (os.FileInfo, error) { stats++; return os.Stat(p) }
	res, err := Apply(root, in, Options{Seen: ledger})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || res.Failed != 20 {
		t.Fatalf("applied %v, failed %d, want twenty refusals", res.Applied, res.Failed)
	}
	if stats > len(ledger) {
		t.Errorf("%d stats of ledger keys for 20 files and %d keys, want at most one per key", stats, len(ledger))
	}
}

// The plan's own record decides when it shows the line after: no spelling of the
// file is looked for, so nothing is stat'ed.
func TestTheLedgerIsNotStattedWhenThePlansOwnRecordShowsTheLineAfter(t *testing.T) {
	const body = "func A() {\n\tif x {\n\t}\n}\n"
	root := t.TempDir()
	sha := seen.SHA([]byte(body))
	ledger := map[string]Seen{}
	var in []Input
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("f%d.go", i)
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		ledger[name] = Seen{SHA: sha, Written: true, Shown: [][2]int{{4, 4}}}
		in = append(in, Input{Path: name, Op: "replace", Start: 1, End: 3, Anchor: "func A", Lines: -1, Body: []string{"func A() {", "}"}})
	}
	var stats int
	restore := aliasStatFn
	t.Cleanup(func() { aliasStatFn = restore })
	aliasStatFn = func(p string) (os.FileInfo, error) { stats++; return os.Stat(p) }
	res, err := Apply(root, in, Options{Seen: ledger})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || stats != 0 {
		t.Errorf("applied %v with %d stats of ledger keys, want applied and none", res.Applied, stats)
	}
}

// Another version's record is no showing and no written file: an alias whose
// record describes an older file must not make a current partial record whole.
func TestAnAliasRecordOfAnotherVersionIsIgnoredByTheLineAfterRule(t *testing.T) {
	const body = "func A() {\n\tif x {\n\t}\n}\n"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("f.go", filepath.Join(root, "link.go")); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	sha := seen.SHA([]byte(body))
	in := []Input{{Path: "link.go", Op: "replace", Start: 1, End: 3, Anchor: "func A", Lines: -1, Body: []string{"func A() {", "}"}}}
	res, err := Apply(root, in, Options{Seen: map[string]Seen{
		"link.go": {SHA: sha, Spans: [][2]int{{1, 4}}},               // a read of the current version
		"f.go":    {SHA: seen.SHA([]byte("older\n")), Written: true}, // whole and written, of an older one
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || res.Failed != 0 {
		t.Errorf("an older version's record made the current one whole: %+v", res.Hunks)
	}
}

// The other direction: the plan's own record is whole and written, nothing shown,
// and another spelling holds a read of line 4 from an older version of the file. A
// showing of another version shows nothing of this one, so the line after is still
// unread and the plan is refused.
func TestAnAliasShowingOfAnotherVersionDoesNotMeetTheLineAfter(t *testing.T) {
	const body = "func A() {\n\tif x {\n\t}\n}\n"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("f.go", filepath.Join(root, "link.go")); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	sha := seen.SHA([]byte(body))
	in := []Input{{Path: "link.go", Op: "replace", Start: 1, End: 3, Anchor: "func A", Lines: -1, Body: []string{"func A() {", "}"}}}
	res, err := Apply(root, in, Options{Seen: map[string]Seen{
		"link.go": {SHA: sha, Written: true},
		"f.go":    {SHA: seen.SHA([]byte("older\n")), Spans: [][2]int{{1, 4}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, "needs a served line after 3") {
		t.Errorf("an older version's showing met the line after: %+v", res.Hunks)
	}
}
