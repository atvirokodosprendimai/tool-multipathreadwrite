package plan

import (
	"strings"
	"testing"
)

// ADR-155 Decision 6. A JSON path is a read address: a plan address in that
// form is refused by name, and a bare `$` still means the last line.
func TestAPlanAddressRefusesAJSONPath(t *testing.T) {
	for _, a := range []string{"$.a.b", `$["a"]`, "$[0]"} {
		_, err := ParseAddr(a)
		if err == nil || !strings.Contains(err.Error(), "a JSON path is a read address") {
			t.Errorf("ParseAddr(%q) = %v, want the form refused by name", a, err)
		}
	}
	if a, err := ParseAddr("$"); err != nil || a.Start != EOF {
		t.Errorf("ParseAddr(\"$\") = %+v, %v; want the last line", a, err)
	}
}
