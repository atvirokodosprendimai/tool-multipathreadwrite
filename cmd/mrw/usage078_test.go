package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// runSplit runs mrw in-process with stdout and stderr captured apart, so a test
// can say which stream something reached.
func runSplit(t *testing.T, argv ...string) (stdout string, err error) {
	t.Helper()
	var w, e bytes.Buffer
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		cmd.Writer, cmd.ErrWriter = &w, &e
		return cmd.Run(context.Background(), append([]string{"mrw"}, argv...))
	})
	return out + w.String(), err
}

// ADR-078 T1. urfave/cli printed a command's whole help to stdout on a flag it
// rejected, and stdout is `mrw mcp`'s protocol stream: `mrw mcp --bogus` in a
// host's config put help where the host reads JSON-RPC. A usage error writes
// nothing to stdout, names the help to read, and exits 2; --help still prints.
func TestAUsageErrorWritesNothingToStdout(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, argv := range [][]string{{"--bogus"}, {"mcp", "--bogus"}, {"read", "--bogus"}, {"write", "--bogus"}, {"--rot", ".", "mcp"}} {
		stdout, err := runSplit(t, argv...)
		if err == nil || exitCode(err) != exitUsage {
			t.Errorf("%v: err %v, want exit %d", argv, err, exitUsage)
			continue
		}
		if stdout != "" {
			t.Errorf("%v wrote to stdout:\n%s", argv, stdout)
		}
		if !strings.Contains(err.Error(), "--help") {
			t.Errorf("%v: the error does not name the help to read: %v", argv, err)
		}
	}
	for _, argv := range [][]string{{"--help"}, {"read", "--help"}, {"mcp", "--help"}} {
		if stdout, err := runSplit(t, argv...); err != nil || !strings.Contains(stdout, "USAGE") {
			t.Errorf("%v: err %v, and stdout lacks the help:\n%s", argv, err, stdout)
		}
	}
}

// ADR-078 T1. `mrw mcp stray` started a server past an argument nobody meant.
func TestMcpTakesNoArguments(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	stdout, err := runSplit(t, "mcp", "stray")
	if err == nil || exitCode(err) != exitUsage || !strings.Contains(err.Error(), "takes no arguments") || stdout != "" {
		t.Errorf("mcp stray: err %v, stdout %q", err, stdout)
	}
}

// ADR-078 T1. -C widens a /pattern/ match and nothing else; on a call with
// none it was ignored in silence. It is refused; with a pattern it serves.
func TestContextWithNoPatternIsRefused(t *testing.T) {
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"})
	out, err := readIn(t, root, "-C", "1", "a.go:2")
	if err == nil || exitCode(err) != exitUsage || !strings.Contains(errString(err), "-C widens a /pattern/ match") {
		t.Errorf("-C on a line range: err %v\n%s", err, out)
	}
	if out, err := readIn(t, root, "-C", "1", "a.go:/func B/"); err != nil || !strings.Contains(out, "func A") {
		t.Errorf("-C on a pattern did not serve its context: %v\n%s", err, out)
	}
	if out, err := readIn(t, root, "--grep", "func B", "-C", "1"); err != nil || !strings.Contains(out, "func A") {
		t.Errorf("-C on a grep did not serve its context: %v\n%s", err, out)
	}
}
