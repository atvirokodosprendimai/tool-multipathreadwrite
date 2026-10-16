package read

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-161 Decision 1. A read in sub mode prints a plan and licenses exactly the
// lines it planned: a line the substitution leaves alone is neither planned nor
// licensed, a text starting with @@ is declared raw, a path is quoted as the
// plan grammar needs, a CRLF line carries no CR, and a path no quoting can
// carry is refused by name with nothing planned.
func TestASubReadPrintsAPlanAndLicensesItsLines(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.go", "Old\nOldX\nxOld\nkeep Old and Old\n")
	write("raw.txt", "@@ Old\n")
	write("crlf.txt", "Old\r\nOldX\r\n")
	write("none.txt", "OldX\n")
	write("has space.txt", "Old\n")
	grep := regexp.MustCompile(`Old`) // wider than the substitution, so OldX is served and left alone
	spec := func(p string) Spec {
		return Spec{Path: p, Raw: p, Ranges: []Range{{Re: grep, Text: "/Old/"}}}
	}
	var lastObs map[string]seen.Observation
	// hdr is a hunk header as the read prints it: pinned to the content it
	// planned from, and a counted raw body whatever the text.
	hdr := func(name string, ln int, content string) string {
		sum := sha256.Sum256([]byte(content))
		return "@@ " + name + " " + strconv.Itoa(ln) + " replace sha=" + hex.EncodeToString(sum[:]) + " body=1 raw=true\n"
	}
	plan := func(paths ...string) (*Sub, map[string][][2]int, string, int) {
		t.Helper()
		sub := &Sub{Re: regexp.MustCompile(`\bOld\b`), Repl: "New"}
		var specs []Spec
		for _, p := range paths {
			specs = append(specs, spec(p))
		}
		var diag bytes.Buffer
		obs, problems := Run(&diag, root, specs, Options{Sub: sub})
		lastObs = obs
		spans := map[string][][2]int{}
		for k, o := range obs {
			spans[k] = o.Spans
		}
		return sub, spans, diag.String(), problems
	}

	sub, spans, diag, problems := plan("a.go", "raw.txt", "crlf.txt", "none.txt", "has space.txt")
	if problems != 0 {
		t.Fatalf("problems=%d:\n%s", problems, diag)
	}
	const aGo = "Old\nOldX\nxOld\nkeep Old and Old\n"
	want := hdr("a.go", 1, aGo) + "New\n" +
		hdr("a.go", 4, aGo) + "keep New and New\n" +
		hdr("raw.txt", 1, "@@ Old\n") + "@@ New\n" +
		hdr("crlf.txt", 1, "Old\r\nOldX\r\n") + "New\n" +
		hdr(`"has space.txt"`, 1, "Old\n") + "New\n"
	if got := sub.Plan.String(); got != want {
		t.Errorf("plan:\n%s\nwant:\n%s", got, want)
	}
	if sub.Sites != 5 || sub.Files != 4 {
		t.Errorf("sites=%d files=%d, want 5 sites in 4 files", sub.Sites, sub.Files)
	}
	if strings.Contains(diag, "==>") && strings.Contains(sub.Plan.String(), "==>") {
		t.Errorf("a header reached the plan:\n%s", sub.Plan.String())
	}
	// The licence is the planned lines and no others.
	if got := spans["a.go"]; !reflect.DeepEqual(got, [][2]int{{1, 1}, {4, 4}}) {
		t.Errorf("a.go licensed %v, want only the planned lines 1 and 4 (OldX and xOld were served but not planned)", got)
	}
	if got := spans["none.txt"]; got == nil || len(got) != 0 {
		t.Errorf("none.txt licensed %#v, want an empty non-nil span set: hashed, nothing planned", got)
	}
	if got := spans["crlf.txt"]; !reflect.DeepEqual(got, [][2]int{{1, 1}}) {
		t.Errorf("crlf.txt licensed %v, want line 1", got)
	}
	// Licensed and not shown (ADR-118): an occurrence count is only good in
	// lines a read showed, and this one showed none.
	for name, o := range lastObs {
		if len(o.Spans) > 0 && (!o.Written || o.Shown == nil || len(o.Shown) != 0) {
			t.Errorf("%s: written=%v shown=%v, want licensed (Written) and nothing shown", name, o.Written, o.Shown)
		}
		if o.ServedLine(1) {
			t.Errorf("%s: line 1 counts as shown", name)
		}
	}

	if runtime.GOOS == "windows" {
		return
	}
	// The header splitter reads a double quote and a backslash as escapes inside
	// a field and a single quote as a letter, so a name holding any of them is
	// double-quoted with those two escaped, whatever mix of quotes it holds.
	write("q\"uote.txt", "Old\n")
	write("b'o\"th.txt", "Old\n")
	for name, want := range map[string]string{
		"q\"uote.txt": hdr(`"q\"uote.txt"`, 1, "Old\n") + "New\n",
		"b'o\"th.txt": hdr(`"b'o\"th.txt"`, 1, "Old\n") + "New\n",
	} {
		sub, _, _, _ = plan(name)
		if got := sub.Plan.String(); got != want {
			t.Errorf("name %q: %q, want %q", name, got, want)
		}
	}
	// A line break cannot be written in a one-line header: the name is refused
	// and the whole read plans and licenses nothing.
	write("n\nl.txt", "Old\n")
	sub, spans, _, _ = plan("a.go", "n\nl.txt")
	if len(sub.Refused) != 1 || !strings.Contains(sub.Refused[0], `n\nl.txt`) {
		t.Errorf("refused = %q, want the one name", sub.Refused)
	}
	if sub.Plan.Len() != 0 || sub.Sites != 0 || len(spans) != 0 {
		t.Errorf("a refused read planned %q (%d sites) and licensed %v, want nothing", sub.Plan.String(), sub.Sites, spans)
	}
}

