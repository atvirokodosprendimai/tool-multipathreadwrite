package adversarial

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Stress of the 2026-09-16 dangling-leftover pins (Codex P2s on PR #196).
// Oracle is the spec, not the pin:
//   JSX: if #inner is not a descendant of #accidental-wrapper, the pin is red.
//   Campaign: R>=50 AND every named corpus has a defined FP rate AND that rate <5%.
// Arm 3 does not apply — these are document/fixture pins, not a CLI.
//
// Placement pool (v2), the tidying class plus the attack:
//   0 inside wrapper (attack)
//   1 sibling after empty wrapper (Codex mutant)
//   2 sibling before wrapper
//   3 child of outer
//   4 decoy empty div then inner still in wrapper (descendant; pin may be conservative)
//   5 inner inside decoy inside wrapper (descendant)
//   6 no wrapper
//   7 no inner
//   8 no intended-parent
// Named corpora (v2): plan Step 5 table rows = this repository, Zeus, Playtrix (3).

func TestRandomJsxMarkupPinFailsWhenInnerIsOutsideTheWrapper(t *testing.T) {
	seeds := []int64{54, 1, 7, 13, 99}
	if s := os.Getenv("MRW_SEED"); s != "" {
		seeds = []int64{mustInt64(t, s)}
	}
	const iterations = 400
	for _, seed := range seeds {
		r := rand.New(rand.NewSource(seed))
		for i := 0; i < iterations; i++ {
			place := i % 9
			markup := randomJsxMarkup(r, place)
			pin := jsxInnerSitsBeforeWrapperCloser(markup)
			parent, found := innerParentID(markup)
			outside := !found || !innerHasAncestor(markup, "accidental-wrapper")
			label := fmt.Sprintf("seed=%d iter=%d place=%d parent=%q found=%v pin=%v markup=%q", seed, i, place, parent, found, pin, markup)
			if outside && pin {
				t.Fatalf("%s: pin passed but #inner is not a descendant of #accidental-wrapper", label)
			}
			if !strings.Contains(markup, `id="intended-parent"`) && pin {
				t.Fatalf("%s: pin passed without #intended-parent", label)
			}
		}
	}
}

func TestAJsxCodexMutantIsRejectedAndTheAttackFixtureStillPasses(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "break", "jsx-nest", "App.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	if !jsxInnerSitsBeforeWrapperCloser(string(b)) {
		t.Fatal("attack fixture pin failed")
	}
	mutant := `<div id="outer"><section id="intended-parent"><div id="accidental-wrapper"></div><p id="inner">hello</p></section></div>`
	if jsxInnerSitsBeforeWrapperCloser(mutant) {
		t.Fatal("empty-wrapper sibling #inner still passes the pin")
	}
	if p, ok := innerParentID(mutant); !ok || p != "intended-parent" {
		t.Fatalf("oracle parent=%q ok=%v, want intended-parent", p, ok)
	}
}

func FuzzJsxWrapperPinNeverPanics(f *testing.F) {
	// Panic-only. The descendant property is TestRandomJsxMarkupPinFailsWhenInnerIsOutsideTheWrapper
	// on a well-formed 9-place pool; go-fuzz bytes invent `<id="inner"` tokens the Index pin
	// never claimed to parse.
	f.Add(`<div id="outer"><section id="intended-parent"><div id="accidental-wrapper"><p id="inner">hello</p></div></section></div>`)
	f.Add(`<div id="outer"><section id="intended-parent"><div id="accidental-wrapper"></div><p id="inner">hello</p></section></div>`)
	f.Add(``)
	f.Add(`id="inner"`)
	f.Fuzz(func(t *testing.T, s string) {
		_ = jsxInnerSitsBeforeWrapperCloser(s)
	})
}

func TestAOneCorpusCampaignDoesNotQualify(t *testing.T) {
	loophole := [3]corpusCounts{
		{},
		{would: 50, broke: 50},
		{},
	}
	if campaignQualifiesPlan(loophole) || campaignQualifiesInt(loophole) {
		t.Fatal("Zeus-only 50 TPs with two empty corpora qualified")
	}
	ok := [3]corpusCounts{
		{would: 20, broke: 20},
		{would: 20, broke: 20},
		{would: 20, broke: 20},
	}
	if !campaignQualifiesPlan(ok) || !campaignQualifiesInt(ok) {
		t.Fatal("three corpora of 20 broke/0 held did not qualify as evidence")
	}
	edge := [3]corpusCounts{
		{would: 20, broke: 19, held: 1},
		{would: 20, broke: 20},
		{would: 20, broke: 20},
	}
	if campaignQualifiesPlan(edge) || campaignQualifiesInt(edge) {
		t.Fatal("FP exactly 5% (1/20) qualified; the bar is < 5%")
	}
	fortyNine := [3]corpusCounts{
		{would: 17, broke: 17},
		{would: 16, broke: 16},
		{would: 16, broke: 16},
	}
	if campaignQualifiesPlan(fortyNine) || campaignQualifiesInt(fortyNine) {
		t.Fatal("R=49 qualified; the bar is >= 50")
	}
}

