package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

// shortRootNote is the stderr line for an all-digit short -C before the
// subcommand, or "" (ADR-151). Before the subcommand -C is --root, and after
// `read` it is --context, so `mrw -C 3 read f` reads under a directory named 3 —
// refused when there is none, and silent when there is one. The parser says
// which value --root got but not which alias carried it, so argv is walked here,
// with the classifier refusePaddedFlagValues uses, up to the first positional.
func shortRootNote(root *cli.Command, argv []string) string {
	kinds := flagKinds(root)
	for i := 0; i < len(argv); i++ {
		role, next := flagRole(argv[i], kinds)
		if role != roleFlag {
			return "" // the subcommand, a "--" or a lone "-": root flags are over
		}
		name, val, attached := strings.Cut(strings.TrimLeft(strings.TrimSpace(argv[i]), "-"), "=")
		if next {
			i++
			if i < len(argv) {
				val = argv[i]
			}
		}
		if name == "C" && (attached || next) && allDigits(val) {
			return fmt.Sprintf("mrw: -C before the subcommand is --root, so %q is taken as the root directory; "+
				"for lines of context, put -C after read: mrw read -C %s …", val, val)
		}
	}
	return ""
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
