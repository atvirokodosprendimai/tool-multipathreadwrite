package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ADR-055 T1: the summary line — the one line a caller reads every time —
// carries the advisory count, zero included, and the receipt carries it as a
// key. The balance row fired correctly in the field run and was not acted
// on twice in one hour, because the summary omitted it and the JSON consumer
// printed only the keys the summary had taught it to care about.

// braceTree is a root with one Go file whose first line opens a brace the
// third line closes. Replacing line 1 with a balanced body is the wrap-tail
// shape: one advisory. Replacing it with another `{` line is clean: none.
func braceTree(t *testing.T) string {
	t.Helper()
	return grepTree(t, map[string]string{"f.go": "func A() {\n\treturn\n}\n"})
}

const deltaPlan = "@@ f.go 1 replace anchor=\"func A\"\nfunc A() { return }\n"
const cleanPlan = "@@ f.go 1 replace anchor=\"func A\"\nfunc B() {\n"

func TestTheSummaryLineCountsAdvisories(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := braceTree(t)
	if _, err := readIn(t, root, "f.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--no-check", planFile(t, deltaPlan))
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "0 failed, 1 advisory — applied") {
		t.Errorf("summary does not carry the advisory count:\n%s", out)
	}

	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root = braceTree(t)
	if _, err := readIn(t, root, "f.go"); err != nil {
		t.Fatal(err)
	}
	out, code = writeIn(t, root, "--no-check", planFile(t, cleanPlan))
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "0 failed, 0 advisories — applied") {
		t.Errorf("a clean write must say 0 advisories, not omit the clause:\n%s", out)
	}
}

// --quiet prints only failures and the summary, so the summary is the whole
// receipt — and it must carry the count. --json carries it as a top-level key.
func TestQuietAndJSONCarryTheAdvisoryCount(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := braceTree(t)
	if _, err := readIn(t, root, "f.go"); err != nil {
		t.Fatal(err)
	}
	out, code := writeIn(t, root, "--no-check", "--quiet", planFile(t, deltaPlan))
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if !strings.Contains(out, "1 advisory") {
		t.Errorf("--quiet dropped the advisory count:\n%s", out)
	}

	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root = braceTree(t)
	if _, err := readIn(t, root, "f.go"); err != nil {
		t.Fatal(err)
	}
	js, code := writeIn(t, root, "--no-check", "--json", planFile(t, deltaPlan))
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, js)
	}
	var got struct {
		Advisories *int `json:"advisories"`
	}
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("receipt is not JSON: %v\n%s", err, js)
	}
	if got.Advisories == nil || *got.Advisories != 1 {
		t.Errorf("json receipt lacks advisories: 1:\n%s", js)
	}
}
