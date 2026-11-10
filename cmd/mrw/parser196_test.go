package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// ADR-196: urfave/cli v3.12.0 and later hand an Action every positional and
// every attached flag value as typed, so mrw no longer models the parser
// (ADR-069's guards and ADR-161's lone-dash lift are gone). The promise ADR-069
// made, that a caller-supplied path reaches mrw exactly as typed, is held here
// as a property of the parser itself: the command tree with every Action
// replaced by a recorder, driven by random argv, may deliver no string the
// caller did not type. A later urfave release that trims again fails this
// test in the Renovate pull request that brings it (v3.11.0 fails it at once:
// `iter add 'x '` delivers x).
//
// Pools come from the live command tree. Excluded: mcp (blocks on stdin), seen
// (--prune walks the state base), instructions and version (no paths), hook (a
// payload on stdin) and skill, whose --dir writes outside the fixture.
// MRW_STRESS_N cases (default 300), MRW_STRESS_SEED (default 1).
func TestTheParserHandsMrwOnlyWhatWasTyped(t *testing.T) {
	// A mistyped MRW_STRESS_N must not become zero cases and a green run.
	n, seed := 300, int64(1)
	if s := os.Getenv("MRW_STRESS_N"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			t.Fatalf("MRW_STRESS_N=%q is not a positive case count", s)
		}
		n = v
	}
	if s := os.Getenv("MRW_STRESS_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatalf("MRW_STRESS_SEED=%q is not an integer", s)
		}
		seed = v
	}
	rng := rand.New(rand.NewSource(seed))

	// A stdin that ends at once: `write -` and `--files-from -` read it.
	empty, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer empty.Close()
	oldStdin := os.Stdin
	os.Stdin = empty
	defer func() { os.Stdin = oldStdin }()

	tree := t.TempDir()
	names := []string{"x", "x ", " x", "x\t", "x\n", "x\u00a0", "-1=", " -1= ", "y", "y ", "-", " - ", "- "}
	for _, nm := range names[:10] {
		// A name this filesystem cannot hold is still a token to parse.
		_ = os.WriteFile(filepath.Join(tree, nm), []byte("file "+strconv.Quote(nm)+"\n"), 0o644)
	}
	t.Chdir(tree)

	root := rootCommand()
	var subs []*cli.Command
	for _, c := range root.Commands {
		switch c.Name {
		case "mcp", "seen", "instructions", "version", "skill", "hook":
			continue
		}
		subs = append(subs, c)
	}
	pad := func(s string) string {
		switch rng.Intn(6) {
		case 0:
			return s + " "
		case 1:
			return " " + s
		case 2:
			return s + "\n"
		default:
			return s
		}
	}
	pick := func(xs []string) string { return xs[rng.Intn(len(xs))] }
	flagToken := func(f cli.Flag, attached bool) []string {
		name := pick(f.Names())
		dash := "--"
		if len(name) == 1 {
			dash = "-"
		}
		if _, isBool := f.(*cli.BoolFlag); isBool {
			return []string{pad(dash + name)}
		}
		val := pick(names)
		if _, isInt := f.(*cli.IntFlag); isInt {
			val = pick([]string{"1", "2 ", " 3"})
		}
		if attached {
			return []string{dash + name + "=" + val}
		}
		return []string{pad(dash + name), val}
	}

	untyped, dispatched := 0, 0
	for i := 0; i < n && untyped < 20; i++ {
		sub := subs[rng.Intn(len(subs))]
		var argv []string
		if rng.Intn(4) == 0 { // root flags before the verb, sometimes
			argv = append(argv, flagToken(root.Flags[0], rng.Intn(2) == 0)...)
		}
		argv = append(argv, pad(sub.Name))
		if sub.Name == "iter" {
			argv = append(argv, pick([]string{"add", "rm", "clear"}))
		}
		flags := append(append([]cli.Flag{}, sub.Flags...), root.Flags...)
		for k := rng.Intn(5); k > 0; k-- {
			switch rng.Intn(7) {
			case 0, 1:
				argv = append(argv, flagToken(pick2(rng, flags), rng.Intn(3) == 0)...)
			case 2:
				argv = append(argv, pad("--"))
			case 3:
				argv = append(argv, pad("-"))
			default:
				argv = append(argv, pick(names))
			}
		}
		got, ok, err := parserDelivers(tree, argv)
		if err != nil || !ok {
			continue // the parser refused this argv, or found no command: nothing reached mrw
		}
		dispatched++
		// What the caller typed: each token, and the text after "=" in one,
		// which is an attached value as typed. -C tree is the test's own.
		typed := map[string]bool{tree: true}
		for _, a := range argv {
			typed[a] = true
			if _, v, has := strings.Cut(a, "="); has {
				typed[v] = true
			}
		}
		for _, g := range got {
			if !typed[g] {
				untyped++
				t.Errorf("seed %d, case %d: argv %q\n  parser delivers %q, which was not typed", seed, i, argv, g)
				break
			}
		}
	}
	t.Logf("seed %d: %d of %d cases dispatched", seed, dispatched, n)
	if dispatched == 0 {
		t.Fatal("no case reached a command: the test checked nothing")
	}
}

