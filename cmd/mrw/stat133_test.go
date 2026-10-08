package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// ADR-133. The null-device line says "a write to these lines": a read that
// served no lines — a --stat — has none, and printed it anyway (the
// quality-blueprints stress run of 25f4f04). Sent to the null device, a
// --stat must exit 0 and say nothing on stderr.
func TestAStatSentToTheNullDeviceSaysNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	stderr, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stderr.Close() }()
	savedOut, savedErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = null, stderr
	cmd := rootCommand()
	var errBuf bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &errBuf, &errBuf
	runErr := cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "--stat", "f.txt"})
	os.Stdout, os.Stderr = savedOut, savedErr
	if runErr != nil {
		t.Fatalf("a --stat to the null device failed: %v", runErr)
	}
	said, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 0 {
		t.Errorf("a --stat to the null device, which served no lines, wrote to stderr: %q", said)
	}
}
