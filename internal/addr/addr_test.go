package addr

import "testing"

// The table is the point of this test: every one of these strings reaches BOTH
// parsers, and before ADR-026's review they did not all mean the same thing on
// each. `,+3` was served as lines 1-4 by the read path and refused as an empty
// address by the plan path.
func TestARelativeEndIsCutOnlyFromASingleStart(t *testing.T) {
	ok := []struct {
		in, base string
		n        int
	}{
		{"5,+3", "5", 3},
		{"$,+1", "$", 1},
		{"/func A/,+12", "/func A/", 12},
		{"/a-b/,+2", "/a-b/", 2}, // a "-" inside a pattern is not an end
		{"5", "5", 0},
		{"5-7", "5-7", 0},
		{"$", "$", 0},
		{"/a,+3/", "/a,+3/", 0}, // the tail is "3/", not a number
		{"/a/,/b/", "/a/,/b/", 0},
		{"-", "-", 0},
		{"0", "0", 0},
	}
	for _, c := range ok {
		base, n, err := CutRelative(c.in)
		if err != nil {
			t.Errorf("CutRelative(%q) refused: %v", c.in, err)
			continue
		}
		if base != c.base || n != c.n {
			t.Errorf("CutRelative(%q) = (%q, %d), want (%q, %d)", c.in, base, n, c.base, c.n)
		}
	}

	// Each refusal names the string the caller wrote, so the message can be
	// read without the plan in front of you.
	bad := []struct{ in, names string }{
		{"+3", "A,+3"},
		{",+3", "12,+3"},
		{"5,+0", "+0"},
		{"5-7,+3", "already has an end"},
		{"5-,+3", "already has an end"},
		{"$-5,+3", "already has an end"},
		{"0,+3", "names no line"},
		{"-,+3", "names no line"},
		{"/a/,/b/,+2", "not both"},
	}
	for _, c := range bad {
		got, n, err := CutRelative(c.in)
		if err == nil {
			t.Errorf("CutRelative(%q) = (%q, %d), want a refusal", c.in, got, n)
			continue
		}
		if !contains(err.Error(), c.names) {
			t.Errorf("the refusal of %q does not say %q: %v", c.in, c.names, err)
		}
		if !contains(err.Error(), c.in) {
			t.Errorf("the refusal of %q does not quote what the caller wrote: %v", c.in, err)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
