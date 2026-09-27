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

// ADR-088 T4. The read action recorded what it served, then flushed its
// buffered answer on return with the error unchecked, so a read whose output
// could not be written licensed a write to lines nobody saw. A stdout that
// refuses writes — a pipe whose reader is gone, standing in for a full disk —
// must exit 2 and leave the ledger without the file; a working stdout records
// as before.
func TestAReadWhoseAnswerCannotBeWrittenRecordsNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read := func(stdout *os.File) error {
		saved := os.Stdout
		os.Stdout = stdout
		defer func() { os.Stdout = saved }()
		cmd := rootCommand()
		var errBuf bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &errBuf, &errBuf
		return cmd.Run(context.Background(), []string{"mrw", "-C", root, "read", "f.txt"})
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

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	err = read(w)
	_ = w.Close()
	if err == nil {
		t.Fatal("a read whose answer could not be written exited 0")
	}
	if got := exitCode(err); got != exitUsage {
		t.Fatalf("a read whose answer could not be written exited %d (%v), want %d", got, err, exitUsage)
	}
	if !strings.Contains(err.Error(), "nothing was recorded") {
		t.Errorf("the refusal does not say nothing was recorded: %v", err)
	}
	if recorded() {
		t.Fatal("a read whose answer never reached the caller licensed its lines")
	}

	sink, err := os.Create(filepath.Join(t.TempDir(), "answer"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sink.Close() }()
	if err := read(sink); err != nil {
		t.Fatalf("a read to a working stdout failed: %v", err)
	}
	if !recorded() {
		t.Fatal("a read that reached its caller recorded nothing")
	}
}
