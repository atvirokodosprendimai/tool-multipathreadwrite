package main

import (
	"fmt"
	"io"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// terseFiles is the file section of a --terse receipt (ADR-179 Decision f): one
// line per file the plan changed, from the hunks that landed and the files they
// reached. A dry run lists the files its hunks name; a rename is one line for
// the old path, and the path it created has none of its own.
func terseFiles(out io.Writer, res apply.Result) {
	type tally struct{ hunks, removed, added int }
	var order []string
	byPath := map[string]*tally{}
	for _, h := range res.Hunks {
		if h.Status != apply.StatusOK {
			continue
		}
		t := byPath[h.Path]
		if t == nil {
			t = &tally{}
			byPath[h.Path] = t
			order = append(order, h.Path)
		}
		t.hunks++
		t.removed += h.Removed
		t.added += h.Added
	}
	if res.DryRun {
		for _, p := range order {
			fmt.Fprintf(out, "dry  %s  %d hunk(s)\n", p, byPath[p].hunks)
		}
		return
	}
	renamedTo := map[string]bool{}
	byFile := map[string]apply.FileResult{}
	for _, f := range res.Files {
		byFile[f.Path] = f
		if f.Written && f.Removed && f.RenamedTo != "" {
			renamedTo[f.RenamedTo] = true
		}
	}
	for _, f := range res.Files {
		switch {
		case !f.Written || renamedTo[f.Path]:
		case f.Removed && f.RenamedTo != "":
			// Edits made beside the rename ride on its line: their tally, and
			// the sha of the file that now holds them.
			line := fmt.Sprintf("ok   %s -> %s  rename", f.Path, f.RenamedTo)
			var lost, gained int
			for _, p := range []string{f.Path, f.RenamedTo} {
				if t := byPath[p]; t != nil {
					lost += t.removed
					gained += t.added
				}
			}
			if lost+gained > 0 {
				line += fmt.Sprintf("  -%d +%d  sha %s", lost, gained, short(byFile[f.RenamedTo].SHAAfter))
			}
			fmt.Fprintln(out, line)
		case f.Removed:
			fmt.Fprintf(out, "ok   %s  unlink  was sha %s\n", f.Path, short(f.SHABefore))
		default:
			t := byPath[f.Path]
			if t == nil {
				t = &tally{}
			}
			through := ""
			if f.Target != "" { // ADR-105: a write through a symlink names where it went
				through = "  target " + f.Target
			}
			fmt.Fprintf(out, "ok   %s  %d hunk(s)  -%d +%d  sha %s%s\n", f.Path, t.hunks, t.removed, t.added, short(f.SHAAfter), through)
		}
	}
}
