package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// #754. The CLI's check.outside_scope, its human line and its drift are spelled
// with "/" as the rest of the receipt is (ADR-091, ADR-132); driven with `\`.
func TestTheCLIReceiptSpellsOutsideScopeAndDriftWithSlash(t *testing.T) {
	prev := receiptSep
	receiptSep = '\\'
	t.Cleanup(func() { receiptSep = prev })
	c := shownCheck(&check.Result{Ran: true, OutsideScope: []string{`d\x.go`}})
	if want := []string{"d/x.go"}; !slices.Equal(c.OutsideScope, want) {
		t.Errorf("outside_scope = %q, want %q", c.OutsideScope, want)
	}
	if line := c.OutsideLine(); strings.Contains(line, `\`) || !strings.Contains(line, "d/x.go") {
		t.Errorf("the outside line: %s", line)
	}
	text := captured(t, func(f *os.File) { _ = reportDrift(f, shownPaths([]string{`d\y.go`}), 0) })
	if strings.Contains(text, `\`) || !strings.Contains(text, "d/y.go") {
		t.Errorf("the drift line: %s", text)
	}
}

// The wiring, which the helper test cannot see: a write's --json receipt carries
// check.outside_scope through the same conversion. A file whose NAME holds a
// backslash stands for a Windows path here, so the run is real on unix.
func TestTheCLIWriteReceiptSpellsOutsideScopeWithSlash(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no sh: the check runs its command with sh -c")
	}
	prev := receiptSep
	receiptSep = '\\'
	t.Cleanup(func() { receiptSep = prev })
	root := t.TempDir()
	for name, body := range map[string]string{
		`d\x.sh`:                "true\n",
		"a.go":                  "package a\n",
		".quality-harness.json": `{"check":"echo WHOLE","scoped_check":"echo {packages}"}`,
		"p.plan":                "@@ d\\x.sh 1 replace\nfalse\n@@ a.go 1 replace\npackage a // x\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, code := runIn(t, root, "read", `d\x.sh`, "a.go", ".quality-harness.json"); code != 0 {
		t.Fatalf("read exit %d: %s", code, out)
	}
	out, code := runIn(t, root, "write", "--check", "--json", filepath.Join(root, "p.plan"))
	if code != 0 {
		t.Fatalf("write --json exit %d: %s", code, out)
	}
	var r struct {
		Check struct {
			OutsideScope []string `json:"outside_scope"`
		} `json:"check"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &r); err != nil {
		t.Fatalf("receipt: %v\n%s", err, out)
	}
	if want := []string{"d/x.sh"}; !slices.Equal(r.Check.OutsideScope, want) {
		t.Errorf("outside_scope = %q, want %q\n%s", r.Check.OutsideScope, want, out)
	}
}
