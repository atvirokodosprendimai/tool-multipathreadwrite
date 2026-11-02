package apply

import (
	"strings"
	"testing"
)

// ADR-007: mrw says which path it looked for. A plan path that does not exist,
// where a file mrw served the caller ends in that path, names the served file:
// in the native-versus-MRW run 7 an agent's plan said parser_0.py for
// parsers/parser_0.py, which it had read, and two turns went on "does not
// exist". The pairs: an empty ledger, and a ledger entry whose file is gone,
// name nothing.
func TestAMissingPlanPathNamesTheServedFileItEndsLike(t *testing.T) {
	for _, c := range []struct {
		name string
		seen map[string]Seen
		want string
	}{
		{"served under a directory", map[string]Seen{"parsers/parser_0.py": {}}, "you were served parsers/parser_0.py"},
		{"nothing served", map[string]Seen{}, ""},
		{"served, since removed", map[string]Seen{"old/parser_0.py": {}}, ""},
	} {
		root := t.TempDir()
		write(t, root, "parsers/parser_0.py", "a\nb\n")
		res, err := Apply(root, []Input{
			{Path: "parser_0.py", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 0},
		}, Options{Seen: c.seen})
		if err != nil {
			t.Fatal(err)
		}
		r := res.Hunks[0].Reason
		if res.Failed != 1 || !strings.Contains(r, "parser_0.py does not exist") {
			t.Fatalf("%s: failed=%d reason %q", c.name, res.Failed, r)
		}
		if got := strings.Contains(r, "you were served"); got != (c.want != "") || c.want != "" && !strings.Contains(r, c.want) {
			t.Errorf("%s: reason %q, want %q", c.name, r, c.want)
		}
	}
}
