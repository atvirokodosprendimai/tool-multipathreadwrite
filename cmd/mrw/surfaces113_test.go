package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// surfaceRow is one plan in one tree, sent once through `mrw write --json` and
// once through mrw_write, each in a fresh copy of the tree.
type surfaceRow struct {
	name      string
	harness   string
	files     map[string]string
	plan      string
	optOut    bool // --no-check / check: false
	dryRun    bool
	noRead    bool
	ledger    bool // the ledger made read-only after the read
	emptyPATH bool
	badTMP    bool
}

var surfaceRows = []surfaceRow{
	{name: "prose_no_check_due", harness: `{"check":"exit 3"}`, plan: mdPlan},
	{name: "code_no_harness", plan: goPlan},
	{name: "check_passed", harness: `{"check":"exit 0"}`, plan: goPlan},
	{name: "check_failed", harness: `{"check":"exit 3"}`, plan: goPlan},
	{name: "check_timed_out", harness: `{"check":"sleep 5","timeout_seconds":1}`, plan: goPlan},
	{name: "check_could_not_start", harness: `{"check":"exit 0"}`, emptyPATH: true, plan: goPlan},
	{name: "check_could_not_run", harness: `{"check":"exit 0"}`, badTMP: true, plan: goPlan},
	{name: "check_drift", harness: `{"check":"printf x >> a.go"}`, plan: goPlan},
	{name: "opted_out", harness: `{"check":"exit 3"}`, optOut: true, plan: goPlan},
	{name: "hunk_unread", harness: `{"check":"exit 0"}`, noRead: true, plan: goPlan},
	{name: "dry_run", harness: `{"check":"exit 3"}`, dryRun: true, plan: goPlan},
	{name: "harness_malformed", harness: `{`, plan: goPlan},
	{name: "ledger_failed_check_due", harness: `{"check":"exit 0"}`, ledger: true, plan: goPlan},
	{name: "pricing_would_refuse_broke", harness: `{"check":"exit 3"}`, files: map[string]string{"b.go": wrapTail}, plan: "@@ b.go 2 replace anchor=\"func B\"\nfunc B() {}\n"},
}

// ADR-113 T2. The two surfaces decided the same outcome in two places, and the
// MCP copy ran no check. One plan through each surface now leaves the same
// tally and pricing, the same hunk verdicts, and the same check verdict.
func TestTheTwoSurfacesCountAWriteAlike(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the rows drive POSIX checks, read-only modes and PATH")
	}
	needShell(t)
	for _, row := range surfaceRows {
		cli := surfaceRun(t, row, false)
		srv := surfaceRun(t, row, true)
		if cli != srv {
			t.Errorf("%s: the surfaces disagree:\n cli: %s\n mcp: %s", row.name, cli, srv)
		}
	}
}

// surfaceRun sends row's plan through one surface in a fresh tree and returns
// what it left: counts, pricing, hunk statuses, and the check's verdict.
func surfaceRun(t *testing.T, row surfaceRow, viaMCP bool) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	path := os.Getenv("PATH")
	root := t.TempDir()
	files := map[string]string{"a.go": "package a\nfunc A() {}\n", "notes.md": "# notes\nline two\n"}
	reads := []string{"read", "a.go", "notes.md"}
	for n, b := range row.files {
		files[n] = b
		reads = append(reads, n)
	}
	if row.harness != "" {
		files[".quality-harness.json"] = row.harness
	}
	writeFiles(t, root, files)
	if !row.noRead {
		if out, code := runIn(t, root, reads...); code != 0 {
			t.Fatalf("%s: read: exit %d:\n%s", row.name, code, out)
		}
	}
	if row.ledger {
		ledger, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ledger, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
	}
	if row.badTMP {
		t.Setenv("TMPDIR", filepath.Join(tmp, "missing"))
	}
	if row.emptyPATH {
		t.Setenv("PATH", t.TempDir())
	}
	var doc map[string]any
	if viaMCP {
		args := map[string]any{"plan": row.plan}
		if row.optOut {
			args["check"] = false
		}
		if row.dryRun {
			args["dry_run"] = true
		}
		doc = mcpWrite(t, root, args)
	} else {
		argv := []string{"write", "--json"}
		if row.optOut {
			argv = append(argv, "--no-check")
		}
		if row.dryRun {
			argv = append(argv, "--dry-run")
		}
		out, _ := runIn(t, root, append(argv, planFile(t, row.plan))...)
		// runIn appends the exit message after the document; the document is
		// the first JSON value.
		_ = json.NewDecoder(strings.NewReader(out)).Decode(&doc)
	}
	t.Setenv("PATH", path)
	t.Setenv("TMPDIR", tmp)
	stats, _ := runIn(t, root, "stats", "--json")
	var s struct {
		Counts  map[string]int `json:"counts"`
		Pricing map[string]int `json:"pricing"`
	}
	if err := json.Unmarshal([]byte(stats), &s); err != nil {
		t.Fatalf("%s: stats: %v\n%s", row.name, err, stats)
	}
	var hunks []string
	if hs, ok := doc["hunks"].([]any); ok {
		for _, h := range hs {
			if m, ok := h.(map[string]any); ok {
				hunks = append(hunks, fmt.Sprint(m["status"]))
			}
		}
	}
	verdict := "none"
	if c, ok := doc["check"].(map[string]any); ok {
		verdict = fmt.Sprintf("ran=%v exit=%v", c["ran"], c["exit_code"])
	}
	_, drift := doc["drift"]
	return fmt.Sprintf("counts %s pricing %s hunks %v check %s drift %v", sortedCounts(s.Counts), sortedCounts(s.Pricing), hunks, verdict, drift)
}

// mcpWrite sends one mrw_write through the server's own wire and returns its
// structured value, or nil when the answer carried none (a refusal).
func mcpWrite(t *testing.T, root string, args map[string]any) map[string]any {
	t.Helper()
	params, _ := json.Marshal(map[string]any{"name": "mrw_write", "arguments": args})
	req, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": json.RawMessage(params)})
	var out bytes.Buffer
	if err := mcp.Serve(strings.NewReader(string(req)+"\n"), &out, root); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	var resp struct {
		Result struct {
			Structured map[string]any `json:"structuredContent"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil || resp.Error != nil {
		t.Fatalf("mrw_write answered %v / %v:\n%s", err, resp.Error, out.String())
	}
	return resp.Result.Structured
}
