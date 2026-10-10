package main

// tellsLedgerDamage reports whether a command tells a damaged read ledger
// (ADR-144): the ones that consume it and save it back, so the call heals what
// it tells, and seen, where one goes to look. check and iter neither read nor
// save it, and mcp resolves its root after Before has run.
func tellsLedgerDamage(verb string) bool {
	switch verb {
	case "read", "write", "seen":
		return true
	}
	return false
}
