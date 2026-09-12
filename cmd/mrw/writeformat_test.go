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

const twoHunkSearchReplace = "a.go\n" +
	"<<<<<<< SEARCH\n" +
	"func A() int { return 1 }\n" +
	"=======\n" +
	"func A() int { return 10 }\n" +
	">>>>>>> REPLACE\n" +
	"<<<<<<< SEARCH\n" +
	"func C() int { return 3 }\n" +
	"=======\n" +
	"func C() int { return 30 }\n" +
	">>>>>>> REPLACE\n"

func TestWriteFormatSearchReplaceUnreadWritesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "p.sr")
	if err := os.WriteFile(patch, []byte(twoHunkSearchReplace), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := readIn(t, root, "a.go:3"); err != nil {
		t.Fatalf("read a.go:3: %v", err)
	}

	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--format=search_replace", patch})
	})
	if got := exitCode(err); got != exitNotApplied {
		t.Fatalf("want exit 1, got %d (%v)\n%s", got, err, out)
	}
	if !strings.Contains(out, "FAIL") || !strings.Contains(out, "skip") || !strings.Contains(out, "has not been read") {
		t.Errorf("not FAIL+skip ledger refusal:\n%s", out)
	}
	got, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != demoFile {
		t.Errorf("the file was written:\n%s", got)
	}
}

func TestWriteFormatApplyPatchServedWritesBoth(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "p.apply")
	if err := os.WriteFile(patch, []byte(twoHunkApplyPatch), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatalf("read a.go: %v", err)
	}
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--format=apply_patch", patch})
	})
	if err != nil {
		t.Fatalf("want success, got %v\n%s", err, out)
	}
	body, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "return 10") || !strings.Contains(string(body), "return 30") {
		t.Errorf("served apply_patch missed a replacement:\n%s", body)
	}
}

func TestWriteFormatApplyPatchNoFlagIsUsage(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "p.apply")
	if err := os.WriteFile(patch, []byte(twoHunkApplyPatch), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", patch})
	})
	if got := exitCode(err); got != exitUsage {
		t.Fatalf("want exit 2, got %d (%v)\n%s", got, err, out)
	}
}

func TestWriteFormatGitIsUsage(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	patch := filepath.Join(root, "p.apply")
	if err := os.WriteFile(patch, []byte(twoHunkApplyPatch), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--format=git", patch})
	})
	if got := exitCode(err); got != exitUsage {
		t.Fatalf("want exit 2, got %d (%v)\n%s", got, err, out)
	}
}

func TestWriteFormatApplyPatchCompileRefusalIsUsage(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "p.apply")
	if err := os.WriteFile(patch, []byte("not a patch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--format=apply_patch", patch})
	})
	if got := exitCode(err); got != exitUsage {
		t.Fatalf("want exit 2, got %d (%v)\n%s", got, err, out)
	}
}

func TestWriteFormatApplyPatchForceWritesUnread(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "p.apply")
	if err := os.WriteFile(patch, []byte(twoHunkApplyPatch), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--force", "--format=apply_patch", patch})
	})
	if err != nil {
		t.Fatalf("want success, got %v\n%s", err, out)
	}
	body, err := os.ReadFile(filepath.Join(root, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "return 10") || !strings.Contains(string(body), "return 30") {
		t.Errorf("--force did not write compiled hunks:\n%s", body)
	}
}
