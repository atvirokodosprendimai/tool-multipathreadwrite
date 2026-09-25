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
	"unicode"

	"github.com/urfave/cli/v3"
)

// ADR-069's promise, as a property: a caller-supplied path reaches mrw
// exactly as typed, or is refused. The guards model the parser edge by edge
// (T5-T10), and every Codex round found one more edge; this test asks the
// PARSER instead. The oracle is urfave itself: the same command tree with
// every Action replaced by a recorder, so it reports the strings the parser
// would hand mrw. The guard must refuse exactly when one of those strings is
// not among the tokens the caller typed (as a multiset: `read 'x ' x`
// delivers x twice for one x typed), the words of an `iter note` excepted.
//
// Pools are enumerated from the live command tree, not from memory (the
// stress-testing skill's v2 rule): subcommands from root.Commands, flags
// from each command's own and inherited sets, iter verbs from the refusal
// at main.go:1355. Excluded on purpose: mcp (blocks on stdin), seen
// (--prune walks the state base), instructions and version (no paths).
//
// MRW_STRESS_N cases (default 300), MRW_STRESS_SEED (default 1). A failure
// prints the seed, the iteration, the argv, what the parser delivered and
// what mrw said, so it replays by hand.
func TestTheGuardsAgreeWithTheParserOnRandomArgv(t *testing.T) {
	n, seed := 300, int64(1)
	if s := os.Getenv("MRW_STRESS_N"); s != "" {
		n, _ = strconv.Atoi(s)
	}
	if s := os.Getenv("MRW_STRESS_SEED"); s != "" {
		seed, _ = strconv.ParseInt(s, 10, 64)
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

	tree := paddedTree(t)
	names := []string{"x", "x ", " x", "x\t", "x\n", "x ", "-1=", " -1= ", "y", "y "}
	for _, nm := range names {
		if err := os.WriteFile(filepath.Join(tree, nm), []byte("file "+strconv.Quote(nm)+"\n"), 0o644); err != nil {
			t.Skipf("this filesystem cannot hold %q: %v", nm, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(tree, "d "), 0o755); err != nil {
		t.Skip(err)
	}
	t.Chdir(tree)

	root := rootCommand()
	var subs []*cli.Command
	for _, c := range root.Commands {
		switch c.Name {
		case "mcp", "seen", "instructions", "version":
			continue
		}
		subs = append(subs, c)
	}
	verbs := []string{"add", "rm", "clear", "note"}
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
		_, isBool := f.(*cli.BoolFlag)
		if isBool {
			return []string{pad(dash + name)}
		}
		val := pick(names)
		if _, isInt := f.(*cli.IntFlag); isInt {
			val = pick([]string{"1", "2 ", " 3", "-1"})
		}
		if attached {
			return []string{dash + name + "=" + val}
		}
		return []string{pad(dash + name), val}
	}

	mismatches := 0
	t.Logf("%d cases, seed %d", n, seed) // so a fast PASS is not mistaken for the 300-case default
	for i := 0; i < n && mismatches < 20; i++ {
		sub := subs[rng.Intn(len(subs))]
		var argv []string
		// Root flags before the verb, sometimes.
		if rng.Intn(4) == 0 {
			argv = append(argv, flagToken(root.Flags[0], rng.Intn(2) == 0)...)
		}
		argv = append(argv, pad(sub.Name))
		verbIdx, isNote := -1, false
		if sub.Name == "iter" {
			verb := pick(verbs)
			argv = append(argv, pad(verb))
			verbIdx, isNote = len(argv)-1, verb == "note"
		}
		flags := append([]cli.Flag{}, sub.Flags...)
		flags = append(flags, root.Flags...) // the inherited --root, where it applies
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

		delivered, dispatched, parseErr := parserDelivers(tree, argv)
		if parseErr != nil || !dispatched {
			continue // the parser refuses this argv itself, or finds no command: nothing reaches mrw
		}
		// The Decision's refusals, by PROVENANCE rather than by coincidence:
		// a padded token is replaced by a unique sentinel and the argv
		// re-parsed. If the sentinel lands where the token itself was, the
		// parser kept the token as given (a separate value, a path after --)
		// and nothing is owed. Otherwise: an attached value whose sentinel is
		// delivered was consumed as a flag and its value trimmed with the
		// token (item 5); a lone "-" the parser trims and keeps is judged by
		// its presence, since the sentinel is not special the way "-" is
		// (item 9); any other token whose trimmed spelling the sentinel
		// replaces was trimmed and acted on (item 1). The iter verb and a
		// note's words are not paths.
		got := multiset(delivered)
		var trimmed []string
		for idx, p := range argv {
			if idx == verbIdx || (isNote && idx > verbIdx && !flagShaped(strings.TrimSpace(p))) {
				continue
			}
			t := strings.TrimSpace(p)
			if t == p || t == "" || t == "--" {
				continue // unpadded, blank, or the terminator: never a path
			}
			u := fmt.Sprintf("sentinel%dq", idx)
			isAttached := flagShaped(t) && strings.Contains(t, "=")
			// Did the parser read this token as a flag (or as the lone "-")
			// at all? An unknown flag in its place is refused by name when
			// the parser reached it, and kept or dropped as a positional when
			// it did not (after a -- or a stop). This needs no re-parse of
			// the rest, which a later padded int value could make fail.
			argv3 := append([]string{}, argv...)
			argv3[idx] = "--" + u
			_, _, err3 := parserDelivers(tree, argv3)
			consumed := err3 != nil && strings.Contains(err3.Error(), u)
			if isAttached || t == "-" {
				if consumed && (isAttached || got["-"] > 0) {
					trimmed = append(trimmed, p)
				}
				continue
			}
			argv2 := append([]string{}, argv...)
			argv2[idx] = u
			d2, ok2, err2 := parserDelivers(tree, argv2)
			if err2 != nil || !ok2 {
				continue
			}
			m2 := multiset(d2)
			asGiven := multiset(delivered)
			asGiven[p]--
			asGiven[u]++
			if sameMultiset(m2, asGiven) {
				continue
			}
			asTrimmed := multiset(delivered)
			asTrimmed[t]--
			asTrimmed[u]++
			if got[t] > 0 && sameMultiset(m2, asTrimmed) {
				trimmed = append(trimmed, p)
			}
		}
		expectRefuse := len(trimmed) > 0

		refused := false
		if err := refusePaddedFlagValues(rootCommand(), append([]string{"-C", tree}, argv...)); err != nil {
			refused = strings.Contains(err.Error(), "whitespace")
		}
		out := ""
		if !refused {
			var code int
			out, code = runIn(t, tree, argv...)
			refused = code == exitUsage && strings.Contains(out, "whitespace the argument parser strips")
		}
		if refused != expectRefuse {
			mismatches++
			t.Errorf("seed %d, case %d: argv %q\n  parser delivers %q\n  not typed: %q\n  guard refused: %v, want %v\n  mrw said: %s",
				seed, i, argv, delivered, trimmed, refused, expectRefuse, strings.TrimSpace(out))
		}
	}
}

func pick2(rng *rand.Rand, fs []cli.Flag) cli.Flag { return fs[rng.Intn(len(fs))] }

// typedTokens is every token the caller typed, plus the value inside an
// attached `--flag=value`: the caller typed that value too, and the parser
// hands it on untrimmed unless the whole token ends in whitespace.
func typedTokens(argv []string) []string {
	out := append([]string{}, argv...)
	for _, tok := range argv {
		if strings.HasPrefix(tok, "-") {
			if _, v, ok := strings.Cut(tok, "="); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

// readsArgs names the commands that read their positionals as paths or a
// verb: read (main.go:631), write (:934), iter (:1297) and check (:1397).
// stats and seen never call cmd.Args(); version and instructions only refuse
// any (:319, :337). A positional handed to the others reaches nothing.
var readsArgs = map[string]bool{"read": true, "write": true, "iter": true, "check": true}

// flagShaped is what the parser reads as a flag: two dashes, or one dash
// before a letter (command_parse.go:131-139). ` -1= ` is not one.
func flagShaped(t string) bool {
	return strings.HasPrefix(t, "--") || (len(t) >= 2 && t[0] == '-' && unicode.IsLetter(rune(t[1])))
}

func sameMultiset(a, b map[string]int) bool {
	for k, v := range a {
		if v != 0 && b[k] != v {
			return false
		}
	}
	for k, v := range b {
		if v != 0 && a[k] != v {
			return false
		}
	}
	return true
}

func multiset(xs []string) map[string]int {
	m := map[string]int{}
	for _, x := range xs {
		m[x]++
	}
	return m
}

// parserDelivers runs argv through the same command tree with every Action
// replaced by a recorder, and returns every string the parser hands the
// command: its positionals, where the command reads them, and every string
// flag it set (own and inherited). The words of an `iter note` are free text
// and are left out. dispatched is false when no command's Action ran: the
// parser found no command (a padded name is looked up as given), so nothing
// reached mrw and main reports the unknown command.
func parserDelivers(tree string, argv []string) (got []string, dispatched bool, err error) {
	root := rootCommand()
	for _, c := range root.Commands {
		c := c
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
