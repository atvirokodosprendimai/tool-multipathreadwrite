package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// outcomesGolden is what every outcome of `mrw write` printed, exited and
// counted when ADR-113 T1 began, generated from the binary BEFORE the write
// path moved into internal/writer. MRW_UPDATE_OUTCOMES=1 rewrites it; a change
// to it is a change to the CLI, which T1 promises not to make.
const outcomesGolden = "testdata/outcomes113.golden"

// outcomeRow is one branch of the CLI write action: the tree it starts from,
// what it reads first, the flags, the plan, and the environment it runs under.
type outcomeRow struct {
	name      string
	harness   string            // .quality-harness.json, or "" for none
	files     map[string]string // more files beside a.go and notes.md, read with them
	dir       string            // a directory to create
	gomod     bool              // a go.mod beside a.go
	noRead    bool              // skip the read, so the plan's lines are unread
	partial   bool              // partialTree's read-only directory
	ledger    bool              // the ledger made read-only after the read
	noLedger  bool              // the ledger made unreadable after the read
	emptyPATH bool              // PATH holds no sh
	badTMP    bool              // TMPDIR names a directory that is not there
	depth     bool              // MRW_STEP_DEPTH at the limit
	pre       []string          // plans written first, --no-check, to fill the recent ring
	raw       bool              // args are the whole command line: no plan file is added
	args      []string
	plan      string
}

const wrapTail = "package b\nfunc B() {\n}\n"

