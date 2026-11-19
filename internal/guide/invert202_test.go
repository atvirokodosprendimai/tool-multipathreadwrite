package guide

import (
	"strings"
	"testing"
)

// ADR-202. The full instructions teach the inverted match of a stream, and say
// it is standard input only; the core, held to eight short rules, does not.
func TestTheInstructionsNameTheInvertedMatch(t *testing.T) {
	const form = "--grep PATTERN --invert -"
	full := CLI()
	i := strings.Index(full, form)
	if i < 0 {
		t.Fatalf("the full instructions do not name %q", form)
	}
	// The sentence that names the form says where it works.
	sentence := full[strings.LastIndex(full[:i], "\n")+1:]
	sentence = sentence[:strings.Index(sentence, "\n")]
	if !strings.Contains(sentence, "standard input only") {
		t.Errorf("the sentence naming %q does not say standard input only:\n%s", form, sentence)
	}
	if strings.Contains(Core(), "--invert") {
		t.Errorf("the core holds --invert:\n%s", Core())
	}
}
