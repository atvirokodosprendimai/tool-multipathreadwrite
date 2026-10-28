package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli/v3"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// effectiveMaxCols is the --max-cols a read runs with (ADR-179 Decision a):
// the flag when given, else MRW_MAX_COLS, else none. fixed says the caller
// already chose, so no hint is due: a flag, a valid variable, or a read the
// variable must not touch. The variable is read with os.Getenv and never as a
// flag source, which would make it IsSet and trip --sub's and --rewrite's
// refusals of --max-cols. It does not apply to a read that licenses (--quiet,
// which would license less without saying so) or that serves no line. A value
// that is no non-negative integer is named on stderr, once, and ignored.
func effectiveMaxCols(cmd *cli.Command) (cols int, fixed bool) {
	if cmd.IsSet("max-cols") {
		return cmd.Int("max-cols"), true
	}
	// quiet, stat and count are booleans: given false they are not the flag.
	if cmd.Bool("quiet") || cmd.Bool("stat") || cmd.Bool("count") || cmd.Bool("outline") {
		return 0, true
	}
	for _, f := range []string{"sub", "rewrite", "json-path"} {
		if cmd.IsSet(f) {
			return 0, true
		}
	}
	v := os.Getenv("MRW_MAX_COLS")
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		fmt.Fprintf(os.Stderr, "mrw: MRW_MAX_COLS=%s is not a non-negative integer, such as 200; ignored\n", v)
		return 0, false
	}
	return n, true
}

// longLineHint is the one stderr line a read earns for the lines it served
// whole past read.LongLineCols (ADR-179 Decision a), or "" when there were none.
func longLineHint(l read.LongLines) string {
	if l.Count == 0 {
		return ""
	}
	return fmt.Sprintf("mrw: %d line(s) over %d columns served whole, the longest %d; --max-cols 200 cuts each to a window round the match, and a cut line licenses nothing until it is read whole",
		l.Count, read.LongLineCols, l.Longest)
}
