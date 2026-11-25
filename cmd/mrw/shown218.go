package main

import (
	"fmt"
	"os"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// shownOffNote is what a read says, once, when MRW_SHOWN=off made it record its
// lines as licensed and not shown (ADR-218 Decision 2).
const shownOffNote = "mrw: recorded as licensed, not shown (MRW_SHOWN=off): name a line by pattern or anchor=, and send a multi-line replace as a search_replace block whose SEARCH and REPLACE end in the same line"

// shownSwitch reads MRW_SHOWN, the one place it is read (ADR-218). off is true
// for `off`: a program stands between mrw's output and the reader who writes the
// plan, so a read records no showing and a write asks for none it cannot have.
// `on` and unset are false. Any other value is false too and is returned as
// unknown, for the caller to name. No host's own variable is consulted
// (Decision 6).
func shownSwitch() (off bool, unknown string) {
	switch v := os.Getenv("MRW_SHOWN"); v {
	case "off":
		return true, ""
	case "", "on":
		return false, ""
	default:
		return false, v
	}
}

// shownOff is shownSwitch for a command a person reads the standard error of: an
// unknown value is named there and ignored, as MRW_RECEIPT's is (ADR-188).
func shownOff() bool {
	off, unknown := shownSwitch()
	if unknown != "" {
		fmt.Fprintf(os.Stderr, "mrw: MRW_SHOWN=%s is not off or on; ignored\n", unknown)
	}
	return off
}

// servedALine reports whether a read served any line of any file: a --stat or
// --max-lines 0 serves none. Nil Spans is a whole-file read, every line served
// (seen.Observation).
func servedALine(observed map[string]seen.Observation) bool {
	for _, o := range observed {
		if o.Spans == nil || len(o.Spans) > 0 {
			return true
		}
	}
	return false
}

// emptySHA is the sha of a file of no bytes.
var emptySHA = seen.SHA(nil)

// showedLines reports whether an observation holds a line a read printed: a
// span, or a whole file that has lines. A whole read of an empty file printed a
// header and no line, and its observation still licenses the first insert.
func showedLines(o seen.Observation) bool {
	return len(o.Spans) > 0 || (o.Spans == nil && o.SHA != emptySHA)
}

// showedALine reports whether a read printed a line of any file: what the
// switch needs to know, since a read that printed none has no showing to
// withhold.
func showedALine(observed map[string]seen.Observation) bool {
	for _, o := range observed {
		if showedLines(o) {
			return true
		}
	}
	return false
}

// shownOffRecord is what a switched read records: each file it served a line of
// licensed and not shown, as a --quiet read's is; a file it served no line of
// (a --stat or a withheld range beside a served one) as the read observed it,
// so that record keeps its shape.
func shownOffRecord(observed map[string]seen.Observation) map[string]seen.Observation {
	out := make(map[string]seen.Observation, len(observed))
	for path, o := range observed {
		if showedLines(o) {
			o = mcp.LicensedNotShown(map[string]seen.Observation{path: o})[path]
		}
		out[path] = o
	}
	return out
}
