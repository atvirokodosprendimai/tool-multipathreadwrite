package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// ADR-097 T1. `mrw --instructions` was refused as "flag provided but not
// defined: -instructions", which is true and names nothing the caller can do:
// the thing they wanted is the subcommand `mrw instructions`. An undefined flag
// whose name is exactly a subcommand of the command it was given to now names
// that subcommand, at exit 2 with nothing on stdout, still pointing at --help.
func TestAFlagNamedForASubcommandNamesTheSubcommand(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const want = "--instructions is not a flag; the subcommand is `mrw instructions` (see: mrw --help)"
	for _, argv := range [][]string{{"--instructions"}, {"-instructions"}, {"--instructions=x"}, {"--instructions "}, {"--instructions", "read"}} {
		stdout, err := runSplit(t, argv...)
		if err == nil || exitCode(err) != exitUsage || err.Error() != want || stdout != "" {
			t.Errorf("%q: err %v (exit %d), stdout %q; want exit %d and %q", argv, err, exitCode(err), stdout, exitUsage, want)
		}
	}
	if _, err := runSplit(t, "--seen"); err == nil || err.Error() != "--seen is not a flag; the subcommand is `mrw seen` (see: mrw --help)" {
		t.Errorf("--seen: %v", err)
	}
	// Below the root: the lookup is the children of the command the flag was
	// given to, and an alias is echoed as typed while the child is named by its
	// name. mrw declares no such subcommand today, so the tree is built here.
	for _, c := range []struct{ flag, want string }{
		{"--inner", "--inner is not a flag; the subcommand is `mrw outer inner` (see: mrw outer --help)"},
		{"--i", "--i is not a flag; the subcommand is `mrw outer inner` (see: mrw outer --help)"},
	} {
		if err := runNested(t, "outer", c.flag); err == nil || exitCode(err) != exitUsage || err.Error() != c.want {
			t.Errorf("outer %s: err %v, want %q", c.flag, err, c.want)
		}
	}
	if stdout, err := runSplit(t, "instructions"); err != nil || stdout == "" {
		t.Errorf("mrw instructions no longer works: err %v, stdout %q", err, stdout)
	}
}

// ADR-097 T1. Everything outside that class keeps its v1.31.0 wording: an
// unknown name, a prefix or a near-miss, another case, a sibling, a verb that is
// a positional argument, and any usage error that is not "not defined".
func TestAnUnknownFlagNamingNoSubcommandIsRefusedAsBefore(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, c := range []struct {
		argv []string
		want string
	}{
		{[]string{"--bogus"}, "flag provided but not defined: -bogus (see: mrw --help)"},
		{[]string{"--instruction"}, "flag provided but not defined: -instruction (see: mrw --help)"},
		{[]string{"--instructionsx"}, "flag provided but not defined: -instructionsx (see: mrw --help)"},
		{[]string{"--Instructions"}, "flag provided but not defined: -Instructions (see: mrw --help)"},
		{[]string{"check", "--stats"}, "flag provided but not defined: -stats (see: mrw check --help)"},
		{[]string{"read", "--instructions"}, "flag provided but not defined: -instructions (see: mrw read --help)"},
		{[]string{"iter", "--add"}, "flag provided but not defined: -add (see: mrw iter --help)"},
	} {
		stdout, err := runSplit(t, c.argv...)
		if err == nil || exitCode(err) != exitUsage || err.Error() != c.want || stdout != "" {
			t.Errorf("%q: err %v (exit %d), stdout %q; want %q", c.argv, err, exitCode(err), stdout, c.want)
		}
	}
	if _, err := runSplit(t, "--help=x"); err == nil || !strings.Contains(err.Error(), "invalid value") || strings.Contains(err.Error(), "is not a flag") {
		t.Errorf("--help=x: %v", err)
	}
	for _, argv := range [][]string{{"--help"}, {"-h"}, {"--version"}, {"-v"}} {
		if _, err := runSplit(t, argv...); err != nil {
			t.Errorf("%q: %v", argv, err)
		}
	}
	if err := refusePaddedFlagValues(rootCommand(), []string{"--instructions= "}); err == nil || !strings.Contains(err.Error(), "ends in whitespace") {
		t.Errorf("a padded attached value is no longer refused first: %v", err)
	}
	// Only urfave's not-defined text is read (Decision 4): a message without that
	// prefix — a reworded one after an upgrade — names nothing, even when its
	// text is a subcommand's name.
	for _, msg := range []string{"instructions", "unknown flag: -instructions"} {
		if flag, sub := subcommandForFlag(rootCommand(), errors.New(msg)); flag != "" || sub != "" {
			t.Errorf("%q named %q / %q", msg, flag, sub)
		}
	}
}

// runNested runs a test-built `mrw outer inner` tree (inner also answers to
// `i`) with mrw's usage errors installed. ExitErrHandler is set as rootCommand
// sets it: without it urfave's HandleExitCoder calls os.Exit.
func runNested(t *testing.T, argv ...string) error {
	t.Helper()
	var w, e bytes.Buffer
	noop := func(context.Context, *cli.Command) error { return nil }
	root := &cli.Command{
		Name:           "mrw",
		Writer:         &w,
		ErrWriter:      &e,
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Commands: []*cli.Command{{
			Name:     "outer",
			Action:   noop,
			Commands: []*cli.Command{{Name: "inner", Aliases: []string{"i"}, Action: noop}},
		}},
	}
	installUsageErrors(root)
	return root.Run(context.Background(), append([]string{"mrw"}, argv...))
}
