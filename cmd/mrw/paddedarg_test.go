package main

import (
	"bytes"
	"context"
	"os"
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

// ADR-069 T1, retargeted by ADR-196 (the name is the done task's): urfave/cli
// v3.11.0 trimmed `read 'x '` to x, and ADR-069 refused it. v3.14.0 hands the
// path over as typed: `read 'x '` serves the file `x `, never x, with or
// without --.
func TestAPaddedPositionalWithoutDashDashIsRefused(t *testing.T) {
	root := paddedTree(t)
	for _, argv := range [][]string{{"read", "x "}, {"read", "--", "x "}} {
		out, code := runIn(t, root, argv...)
		if code != 0 || !strings.Contains(out, "| padded") || strings.Contains(out, "| plain") {
			t.Errorf("%q: exit %d, want the file `x ` and never x:\n%s", argv, code, out)
		}
	}
}
