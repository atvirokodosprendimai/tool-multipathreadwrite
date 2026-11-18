package apply

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/quote"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-200 T1. After a write that moved lines, a hunk addressed by a number
// from an earlier read names another line, and the ledger cannot tell. Apply
// COUNTS the numbered hunks a guard would refuse and refuses nothing: every
// case below lands with the status and bytes it had before the marks existed.
func TestANumberedHunkAfterMrwsOwnWriteIsCountedNotRefused(t *testing.T) {
	const body = "a\nb\nc\nd\ne\n"
	sha := seen.SHA([]byte(body))
	wrote := func(shown ...[2]int) *Seen { return &Seen{SHA: sha, Written: true, Shown: shown} }
	rep := func(n int, anchor string) Input {
		return Input{Path: "f.txt", Start: n, End: n, Op: "replace", Body: []string{"X"}, Lines: -1, Anchor: anchor}
	}
	del := func(a, b int) Input { return Input{Path: "f.txt", Start: a, End: b, Op: "delete", Lines: -1} }
	ins := func(op string, n int) Input {
		return Input{Path: "f.txt", Start: n, End: n, Op: op, Body: []string{"X"}, Lines: -1}
	}
	with := func(in Input, f func(*Input)) Input { f(&in); return in }

	cases := []struct {
		name            string
		in              Input
		obs             *Seen
		opt             Options
		numbered, unshn int
	}{
		{"the reproduction: a replace of line 4 after a write", rep(4, ""), wrote(), Options{}, 1, 1},
		{"a delete", del(4, 4), wrote(), Options{}, 1, 1},
		{"a delete range", del(3, 4), wrote(), Options{}, 1, 1},
		{"insert-after", ins("insert-after", 4), wrote(), Options{}, 1, 1},
		{"insert-before", ins("insert-before", 4), wrote(), Options{}, 1, 1},
		{"N-$", with(del(4, 5), func(i *Input) { i.End = EOF }), wrote(), Options{}, 1, 1},
		{"N,+K", with(del(4, 4), func(i *Input) { i.RelEnd = 1 }), wrote(), Options{}, 1, 1},
		{"$ alone", with(rep(5, ""), func(i *Input) { i.Start, i.End = EOF, EOF }), wrote(), Options{}, 0, 0},
		{"$-N starts at the last line", with(del(5, 5), func(i *Input) { i.Start, i.End = EOF, 5 }), wrote(), Options{}, 0, 0},
		{"insert-after 0", ins("insert-after", 0), wrote(), Options{}, 0, 0},
		{"insert-before 1", ins("insert-before", 1), wrote(), Options{}, 0, 0},
		{"insert-before one past the last line", ins("insert-before", 6), wrote(), Options{}, 0, 0},
		{"insert-after the last line names it", ins("insert-after", 5), wrote(), Options{}, 1, 1},
		{"anchor=", rep(4, "d"), wrote(), Options{}, 0, 0},
		{"a pattern", with(rep(4, ""), func(i *Input) { i.Start, i.End, i.StartPat = 0, 0, regexp.MustCompile(`^d$`) }), wrote(), Options{}, 0, 0},
		{"a quoted hunk", with(rep(4, ""), func(i *Input) { i.Quote = quote.Mark("f.txt", 4, 4, 0, []string{"d"}) }), wrote(), Options{}, 0, 0},
		{"a computed plan", rep(4, ""), wrote(), Options{Computed: true}, 0, 0},
		{"force", rep(4, ""), wrote(), Options{Force: true}, 0, 0},
		{"no ledger", rep(4, ""), nil, Options{}, 0, 0},
		{"a whole read since the write", rep(4, ""), &Seen{SHA: sha}, Options{}, 0, 0},
		{"a partial read", rep(4, ""), &Seen{SHA: sha, Spans: [][2]int{{4, 4}}}, Options{}, 0, 0},
		{"a written record licensed in part", rep(4, ""), &Seen{SHA: sha, Written: true, Spans: [][2]int{{4, 4}}}, Options{}, 0, 0},
		{"a written record with line 4 shown", rep(4, ""), wrote([2]int{4, 4}), Options{}, 1, 0},
		{"line 3 when only 4 is shown", rep(3, ""), wrote([2]int{4, 4}), Options{}, 1, 1},
		{"line 5 when only 4 is shown", rep(5, ""), wrote([2]int{4, 4}), Options{}, 1, 1},
		{"a range is unshown when its last line is", del(4, 5), wrote([2]int{4, 4}), Options{}, 1, 1},
		{"a range is unshown when its first line is", del(3, 4), wrote([2]int{4, 4}), Options{}, 1, 1},
		{"a range of shown lines", del(4, 5), wrote([2]int{4, 5}), Options{}, 1, 0},
		{"insert-after 4 is judged on line 4", ins("insert-after", 4), wrote([2]int{4, 4}), Options{}, 1, 0},
		{"insert-after 3 is judged on line 3", ins("insert-after", 3), wrote([2]int{4, 4}), Options{}, 1, 1},
		{"insert-before 4 is judged on line 4", ins("insert-before", 4), wrote([2]int{4, 4}), Options{}, 1, 0},
		{"insert-before 5 is judged on line 5", ins("insert-before", 5), wrote([2]int{4, 4}), Options{}, 1, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			run := func(obs *Seen, opt Options) (Result, string) {
				root := t.TempDir()
				if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
				if obs != nil {
					opt.Seen = map[string]Seen{"f.txt": *obs}
				}
				res, err := Apply(root, []Input{c.in}, opt)
				if err != nil {
					t.Fatal(err)
				}
				got, _ := os.ReadFile(filepath.Join(root, "f.txt"))
				return res, string(got)
			}
			res, got := run(c.obs, c.opt)
			if res.Failed != 0 || !res.Applied || res.Hunks[0].Status != StatusOK {
				t.Fatalf("the hunk was not applied: %+v", res.Hunks)
			}
			// The same plan with no ledger is the unmarked engine: same bytes.
			base, want := run(nil, Options{})
			if got != want || base.Hunks[0].Status != res.Hunks[0].Status || base.Hunks[0].Reason != res.Hunks[0].Reason {
				t.Errorf("a mark changed the outcome: bytes %q, want %q", got, want)
			}
			if res.StaleNumbered != c.numbered || res.StaleUnshown != c.unshn {
				t.Errorf("numbered %d unshown %d, want %d and %d", res.StaleNumbered, res.StaleUnshown, c.numbered, c.unshn)
			}
		})
	}

	t.Run("an alias observation is the one apply selected", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(filepath.Join(root, "f.txt"), filepath.Join(root, "g.txt")); err != nil {
			t.Skipf("no hard links here: %v", err)
		}
		res, err := Apply(root, []Input{rep(4, "")}, Options{Seen: map[string]Seen{"g.txt": *wrote()}})
		if err != nil || res.Failed != 0 || res.StaleNumbered != 1 || res.StaleUnshown != 1 {
			t.Errorf("err %v failed %d numbered %d unshown %d, want a landing counted 1 and 1", err, res.Failed, res.StaleNumbered, res.StaleUnshown)
		}
	})

	t.Run("a partial commit keeps the mark of a hunk that did not reach disk", func(t *testing.T) {
		root := t.TempDir()
		for _, name := range []string{"a.txt", "f.txt"} {
			if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		calls := 0
		real := commitRenameFn
		defer func() { commitRenameFn = real }()
		commitRenameFn = func(tr *tree, from, to string) error {
			if calls++; calls >= 2 { // the second file's commit, then the undo of the first
				return errors.New("stale200: rename refused")
			}
			return real(tr, from, to)
		}
		clean := Input{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1}
		res, err := Apply(root, []Input{clean, rep(4, "")}, Options{Seen: map[string]Seen{"a.txt": {SHA: sha}, "f.txt": *wrote()}})
		commitRenameFn = real
		landed := map[string]bool{}
		for _, f := range res.Files {
			landed[f.Path] = f.Written
		}
		if err == nil || !landed["a.txt"] || landed["f.txt"] {
			t.Fatalf("err %v, written %v: the fixture must land a.txt and not the candidate's f.txt", err, landed)
		}
		if res.StaleNumbered != 1 || res.StaleUnshown != 1 {
			t.Errorf("numbered %d unshown %d, want 1 and 1: a guard at validation would have stopped this plan", res.StaleNumbered, res.StaleUnshown)
		}
	})
}
