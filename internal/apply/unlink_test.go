package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnlinkRemovesThePath(t *testing.T) {
	root := t.TempDir()
	write(t, root, "gone.txt", "keep me\n")

	res, err := Apply(root, []Input{
		{Path: "gone.txt", Op: "unlink", Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "gone.txt")); !os.IsNotExist(err) {
		t.Fatalf("unlink left the path (failed=%d hunks=%+v): %v", res.Failed, res.Hunks, err)
	}
	var fr FileResult
	for _, f := range res.Files {
		if f.Path == "gone.txt" {
			fr = f
		}
	}
	if !fr.Removed || !fr.Written {
		t.Fatalf("FileResult = %+v, want Removed and Written", fr)
	}
}

func TestUnlinkWithoutWholeFileReadIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "gone.txt", abcde)
	sha := shaOfFile(t, root, "gone.txt")

	res, err := Apply(root, []Input{
		{Path: "gone.txt", Op: "unlink", Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{
		"gone.txt": {SHA: sha, Spans: [][2]int{{1, 1}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("partial read licensed unlink: failed=%d hunks=%+v", res.Failed, res.Hunks)
	}
	if !strings.Contains(res.Hunks[0].Reason, "takes no line address") {
		t.Fatalf("reason = %q, want path-op unread wording", res.Hunks[0].Reason)
	}
	if strings.Contains(strings.ToLower(res.Hunks[0].Reason), "line address means nothing") {
		t.Fatalf("reason = %q, unlink has no line address", res.Hunks[0].Reason)
	}
	if read(t, root, "gone.txt") != abcde {
		t.Fatal("a refused unlink wrote the tree")
	}
}

func TestUnlinkOfAnUnreadFileDoesNotTalkAboutALineAddress(t *testing.T) {
	root := t.TempDir()
	write(t, root, "gone.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "gone.txt", Op: "unlink", Lines: -1, Index: 0},
	}, Options{Seen: map[string]Seen{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("unread unlink applied: failed=%d hunks=%+v", res.Failed, res.Hunks)
	}
	reason := res.Hunks[0].Reason
	if !strings.Contains(reason, "takes no line address") {
		t.Fatalf("reason = %q, want path-op unread wording", reason)
	}
	if strings.Contains(strings.ToLower(reason), "line address means nothing") {
		t.Fatalf("reason = %q, unlink has no line address", reason)
	}
	if read(t, root, "gone.txt") != abcde {
		t.Fatal("a refused unlink wrote the tree")
	}
}

func TestUnlinkRestoresWhenASiblingFails(t *testing.T) {
	root := t.TempDir()
	const orig = "stay\n"
	write(t, root, "a.txt", orig)
	write(t, root, "b.txt", abcde)

	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "unlink", Lines: -1, Index: 0},
		{Path: "b.txt", Start: 1, End: 1, Op: "replace", Body: []string{"X"}, Lines: -1, Index: 1},
	}, Options{Seen: map[string]Seen{
		"a.txt": {SHA: shaOfFile(t, root, "a.txt")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("sibling fail applied")
	}
	if got := read(t, root, "a.txt"); got != orig {
		t.Fatalf("unlink committed despite sibling fail: %q", got)
	}
	var unlink, replace HunkResult
	for _, h := range res.Hunks {
		switch h.Path {
		case "a.txt":
			unlink = h
		case "b.txt":
			replace = h
		}
	}
	if replace.Status != StatusFailed {
		t.Fatalf("unread replace status %s, want failed", replace.Status)
	}
	if unlink.Status != StatusSkipped {
		t.Fatalf("unlink status %s want skipped (reason %q)", unlink.Status, unlink.Reason)
	}
}

func TestRenameMovesThePath(t *testing.T) {
	root := t.TempDir()
	const body = "payload\n"
	write(t, root, "old.txt", body)

	res, err := Apply(root, []Input{
		{Path: "old.txt", Op: "rename", Body: []string{"dest.txt"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("rename left the source (failed=%d hunks=%+v): %v", res.Failed, res.Hunks, err)
	}
	if got := read(t, root, "dest.txt"); got != body {
		t.Fatalf("dest = %q, want %q", got, body)
	}
	var src FileResult
	for _, f := range res.Files {
		if f.Path == "old.txt" {
			src = f
		}
	}
	if !src.Removed || src.RenamedTo != "dest.txt" {
		t.Fatalf("source FileResult = %+v, want Removed and RenamedTo dest.txt", src)
	}
}

func TestRenameOntoExistingDestIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "old.txt", "src\n")
	write(t, root, "dest.txt", "stay\n")

	res, err := Apply(root, []Input{
		{Path: "old.txt", Op: "rename", Body: []string{"dest.txt"}, Lines: -1, Index: 0},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("rename onto existing dest applied: failed=%d hunks=%+v", res.Failed, res.Hunks)
	}
	if !strings.Contains(res.Hunks[0].Reason, "already exists") {
		t.Fatalf("reason = %q, want already exists", res.Hunks[0].Reason)
	}
	if read(t, root, "old.txt") != "src\n" || read(t, root, "dest.txt") != "stay\n" {
		t.Fatal("a refused rename wrote the tree")
	}
}
