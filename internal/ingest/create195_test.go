package ingest

import (
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ADR-195. Zero bytes on --create's standard input read exactly like a producer
// that failed before it printed, so they are refused and the refusal names the
// plan that makes an empty file on purpose (ADR-027's body=0). One byte or more
// is created as before: a lone newline is one empty line, and the CRLF a
// PowerShell pipe sends for an empty string is never trimmed to nothing. The
// content goes through CreateContent first, as cmd/mrw does.
func TestCreateRefusesEmptyInputAndNamesTheEmptyFilePlan(t *testing.T) {
	compile := func(path, raw string) ([]byte, error) {
		content, _ := CreateContent([]byte(raw))
		return CompileCreate(path, content)
	}
	for _, path := range []string{"e.txt", "with space.txt", `q"b\c.txt`} {
		_, err := compile(path, "")
		if err == nil {
			t.Fatalf("%s: zero bytes were compiled, want a refusal", path)
		}
		msg := err.Error()
		if !strings.Contains(msg, "standard input was empty") || !strings.Contains(msg, "body=0") {
			t.Errorf("%s: the refusal does not say the input was empty and name body=0: %s", path, msg)
		}
		// The suggested plan sits between "send the plan " and " to mrw write -".
		_, rest, ok := strings.Cut(msg, "send the plan ")
		suggested, _, ok2 := strings.Cut(rest, " to mrw write -")
		if !ok || !ok2 {
			t.Fatalf("%s: no plan to send in the refusal: %s", path, msg)
		}
		hunks, perr := plan.Parse(strings.NewReader(suggested + "\n"))
		if perr != nil || len(hunks) != 1 {
			t.Fatalf("%s: the suggested plan %q does not parse to one hunk: %v", path, suggested, perr)
		}
		if h := hunks[0]; h.Path != path || string(h.Op) != "create" || len(h.Body) != 0 {
			t.Errorf("%s: the suggested plan is not an empty create of that path: %+v", path, h)
		}
	}
	for _, raw := range []string{"\n", "\r\n", "x"} {
		out, err := compile("ok.txt", raw)
		if err != nil {
			t.Errorf("%q was refused, want it created as before: %v", raw, err)
			continue
		}
		if h := parseOne(t, out); len(h.Body) != 1 {
			t.Errorf("%q: want one line, got %+v", raw, h)
		}
	}
}
