package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const demoFile = "package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n"

const twoHunkApplyPatch = "*** Begin Patch\n" +
	"*** Update File: a.go\n" +
	"@@\n" +
	"-func A() int { return 1 }\n" +
	"+func A() int { return 10 }\n" +
	"@@\n" +
	"-func C() int { return 3 }\n" +
	"+func C() int { return 30 }\n" +
	"*** End Patch\n"

// TestWriteFormatApplyPatchUnreadWritesNothing is ADR-051 T2: the CLI flag
// compiles, Apply refuses the unread sibling, and the file is unchanged.
func TestWriteFormatApplyPatchUnreadWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "p.apply")
	if err := os.WriteFile(patch, []byte(twoHunkApplyPatch), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := readIn(t, root, "a.go:3"); err != nil {
		t.Fatalf("read a.go:3: %v", err)
	}

	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--format=apply_patch", patch})
	})
	if got := exitCode(err); got != exitNotApplied {
		t.Fatalf("want exit 1, got %d (%v)\n%s", got, err, out)
	}
	if !strings.Contains(out, "FAIL") {
		t.Errorf("no FAIL:\n%s", out)
	}
	if !strings.Contains(out, "skip") {
		t.Errorf("no skip:\n%s", out)
	}
	if !strings.Contains(out, "has not been read") {
		t.Errorf("not the ledger refusal:\n%s", out)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != demoFile {
		t.Errorf("the file was written:\n%s", got)
	}
}
