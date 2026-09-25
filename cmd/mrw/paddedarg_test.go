package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runIn runs `mrw -C root <argv...>` in-process and returns everything
// printed plus the exit code the binary would have used.
func runIn(t *testing.T, root string, argv ...string) (string, int) {
	t.Helper()
	if os.Getenv("XDG_STATE_HOME") == "" {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
	}
	full := append([]string{"mrw", "-C", root}, argv...)
	var sink bytes.Buffer
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), full)
	})
	code := 0
	if err != nil {
		code = exitCode(err)
		out += err.Error()
	}
	return out + sink.String(), code
}

// ADR-069 T1. urfave/cli v3.11.0 trims every positional before `--`
// (command_parse.go:81, :117), so `mrw read 'x '` served x: a file the caller
// did not name. The padded positional is refused, and the refusal names the
// `--` that reaches it; `--` itself is unchanged.
func TestAPaddedPositionalWithoutDashDashIsRefused(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"x", "x ", "p "} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("the same\n"), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", n, err)
		}
	}
	if fi, err := os.ReadDir(root); err != nil || len(fi) != 3 {
		t.Skipf("this filesystem folds names that differ by a trailing space")
	}

	for _, argv := range [][]string{
		{"read", "x "},
		{"write", "--no-check", filepath.Join(root, "p ")},
		{"check", "x "},
	} {
		out, code := runIn(t, root, argv...)
		padded := argv[len(argv)-1]
		if code != exitUsage {
			t.Errorf("%q exited %d, want %d:\n%s", argv, code, exitUsage, out)
		}
		if !strings.Contains(out, "'"+padded+"'") || !strings.Contains(out, "--") {
			t.Errorf("%q: the refusal does not name %q and --:\n%s", argv, padded, out)
		}
	}

	out, code := runIn(t, root, "read", "--", "x ")
	if code != 0 || !strings.Contains(out, "==> x  ") {
		t.Errorf("read -- 'x ' exited %d or did not serve x-space:\n%s", code, out)
	}
	// A token after -- is never inspected: here "x " trims to a positional,
	// x, and refusing it would refuse the very spelling the message teaches.
	if out, code := runIn(t, root, "read", "x", "--", "x "); code != 0 {
		t.Errorf("read x -- 'x ' was refused, exit %d:\n%s", code, out)
	}
	// A padded flag value whose trim is no positional passes: this kills a
	// helper that refuses every padded token.
	if out, code := runIn(t, root, "read", "--grep", " same", "x"); code != 0 {
		t.Errorf("a padded --grep value was refused, exit %d:\n%s", code, out)
	}
	// A note is free text, not a path.
	if out, code := runIn(t, root, "iter", "note", "wip "); code != 0 {
		t.Errorf("iter note 'wip ' was refused, exit %d:\n%s", code, out)
	}
}
