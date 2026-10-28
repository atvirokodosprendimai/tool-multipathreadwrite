package read

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// A reference that reached the caller stays listed when a missed range's
// diagnostic, printed after it, uses up --max-chars: delivery is judged right
// after the reference prints, as the IfChanged answer does.
func TestIfServedKeepsADeliveredReferenceWhenADiagnosticOverrunsTheBudget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(ifServed179Body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(ifServed179Body))
	full := hex.EncodeToString(sum[:])
	spec := []Spec{{Path: "a.txt", Raw: "a.txt", Ranges: []Range{{Start: 1, End: 2, Text: "1-2"}, {Re: regexp.MustCompile("nomatch"), Text: "/nomatch/"}}}}
	opt := func(refs *[]SameAsServed, withheld *bytes.Buffer) Options {
		return Options{
			Numbers: true, IfServed: map[string]string{"a.txt": full[:8]},
			Served:       func(string) seen.Observation { return seen.Observation{SHA: full, Spans: [][2]int{{1, 6}}} },
			SameAsServed: refs, Withheld: withheld,
		}
	}
	var probe bytes.Buffer
	var none []SameAsServed
	Run(&probe, root, spec, opt(&none, &bytes.Buffer{}))
	cut := strings.Index(probe.String(), "!! no match")
	if cut < 0 || len(none) != 1 {
		t.Fatalf("the unbudgeted read printed\n%s", probe.String())
	}
	var sink, withheld bytes.Buffer
	var refs []SameAsServed
	_, problems := Run(NewBudget(&sink, cut), root, spec, opt(&refs, &withheld))
	if len(refs) != 1 || !strings.Contains(sink.String(), "-- same as served") || problems == 0 {
		t.Errorf("the delivered reference is listed %d times (problems %d):\n%s\n%s", len(refs), problems, sink.String(), withheld.String())
	}
}

// -C widens a pattern match, and the reference is judged on the widened span: the
// match at line 3 with one line of context is lines 2-4, and line 4 was not shown.
func TestIfServedJudgesTheContextAWidenedMatchServes(t *testing.T) {
	held := seen.Observation{SHA: "same", Spans: [][2]int{{1, 3}}}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(ifServed179Body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(ifServed179Body))
	full := hex.EncodeToString(sum[:])
	held.SHA = full
	var out bytes.Buffer
	var refs []SameAsServed
	Run(&out, root, []Spec{{Path: "a.txt", Raw: "a.txt", Ranges: []Range{{Re: regexp.MustCompile("l3"), Text: "/l3/"}}}}, Options{
		Numbers: true, Context: 1, IfServed: map[string]string{"a.txt": full[:8]},
		Served: func(string) seen.Observation { return held }, SameAsServed: &refs,
	})
	if strings.Contains(out.String(), "same as served") || !strings.Contains(out.String(), "l4") || len(refs) != 0 {
		t.Errorf("a widened match reaching an unshown line was answered by reference:\n%s", out.String())
	}
}

// A reference whose own line does not fit --max-chars is withheld, named, and
// not listed.
func TestIfServedNamesAReferenceTheBudgetWithholds(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(ifServed179Body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(ifServed179Body))
	full := hex.EncodeToString(sum[:])
	opt := func(refs *[]SameAsServed, withheld *bytes.Buffer) Options {
		return Options{
			Numbers: true, IfServed: map[string]string{"a.txt": full[:8]}, Withheld: withheld, SameAsServed: refs,
			Served: func(string) seen.Observation { return seen.Observation{SHA: full, Spans: [][2]int{{1, 6}}} },
		}
	}
	spec := []Spec{{Path: "a.txt", Raw: "a.txt"}}
	var probe bytes.Buffer
	var none []SameAsServed
	Run(&probe, root, spec, opt(&none, &bytes.Buffer{}))
	headerOnly := strings.Index(probe.String(), "\n") + 1
	var sink, withheld bytes.Buffer
	var refs []SameAsServed
	_, problems := Run(NewBudget(&sink, headerOnly), root, spec, opt(&refs, &withheld))
	if !strings.Contains(withheld.String(), "a.txt: WITHHELD: --max-chars reached") || len(refs) != 0 || problems != 1 {
		t.Errorf("a reference the budget cut (problems %d, listed %d):\n%q\n%q", problems, len(refs), sink.String(), withheld.String())
	}
}