// ADR-196: with the guard gone, nothing but the parser and the boundary stands
// between `read 'x '` and x, and on NTFS Win32 opens x for "x " (ADR-071
// refuses that spelling). The tree holds x alone, so this runs on Windows too:
// no spelling of "x " may serve or change x. The pair: x itself is served.
func TestAPaddedPathIsNeverServedAsItsTrimmedSibling(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x"), []byte("plain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "read", "x")
	if code != 0 || !strings.Contains(out, "plain") {
		t.Fatalf("read x: exit %d, want 0 serving x:\n%s", code, out)
	}
	for _, argv := range [][]string{
		{"read", "x "},
		{"read", " x"},
		{"read", "x\t"},
		{"read", "--", "x "},
		{"read", "--grep", "plain", "x "},
	} {
		out, code := runIn(t, root, argv...)
		if code == 0 || strings.Contains(out, "| plain") {
			t.Errorf("%q: exit %d, and it must neither succeed nor serve x:\n%s", argv, code, out)
		}
	}
	plan := filepath.Join(t.TempDir(), "p.mrw")
	if err := os.WriteFile(plan, []byte("@@ \"x \" 1 replace\nNEW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "write", plan); code == 0 {
		t.Errorf("a write to \"x \" exited 0:\n%s", out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "x")); string(b) != "plain\n" {
		t.Errorf("x now holds %q", b)
	}
}

// ADR-196: v3.14.0 keeps a lone - and parses on, so the lift that moved it last
// is gone. `write - --` is the piped plan and an empty terminator, and it
// applies (the lift refused the -- with exit 2); flags after - are honoured;
// a path after - is still refused by write's own one-plan rule.
func TestALoneDashThenATerminatorWritesThePipedPlan(t *testing.T) {
	const plan = "@@ created.txt 0 create\ncreated\n"
	root := t.TempDir()
	target := filepath.Join(root, "created.txt")

	withStdin(t, plan, func() {
		out, code := runIn(t, root, "write", "-", "--dry-run", "--")
		if code != 0 {
			t.Fatalf("write - --dry-run --: exit %d, want 0:\n%s", code, out)
		}
	})
	if _, err := os.Stat(target); err == nil {
		t.Fatal("a dry run after a lone - wrote the file: the flag after - was not honoured")
	}
	withStdin(t, plan, func() {
		out, code := runIn(t, root, "write", "-", "x")
		if code != exitUsage {
			t.Fatalf("write - x: exit %d, want %d:\n%s", code, exitUsage, out)
		}
	})
	if _, err := os.Stat(target); err == nil {
		t.Fatal("write - x wrote the file")
	}
	withStdin(t, plan, func() {
		out, code := runIn(t, root, "write", "-", "--")
		if code != 0 {
			t.Fatalf("write - --: exit %d, want 0:\n%s", code, out)
		}
	})
	if b, err := os.ReadFile(target); err != nil || string(b) != "created\n" {
		t.Fatalf("write - -- left %q, %v", b, err)
	}
}

func pick2(rng *rand.Rand, fs []cli.Flag) cli.Flag { return fs[rng.Intn(len(fs))] }

// readsArgs names the commands that read their positionals as paths or a
// verb. stats and seen never call cmd.Args(); version and instructions only
// refuse any. A positional handed to the others reaches nothing.
var readsArgs = map[string]bool{"read": true, "write": true, "iter": true, "check": true, "run": true}

// parserDelivers runs argv through the command tree with every Action
// replaced by a recorder, and returns every string the parser hands the
// command: its positionals, where the command reads them, and every string
// flag it set (own and inherited). The words of an `iter note` are free text
// and are left out. dispatched is false when no command's Action ran.
func parserDelivers(tree string, argv []string) (got []string, dispatched bool, err error) {
	root := rootCommand()
	for _, c := range root.Commands {
		c.Action = func(_ context.Context, cmd *cli.Command) error {
			dispatched = true
			args := cmd.Args().Slice()
			if cmd.Name == "iter" && len(args) > 0 && args[0] == "note" {
				args = args[:1]
			}
			if readsArgs[cmd.Name] {
				got = append(got, args...)
			}
			seen := map[string]bool{}
			for _, f := range append(append([]cli.Flag{}, cmd.Flags...), root.Flags...) {
				name := f.Names()[0]
				if seen[name] || !cmd.IsSet(name) {
					continue
				}
				seen[name] = true
				switch f.(type) {
				case *cli.StringFlag:
					got = append(got, cmd.String(name))
				case *cli.StringSliceFlag:
					got = append(got, cmd.StringSlice(name)...)
				case *cli.GenericFlag:
					// --then and --then-sh (ADR-092): each value the parser set.
					if v, ok := cmd.Generic(name).(interface{ Values() []string }); ok {
						got = append(got, v.Values()...)
					}
				}
			}
			return nil
		}
	}
	var sink bytes.Buffer
	root.Writer, root.ErrWriter = &sink, &sink
	if err := root.Run(context.Background(), append([]string{"mrw", "-C", tree}, argv...)); err != nil {
		return nil, false, fmt.Errorf("parser: %w", err)
	}
	return got, dispatched, nil
}