var outcomeRows = []outcomeRow{
	// Refused before the plan is read: no tally.
	{name: "flags_check_and_dry_run", harness: `{"check":"exit 0"}`, args: []string{"--check", "--dry-run"}, plan: goPlan},
	{name: "flags_check_and_no_check", harness: `{"check":"exit 0"}`, args: []string{"--check", "--no-check"}, plan: goPlan},
	{name: "then_sh_empty", harness: `{"check":"exit 0"}`, args: []string{"--then-sh", ""}, plan: goPlan},
	{name: "plan_unreadable", harness: `{"check":"exit 0"}`, raw: true, args: []string{"no-such-plan.mrw"}},
	{name: "format_unknown", harness: `{"check":"exit 0"}`, args: []string{"--format", "bogus"}, plan: goPlan},
	// Refused as a document: refused_parse.
	{name: "parse_refused", harness: `{"check":"exit 0"}`, plan: "not a plan\n"},
	{name: "apply_patch_refused", harness: `{"check":"exit 0"}`, args: []string{"--format", "apply_patch"}, plan: "not a patch\n"},
	{name: "search_replace_refused", harness: `{"check":"exit 0"}`, args: []string{"--format", "search_replace"}, plan: "not a block\n"},
	{name: "body_file_missing", harness: `{"check":"exit 0"}`, plan: "@@ a.go 2 replace body=@nope.txt\n"},
	// Refused after it parsed, before anything was written: refused_apply.
	{name: "pointer_unresolved", harness: `{"check":"exit 0"}`, plan: "@@ @1 2 replace\nx\n"},
	{name: "json_path_not_utf8", harness: `{"check":"exit 0"}`, plan: "@@ \xff.go 0 create\nx\n"},
	{name: "harness_malformed", harness: `{`, plan: goPlan},
	{name: "then_undeclared", harness: `{"check":"exit 0"}`, args: []string{"--then", "nosuch"}, plan: goPlan},
	{name: "depth_refused", harness: `{"check":"exit 0"}`, depth: true, plan: goPlan},
	{name: "ledger_unloadable", harness: `{"check":"exit 0"}`, noLedger: true, plan: goPlan},
	{name: "hunk_failed_unread", harness: `{"check":"exit 0"}`, noRead: true, plan: goPlan},
	{name: "dry_run_refused", harness: `{"check":"exit 0"}`, noRead: true, args: []string{"--dry-run"}, plan: goPlan},
	{name: "names_a_directory", harness: `{"check":"exit 0"}`, dir: "d", plan: "@@ d 1 replace\nx\n"},
	// Landed, and what followed.
	{name: "partial_commit", partial: true, args: []string{"--no-check"}, plan: partialPlan},
	{name: "ledger_failed_no_check", ledger: true, harness: `{"check":"exit 0"}`, args: []string{"--no-check"}, plan: goPlan},
	{name: "ledger_failed_check_due", ledger: true, harness: `{"check":"exit 0"}`, plan: goPlan},
	{name: "ledger_failed_with_step", ledger: true, harness: `{"check":"exit 0"}`, args: []string{"--then-sh", "exit 0"}, plan: goPlan},
	{name: "dry_run", harness: `{"check":"exit 0"}`, args: []string{"--dry-run"}, plan: goPlan},
	{name: "applied_prose_no_check_due", harness: `{"check":"exit 3"}`, plan: mdPlan},
	{name: "applied_no_check", harness: `{"check":"exit 3"}`, args: []string{"--no-check"}, plan: goPlan},
	{name: "applied_no_harness", plan: goPlan},
	{name: "check_passed", harness: `{"check":"exit 0"}`, plan: goPlan},
	{name: "check_failed", harness: `{"check":"echo boom; exit 3"}`, plan: goPlan},
	{name: "check_demanded_on_prose", harness: `{"check":"exit 0"}`, args: []string{"--check"}, plan: mdPlan},
	{name: "check_demanded_none_declared", args: []string{"--check"}, plan: mdPlan},
	{name: "check_could_not_run", harness: `{"check":"exit 0"}`, badTMP: true, plan: goPlan},
	{name: "check_could_not_start", harness: `{"check":"exit 0"}`, emptyPATH: true, plan: goPlan},
	{name: "check_timed_out", harness: `{"check":"sleep 5","timeout_seconds":1}`, plan: goPlan},
	{name: "check_inferred_from_gomod", gomod: true, harness: `{"tail_lines":1}`, plan: goPlan},
	{name: "check_scoped_unlink", harness: `{"scoped_check":"echo scoped {files}"}`, plan: "@@ a.go - unlink\n"},
	{name: "check_scoped_rename", harness: `{"scoped_check":"echo scoped {files}"}`, plan: "@@ a.go - rename\nb.txt\n"},
	{name: "check_drift", harness: `{"check":"printf x >> a.go"}`, plan: goPlan},
	{name: "step_passed", harness: `{"check":"exit 0"}`, args: []string{"--then-sh", "echo fine"}, plan: goPlan},
	{name: "step_named_passed", harness: `{"check":"exit 0","steps":{"vet":"echo vetted"}}`, args: []string{"--then", "vet"}, plan: goPlan},
	{name: "step_failed", harness: `{"check":"exit 0"}`, args: []string{"--then-sh", "exit 4", "--then-sh", "echo never"}, plan: goPlan},
	{name: "step_timed_out", harness: `{"check":"exit 0","timeout_seconds":1}`, args: []string{"--then-sh", "sleep 5"}, plan: goPlan},
	{name: "step_after_failed_check", harness: `{"check":"exit 3"}`, args: []string{"--then-sh", "exit 0"}, plan: goPlan},
	{name: "step_could_not_start", harness: `{"check":"exit 0"}`, emptyPATH: true, args: []string{"--then-sh", "true"}, plan: mdPlan},
	{name: "strict_balance_advisory", harness: `{"check":"exit 0"}`, args: []string{"--strict-balance"}, plan: "@@ a.go 2 replace anchor=\"func A\"\nfunc A() {\n"},
	{name: "strict_balance_refused", harness: `{"check":"exit 0"}`, files: map[string]string{"b.go": wrapTail}, args: []string{"--strict-balance"}, plan: "@@ b.go 2 replace anchor=\"func B\"\nfunc B() {}\n"},
	{name: "pricing_would_refuse_held", harness: `{"check":"exit 0"}`, files: map[string]string{"b.go": wrapTail}, plan: "@@ b.go 2 replace anchor=\"func B\"\nfunc B() {}\n"},
	{name: "pricing_would_refuse_broke", harness: `{"check":"exit 3"}`, files: map[string]string{"b.go": wrapTail}, plan: "@@ b.go 2 replace anchor=\"func B\"\nfunc B() {}\n"},
	{name: "pattern_fires", harness: `{"check":"exit 0"}`, pre: []string{"@@ a.go 2 replace\nfunc A() {\n", "@@ a.go 2 replace\nfunc A() {}\n"}, plan: "@@ a.go 2 replace\nfunc A() {\n"},
}

