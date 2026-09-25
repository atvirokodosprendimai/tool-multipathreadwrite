package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-074 T4. A --files-from line over the scanner's 8 MiB limit died with
// "bufio.Scanner: token too long" and no line number, so the caller could not
// tell which spec in a generated list to fix.
func TestAFilesFromLineOver8MiBNamesTheLine(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	list := filepath.Join(t.TempDir(), "list")
	body := "a.go\n# a comment\n" + strings.Repeat("x", 9<<20) + "\n"
	if err := os.WriteFile(list, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var sink bytes.Buffer
	cmd := rootCommand()
	cmd.Writer, cmd.ErrWriter = &sink, &sink
	err := cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "--files-from", list})
	msg := errString(err) + sink.String()
	if err == nil {
		t.Fatalf("an over-long spec line was accepted:\n%.300s", msg)
	}
	if !strings.Contains(msg, "line 3") || !strings.Contains(msg, "8 MiB") {
		t.Fatalf("the error does not name line 3 and the 8 MiB limit:\n%.300s", msg)
	}
}
