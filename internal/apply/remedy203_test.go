package apply

import "testing"

// The stale-number refusal prints a read to paste only for a path that is one
// word in every shell mrw meets: single quotes do not quote in cmd.exe, so the
// command is never quoted, and a path that would need quoting gets a
// description instead (ADR-203).
func TestTheStaleNumberRemedyIsACommandOnlyForAPlainPath(t *testing.T) {
	for p, want := range map[string]string{
		"f.txt":            "mrw read --max-cols 0 f.txt:8",
		"sub/dir/a_b-c.go": "mrw read --max-cols 0 sub/dir/a_b-c.go:8",
		"a":                "a read that shows line 8",
		"ab":               "mrw read --max-cols 0 ab:8",
		"my file.txt":      "a read that shows line 8",
		"it's.txt":         "a read that shows line 8",
		"-f.txt":           "a read that shows line 8",
		"a$b.txt":          "a read that shows line 8",
		"a%b.txt":          "a read that shows line 8",
		"café.txt":         "a read that shows line 8",
		"a\x1bb.txt":       "a read that shows line 8",
		"":                 "a read that shows line 8",
	} {
		if got := staleRemedy("", p, 8); got != want {
			t.Errorf("staleRemedy(%q, 8) = %q, want %q", p, got, want)
		}
	}
}
