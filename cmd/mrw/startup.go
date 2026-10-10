package main

import (
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// startsWithoutState reports whether args begin with a command that touches no
// state (ADR-145): the install check and the instructions print. main runs the
// legacy migration before it parses, so only the first argument is known. The
// parser prints the version for -v, --v, -version, --version and either with
// =true, so each of those spellings is the install check.
func startsWithoutState(args []string) bool {
	if len(args) == 0 {
		return false
	}
	a := args[0]
	if a == "version" || a == "instructions" {
		return true
	}
	flag := strings.TrimPrefix(strings.TrimPrefix(a, "-"), "-")
	if flag == a {
		return false
	}
	name, val, hasVal := strings.Cut(flag, "=")
	return (name == "v" || name == "version") && (!hasVal || val == "true")
}

// migrateLegacyState copies a legacy ./.mrw/ into the state directory, unless
// args begin with a command that touches no state.
func migrateLegacyState(args []string) ([]string, error) {
	if startsWithoutState(args) {
		return nil, nil
	}
	return state.Migrate(".")
}
