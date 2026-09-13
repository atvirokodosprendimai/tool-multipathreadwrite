package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateThenRenameOntoThatPathIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "old.txt", "src\n")
	res, err := Apply(root, []Input{
		{Path: "new.txt", Op: "create", Body: []string{"created"}, Index: 0},
		{Path: "old.txt", Op: "rename", Body: []string{"new.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("create plus rename onto the create dest applied")
	}
	if exists(t, root, "new.txt") {
		t.Fatal("create dest was written despite the refusal")
	}
	if read(t, root, "old.txt") != "src\n" {
		t.Fatal("source was moved despite the refusal")
	}
}

func TestTwoRenamesOntoTheSameDestAreRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "aaa\n")
	write(t, root, "b.txt", "bbb\n")
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, Index: 0},
		{Path: "b.txt", Op: "rename", Body: []string{"c.txt"}, Lines: -1, Index: 1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("two renames onto one dest applied")
	}
	if exists(t, root, "c.txt") {
		t.Fatal("shared dest was written")
	}
}

func TestRenameOntoDanglingSymlinkIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "old.txt", "src\n")
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(root, "dest.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	res, err := Apply(root, []Input{
		{Path: "old.txt", Op: "rename", Body: []string{"dest.txt"}, Lines: -1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("rename onto a dangling symlink applied")
	}
	if read(t, root, "old.txt") != "src\n" {
		t.Fatal("source moved onto a dangling symlink")
	}
}

func TestRenameDestUnderOutboundSymlinkIsRefused(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	outside := filepath.Join(outer, "out")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	write(t, root, "old.txt", "src\n")
	res, err := Apply(root, []Input{
		{Path: "old.txt", Op: "rename", Body: []string{"link/new.txt"}, Lines: -1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("rename through an outbound symlink applied")
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); err == nil {
		t.Fatal("file appeared outside the root")
	}
}

func TestOverlongSHADoesNotPanicOnUnlink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "x\n")
	res, err := Apply(root, []Input{
		{Path: "a.txt", Op: "unlink", SHA: strings.Repeat("a", 80), Lines: -1},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied {
		t.Fatal("unlink with an overlong sha applied")
	}
	if !exists(t, root, "a.txt") {
		t.Fatal("overlong sha still unlinked the path")
	}
}
