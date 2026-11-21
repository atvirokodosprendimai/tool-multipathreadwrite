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

// outside754Root is a root whose file NAME holds a backslash, which stands for a
// Windows path here so the run is real on unix, beside a Go file the
// {packages}-only scoped check places; the shell file is left outside it.
func outside754Root(t *testing.T, convert bool) string {
	t.Helper()
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("no sh: the check runs its command with sh -c")
	}
	if convert { // the Windows separator, on any platform; off, the production one
		prev := receiptSep
		receiptSep = '\\'
		t.Cleanup(func() { receiptSep = prev })
	}
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
	return root
}

// outsideLine754 is the human `outside the scoped check:` line of out, or fails.
func outsideLine754(t *testing.T, out string) string {
	t.Helper()
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "outside the scoped check:") {
			return l
		}
	}
	t.Fatalf("no outside line in:\n%s", out)
	return ""
}

// outsideScope754 is the check.outside_scope of the --json document in out.
func outsideScope754(t *testing.T, out string) []string {
	t.Helper()
	var r struct {
		Check struct {
			OutsideScope []string `json:"outside_scope"`
		} `json:"check"`
		OutsideScope []string `json:"outside_scope"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &r); err != nil {
		t.Fatalf("receipt: %v\n%s", err, out)
	}
	return append(r.Check.OutsideScope, r.OutsideScope...)
}

// The wiring, which the helper test cannot see: a write's receipt carries
// check.outside_scope, in --json and in its human line, through the same
// conversion.
func TestTheCLIWriteReceiptSpellsOutsideScopeWithSlash(t *testing.T) {
	for _, mode := range []string{"json", "human"} {
		t.Run(mode, func(t *testing.T) {
			root := outside754Root(t, true)
			args := []string{"write", "--check", filepath.Join(root, "p.plan")}
			if mode == "json" {
				args = []string{"write", "--check", "--json", filepath.Join(root, "p.plan")}
			}
			out, code := runIn(t, root, args...)
			if code != 0 {
				t.Fatalf("write exit %d: %s", code, out)
			}
			if mode == "json" {
				if got := outsideScope754(t, out); !slices.Equal(got, []string{"d/x.sh"}) {
					t.Errorf("outside_scope = %q, want [d/x.sh]\n%s", got, out)
				}
				return
			}
			if line := outsideLine754(t, out); strings.Contains(line, `\`) || !strings.Contains(line, "d/x.sh") {
				t.Errorf("the outside line: %s", line)
			}
		})
	}
}

// #754. `mrw check` names its paths in the same receipt, so it spells them the
// same way: the engine keeps the caller's path, which the platform separates.
func TestTheCLICheckReceiptSpellsOutsideScopeWithSlash(t *testing.T) {
	for _, mode := range []string{"json", "human"} {
		t.Run(mode, func(t *testing.T) {
			root := outside754Root(t, true)
			args := []string{"check", "a.go", `d\x.sh`}
			if mode == "json" {
				args = []string{"check", "--json", "a.go", `d\x.sh`}
			}
			out, code := runIn(t, root, args...)
			if code != 0 {
				t.Fatalf("check exit %d: %s", code, out)
			}
			if mode == "json" {
				if got := outsideScope754(t, out); !slices.Equal(got, []string{"d/x.sh"}) {
					t.Errorf("outside_scope = %q, want [d/x.sh]\n%s", got, out)
				}
				return
			}
			if line := outsideLine754(t, out); strings.Contains(line, `\`) || !strings.Contains(line, "d/x.sh") {
				t.Errorf("the outside line: %s", line)
			}
		})
	}
}

// #754. The conversion is by the platform's separator, not by every backslash:
// on a build whose separator is "/" a file really named `d\x.sh` is that name,
// and a receipt must give it back as it is. Driven with the production
// separator, so a Slashed that replaced backslashes whatever the platform goes
// red here and not in the tests above, which force `\`.
func TestTheCLIReceiptKeepsABackslashInAUnixFileName(t *testing.T) {
	if filepath.Separator == '\\' {
		t.Skip("on Windows a backslash is the separator, which the tests above cover")
	}
	for _, cmd := range []string{"write", "check"} {
		for _, mode := range []string{"json", "human"} {
			t.Run(cmd+"/"+mode, func(t *testing.T) {
				root := outside754Root(t, false)
				args := []string{"write", "--check", filepath.Join(root, "p.plan")}
				if cmd == "check" {
					args = []string{"check", "a.go", `d\x.sh`}
				}
				if mode == "json" {
					args = slices.Insert(args, 1, "--json")
				}
				out, code := runIn(t, root, args...)
				if code != 0 {
					t.Fatalf("%s exit %d: %s", cmd, code, out)
				}
				if mode == "json" {
					if got := outsideScope754(t, out); !slices.Equal(got, []string{`d\x.sh`}) {
						t.Errorf("outside_scope = %q, want [d\\x.sh]\n%s", got, out)
					}
					return
				}
				if line := outsideLine754(t, out); !strings.Contains(line, `d\x.sh`) || strings.Contains(line, "d/x.sh") {
					t.Errorf("the outside line changed the file's name: %s", line)
				}
			})
		}
	}
}
