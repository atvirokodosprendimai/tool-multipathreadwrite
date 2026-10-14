package mcp

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"

// newerLine is the line a served read carries when the ledger was written by a
// newer mrw, and "" otherwise (ADR-157 Decision 3): the read hands out
// checkpoint ids an agent will ack, and this mrw records nothing from an ack
// while it leaves that ledger untouched (ADR-150). The server has no stderr, so
// the answer says it.
func newerLine(root string) string {
	writer, newer, err := seen.Newer(root)
	if err != nil || !newer {
		return ""
	}
	return "-- the read ledger was written by a newer mrw (" + writer + "): an ack here licenses nothing. Use the newer mrw.\n"
}
