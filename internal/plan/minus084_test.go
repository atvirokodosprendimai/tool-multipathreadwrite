package plan

import (
	"strings"
	"testing"
)

// ADR-084 (blind reading 03). `@@ f -2 delete` failed as `bad line number ""`,
// a parse error that did not name the form: -M is a read's "from the start",
// which a write address does not take. The refusal names the form a write
// takes; a bare -, N-, N-M and N still parse.
func TestAWriteAddressThatStartsWithMinusNamesTheForm(t *testing.T) {
	_, err := ParseAddr("-2")
	if err == nil || !strings.Contains(err.Error(), "1-2") || !strings.Contains(err.Error(), "read range") {
		t.Errorf(`ParseAddr("-2") = %v, want a refusal naming -2 a read range and 1-2 the write form`, err)
	}
	for _, ok := range []string{"-", "2-", "1-2", "3"} {
		if _, err := ParseAddr(ok); err != nil {
			t.Errorf("ParseAddr(%q) = %v, want it to parse", ok, err)
		}
	}
	// Only a positive digit bound earns a recommendation (Codex review of
	// #252): "--2", "-0" and an overflowing bound are refused without one.
	for _, bad := range []string{"--2", "-0", "-99999999999999999999"} {
		_, err := ParseAddr(bad)
		if err == nil || strings.Contains(err.Error(), ": 1-") {
			t.Errorf("ParseAddr(%q) = %v, want a refusal that recommends no 1-M", bad, err)
		}
	}
	if _, err := Parse(strings.NewReader("@@ f.go -2 delete\n")); err == nil || !strings.Contains(err.Error(), "1-2") {
		t.Errorf("a plan addressing -2: %v, want the write form named", err)
	}
}
