package guide

import (
	"strings"
	"testing"
)

// ADR-164 T1. The full contract says sha= does not stand in for anchor=; the
// core does not, because it is at its word ceiling (ADR-141) and the refusal
// reaches the caller who made the mistake.
func TestTheFullGuideSaysShaDoesNotStandInForAnAnchor(t *testing.T) {
	const s = "sha= does not stand in for it"
	if !strings.Contains(CLI(), "A multi-line replace requires anchor= taken from the served first line. "+s+".") {
		t.Errorf("the full guide does not say %q beside the anchor rule", s)
	}
	if strings.Contains(Core(), s) {
		t.Errorf("the core says %q; ADR-164 keeps it out of the core", s)
	}
}
