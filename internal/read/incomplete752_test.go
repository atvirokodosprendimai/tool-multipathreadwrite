package read

import (
	"strings"
	"testing"
)

// #752 review. The last line of a read that found nothing beside paths it could
// not use says where to look for every one of them: some are REFUSED, some
// UNREADABLE. ADR-146's earlier phrase survives inside the line.
func TestTheIncompleteLineNamesBothKindsOfLine(t *testing.T) {
	got := NoMatchLine("X", 2, "")
	if !strings.HasPrefix(got, "incomplete: no file matched /X/, and 2 path(s) were refused") {
		t.Errorf("the ADR-146 phrase did not survive: %q", got)
	}
	if !strings.Contains(got, "(see the REFUSED and UNREADABLE lines)") {
		t.Errorf("the line points only at REFUSED: %q", got)
	}
}
