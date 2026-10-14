package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

// shortRoot reports the value of the last root option before the subcommand,
// and whether that option was the short -C with an all-digit value (ADR-151).
// Before the subcommand -C is --root, and after `read` it is --context, so
// `mrw -C 3 read f` reads under a directory named 3 — refused when there is
// none, and silent when there is one. The parser says which value --root got
// but not which alias carried it, so argv is walked here with the classifier
// refusePaddedFlagValues uses. The last option wins, as it does in the parser,
// and an empty token is passed over, as the parser passes over it.
func shortRoot(root *cli.Command, argv []string) (string, bool) {
	kinds := flagKinds(root)
	val, short := "", false
	for i := 0; i < len(argv); i++ {
		tok := strings.TrimSpace(argv[i])
		if tok == "" {
			continue
		}
		role, next := flagRole(argv[i], kinds)
		if role != roleFlag {
			break // the subcommand, a "--" or a lone "-": root options are over
		}
		// A spelling the parser refuses (---C=3) never reaches Before, so the
		// note it would install is never printed.
		name, v, attached := strings.Cut(strings.TrimLeft(tok, "-"), "=")
		if next {
			i++
			if i < len(argv) {
				v = argv[i]
			}
		}
		if name == "C" || name == "root" {
			val, short = v, name == "C" && (attached || next) && allDigits(v)
		}
	}
	return val, short
}

// noteShortRoot has the root command name an all-digit short root v on stderr
// once its own options have parsed. A call the parser refuses first never
// reaches Before, so it says nothing about a root it never used.
func noteShortRoot(root *cli.Command, v string) {
	before := root.Before
	root.Before = func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		fmt.Fprintf(os.Stderr, "mrw: -C before the subcommand is --root, so %q is taken as the root directory; "+
			"for lines of context, put -C after read: mrw read -C %s …\n", v, v)
		return before(ctx, cmd)
	}
}

// allDigits reports whether s is one or more ASCII digits.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
