package main

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"

// startsWithoutState reports whether args begin with a command that touches no
// state (ADR-145): the install check and the instructions print. main runs the
// legacy migration before it parses, so only the first argument is known.
func startsWithoutState(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "version", "-v", "--version", "instructions":
		return true
	}
	return false
}

// migrateLegacyState copies a legacy ./.mrw/ into the state directory, unless
// args begin with a command that touches no state.
func migrateLegacyState(args []string) ([]string, error) {
	if startsWithoutState(args) {
		return nil, nil
	}
	return state.Migrate(".")
}
