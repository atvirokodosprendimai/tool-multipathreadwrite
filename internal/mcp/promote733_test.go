package mcp

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// promote is promoteAcks without the kept ids, the shape the older tests were
// written against.
func promote(root string, acks []string) ([]string, error) {
	unknown, _, err := promoteAcks(root, acks)
	return unknown, err
}

// Issue #733. A read that failed for a reason that says nothing about the file's
// version (a permission, a descriptor limit) dropped the span, yet the checkpoint
// was deleted from the pending store and the store saved: the acknowledgement was
// consumed, no licence was recorded and nothing said so. Now the id is kept,
// reported in the answer's note, and a retry once the file reads licenses it.
func TestAnAckIsKeptWhenTheFileCannotBeReadJustNow(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 file is still readable here")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	f := filepath.Join(root, "f.txt")
	if err := os.WriteFile(f, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, err := currentSHA(root, "f.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := hold(root, "f.txt", sha, map[string][2]int{"ck-1": {1, 2}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f, 0o000); err != nil {
		t.Fatal(err)
	}
	unknown, kept, perr := promoteAcks(root, []string{"ck-1"})
	if perr != nil || len(unknown) != 0 || len(kept) != 1 || kept[0].id != "ck-1" {
		t.Fatalf("unreadable file: unknown=%v kept=%v err=%v; want ck-1 kept, no error", unknown, kept, perr)
	}
	if note := keptNote(kept); !strings.Contains(note, "ck-1") || !strings.Contains(note, "kept") {
		t.Errorf("the note does not name the kept id: %q", note)
	}
	if err := os.Chmod(f, 0o644); err != nil {
		t.Fatal(err)
	}
	unknown, kept, perr = promoteAcks(root, []string{"ck-1"})
	if perr != nil || len(unknown) != 0 || len(kept) != 0 {
		t.Fatalf("retry: unknown=%v kept=%v err=%v; want the id consumed", unknown, kept, perr)
	}
	snap, _ := seen.Snapshot(root)
	if _, ok := snap["f.txt"]; !ok {
		t.Error("the retry did not license f.txt")
	}
}

// The pair: a version known to be gone (the file changed, or went away) still
// consumes the id, as before, and is not reported as kept.
func TestAnAckForAChangedOrRemovedFileIsStillConsumed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for name, mutate := range map[string]func(string){
		"changed": func(f string) { _ = os.WriteFile(f, []byte("changed\n"), 0o644) },
		"removed": func(f string) { _ = os.Remove(f) },
	} {
		root := t.TempDir()
		f := filepath.Join(root, "f.txt")
		if err := os.WriteFile(f, []byte("a\nb\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sha, _ := currentSHA(root, "f.txt")
		if err := hold(root, "f.txt", sha, map[string][2]int{"ck-" + name: {1, 2}}); err != nil {
			t.Fatal(err)
		}
		mutate(f)
		unknown, kept, err := promoteAcks(root, []string{"ck-" + name})
		if err != nil || len(unknown) != 0 || len(kept) != 0 {
			t.Errorf("%s: unknown=%v kept=%v err=%v; want consumed", name, unknown, kept, err)
		}
		if again, _, _ := promoteAcks(root, []string{"ck-" + name}); len(again) != 1 {
			t.Errorf("%s: the id was not consumed: second promote unknown=%v", name, again)
		}
	}
}

// Review of #736. An id is kept only for a failure known to be transient. A path
// whose parent is a file now (ENOTDIR), or whose file became a directory or
// another thing that is not a regular file, is a fact about the version: the id is
// consumed, not kept for ever behind a "send them again" note.
func TestAnAckForAVersionThatIsGoneByAnyOtherErrorIsConsumed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for name, swap := range map[string]func(root string){
		"a parent that is a file now (ENOTDIR)": func(root string) {
			_ = os.RemoveAll(filepath.Join(root, "d"))
			_ = os.WriteFile(filepath.Join(root, "d"), []byte("now a file\n"), 0o644)
		},
		"a file that is a directory now": func(root string) {
			_ = os.Remove(filepath.Join(root, "d", "f.txt"))
			_ = os.Mkdir(filepath.Join(root, "d", "f.txt"), 0o755)
		},
	} {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "d", "f.txt"), []byte("a\nb\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sha, err := currentSHA(root, "d/f.txt")
		if err != nil {
			t.Fatal(err)
		}
		if err := hold(root, "d/f.txt", sha, map[string][2]int{"ck": {1, 2}}); err != nil {
			t.Fatal(err)
		}
		swap(root)
		unknown, kept, err := promoteAcks(root, []string{"ck"})
		if err != nil || len(unknown) != 0 || len(kept) != 0 {
			t.Errorf("%s: unknown=%v kept=%v err=%v; want the id consumed", name, unknown, kept, err)
		}
		if again, _, _ := promoteAcks(root, []string{"ck"}); len(again) != 1 {
			t.Errorf("%s: the id is still pending: second promote unknown=%v", name, again)
		}
	}
}

// Review of #736. The kept note names each kept id with its own cause, and the
// paths in it are the checkout's own, never the absolute ones.
func TestTheKeptNoteNamesEachIdWithItsOwnCauseAndRelativePaths(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 file is still readable here")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, n := range []string{"a.txt", "b.txt"} {
		f := filepath.Join(root, n)
		if err := os.WriteFile(f, []byte("x\ny\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		sha, _ := currentSHA(root, n)
		if err := hold(root, n, sha, map[string][2]int{"ck-" + n: {1, 2}}); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(f, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	}
	_, kept, err := promoteAcks(root, []string{"ck-a.txt", "ck-b.txt"})
	if err != nil || len(kept) != 2 {
		t.Fatalf("kept=%v err=%v, want both ids kept", kept, err)
	}
	note := keptNote(kept)
	for _, want := range []string{"ck-a.txt (open a.txt: permission denied)", "ck-b.txt (open b.txt: permission denied)"} {
		if !strings.Contains(note, want) {
			t.Errorf("the note lacks %q:\n%s", want, note)
		}
	}
	if strings.Contains(note, root) {
		t.Errorf("the note carries the absolute root %q:\n%s", root, note)
	}
}

// Review of #736. The kept note reaches the answer of the call that sent the ack.
func TestTheKeptNoteReachesTheMCPAnswer(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 file is still readable here")
	}
	root := licensed(t, map[string]string{"a.txt": "a\n", "b.txt": "b\n"})
	sha, err := currentSHA(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := hold(root, "a.txt", sha, map[string][2]int{"ck-kept": {1, 1}}); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(root, "a.txt")
	if err := os.Chmod(f, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	res := call(t, root, "mrw_read", map[string]any{"specs": []string{"b.txt"}, "ack": []string{"ck-kept"}})
	content, _ := res["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("no content: %v", res)
	}
	first, _ := content[0].(map[string]any)
	text := asString(first["text"])
	if !strings.Contains(text, "-- ack: 1 id(s) were kept") || !strings.Contains(text, "ck-kept") {
		t.Errorf("the answer does not carry the kept note:\n%s", text)
	}
}

// Review of #736 (Windows). Win32 returns ERROR_TOO_MANY_OPEN_FILES as errno 4,
// which syscall.EMFILE does not match there; it and the other native codes that
// clear on their own are transient on Windows, and mean other things elsewhere.
// The classifier is pure, so this runs on every OS.
func TestTheWindowsNativeCodesThatClearOnTheirOwnAreTransient(t *testing.T) {
	open := func(e error) error { return &fs.PathError{Op: "open", Path: "f.txt", Err: e} }
	for _, tc := range []struct {
		goos string
		err  error
		want bool
	}{
		{"windows", open(syscall.Errno(4)), true},  // ERROR_TOO_MANY_OPEN_FILES
		{"windows", open(syscall.Errno(8)), true},  // ERROR_NOT_ENOUGH_MEMORY
		{"windows", open(syscall.Errno(32)), true}, // ERROR_SHARING_VIOLATION
		{"windows", open(syscall.Errno(33)), true}, // ERROR_LOCK_VIOLATION
		{"windows", open(os.ErrPermission), true},
		{"windows", open(syscall.Errno(3)), false}, // ERROR_PATH_NOT_FOUND: the version is gone
		{"windows", open(os.ErrNotExist), false},
		{"linux", open(syscall.Errno(8)), false},  // ENOEXEC
		{"linux", open(syscall.Errno(32)), false}, // EPIPE
		{"linux", open(syscall.EMFILE), true},
		{"linux", open(syscall.ENOTDIR), false},
	} {
		if got := transientReadOn(tc.goos, tc.err); got != tc.want {
			t.Errorf("%s: transientReadOn(%v) = %v, want %v", tc.goos, tc.err, got, tc.want)
		}
	}
}

// Review of #736. With the root spelled with a trailing slash the open error's
// path is cleaned, so stripping "root/" never matched and the note carried the
// absolute path. The note uses the checkpoint's own stored relative path.
func TestTheKeptNoteNamesTheStoredPathWhateverWayTheRootIsSpelled(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 file is still readable here")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	f := filepath.Join(root, "a.txt")
	if err := os.WriteFile(f, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sha, _ := currentSHA(root, "a.txt")
	if err := hold(root, "a.txt", sha, map[string][2]int{"ck": {1, 1}}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(f, 0o644) })
	_, kept, err := promoteAcks(root+string(filepath.Separator), []string{"ck"})
	if err != nil || len(kept) != 1 {
		t.Fatalf("kept=%v err=%v", kept, err)
	}
	if note := keptNote(kept); !strings.Contains(note, "ck (open a.txt: permission denied)") || strings.Contains(note, root) {
		t.Errorf("the note does not carry the stored relative path alone:\n%s", note)
	}
}

// Review of #736. While the answer has room, every kept id is named, each with
// its cause; only the copy that goes in under the ceiling is cut to five.
func TestEveryKeptIdIsListedWhileThereIsRoom(t *testing.T) {
	var kept []keptAck
	for i := 1; i <= 6; i++ {
		kept = append(kept, keptAck{id: fmt.Sprintf("ck-%d", i), cause: fmt.Sprintf("open f%d.txt: permission denied", i)})
	}
	noteAcks(nil, kept)
	t.Cleanup(func() { callAckNote, callAckNoteShort = "", "" })
	if !strings.Contains(callAckNote, "ck-6 (open f6.txt: permission denied)") || strings.Contains(callAckNote, "more)") {
		t.Errorf("the whole note hides a kept id:\n%s", callAckNote)
	}
	if !strings.Contains(callAckNoteShort, "(and 1 more)") || strings.Contains(callAckNoteShort, "ck-6") {
		t.Errorf("the note for a tight ceiling is not the cut one:\n%s", callAckNoteShort)
	}
}