func TestRandomCampaignTablesMatchTheIntegerOracle(t *testing.T) {
	seeds := []int64{54, 1, 7, 13, 99}
	if s := os.Getenv("MRW_SEED"); s != "" {
		seeds = []int64{mustInt64(t, s)}
	}
	const iterations = 400
	for _, seed := range seeds {
		r := rand.New(rand.NewSource(seed))
		for i := 0; i < iterations; i++ {
			var cs [3]corpusCounts
			for c := 0; c < 3; c++ {
				if r.Intn(10) == 0 {
					continue
				}
				cs[c].broke = r.Intn(40)
				cs[c].held = r.Intn(8)
				cs[c].unchecked = r.Intn(5)
				cs[c].would = cs[c].broke + cs[c].held + cs[c].unchecked
			}
			if r.Intn(8) == 0 {
				cs = [3]corpusCounts{
					{would: 20, broke: 20},
					{would: 20, broke: 20},
					{would: 20, broke: 20},
				}
			}
			plan := campaignQualifiesPlan(cs)
			got := campaignQualifiesInt(cs)
			if plan != got {
				t.Fatalf("seed=%d iter=%d plan=%v int=%v cs=%+v", seed, i, plan, got, cs)
			}
		}
	}
}

type corpusCounts struct {
	would, broke, held, unchecked int
}

func campaignQualifiesPlan(cs [3]corpusCounts) bool {
	r := 0
	for _, c := range cs {
		r += c.would
		d := c.broke + c.held
		if d == 0 {
			return false
		}
		if float64(c.held)/float64(d) >= 0.05 {
			return false
		}
	}
	return r >= 50
}

func campaignQualifiesInt(cs [3]corpusCounts) bool {
	r := 0
	for _, c := range cs {
		r += c.would
		d := c.broke + c.held
		if d == 0 {
			return false
		}
		if c.held*20 >= d {
			return false
		}
	}
	return r >= 50
}

func mustInt64(t *testing.T, s string) int64 {
	t.Helper()
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func randomJsxMarkup(r *rand.Rand, place int) string {
	pad := strings.Repeat(" ", r.Intn(3))
	inner := pad + `<p id="inner">hello</p>`
	wrapOpen := `<div id="accidental-wrapper">`
	wrapClose := `</div>`
	intendedOpen := `<section id="intended-parent">`
	intendedClose := `</section>`
	outerOpen := `<div id="outer">`
	outerClose := `</div>`
	switch place {
	case 0:
		return outerOpen + intendedOpen + wrapOpen + inner + wrapClose + intendedClose + outerClose
	case 1:
		return outerOpen + intendedOpen + wrapOpen + wrapClose + inner + intendedClose + outerClose
	case 2:
		return outerOpen + intendedOpen + inner + wrapOpen + wrapClose + intendedClose + outerClose
	case 3:
		return outerOpen + inner + intendedOpen + wrapOpen + wrapClose + intendedClose + outerClose
	case 4:
		return outerOpen + intendedOpen + wrapOpen + `<div id="decoy"></div>` + inner + wrapClose + intendedClose + outerClose
	case 5:
		return outerOpen + intendedOpen + wrapOpen + `<div id="decoy">` + inner + `</div>` + wrapClose + intendedClose + outerClose
	case 6:
		return outerOpen + intendedOpen + inner + intendedClose + outerClose
	case 7:
		return outerOpen + intendedOpen + wrapOpen + wrapClose + intendedClose + outerClose
	default:
		return outerOpen + wrapOpen + inner + wrapClose + outerClose
	}
}

func innerParentID(markup string) (string, bool) {
	type frame struct{ id string }
	var stack []frame
	for i := 0; i < len(markup); {
		if strings.HasPrefix(markup[i:], "</") {
			gt := strings.Index(markup[i:], ">")
			if gt < 0 {
				return "", false
			}
			if len(stack) == 0 {
				return "", false
			}
			stack = stack[:len(stack)-1]
			i += gt + 1
			continue
		}
		if markup[i] == '<' && i+1 < len(markup) && markup[i+1] != '!' && markup[i+1] != '/' {
			gt := strings.Index(markup[i:], ">")
			if gt < 0 {
				return "", false
			}
			open := markup[i+1 : i+gt]
			id := tagID(open)
			if id == "inner" {
				if len(stack) == 0 {
					return "", true
				}
				return stack[len(stack)-1].id, true
			}
			if !strings.HasSuffix(strings.TrimSpace(open), "/") {
				stack = append(stack, frame{id: id})
			}
			i += gt + 1
			continue
		}
		i++
	}
	return "", false
}

func innerHasAncestor(markup, want string) bool {
	type frame struct{ id string }
	var stack []frame
	for i := 0; i < len(markup); {
		if strings.HasPrefix(markup[i:], "</") {
			gt := strings.Index(markup[i:], ">")
			if gt < 0 {
				return false
			}
			if len(stack) == 0 {
				return false
			}
			stack = stack[:len(stack)-1]
			i += gt + 1
			continue
		}
		if markup[i] == '<' && i+1 < len(markup) && markup[i+1] != '!' && markup[i+1] != '/' {
			gt := strings.Index(markup[i:], ">")
			if gt < 0 {
				return false
			}
			open := markup[i+1 : i+gt]
			id := tagID(open)
			if id == "inner" {
				for _, f := range stack {
					if f.id == want {
						return true
					}
				}
				return false
			}
			if !strings.HasSuffix(strings.TrimSpace(open), "/") {
				stack = append(stack, frame{id: id})
			}
			i += gt + 1
			continue
		}
		i++
	}
	return false
}

func tagID(open string) string {
	const key = `id="`
	j := strings.Index(open, key)
	if j < 0 {
		return ""
	}
	rest := open[j+len(key):]
	k := strings.Index(rest, `"`)
	if k < 0 {
		return ""
	}
	return rest[:k]
}
