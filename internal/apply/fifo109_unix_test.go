//go:build unix

package apply

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// ADR-109 T2. Validation judges a file by a path Stat, and then readLines
// opened it blocking: a file swapped for a FIFO in between hung the write.
// readLines asks the descriptor, and the file is refused on its hunk with every
// verdict kept, as ADR-107 refuses one that grew.
func TestApplyLoadRefusesAFIFOAtOnce(t *testing.T) {
	root := t.TempDir()
	pipe := filepath.Join(root, "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	release := func() {
		if w, err := os.OpenFile(pipe, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := readLines(pipe)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, regular.ErrNotRegular) {
			t.Errorf("readLines read a FIFO, or refused it with something else: %v", err)
		}
	case <-time.After(5 * time.Second):
		release()
		t.Fatal("readLines blocked on a FIFO")
	}

	write(t, root, "swap.txt", "s\n")
	write(t, root, "small.txt", "a\n")
	real := loadFn
	t.Cleanup(func() { loadFn = real })
	loadFn = func(path string) (text, bool, error) {
		if filepath.Base(path) == "swap.txt" {
			if err := os.Remove(path); err != nil {
				t.Error(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Error(err)
			}
		}
		return real(path)
	}
	type outcome struct {
		res Result
		err error
	}
	got := make(chan outcome, 1)
	go func() {
		res, err := Apply(root, []Input{
			{Path: "swap.txt", Start: 1, End: 1, Op: "replace", Body: []string{"S"}, Lines: -1, Index: 0},
			{Path: "small.txt", Start: 1, End: 1, Op: "replace", Body: []string{"b"}, Lines: -1, Index: 1},
		}, Options{Force: true})
		got <- outcome{res, err}
	}()
	select {
	case o := <-got:
		if s := hunkFor(t, o.res, "swap.txt"); o.err != nil || o.res.Applied || s.Status != StatusFailed || !strings.Contains(s.Reason, "not a regular file") {
			t.Errorf("a file swapped for a FIFO was not refused on its hunk: err %v, %+v", o.err, s)
		}
		if s := hunkFor(t, o.res, "small.txt"); s.Status != StatusSkipped || read(t, root, "small.txt") != "a\n" {
			t.Errorf("the sibling was not skipped and left alone: %+v", s)
		}
	case <-time.After(5 * time.Second):
		if w, err := os.OpenFile(filepath.Join(root, "swap.txt"), os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("Apply blocked on a file swapped for a FIFO")
	}

	// The Codex review of #306: a file swapped for a socket cannot be opened
	// at all, and is refused on its hunk all the same, verdicts kept.
	short, err := os.MkdirTemp("/tmp", "mrw")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(short) })
	write(t, short, "s.txt", "s\n")
	write(t, short, "o.txt", "o\n")
	var l net.Listener
	t.Cleanup(func() {
		if l != nil {
			_ = l.Close()
		}
	})
	loadFn = func(path string) (text, bool, error) {
		if filepath.Base(path) == "s.txt" {
			if err := os.Remove(path); err != nil {
				t.Error(err)
			}
			var lerr error
			if l, lerr = net.Listen("unix", path); lerr != nil {
				t.Skipf("no unix sockets here: %v", lerr)
			}
		}
		return real(path)
	}
	res, err := Apply(short, []Input{
		{Path: "s.txt", Start: 1, End: 1, Op: "replace", Body: []string{"S"}, Lines: -1, Index: 0},
		{Path: "o.txt", Start: 1, End: 1, Op: "replace", Body: []string{"O"}, Lines: -1, Index: 1},
	}, Options{Force: true})
	if s := hunkFor(t, res, "s.txt"); err != nil || res.Applied || s.Status != StatusFailed || !strings.Contains(s.Reason, "not a regular file") {
		t.Errorf("a file swapped for a socket was not refused on its hunk: err %v, %+v", err, s)
	}
	if s := hunkFor(t, res, "o.txt"); s.Status != StatusSkipped || read(t, short, "o.txt") != "o\n" {
		t.Errorf("the sibling was not skipped and left alone: %+v", s)
	}
}
