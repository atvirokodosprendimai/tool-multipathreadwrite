package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var padLine = regexp.MustCompile(`^\s+\d+\|`)

func countPadLines(out string) int {
	n := 0
	for _, line := range strings.Split(out, "\n") {
		if padLine.MatchString(line) {
			n++
		}
	}
	return n
}

// TestWriteEchoPadShowsTheLineAfterTheBody is ADR-052 T2: --echo-pad 1
// prints the line after the body on an ok hunk.
func TestWriteEchoPadShowsTheLineAfterTheBody(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("1\n2\n3\n</div>\n5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(root, "plan.mrw")
	if err := os.WriteFile(plan, []byte("@@ f.txt 2-3 replace anchor=\"2\"\nX\nY\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{
			"mrw", "-C", root, "write", "--force", "--echo-pad", "1", plan,
		})
	})
	if err != nil {
		t.Fatalf("write --echo-pad 1 failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("receipt is not ok:\n%s", out)
	}
	if strings.Contains(out, "FAIL") {
		t.Errorf("a closer in the pad failed the hunk:\n%s", out)
	}
	if !strings.Contains(out, "</div>") {
		t.Errorf("pad does not show the closer:\n%s", out)
	}
	if got := countPadLines(out); got != 1 {
		t.Errorf("pad lines = %d, want 1\n%s", got, out)
	}
}

// Default --echo-pad is 0: the receipt prints no NNN| lines after the hunk.
func TestWriteEchoPadDefaultPrintsNoLines(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("1\n2\n3\n</div>\n5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(root, "plan.mrw")
	if err := os.WriteFile(plan, []byte("@@ f.txt 2-3 replace anchor=\"2\"\nX\nY\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{
			"mrw", "-C", root, "write", "--force", plan,
		})
	})
	if err != nil {
		t.Fatalf("write failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "</div>") {
		t.Errorf("default 0 printed a pad:\n%s", out)
	}
	if got := countPadLines(out); got != 0 {
		t.Errorf("pad lines = %d, want 0\n%s", got, out)
	}
}

// --echo-pad N prints exactly N numbered lines after the hunk.
func TestWriteEchoPadNPrintsNNumberedLines(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("1\n2\n3\na\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(root, "plan.mrw")
	if err := os.WriteFile(plan, []byte("@@ f.txt 2-3 replace anchor=\"2\"\nX\nY\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{
			"mrw", "-C", root, "write", "--force", "--echo-pad", "3", plan,
		})
	})
	if err != nil {
		t.Fatalf("write --echo-pad 3 failed: %v\n%s", err, out)
	}
	if got := countPadLines(out); got != 3 {
		t.Errorf("pad lines = %d, want 3\n%s", got, out)
	}
	if !strings.Contains(out, "4|") || !strings.Contains(out, "5|") || !strings.Contains(out, "6|") {
		t.Errorf("pad is not three numbered lines:\n%s", out)
	}
}

// A pad that runs past the last line is clamped; the hunk stays ok.
func TestWriteEchoPadClampsAtEOF(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("1\n2\n3\n4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(root, "plan.mrw")
	if err := os.WriteFile(plan, []byte("@@ f.txt 2-3 replace anchor=\"2\"\nX\nY\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{
			"mrw", "-C", root, "write", "--force", "--echo-pad", "10", plan,
		})
	})
	if err != nil {
		t.Fatalf("write --echo-pad 10 failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("clamped pad is not ok:\n%s", out)
	}
	if got := countPadLines(out); got != 1 {
		t.Errorf("clamped pad lines = %d, want 1\n%s", got, out)
	}
}