// ADR-113 T1. The CLI write's gates, tallies, check, drift, steps and pricing
// move out of cmd/mrw into internal/writer, and none of its outcomes may move
// with them: every row's exit code, stdout and stats are compared with what the
// binary produced before the move, in --json and in human form.
//
// Not driven here: a check or step interrupted by a signal (the signal tests in
// internal/check and checklog080_test.go), mrw killed during its check
// (TestTheReceiptIsOnStdoutBeforeTheCheckStarts), and a --json encoder that
// fails on stdout, whose order — rendered before Settle counts — is the
// sequence's shape.
func TestTheCLIWriteOutcomesAreUnchanged(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the rows drive POSIX checks, read-only modes and PATH; Linux and macOS cover the shared code")
	}
	needShell(t)
	var got strings.Builder
	for _, row := range outcomeRows {
		for _, mode := range []string{"json", "human"} {
			fmt.Fprintf(&got, "== %s %s\n%s\n", row.name, mode, runOutcome(t, row, mode == "json"))
		}
	}
	if os.Getenv("MRW_UPDATE_OUTCOMES") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(outcomesGolden, []byte(strings.TrimRight(got.String(), "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(outcomesGolden)
	if err != nil {
		t.Fatal(err)
	}
	if g := strings.TrimRight(got.String(), "\n") + "\n"; g != string(want) {
		gs, ws := strings.Split(g, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gs) || i < len(ws); i++ {
			var g, w string
			if i < len(gs) {
				g = gs[i]
			}
			if i < len(ws) {
				w = ws[i]
			}
			if g != w {
				t.Fatalf("outcome moved at golden line %d:\n got: %q\nwant: %q", i+1, g, w)
			}
		}
	}
}

// runOutcome runs one row in a fresh tree and state directory and renders what
// it did: the exit code, the counts stats keeps, and stdout with every path and
// duration that differs between runs replaced by a placeholder.
func runOutcome(t *testing.T, row outcomeRow, asJSON bool) string {
	t.Helper()
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	path := os.Getenv("PATH")
	// The inferred check is `go test`; -count=1 keeps a cached result from
	// printing "(cached)" on one run and not the next.
	t.Setenv("GOFLAGS", "-count=1")
	t.Setenv("MRW_STEP_DEPTH", "")
	var root string
	if row.partial {
		root = partialTree(t)
		t.Setenv("XDG_STATE_HOME", state) // partialTree sets its own; keep ours
		if out, code := runIn(t, root, "read", "a.go", "d/x.txt"); code != 0 {
			t.Fatalf("%s: read: exit %d:\n%s", row.name, code, out)
		}
	} else {
		root = t.TempDir()
		files := map[string]string{"a.go": "package a\nfunc A() {}\n", "notes.md": "# notes\nline two\n"}
		reads := []string{"read", "a.go", "notes.md"}
		for n, b := range row.files {
			files[n] = b
			reads = append(reads, n)
		}
		sort.Strings(reads[3:])
		if row.harness != "" {
			files[".quality-harness.json"] = row.harness
		}
		if row.gomod {
			files["go.mod"] = "module a\n\ngo 1.26\n"
		}
		writeFiles(t, root, files)
		if row.dir != "" {
			if err := os.Mkdir(filepath.Join(root, row.dir), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if !row.noRead {
			if out, code := runIn(t, root, reads...); code != 0 {
				t.Fatalf("%s: read: exit %d:\n%s", row.name, code, out)
			}
		}
	}
	for i, p := range row.pre {
		if out, code := runIn(t, root, "write", "--no-check", planFile(t, p)); code != 0 {
			t.Fatalf("%s: pre-write %d: exit %d:\n%s", row.name, i, code, out)
		}
	}
	if row.ledger || row.noLedger {
		ledger, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		mode := os.FileMode(0o444)
		if row.noLedger {
			mode = 0
		}
		if err := os.Chmod(ledger, mode); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
	}
	var args []string
	if asJSON {
		args = append(args, "--json")
	}
	args = append(args, row.args...)
	plan := ""
	if !row.raw {
		plan = planFile(t, row.plan)
		args = append(args, plan)
	}
	if row.badTMP {
		t.Setenv("TMPDIR", filepath.Join(tmp, "missing"))
	}
	if row.depth {
		t.Setenv("MRW_STEP_DEPTH", "8")
	}
	if row.emptyPATH {
		t.Setenv("PATH", t.TempDir())
	}
	out, code := runIn(t, root, append([]string{"write"}, args...)...)
	t.Setenv("PATH", path)
	t.Setenv("TMPDIR", tmp)
	t.Setenv("MRW_STEP_DEPTH", "")
	if row.noLedger {
		ledger, _ := seen.ReadPath(root)
		_ = os.Chmod(ledger, 0o600)
	}
	stats, scode := runIn(t, root, "stats", "--json")
	var s struct {
		Counts  map[string]int `json:"counts"`
		Landed  int            `json:"landed"`
		Pricing map[string]int `json:"pricing"`
	}
	if err := json.Unmarshal([]byte(stats), &s); scode != 0 || err != nil {
		t.Fatalf("%s: stats --json: exit %d, %v:\n%s", row.name, scode, err, stats)
	}
	cwd, _ := os.Getwd()
	return fmt.Sprintf("exit %d\ncounts %s landed %d\npricing %s\n%s", code, sortedCounts(s.Counts), s.Landed,
		sortedCounts(s.Pricing), normalizeOutcome(out, root, state, tmp, plan, cwd))
}

func sortedCounts(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%d", k, m[k])
	}
	return strings.Join(parts, " ")
}

var (
	durationJSON  = regexp.MustCompile(`"duration_ms": [0-9]+`)
	durationHuman = regexp.MustCompile(`[0-9]+(\.[0-9]+)?(ms|s)\b`)
	checkLog      = regexp.MustCompile(`mrw-(check|step)-[0-9]+\.log`)
	pidNote       = regexp.MustCompile(`pid [0-9]+`)
	asideName     = regexp.MustCompile(`\.mrw-aside-[0-9a-f]+`)
	stateKey      = regexp.MustCompile(`<STATE>/mrw/[0-9a-f]+`)
)

// normalizeOutcome replaces what differs between two runs of the same row —
// temp paths, in both spellings macOS gives them, the test's own directory,
// log names and durations.
func normalizeOutcome(out, root, state, tmp, plan, cwd string) string {
	subs := []struct{ from, to string }{{root, "<ROOT>"}, {state, "<STATE>"}, {tmp, "<TMP>"}, {cwd, "<CWD>"}}
	if plan != "" {
		subs = append([]struct{ from, to string }{{plan, "<PLAN>"}}, subs...)
	}
	for _, p := range subs {
		// The canonical spelling first: on macOS /private/var/… holds /var/…,
		// and replacing the shorter one first leaves /private<ROOT>.
		if real, err := filepath.EvalSymlinks(p.from); err == nil && real != p.from {
			out = strings.ReplaceAll(out, real, p.to)
		}
		out = strings.ReplaceAll(out, p.from, p.to)
	}
	out = durationJSON.ReplaceAllString(out, `"duration_ms": 0`)
	out = checkLog.ReplaceAllString(out, "<LOG>")
	out = normalizeDurations(out)
	out = asideName.ReplaceAllString(out, ".mrw-aside-<X>")
	out = stateKey.ReplaceAllString(out, "<STATE>/mrw/<KEY>")
	return pidNote.ReplaceAllString(out, "pid <PID>")
}

// normalizeDurations replaces each measured duration with <DUR>, and leaves a
// configured one — "timed out after 1s" — as written, so a change to the
// timeout or to how it is said shows in the golden.
func normalizeDurations(s string) string {
	var b strings.Builder
	last := 0
	for _, m := range durationHuman.FindAllStringIndex(s, -1) {
		b.WriteString(s[last:m[0]])
		if strings.HasSuffix(s[:m[0]], "after ") {
			b.WriteString(s[m[0]:m[1]])
		} else {
			b.WriteString("<DUR>")
		}
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}
