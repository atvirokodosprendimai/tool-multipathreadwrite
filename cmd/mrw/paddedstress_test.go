package main

import "testing"

// ADR-069 T11, retargeted by ADR-196 (the name is the done task's, and ADR-092
// T2's and ADR-168 T2's): the random differential tested the guard against the
// parser. The guard is gone, so the property it served is tested directly: the
// parser hands mrw only what was typed, --then values and the skill exclusion
// included (parser196_test.go).
func TestTheGuardsAgreeWithTheParserOnRandomArgv(t *testing.T) {
	TestTheParserHandsMrwOnlyWhatWasTyped(t)
}
