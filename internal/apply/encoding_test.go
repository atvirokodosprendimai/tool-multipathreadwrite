package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-073. A UTF-16LE file was served as byte-split lines and a replace
// rewrote it at exit 0, BOM gone and encodings mixed: nothing refused a line
// edit to a file mrw cannot split into lines. Each encoding is refused by
// name, the bytes untouched, with --force too (it overrides the read ledger,
// not the file's encoding).
func TestAUTF16FileIsRefusedNotRewrittenAsMixedEncodings(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"utf16le.txt", "\xff\xfea\x00\n\x00b\x00\n\x00", "UTF-16 (BOM FF FE)"},
		{"utf16be.txt", "\xfe\xff\x00a\x00\n\x00b\x00\n", "UTF-16 (BOM FE FF)"},
		{"utf32le.txt", "\xff\xfe\x00\x00a\x00\x00\x00\n\x00\x00\x00", "UTF-32 (BOM FF FE 00 00)"},
		{"utf32be.txt", "\x00\x00\xfe\xff\x00\x00\x00a\x00\x00\x00\n", "UTF-32 (BOM 00 00 FE FF)"},
		{"nulbyte.bin", "ab\x00cd\nef\n", "NUL byte at offset 2"}, // not nul.bin: a Windows device name (ADR-081)
	} {
		for _, force := range []bool{false, true} {
			root := t.TempDir()
			write(t, root, tc.name, tc.body)
			res, err := Apply(root, []Input{
				{Path: tc.name, Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1},
			}, Options{Force: force, Seen: map[string]Seen{tc.name: {SHA: shaOfFile(t, root, tc.name)}}})
			if err != nil {
				t.Fatal(err)
			}
			if res.Applied || res.Failed != 1 || !strings.Contains(res.Hunks[0].Reason, tc.want) {
				t.Errorf("%s (force %v): applied %v, reason %q, want a refusal naming %q", tc.name, force, res.Applied, res.Hunks[0].Reason, tc.want)
			}
			if got := read(t, root, tc.name); got != tc.body {
				t.Errorf("%s (force %v) changed under a refused write: %q", tc.name, force, got)
			}
		}
	}
}

// The bound is 8 KiB of BYTES: a NUL at offset 8191 is inside it, one at 8192
// is not (and that file is edited as text, as before).
func TestANULInTheFirst8KiBIsRefusedAndOneAfterItIsNot(t *testing.T) {
	for _, tc := range []struct {
		at      int
		refused bool
	}{{8191, true}, {8192, false}} {
		root := t.TempDir()
		b := []byte(strings.Repeat("a", tc.at) + "\x00\nsecond\n")
		b[0] = 'x'
		if err := os.WriteFile(filepath.Join(root, "f.txt"), b, 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := Apply(root, []Input{
			{Path: "f.txt", Start: 2, End: 2, Op: "replace", Body: []string{"SECOND"}, Lines: -1},
		}, Options{Seen: map[string]Seen{"f.txt": {SHA: shaOfFile(t, root, "f.txt")}}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Applied == tc.refused {
			t.Errorf("a NUL at offset %d: applied %v, want refused %v (%+v)", tc.at, res.Applied, tc.refused, res.Hunks)
		}
	}
}

// unlink and rename do not split lines, so an encoded file can still be
// removed or moved; a create has no existing bytes to be foreign.
func TestUnlinkAndRenameOfAForeignFileStillApply(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "\xff\xfea\x00\n\x00")
	write(t, root, "b.txt", "\xff\xfeb\x00\n\x00")
	seen := map[string]Seen{"a.txt": {SHA: shaOfFile(t, root, "a.txt")}, "b.txt": {SHA: shaOfFile(t, root, "b.txt")}}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, Index: 1},
		{Path: "new.txt", Op: "create", Body: []string{"fresh"}, Lines: -1, Index: 2},
	}, Options{Seen: seen})
	if err != nil || !res.Applied {
		t.Fatalf("an unlink, a rename and a create beside encoded files were refused: %v %+v", err, res.Hunks)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Errorf("a.txt was not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "b.txt")); !os.IsNotExist(err) {
		t.Errorf("b.txt was not moved away: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "c.txt")); string(got) != "\xff\xfeb\x00\n\x00" {
		t.Errorf("c.txt holds %q, want b.txt's bytes unchanged", got)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "new.txt")); string(got) != "fresh\n" {
		t.Errorf("new.txt holds %q", got)
	}
}

// Review of #230. The refusal is carried by the hunk that would split the file,
// not by a path op beside it, and the receipt reports the refused file as it
// is. A create over an existing encoded file keeps its own reason.
func TestTheEncodingRefusalNamesTheLineEditAndTheFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "\xff\xfea\x00\n\x00")
	sha := shaOfFile(t, root, "a.txt")
	seen := map[string]Seen{"a.txt": {SHA: sha}}
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 1},
	}, Options{Seen: seen})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || len(res.Hunks) != 2 || res.Hunks[0].Status != StatusSkipped ||
		res.Hunks[1].Status != StatusFailed || !strings.Contains(res.Hunks[1].Reason, "UTF-16") {
		t.Fatalf("the refusal is not on the line edit: %+v", res.Hunks)
	}
	var reported bool
	for _, f := range res.Files {
		if f.Path == "a.txt" {
			reported = f.SHABefore == sha && f.LinesFrom > 0
		}
	}
	if !reported {
		t.Fatalf("the refused file is not reported with its sha and line count: %+v", res.Files)
	}
	res, err = Apply(root, []Input{{Path: "a.txt", Op: "create", Body: []string{"x"}, Lines: -1, Index: 0}}, Options{Seen: seen})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied || len(res.Hunks) != 1 || !strings.Contains(res.Hunks[0].Reason, "already exists") {
		t.Fatalf("a create over an existing encoded file does not say it exists: %+v", res.Hunks)
	}
}
