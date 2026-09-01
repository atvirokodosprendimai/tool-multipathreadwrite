package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A plan file is a SHELL argument, not a path inside the tree being edited, so
// it resolves against the working directory while `-C` moves only the paths
// INSIDE the plan. That split is defensible and it is invisible: `-C repo write
// plan.mrw` fails with a bare "no such file or directory" naming a path the
// caller believes they are standing in.
//
// The refusal has to say which directory it looked in, and that -C did not
// apply — otherwise the caller's next move is to doubt the plan, not the path.
func TestAMissingPlanFileSaysWhereItLooked(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "plan.mrw"), []byte("@@ a.go 1 delete\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stand somewhere the plan is NOT, and point --root at where it is.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(elsewhere); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &buf, &buf

	err = cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--dry-run", "plan.mrw"})
	if err == nil {
		t.Fatal("a plan file that is not in the working directory was somehow read")
	}
	msg := err.Error() + buf.String()
	if !strings.Contains(msg, elsewhere) {
		t.Errorf("the error does not name the directory it looked in (%s):\n%s", elsewhere, msg)
	}
	if !strings.Contains(msg, "--root") && !strings.Contains(msg, "-C") {
		t.Errorf("the error does not say that -C does not apply to the plan file:\n%s", msg)
	}
}

// And the ordinary case still works: a plan beside the caller, editing a tree
// somewhere else entirely.
func TestAPlanIsReadRelativeToTheWorkingDirectory(t *testing.T) {
	// The ledger lives outside the tree; point it somewhere disposable, and use
	// --force so this test is about the PLAN PATH and nothing else.
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	here := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(here, "plan.mrw"), []byte("@@ a.txt 1 replace lines=1\nONE\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(here); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &buf, &buf

	// Not a dry run: the file at --root is the evidence that the plan beside the
	// caller was read AND that its paths resolved into the tree, which is the
	// whole of the split this pair of tests documents. mrw prints its receipt on
	// stdout rather than through cmd.Writer, so the tree is the thing to assert.
	if err := cmd.Run(context.Background(), []string{"mrw", "-C", root, "write", "--force", "plan.mrw"}); err != nil {
		t.Fatalf("a plan in the working directory was not read: %v\n%s", err, buf.String())
	}
	got, err := os.ReadFile(filepath.Join(root, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ONE\n" {
		t.Errorf("the plan did not edit the tree at --root: %q", got)
	}
}

// The explanation WRAPS the original, so a caller can still ask what went
// wrong rather than matching on prose. Flagged in review: every other failure
// in main.go is an error, and this one was a string only because cli.Exit
// happened to accept both.
func TestThePlanErrorWrapsTheCause(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got := planOpenError("plan.mrw", filepath.Join(wd, "elsewhere"), fs.ErrNotExist)

	if !errors.Is(got, fs.ErrNotExist) {
		t.Errorf("the cause was flattened into prose: %v", got)
	}
	if !strings.Contains(got.Error(), "working directory") {
		t.Errorf("the explanation was lost: %v", got)
	}
	// And the branches that have nothing to explain hand the cause straight
	// back, so an ordinary `mrw write plan.mrw` gains no paragraph.
	if plain := planOpenError("/abs/plan.mrw", wd, fs.ErrNotExist); plain != fs.ErrNotExist {
		t.Errorf("an absolute path was given an explanation it does not need: %v", plain)
	}
}
