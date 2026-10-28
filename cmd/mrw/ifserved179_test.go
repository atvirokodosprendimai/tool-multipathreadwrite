package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

const ifServed179Body = "l1\nl2\nl3\nl4\nl5\nl6\n"

var ifServed179Sha = regexp.MustCompile(`sha ([0-9a-f]{8})`)

func ifServed179Tree(t *testing.T) (root, sha string) {
	t.Helper()
	root = grepTree(t, map[string]string{"a.txt": ifServed179Body, "f.json": `{"a": 1}` + "\n"})
	out, code := runIn(t, root, "read", "a.txt")
	m := ifServed179Sha.FindStringSubmatch(out)
	if code != 0 || m == nil {
		t.Fatalf("the first read exited %d:\n%s", code, out)
	}
	return root, m[1]
}

// ADR-179 Decision (d). The caller must quote the sha it holds: a wrong one, a
// malformed one and none at all serve the text (or are refused), however much of
// the ledger covers the lines.
func TestIfServedWithoutTheCallersShaServesTheText(t *testing.T) {
	root, sha := ifServed179Tree(t)
	out, code := runIn(t, root, "read", "--if-served", "a.txt="+sha, "a.txt")
	if code != 0 || !strings.Contains(out, "-- same as served: lines 1-6 at sha "+sha) || strings.Contains(out, "l3") {
		t.Fatalf("by reference exited %d:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "--if-served", "a.txt=deadbeef", "a.txt"); code != 0 || strings.Contains(out, "same as served") || !strings.Contains(out, "l3") {
		t.Errorf("a wrong sha exited %d, want the text:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "--if-served", "a.txt", "a.txt"); code != 2 || !strings.Contains(out, "PATH=SHA") {
		t.Errorf("no sha exited %d, want 2 naming PATH=SHA:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "a.txt"); code != 0 || strings.Contains(out, "same as served") {
		t.Errorf("a read without the flag exited %d:\n%s", code, out)
	}
}

// ADR-179 Decision (d), ADR-118. A file mrw wrote and the caller was not shown is
// served; after a read of lines 1-3 (which serves the line after, 4) those are
// referenced and line 5 is served.
func TestIfServedServesAFileMrwWroteButDidNotShow(t *testing.T) {
	root := grepTree(t, map[string]string{"seed.txt": "x\n"})
	if out, code := runIn(t, root, "write", "--no-check", planFile(t, "@@ w.txt 0 create\n"+ifServed179Body+"@@ x.txt 0 create\n"+ifServed179Body)); code != 0 {
		t.Fatalf("the create exited %d:\n%s", code, out)
	}
	out, code := runIn(t, root, "read", "--stat", "w.txt", "x.txt")
	m := ifServed179Sha.FindStringSubmatch(out)
	if code != 0 || m == nil {
		t.Fatalf("--stat exited %d:\n%s", code, out)
	}
	// Both files hold the same text, so one sha: x.txt shows the written-and-unshown case.
	if out, code = runIn(t, root, "read", "--if-served", "x.txt="+m[1], "x.txt"); code != 0 || strings.Contains(out, "same as served") || !strings.Contains(out, "l6") {
		t.Errorf("a file mrw wrote and nobody was shown exited %d, want the text:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "w.txt:1-3"); code != 0 {
		t.Fatalf("the ranged read exited %d:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "--if-served", "w.txt="+m[1], "w.txt:1-3"); code != 0 || !strings.Contains(out, "-- same as served: lines 1-4 at sha "+m[1]) {
		t.Errorf("the shown lines exited %d, want a reference:\n%s", code, out)
	}
	if out, code = runIn(t, root, "read", "--if-served", "w.txt="+m[1], "w.txt:1-5"); code != 0 || strings.Contains(out, "same as served") || !strings.Contains(out, "l5") {
		t.Errorf("a range reaching an unshown line exited %d, want the text:\n%s", code, out)
	}
}

// ADR-179 Decision (d). `mrw seen --forget` empties the ledger, so a reference
// that held is served again.
func TestIfServedServesAgainAfterForget(t *testing.T) {
	root, sha := ifServed179Tree(t)
	if out, code := runIn(t, root, "read", "--if-served", "a.txt="+sha, "a.txt"); code != 0 || !strings.Contains(out, "same as served") {
		t.Fatalf("by reference exited %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "seen", "--forget"); code != 0 {
		t.Fatalf("--forget exited %d:\n%s", code, out)
	}
	if out, code := runIn(t, root, "read", "--if-served", "a.txt="+sha, "a.txt"); code != 0 || strings.Contains(out, "same as served") || !strings.Contains(out, "l3") {
		t.Errorf("after --forget exited %d, want the text:\n%s", code, out)
	}
}

// ADR-179 Decision (d). --json lists each reference (path, sha, spans), and the
// flag is refused, exit 2, beside every flag that serves no line, prints a plan,
// hides lines or budgets matches, on standard input, and beside --if-changed
// naming the same file.
func TestIfServedIsListedInTheJSONReceipt(t *testing.T) {
	root, sha := ifServed179Tree(t)
	out, code := runIn(t, root, "read", "--json", "--if-served", "a.txt="+sha, "a.txt:2-3")
	var doc struct {
		Receipt struct {
			Same []struct {
				Path  string   `json:"path"`
				SHA   string   `json:"sha"`
				Spans [][2]int `json:"spans"`
			} `json:"same_as_served"`
		} `json:"receipt"`
	}
	if i := strings.Index(out, "{"); code != 0 || i < 0 || json.Unmarshal([]byte(out[i:strings.LastIndex(out, "}")+1]), &doc) != nil ||
		len(doc.Receipt.Same) != 1 || doc.Receipt.Same[0].Path != "a.txt" || doc.Receipt.Same[0].SHA != sha || len(doc.Receipt.Same[0].Spans) != 1 || doc.Receipt.Same[0].Spans[0] != [2]int{2, 4} {
		t.Fatalf("--json exited %d with %+v:\n%s", code, doc.Receipt.Same, out)
	}
	for _, argv := range [][]string{
		{"--quiet", "a.txt"},
		{"--stat", "a.txt"},
		{"--grep", "l", "--count", "a.txt"},
		{"--grep", "l", "--sub", "x", "a.txt"},
		{"--fold", "a.txt"},
		{"--no-comments", "a.txt"},
		{"--outline", "a.txt"},
		{"--json-path", "$.a", "f.json"},
		{"--if-changed", "a.txt=" + sha, "a.txt"},
	} {
		if out, code := runIn(t, root, append([]string{"read", "--if-served", "a.txt=" + sha}, argv...)...); code != 2 || !strings.Contains(out, "--if-served") || strings.Contains(out, "==>") {
			t.Errorf("mrw read --if-served … %v exited %d, want 2 naming --if-served:\n%s", argv, code, out)
		}
	}
	if out, _, code := project178Run(t, root, "x\n", "read", "--if-served", "a.txt="+sha, "-"); code != 2 || !strings.Contains(out, "--if-served") {
		t.Errorf("--if-served on standard input exited %d:\n%s", code, out)
	}
}
