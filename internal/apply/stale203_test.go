package apply

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-203 T1. `f.txt` held L1 to L10 and mrw wrote NEW after line 3, so the file
// below is what a second plan meets. A hunk that names a line by number alone,
// on a line no read has shown in this version of a file mrw wrote, is refused
// and writes nothing; every other way of saying which line still lands.
func TestANumberAloneIsRefusedOnALineNotShownSinceTheWrite(t *testing.T) {
	const body = "L1\nL2\nL3\nNEW\nL4\nL5\nL6\nL7\nL8\nL9\nL10\n"
	sha := seen.SHA([]byte(body))
	wrote := func(shown ...[2]int) *Seen { return &Seen{SHA: sha, Written: true, Shown: shown} }
	rep := func(n int) Input {
		return Input{Path: "f.txt", Start: n, End: n, Op: "replace", Body: []string{"X"}, Lines: -1}
	}
	del := func(a, b int) Input { return Input{Path: "f.txt", Start: a, End: b, Op: "delete", Lines: -1} }
	ins := func(op string, n int) Input {
		return Input{Path: "f.txt", Start: n, End: n, Op: op, Body: []string{"X"}, Lines: -1}
	}
	with := func(in Input, f func(*Input)) Input { f(&in); return in }
	const ways = "has no recorded showing in this version, and a line number alone cannot say which line you mean. Read it whole (mrw read f.txt:"

	fresh := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}
	read := func(t *testing.T, root string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, "f.txt"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	refused := []struct {
		name string
		ins  []Input
		obs  *Seen
		line int // the line the reason names
	}{
		{"a replace of line 8", []Input{rep(8)}, wrote(), 8},
		{"a replace above the insert, too", []Input{rep(2)}, wrote(), 2},
		{"a delete range names its first line", []Input{del(4, 6)}, wrote(), 4},
		{"a range read from 5 names 4", []Input{del(4, 6)}, wrote([2]int{5, 6}), 4},
		{"a range with only its middle line missing names 5", []Input{del(4, 6)}, wrote([2]int{3, 4}, [2]int{6, 7}), 5},
		{"a range whose first line is shown and last is not names the last", []Input{del(4, 6)}, wrote([2]int{4, 5}), 6},
		{"insert-after 5 names line 5", []Input{ins("insert-after", 5)}, wrote(), 5},
		{"insert-before 5 names line 5", []Input{ins("insert-before", 5)}, wrote(), 5},
		{"a whole-file quiet read shows nothing", []Input{rep(8)}, &Seen{SHA: sha, Written: true}, 8},
		{"N-$", []Input{with(del(8, 11), func(i *Input) { i.End = EOF })}, wrote(), 8},
		{"N,+K", []Input{with(del(8, 8), func(i *Input) { i.RelEnd = 1 })}, wrote(), 8},
	}
	for _, c := range refused {
		t.Run("refused: "+c.name, func(t *testing.T) {
			root := fresh(t)
			res, err := Apply(root, c.ins, Options{Seen: map[string]Seen{"f.txt": *c.obs}})
			if err != nil {
				t.Fatal(err)
			}
			h := res.Hunks[0]
			if res.Applied || res.Failed != 1 || h.Status != StatusFailed || h.Kind != refusal.StaleNumber {
				t.Fatalf("not refused as a stale number: applied %v failed %d, %+v", res.Applied, res.Failed, h)
			}
			if want := "line " + strconv.Itoa(c.line) + " of f.txt " + ways + strconv.Itoa(c.line) + "; a --quiet read, or a line --max-cols cut, shows nothing), address it by pattern, or give anchor="; !strings.Contains(h.Reason, want) {
				t.Errorf("the reason is %q, want it to hold %q", h.Reason, want)
			}
			if got := read(t, root); got != body {
				t.Errorf("a refused plan changed the file:\n%s", got)
			}
		})
	}

	lands := []struct {
		name string
		in   Input
		obs  *Seen
		opt  Options
	}{
		{"anchor=", with(rep(8), func(i *Input) { i.Anchor = "L7" }), wrote(), Options{}},
		{"a pattern", with(rep(8), func(i *Input) { i.Start, i.End, i.StartPat = 0, 0, regexp.MustCompile(`^L7$`) }), wrote(), Options{}},
		{"a line shown since the write", rep(8), wrote([2]int{8, 8}), Options{}},
		{"every line of a range shown", del(4, 6), wrote([2]int{4, 6}), Options{}},
		{"force", rep(8), wrote(), Options{Force: true}},
		{"a plan mrw computed", rep(8), wrote(), Options{Computed: true}},
		{"$ replace", with(rep(11), func(i *Input) { i.Start, i.End = EOF, EOF }), wrote(), Options{}},
		{"insert-after 0", ins("insert-after", 0), wrote(), Options{}},
		{"insert-before 1", ins("insert-before", 1), wrote(), Options{}},
		{"insert-before one past the last line", ins("insert-before", 12), wrote(), Options{}},
		{"a file read whole and never written", rep(8), &Seen{SHA: sha}, Options{}},
		{"a partial licence", rep(8), &Seen{SHA: sha, Written: true, Spans: [][2]int{{8, 8}}}, Options{}},
		{"no ledger", rep(8), nil, Options{}},
	}
	for _, c := range lands {
		t.Run("lands: "+c.name, func(t *testing.T) {
			root := fresh(t)
			if c.obs != nil {
				c.opt.Seen = map[string]Seen{"f.txt": *c.obs}
			}
			res, err := Apply(root, []Input{c.in}, c.opt)
			if err != nil || res.Failed != 0 || !res.Applied || res.Hunks[0].Status != StatusOK {
				t.Fatalf("err %v: not applied: %+v", err, res.Hunks)
			}
		})
	}

	t.Run("anchor= lands on the line that holds L7", func(t *testing.T) {
		root := fresh(t)
		_, err := Apply(root, []Input{with(rep(8), func(i *Input) { i.Anchor = "L7" })}, Options{Seen: map[string]Seen{"f.txt": *wrote()}})
		if err != nil || read(t, root) != strings.Replace(body, "L7\n", "X\n", 1) {
			t.Errorf("err %v:\n%s", err, read(t, root))
		}
	})

	t.Run("every stale hunk of a plan is reported, and a valid sibling on another file is skipped", func(t *testing.T) {
		root := fresh(t)
		if err := os.WriteFile(filepath.Join(root, "g.txt"), []byte("g1\ng2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gsha := seen.SHA([]byte("g1\ng2\n"))
		g := Input{Path: "g.txt", Start: 1, End: 1, Op: "replace", Body: []string{"G"}, Lines: -1, Anchor: "g1"}
		res, err := Apply(root, []Input{rep(8), g, del(9, 9)}, Options{Seen: map[string]Seen{"f.txt": *wrote(), "g.txt": {SHA: gsha}}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Applied || res.Failed != 2 || res.Hunks[0].Kind != refusal.StaleNumber || res.Hunks[2].Kind != refusal.StaleNumber || res.Hunks[1].Status != StatusSkipped {
			t.Fatalf("want both stale hunks failed and the sibling skipped: %+v", res.Hunks)
		}
		if got := read(t, root); got != body {
			t.Errorf("f.txt changed:\n%s", got)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "g.txt")); string(b) != "g1\ng2\n" {
			t.Errorf("g.txt changed: %q", b)
		}
	})

	t.Run("an overlap is reported as an overlap, ahead of the stale number", func(t *testing.T) {
		root := fresh(t)
		anchored := with(del(4, 6), func(i *Input) { i.Anchor = "NEW" })
		second := rep(5)
		second.Index = 1
		res, err := Apply(root, []Input{anchored, second}, Options{Seen: map[string]Seen{"f.txt": *wrote()}})
		if err != nil || res.Applied || res.Hunks[1].Status != StatusFailed || res.Hunks[1].Kind == refusal.StaleNumber || !strings.Contains(res.Hunks[1].Reason, "overlaps an earlier hunk") {
			t.Fatalf("err %v: want the second hunk refused as an overlap: %+v", err, res.Hunks)
		}
	})

	t.Run("an alias observation is the one apply selected", func(t *testing.T) {
		root := fresh(t)
		if err := os.Link(filepath.Join(root, "f.txt"), filepath.Join(root, "g.txt")); err != nil {
			t.Skipf("no hard links here: %v", err)
		}
		res, err := Apply(root, []Input{rep(8)}, Options{Seen: map[string]Seen{"g.txt": *wrote()}})
		if err != nil || res.Failed != 1 || res.Hunks[0].Kind != refusal.StaleNumber {
			t.Errorf("err %v: want the refusal through an alias: %+v", err, res.Hunks)
		}
	})
}
