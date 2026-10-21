package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-166 T1. A Codex agent in a sandbox with a read-only home got exit 2 from
// a read that had served its content: the ledger could not be written. A read
// that cannot record now exits as the read itself would and says, last on
// stderr, that nothing was recorded; a read that needs the working set still
// refuses, and no licence exists afterwards. Through the built binary, since
// the exit code and the order of the streams are the subject.
func TestAReadWithAnUnwritableLedgerServesAndSaysNothingWasRecorded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode bits make the read-only directory; the code path is the same on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores mode bits")
	}
	bin := filepath.Join(t.TempDir(), "mrw.exe")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	tree := t.TempDir()
	if err := os.WriteFile(filepath.Join(tree, "f.txt"), []byte("alpha\nbravo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(state, stdin string, argv ...string) (stdout, stderr string, code int) {
		t.Helper()
		cmd := exec.Command(bin, argv...)
		cmd.Dir = tree
		cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		cmd.Stdin = strings.NewReader(stdin)
		var o, e bytes.Buffer
		cmd.Stdout, cmd.Stderr = &o, &e
		err := cmd.Run()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), code
	}
	lastLine := func(s string) string {
		s = strings.TrimRight(s, "\n")
		return s[strings.LastIndex(s, "\n")+1:]
	}

	for name, body := range map[string]string{"plain.list": "f.txt:1\n", "pointer.list": "@1\n"} {
		if err := os.WriteFile(filepath.Join(tree, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// (a) the state directory exists and is read-only, every level of it; (b) it
	// cannot be created. The setup read names another file, so it licenses
	// nothing a write below could use.
	if err := os.WriteFile(filepath.Join(tree, "g.txt"), []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	existing := t.TempDir()
	if _, _, code := run(existing, "", "read", "g.txt"); code != 0 {
		t.Fatalf("setting up the state directory: exit %d", code)
	}
	parent := t.TempDir()
	chmodAll := func(root string, mode os.FileMode) {
		_ = filepath.Walk(root, func(p string, info os.FileInfo, _ error) error {
			if info != nil && info.IsDir() {
				_ = os.Chmod(p, mode)
			}
			return nil
		})
	}
	chmodAll(existing, 0o555)
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		chmodAll(existing, 0o755)
		_ = os.Chmod(parent, 0o755)
	})
	for name, state := range map[string]string{"read-only": existing, "uncreatable": filepath.Join(parent, "state")} {
		out, errOut, code := run(state, "", "read", "f.txt:1")
		if code != 0 || !strings.Contains(out, "1| alpha") || !strings.HasPrefix(lastLine(errOut), "mrw: nothing was recorded") {
			t.Errorf("%s state, a served read: exit %d, out %q, err %q", name, code, out, errOut)
		}
		if _, errOut, code := run(state, "", "read", "f.txt:9"); code != 1 || !strings.Contains(errOut, "nothing was recorded") {
			t.Errorf("%s state, a read that missed: exit %d, err %q", name, code, errOut)
		}
		// --files-from: the effective specs decide, after the list is read.
		if out, errOut, code := run(state, "", "read", "--files-from", "plain.list"); code != 0 || !strings.Contains(out, "1| alpha") || !strings.Contains(errOut, "nothing was recorded") {
			t.Errorf("%s state, --files-from with a plain spec: exit %d, out %q, err %q", name, code, out, errOut)
		}
		// Reads that need the working set, and reads whose purpose is the
		// licence (ADR-166 Decision 3), still refuse.
		for _, argv := range [][]string{{"read", "@1"}, {"read"}, {"read", "--files-from", "pointer.list"}} {
			if out, _, code := run(state, "", argv...); code != 2 || out != "" {
				t.Errorf("%s state, %q needs the working set: exit %d, out %q", name, argv, code, out)
			}
		}
		// A quiet read and a plan read print what they printed before, and
		// still exit 2, as ADR-164 T3 left them.
		for _, argv := range [][]string{{"read", "--quiet", "f.txt"}, {"read", "--grep", "alpha", "--sub", "x", "f.txt"}} {
			if _, errOut, code := run(state, "", argv...); code != 2 || strings.Contains(errOut, "nothing was recorded") {
				t.Errorf("%s state, %q is a licence read: exit %d, err %q", name, argv, code, errOut)
			}
		}
	}
	// No licence was recorded: with the state writable again, a write to the
	// line served above is refused as unread.
	_ = os.Chmod(parent, 0o755)
	chmodAll(existing, 0o755)
	for _, state := range []string{existing, filepath.Join(parent, "state")} {
		if out, _, code := run(state, "@@ f.txt 1 replace\nALPHA\n", "write", "--no-check", "-"); code != 1 {
			t.Errorf("a write after a read that recorded nothing applied (%s): exit %d\n%s", state, code, out)
		}
	}
}
