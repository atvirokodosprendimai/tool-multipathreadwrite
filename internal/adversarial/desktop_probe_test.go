package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheDesktopEnvelopeProbeRecipeIsFiled is UC-3's bindable name: ADR-023
// carries a Verification Log recipe for one Desktop mrw_read of a two-line
// fixture. The observation itself is human; this test only requires the recipe.
func TestTheDesktopEnvelopeProbeRecipeIsFiled(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "adr", "ADR-023-a-reads-answer-is-the-served-text.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "## Verification Log") {
		t.Error("ADR-023 has no Verification Log section for the Desktop probe")
	}
	if !strings.Contains(got, "Desktop envelope probe") {
		t.Error("ADR-023 Verification Log does not name the Desktop envelope probe")
	}
	if !strings.Contains(got, "two-line") {
		t.Error("the recipe does not say to use a two-line fixture")
	}
}

// TestAnUnrunDesktopProbeIsNotAMissRate: BACKLOG keeps "ADR-023: other hosts"
// open and not blocking. An absent observation is not a miss rate.
func TestAnUnrunDesktopProbeIsNotAMissRate(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "adr", "BACKLOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	i := strings.Index(got, "ADR-023: other hosts")
	if i < 0 {
		t.Fatal("BACKLOG has no ADR-023: other hosts entry")
	}
	window := got[i:]
	if len(window) > 800 {
		window = window[:800]
	}
	if !strings.Contains(window, "Not blocking") {
		t.Error("the other-hosts entry no longer says it is not blocking")
	}
}
