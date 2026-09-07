package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pruneBase is one state base holding every class of entry the prune must tell
// apart, built by driving the real command — so the `root` markers under test
// are the ones mrw itself writes.
type pruneBase struct {
	mrw     string // <XDG_STATE_HOME>/mrw
	live    string // entry whose checkout still exists
	dead    string // entry whose checkout was removed
	self    string // entry for the root the command runs in
	selfGo  string
	unknown string // a directory mrw did not create and cannot identify
}

// plantBase is the CLI-level sibling of internal/state's fixture.
//
// ⚠ ONE BASE, THREE OUTCOMES. A base holding only a dead entry passes against a
// command that deletes everything under it — the parent record's High-likelihood
// risk — so every assertion here names something that must go AND something
// that must stay.
func plantBase(t *testing.T) pruneBase {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	b := pruneBase{mrw: filepath.Join(base, "mrw")}

	dirFor := func(root string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		// A real read, so the entry holds a ledger and not only a marker.
		if _, out := runMrw(t, root, "read", "f.txt"); out == "" {
			t.Fatalf("read produced nothing for %q", root)
		}
		items, err := os.ReadDir(b.mrw)
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			d := filepath.Join(b.mrw, it.Name())
			marker, err := os.ReadFile(filepath.Join(d, "root"))
			if err != nil {
				continue
			}
			if real, err := filepath.EvalSymlinks(root); err == nil &&
				strings.TrimSpace(string(marker)) == real {
				return d
			}
		}
		t.Fatalf("no state entry names %q", root)
		return ""
	}

	liveGo, deadGo := t.TempDir(), t.TempDir()
	b.selfGo = t.TempDir()
	b.live = dirFor(liveGo)
	b.dead = dirFor(deadGo)
	b.self = dirFor(b.selfGo)

	if err := os.RemoveAll(deadGo); err != nil {
		t.Fatal(err)
	}
	b.unknown = filepath.Join(b.mrw, "cccccccccccccccc")
	if err := os.MkdirAll(b.unknown, 0o700); err != nil {
		t.Fatal(err)
	}
	return b
}

// runMrw drives the real command in-process and returns its error and whatever
// it wrote to stdout. The commands under test print with fmt, not through
// cli.Command's writer, so stdout is captured rather than the writer swapped —
// a test that read the writer would see nothing and pass on an empty string.
func runMrw(t *testing.T, root string, args ...string) (error, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	cmd := rootCommand()
	var errBuf bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &errBuf, &errBuf
	runErr := cmd.Run(context.Background(), append([]string{"mrw", "-C", root}, args...))

	os.Stdout = saved
	_ = w.Close()
	out := <-done
	_ = r.Close()
	return runErr, out + errBuf.String()
}

func onDisk(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

func TestSeenPruneRemovesOnlyTheDeadEntries(t *testing.T) {
	b := plantBase(t)

	err, out := runMrw(t, b.selfGo, "seen", "--prune")
	if err != nil {
		t.Fatalf("seen --prune: %v\n%s", err, out)
	}

	// ⚠ THE FILESYSTEM IS THE ASSERTION. A command that prints the right
	// sentence and removes nothing passes an output check, which is the way
	// this test would be written wrong.
	if onDisk(t, b.dead) {
		t.Errorf("the dead entry %q survived --prune\n%s", b.dead, out)
	}
	for what, dir := range map[string]string{
		"the live entry":           b.live,
		"the running root's entry": b.self,
		"the unidentifiable entry": b.unknown,
	} {
		if !onDisk(t, dir) {
			t.Errorf("--prune removed %s (%q); only a dead, identified entry may go\n%s", what, dir, out)
		}
	}

	// ADR-008: a delete says what it removed. Both halves — the directory and
	// the checkout it belonged to, which is the only thing that makes a
	// 16-hex-character name mean anything to a reader.
	if !strings.Contains(out, b.dead) {
		t.Errorf("the report does not name the directory it removed:\n%s", out)
	}
	if !strings.Contains(out, "1 kept") {
		t.Errorf("the report does not say the unidentifiable entry was kept:\n%s", out)
	}
}

func TestADryRunPruneRemovesNothing(t *testing.T) {
	b := plantBase(t)

	err, out := runMrw(t, b.selfGo, "seen", "--prune", "--dry-run")
	if err != nil {
		t.Fatalf("seen --prune --dry-run: %v\n%s", err, out)
	}
	for _, dir := range []string{b.dead, b.live, b.self, b.unknown} {
		if !onDisk(t, dir) {
			t.Errorf("a dry run removed %q\n%s", dir, out)
		}
	}
	if !strings.Contains(out, b.dead) {
		t.Errorf("a dry run did not name the entry it would remove:\n%s", out)
	}

	// --dry-run alone is a dry run of the thing that only reads, which is not a
	// request anyone can mean.
	err, out = runMrw(t, b.selfGo, "seen", "--dry-run")
	if err == nil {
		t.Fatalf("seen --dry-run was accepted without --prune\n%s", out)
	}
	if got := exitCode(err); got != exitUsage {
		t.Errorf("seen --dry-run exited %d, want %d — the fix is an argument", got, exitUsage)
	}
}

func TestTheStateDirectoryIsStillTheFirstLine(t *testing.T) {
	b := plantBase(t)

	err, out := runMrw(t, b.selfGo, "seen")
	if err != nil {
		t.Fatalf("seen: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("seen printed %d line(s), want the directory and a count:\n%s", len(lines), out)
	}
	// The contract `scripts/contract.sh` reads with `head -1`, and the reason
	// the count line goes second rather than first.
	if lines[0] != b.self {
		t.Errorf("line 1 is %q, want the state directory %q — `mrw seen | head -1` is documented",
			lines[0], b.self)
	}
	if !onDisk(t, lines[0]) {
		t.Errorf("line 1 %q is not a directory that exists", lines[0])
	}
	if !strings.HasPrefix(lines[1], "# 4 state directories under ") {
		t.Errorf("line 2 is %q, want the count of the 4 entries in the base", lines[1])
	}
	if !strings.Contains(out, "--prune") {
		t.Errorf("nothing in the output points at --prune, so the growth is visible and the remedy is not:\n%s", out)
	}
}
