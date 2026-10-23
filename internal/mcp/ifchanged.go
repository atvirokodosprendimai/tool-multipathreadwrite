package mcp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// guards is the call's if_changed as read.Guards, in path order so a refusal
// names the same one every time. Each is judged as the CLI's --if-changed is
// (ADR-169 Decision 5).
func (a readArgs) guards() ([]read.Guard, error) {
	paths := make([]string, 0, len(a.IfChanged))
	for p := range a.IfChanged {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var out []read.Guard
	for _, p := range paths {
		g, err := read.NewGuard(p, a.IfChanged[p])
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func guardPaths(guards []read.Guard) []string {
	out := make([]string, len(guards))
	for i, g := range guards {
		out[i] = g.Path
	}
	return out
}

// guardNotes names, one line each, the guards a walk did not reach: the file
// may have stopped matching because it changed, so it is named and not refused
// (ADR-169 Decision 1).
func guardNotes(unused []string) string {
	var b strings.Builder
	for _, p := range unused {
		fmt.Fprintf(&b, "\n-- guard not used: %s (this read did not serve it)", p)
	}
	return b.String()
}
