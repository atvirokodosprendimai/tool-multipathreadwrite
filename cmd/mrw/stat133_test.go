package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-133. The null-device line says "a write to these lines", so it follows
// a read that served lines and only that: a --stat, a --max-lines 0 and a
// range past the end serve none and printed it anyway (the peer stress runs of
// 25f4f04), while a whole-file read serves every line and must still print it
// — its observation carries nil Spans, meaning the whole file (the Codex
// review of #351).
func TestTheNullDeviceLineFollowsOnlyServedLines(t *testing.T) {
	cases := []struct {
		args []string
		line bool
	}{
		{[]string{"--stat", "f.txt"}, false},
		{[]string{"--max-lines", "0", "f.txt"}, false},
		{[]string{"f.txt:9"}, false},
		{[]string{"f.txt"}, true},
		{[]string{"f.txt:2"}, true},
	}
	for _, c := range cases {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		stderr, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
		if err != nil {
			t.Fatal(err)
		}
		savedOut, savedErr := os.Stdout, os.Stderr
		os.Stdout, os.Stderr = null, stderr
		cmd := rootCommand()
		var errBuf bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &errBuf, &errBuf
		_ = cmd.Run(context.Background(), append([]string{"mrw", "-C", root, "read"}, c.args...))
		os.Stdout, os.Stderr = savedOut, savedErr
		_ = null.Close()
		_ = stderr.Close()
		said, err := os.ReadFile(stderr.Name())
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(string(said), "null device"); got != c.line {
			t.Errorf("read %v to the null device: the null-device line printed = %v, want %v (stderr %q)", c.args, got, c.line, said)
		}
	}
}
