package main

import "fmt"

// incompleteLine is the kind line of a read that exits 1 with problems unserved
// ranges. When the only one is the match budget's (ADR-179 Decision g) it names
// the budget; `incomplete:` stays the first word (ADR-146).
func incompleteLine(problems, left int, budget *int, reached string) string {
	if problems == 1 && left > 0 && budget != nil {
		return fmt.Sprintf("incomplete: --max-matches %d reached, %d matching line(s) not shown%s", *budget, left, reached)
	}
	return fmt.Sprintf("incomplete: %d range(s) could not be served%s", problems, reached)
}

// cappedLine is the kind line of a read that exits 0 because every problem was
// a cap the caller set (ADR-173): capped ranges, the match budget's one problem
// among them when left matching lines were not shown. `capped:` is the first
// word, where ADR-146 puts `incomplete:`.
func cappedLine(capped, left int, budget *int) string {
	matches := ""
	if left > 0 && budget != nil {
		matches = fmt.Sprintf("--max-matches %d reached, %d matching line(s) not shown", *budget, left)
		capped--
	}
	switch {
	case capped == 0:
		return "capped: " + matches
	case matches == "":
		return fmt.Sprintf("capped: %d range(s) withheld by --max-lines", capped)
	}
	return fmt.Sprintf("capped: %d range(s) withheld by --max-lines; %s", capped, matches)
}
