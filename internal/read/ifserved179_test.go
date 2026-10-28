package read

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

const ifServed179Body = "l1\nl2\nl3\nl4\nl5\nl6\n"

// ifServed179 runs a read of a.txt with a ledger entry and the sha the caller
// quotes, and returns the output, the references listed, the observations and
// the problem count.
func ifServed179(t *testing.T, held seen.Observation, quoted string, ranges ...Range) (string, []SameAsServed, map[string]seen.Observation, int) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(ifServed179Body), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(ifServed179Body))
	full := hex.EncodeToString(sum[:])
	if held.SHA == "same" {
		held.SHA = full
	}
	if quoted == "own" {
		quoted = full[:8]
	}
	var out bytes.Buffer
	var refs []SameAsServed
	observed, problems := Run(&out, root, []Spec{{Path: "a.txt", Raw: "a.txt", Ranges: ranges}}, Options{
		Numbers:      true,
		IfServed:     map[string]string{"a.txt": quoted},
		Served:       func(string) seen.Observation { return held },
		SameAsServed: &refs,
	})
	return out.String(), refs, observed, problems
}

func r179(from, to int) Range { return Range{Start: from, End: to, Text: "x"} }

// ADR-179 Decision (d). A spec whose file has the sha the caller quotes, and
// every line of which the ledger reports shown (the union of two reads counts),
// is answered by reference, lists the spans, and records nothing; one line not
// shown, a file that changed since, and a sha that is not the file's are served.
func TestIfServedAnswersASpanTheCallerAndTheLedgerHold(t *testing.T) {
	held := seen.Observation{SHA: "same", Spans: [][2]int{{1, 3}, {4, 5}}}
	text, refs, observed, problems := ifServed179(t, held, "own", r179(2, 5))
	sha := regexp.MustCompile(`sha ([0-9a-f]{8})`).FindStringSubmatch(text)
	if sha == nil || !strings.Contains(text, "-- same as served: lines 2-5 at sha "+sha[1]+"\n") || strings.Contains(text, "l2") || problems != 0 || len(observed) != 0 {
		t.Fatalf("by reference (problems %d, observed %v):\n%s", problems, observed, text)
	}
	if want := []SameAsServed{{Path: "a.txt", SHA: sha[1], Spans: [][2]int{{2, 5}}}}; !reflect.DeepEqual(refs, want) {
		t.Errorf("listed %v, want %v", refs, want)
	}
	text, _, _, _ = ifServed179(t, held, "own", r179(1, 2), r179(4, 5))
	if !strings.Contains(text, "lines 1-2, 4-5 at sha ") {
		t.Errorf("two ranges are one reference line:\n%s", text)
	}
	for name, c := range map[string]struct {
		held   seen.Observation
		quoted string
		r      Range
	}{
		"a line not shown":        {held, "own", r179(2, 6)},
		"the file changed":        {seen.Observation{SHA: "0123456789abcdef", Spans: [][2]int{{1, 6}}}, "own", r179(2, 3)},
		"a sha that is not its":   {held, "deadbeef", r179(2, 3)},
		"nothing in the ledger":   {seen.Observation{}, "own", r179(2, 3)},
		"a whole read is shown":   {seen.Observation{SHA: "same"}, "own", r179(2, 3)},
		"written, none shown yet": {seen.Observation{SHA: "same", Written: true, Shown: [][2]int{}}, "own", r179(2, 3)},
	} {
		text, refs, _, _ := ifServed179(t, c.held, c.quoted, c.r)
		byRef := strings.Contains(text, "same as served")
		wantRef := name == "a whole read is shown"
		if byRef != wantRef || (!wantRef && !strings.Contains(text, "l2")) || (wantRef && len(refs) != 1) {
			t.Errorf("%s: by reference %v, want %v:\n%s", name, byRef, wantRef, text)
		}
	}
}

// ADR-179 Decision (d). A range that matches nothing beside a referenced one is
// reported as without the flag and is a problem, so the read exits 1.
func TestIfServedKeepsAMissedRangesReport(t *testing.T) {
	held := seen.Observation{SHA: "same", Spans: [][2]int{{1, 6}}}
	text, refs, _, problems := ifServed179(t, held, "own", r179(1, 2), Range{Re: regexp.MustCompile("nomatch"), Text: "/nomatch/"})
	if !strings.Contains(text, "-- same as served: lines 1-2 at sha ") || !strings.Contains(text, "!! no match for /nomatch/") || problems != 1 || len(refs) != 1 {
		t.Errorf("a missed range beside a reference (problems %d):\n%s", problems, text)
	}
}
