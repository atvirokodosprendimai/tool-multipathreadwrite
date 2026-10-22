package main

import (
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// recordedAs is what a read puts in the ledger: what it observed, or, for a
// --quiet read, the same lines licensed and not shown (ADR-167, Zy 2026-10-22).
func recordedAs(quiet bool, observed map[string]seen.Observation) map[string]seen.Observation {
	if quiet {
		return mcp.LicensedNotShown(observed)
	}
	return observed
}
