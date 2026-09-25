package seen

import "testing"

// ADR-068 T1. The ledger writes a path verbatim (`<sha>  <spans>  <path>`)
// and must read it back verbatim: parseLine trimmed the line, so an
// observation of "x " loaded as "x" and licensed a file nobody had read.
func TestALedgerPathKeepsItsSurroundingSpaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	paths := []string{"x ", " y", "a  b"}
	obs := map[string]Observation{}
	for _, p := range paths {
		obs[p] = Observation{SHA: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	}
	if err := Record(root, obs); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if _, ok := l[p]; !ok {
			t.Errorf("%q was recorded and did not load under its own key; the ledger holds %q", p, keys(l))
		}
	}
	for _, trimmed := range []string{"x", "y"} {
		if _, ok := l[trimmed]; ok {
			t.Errorf("%q loaded though only a spaced name was recorded: a read of one file licenses another", trimmed)
		}
	}
}

func keys(l Ledger) []string {
	out := make([]string, 0, len(l))
	for k := range l {
		out = append(out, k)
	}
	return out
}
