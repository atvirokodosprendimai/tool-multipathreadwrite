package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-133. A read whose answer went to the null device was recorded like any
// other, so `mrw read f:2 >/dev/null` licensed a write to line 2 that nobody
// had seen. Sent there, a read must exit 0, say on stderr that nothing was
// recorded, and leave the file out of the ledger; sent to a file, the same
// read records as before.
func TestAReadSentToTheNullDeviceLicensesNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func(stdout *os.File) (error, string) {
		t.Helper()
		stderr, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = stderr.Close() }()
		savedOut, savedErr := os.Stdout, os.Stderr
		os.Stdout, os.Stderr = stdout, stderr
		defer func() { os.Stdout, os.Stderr = savedOut, savedErr }()
		cmd := rootCommand()
		var errBuf bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &errBuf, &errBuf
		runErr := cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "f.txt:2"})
		said, err := os.ReadFile(stderr.Name())
		if err != nil {
			t.Fatal(err)
		}
		return runErr, string(said)
	}
	recorded := func() bool {
		t.Helper()
		l, err := seen.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := l["f.txt"]
		return ok
	}

	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	err, said := read(null)
	if err != nil {
		t.Fatalf("a read to the null device failed: %v", err)
	}
	if recorded() {
		t.Fatal("a read whose answer went to the null device licensed its lines")
	}
	if !strings.Contains(said, "null device") || !strings.Contains(said, "nothing was recorded") {
		t.Errorf("stderr does not say the answer went to the null device and nothing was recorded: %q", said)
	}

	sink, err := os.Create(filepath.Join(t.TempDir(), "answer"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sink.Close() }()
	err, said = read(sink)
	if err != nil {
		t.Fatalf("a read to a file failed: %v", err)
	}
	if !recorded() {
		t.Fatal("a read to a file recorded nothing")
	}
	if said != "" {
		t.Errorf("a read to a file wrote to stderr: %q", said)
	}
}
