package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-218 Decision 4, in the engine. Under Options.NoShowing a hunk that names
// its line by number alone is refused when the ledger records no showing of it,
// on a whole-file licence and on a partial one; a shown line, anchor=, --force
// and a read-shaped record land. ADR-200's count of numbered candidates is the
// same with the option on and off.
func TestANumberAloneUnderNoShowing(t *testing.T) {
	withoutAsides(t)
	const body = "a\nb\nc\nd\ne\n"
	sha := seen.SHA([]byte(body))
	rep := func(anchor string) Input {
		return Input{Path: "f.txt", Start: 4, End: 4, Op: "replace", Body: []string{"X"}, Lines: -1, Anchor: anchor}
	}
	run := func(t *testing.T, in Input, obs Seen, opt Options) (Result, string) {
		t.Helper()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		opt.Seen = map[string]Seen{"f.txt": obs}
		res, err := Apply(root, []Input{in}, opt)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(filepath.Join(root, "f.txt"))
		return res, string(got)
	}
	partial := Seen{SHA: sha, Written: true, Spans: [][2]int{{4, 5}}, Shown: [][2]int{}}
	whole := Seen{SHA: sha, Written: true, Shown: [][2]int{}}
	shown := Seen{SHA: sha, Written: true, Spans: [][2]int{{4, 5}}, Shown: [][2]int{{4, 4}}}
	read := Seen{SHA: sha, Spans: [][2]int{{4, 5}}}

	for _, c := range []struct {
		name    string
		in      Input
		obs     Seen
		opt     Options
		refused bool
	}{
		{"partial_refused", rep(""), partial, Options{NoShowing: true}, true},
		{"whole_refused", rep(""), whole, Options{NoShowing: true}, true},
		{"partial_without_the_option_lands", rep(""), partial, Options{}, false},
		{"shown_line_lands", rep(""), shown, Options{NoShowing: true}, false},
		{"read_shaped_record_lands", rep(""), read, Options{NoShowing: true}, false},
		{"anchor_lands", rep("d"), partial, Options{NoShowing: true}, false},
		{"force_lands", rep(""), partial, Options{NoShowing: true, Force: true}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, got := run(t, c.in, c.obs, c.opt)
			if !c.refused {
				if res.Failed != 0 || !res.Applied || got != "a\nb\nc\nX\ne\n" {
					t.Fatalf("the hunk did not land: %+v, bytes %q", res.Hunks, got)
				}
				return
			}
			h := res.Hunks[0]
			if res.Failed != 1 || res.Applied || h.Kind != refusal.StaleNumber || got != body {
				t.Fatalf("an unshown number was not refused: applied %v, %+v, bytes %q", res.Applied, res.Hunks, got)
			}
			if !strings.Contains(h.Reason, "MRW_SHOWN=off") || strings.Contains(h.Reason, "mrw read") {
				t.Errorf("the reason names a read, or not the switch: %s", h.Reason)
			}
		})
	}
	t.Run("pricing_unmoved", func(t *testing.T) {
		// ADR-200's predicate is not the refusal's: a landing on a partial
		// licence is no numbered candidate with the option or without, and one
		// on a whole licence with the line shown is one either way.
		wholeShown := Seen{SHA: sha, Written: true, Shown: [][2]int{{4, 4}}}
		for _, c := range []struct {
			name string
			obs  Seen
			want int
		}{{"partial, shown", shown, 0}, {"whole, shown", wholeShown, 1}} {
			off, _ := run(t, rep(""), c.obs, Options{})
			on, _ := run(t, rep(""), c.obs, Options{NoShowing: true})
			if !off.Applied || !on.Applied || off.StaleNumbered != c.want || on.StaleNumbered != c.want {
				t.Errorf("%s: numbered %d without the option and %d with it, want %d both (applied %v, %v)", c.name, off.StaleNumbered, on.StaleNumbered, c.want, off.Applied, on.Applied)
			}
		}
	})
}
