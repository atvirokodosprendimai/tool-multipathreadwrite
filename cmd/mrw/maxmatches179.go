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