// The plan a sub read prints is parsed by the same parser a write uses: every
// path, however it is spelled, comes back as the path, and every new text, however it starts, as
// the one counted raw body line (ADR-161).
func TestASubPlanParsesBackToTheSamePathsAndBodies(t *testing.T) {
	root := t.TempDir()
	type c struct{ name, repl string }
	cases := []c{
		{"plain.txt", "@@ y"},
		{"has space.txt", "body=3"},
		{"empty.txt", ""},
		{"bom.txt", string(rune(0xFEFF)) + "z"},
	}
	if runtime.GOOS != "windows" {
		cases = append(cases, c{`both'and".txt`, "x"}, c{`back\slash.txt`, "@@ raw=true"})
	}
	sub := &Sub{Re: regexp.MustCompile(`X`)}
	var specs []Spec
	for _, k := range cases {
		if err := os.WriteFile(filepath.Join(root, k.name), []byte("X\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, Spec{Path: k.name, Raw: k.name, Ranges: []Range{{Re: sub.Re, Text: "/X/"}}})
	}
	for i, k := range cases { // one run per replacement: Repl is the Sub's, not the file's
		sub.Repl = k.repl
		before := sub.Plan.Len()
		var diag bytes.Buffer
		Run(&diag, root, specs[i:i+1], Options{Sub: sub})
		hunks, err := plan.Parse(strings.NewReader(sub.Plan.String()[before:]))
		if err != nil {
			t.Fatalf("%q: the plan does not parse: %v\n%s", k.name, err, sub.Plan.String()[before:])
		}
		h := hunks[0]
		if len(hunks) != 1 || h.Path != k.name || !reflect.DeepEqual(h.Body, []string{k.repl}) || !h.Raw || !h.CountedBody || len(h.SHA) != 64 {
			t.Errorf("%q: parsed back as %+v, want path %q, body %q, raw, counted, a full sha", k.name, h, k.name, k.repl)
		}
	}
}

// A new text a plan line cannot carry is refused, not shortened: a line of a
// mixed-ending file keeps its CR in the text, and the parser would drop it
// (Codex review of ADR-161 T1). Nothing is planned or licensed.
func TestASubRefusesANewTextEndingInACR(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mixed.txt"), []byte("Old\r\nkeep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := &Sub{Re: regexp.MustCompile(`Old`), Repl: "New"}
	var diag bytes.Buffer
	obs, _ := Run(&diag, root, []Spec{{Path: "mixed.txt", Raw: "mixed.txt", Ranges: []Range{{Re: sub.Re, Text: "/Old/"}}}}, Options{Sub: sub})
	if len(sub.Refused) != 1 || !strings.Contains(sub.Refused[0], "mixed.txt:1: the new text ends in a CR byte") {
		t.Errorf("a new text ending in CR: refused %q, want mixed.txt:1 named", sub.Refused)
	}
	if sub.Plan.Len() != 0 || sub.Sites != 0 || len(obs) != 0 {
		t.Errorf("a refused sub planned %q and licensed %v, want nothing", sub.Plan.String(), obs)
	}
}
