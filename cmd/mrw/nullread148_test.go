package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-148. A read sent to the null device records nothing (ADR-133), and said
// only that, so a write that then applied on an EARLIER licence read like a
// broken guard. The read now names every file whose served lines an earlier
// read or write still licenses in this version, and only those: each name is
// paired with the write that must apply, each omission with one that must be
// refused.
func TestANullDeviceReadNamesWhatIsStillLicensed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, f := range []string{"read.txt", "never.txt", "part.txt"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("one\ntwo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	run := func(stdout *os.File, stdin string, args ...string) (error, string) {
		t.Helper()
		stderr, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = stderr.Close() }()
		in := filepath.Join(t.TempDir(), "stdin")
		if err := os.WriteFile(in, []byte(stdin), 0o644); err != nil {
			t.Fatal(err)
		}
		inFile, err := os.Open(in)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = inFile.Close() }()
		savedOut, savedErr, savedIn := os.Stdout, os.Stderr, os.Stdin
		os.Stdout, os.Stderr, os.Stdin = stdout, stderr, inFile
		defer func() { os.Stdout, os.Stderr, os.Stdin = savedOut, savedErr, savedIn }()
		cmd := rootCommand()
		var errBuf bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &errBuf, &errBuf
		runErr := cmd.Run(context.Background(), append([]string{"mrw", "-C", root}, args...))
		said, err := os.ReadFile(stderr.Name())
		if err != nil {
			t.Fatal(err)
		}
		return runErr, string(said) + errBuf.String()
	}
	sink := func() *os.File {
		t.Helper()
		f, err := os.Create(filepath.Join(t.TempDir(), "answer"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	licenceLine := func(said string) string {
		for _, l := range strings.Split(said, "\n") {
			if strings.Contains(l, "still licensed") {
				return l
			}
		}
		return ""
	}

	if err, _ := run(sink(), "", "read", "read.txt:2", "part.txt:1"); err != nil {
		t.Fatalf("a read to a file failed: %v", err)
	}
	err, said := run(null, "", "read", "read.txt:2", "never.txt:2", "part.txt:1-2")
	if err != nil {
		t.Fatalf("a read to the null device failed: %v", err)
	}
	if !strings.Contains(said, "null device, so nothing was recorded") {
		t.Errorf("ADR-133's line is gone: %q", said)
	}
	line := licenceLine(said)
	if !strings.Contains(line, "read.txt") || !strings.Contains(line, "(1 of 3 files served)") {
		t.Fatalf("the file an earlier read still licenses is not named with the count: %q", said)
	}
	if strings.Contains(line, "never.txt") || strings.Contains(line, "part.txt") {
		t.Fatalf("a file whose served lines are not all licensed is named: %q", line)
	}
	if err, out := run(sink(), "@@ read.txt 2 replace\nTWO\n", "write", "--no-check", "-"); err != nil {
		t.Fatalf("the write the named file promises was refused: %v %s", err, out)
	}
	if err, _ := run(sink(), "@@ never.txt 2 replace\nTWO\n", "write", "--no-check", "-"); err == nil {
		t.Fatal("a write to a file the line did not name applied")
	}
	if err, _ := run(sink(), "@@ part.txt 2 replace\nTWO\n", "write", "--no-check", "-"); err == nil {
		t.Fatal("a write to a partly licensed file's uncovered line applied")
	}

	// read.txt is now the version mrw wrote, wholly licensed (ADR-005): it is
	// named. Changed behind mrw's back, nothing licenses it and it is not.
	if _, said := run(null, "", "read", "read.txt:1"); !strings.Contains(licenceLine(said), "read.txt") {
		t.Fatalf("a file mrw wrote is not named: %q", said)
	}
	if err := os.WriteFile(filepath.Join(root, "read.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, said := run(null, "", "read", "read.txt:1"); licenceLine(said) != "" {
		t.Fatalf("a file changed since its licence is named: %q", said)
	}
	// The Codex review of #397. A whole-file read is licensed by an earlier read
	// whose spans reach the last line, and a file is one file under any spelling:
	// the write recovers an in-root link's entry (ADR-029), so the line does too.
	for _, f := range []string{"spans.txt", "target.txt"} {
		if err := os.WriteFile(filepath.Join(root, f), []byte("one\ntwo\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err, _ := run(sink(), "", "read", "spans.txt:1-2"); err != nil {
		t.Fatalf("a read to a file failed: %v", err)
	}
	saved := recountCap
	recountCap = 4 // spans.txt holds 8 bytes
	if _, said := run(null, "", "read", "spans.txt"); licenceLine(said) != "" {
		recountCap = saved
		t.Fatalf("a file over the recount cap was read whole to be named: %q", said)
	}
	recountCap = saved
	if _, said := run(null, "", "read", "spans.txt"); !strings.Contains(licenceLine(said), "spans.txt") {
		t.Fatalf("a whole-file read that earlier spans cover to the last line is not named: %q", said)
	}
	if err, out := run(sink(), "@@ spans.txt 2 replace\nTWO\n", "write", "--no-check", "-"); err != nil {
		t.Fatalf("the write the whole-file name promises was refused: %v %s", err, out)
	}
	if err := os.Symlink("target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Logf("no in-root link on this system, so the spelling case is not driven: %v", err)
		return
	}
	if err, _ := run(sink(), "", "read", "link.txt:1"); err != nil {
		t.Fatalf("a read through the link failed: %v", err)
	}
	if _, said := run(null, "", "read", "target.txt:1"); !strings.Contains(licenceLine(said), "target.txt") {
		t.Fatalf("a file licensed under another spelling is not named: %q", said)
	}
	if err, out := run(sink(), "@@ target.txt 1 replace\nONE\n", "write", "--no-check", "-"); err != nil {
		t.Fatalf("the write under the other spelling was refused: %v %s", err, out)
	}
}
