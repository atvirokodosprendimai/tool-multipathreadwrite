package guide

import (
	"strings"
	"testing"
)

// ADR-203. A number alone is refused on a line of a file mrw wrote that no read
// has shown, so the core and the full instructions say a second edit names its
// line by pattern or anchor=, or follows a read that shows it; the recipes that
// grant or teach the second edit say the same; the core stays under its limit.
func TestTheInstructionsSayASecondEditNamesItsLine(t *testing.T) {
	core, full, recipes := Core(), CLI(), Recipes()
	if n := len(strings.Fields(core)); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}
	const rule = "except in a file mrw just wrote, where a number alone is refused: use a pattern, anchor= or a read"
	if !strings.Contains(core, rule) {
		t.Errorf("core rule 2 does not say a number alone is refused in a written file:\n%s", core)
	}
	const fullRule = "A number alone is refused on a line of it that no read has shown since the write"
	if !strings.Contains(full, fullRule) || !strings.Contains(full, "read the line, or name it by a pattern or anchor=") {
		t.Errorf("the full text does not say what lifts the refusal")
	}
	if !strings.Contains(full, "has no recorded showing in this version") {
		t.Errorf("the full text does not name the reason's words")
	}
	for _, text := range []string{core, full} {
		if strings.Contains(text, "a chain of edits to it needs no re-read") || strings.Contains(text, "whose numbers can move") {
			t.Errorf("the instructions still promise an unread number is safe in a written file")
		}
	}
	for _, n := range []string{"7. ", "21. "} {
		var label string
		for _, l := range strings.Split(recipes, "\n") {
			if strings.HasPrefix(l, n) {
				label = l
			}
		}
		if !strings.Contains(label, "a number alone is refused") {
			t.Errorf("recipe %sdoes not say a number alone is refused:\n%s", n, label)
		}
	}
}
