package adversarial

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ── ADR-054 under stress, through the built binary ─────────────────────────
//
// The record's three arms are promises about EXIT CODES and a TALLY, and both
// are only observable from outside the process. So this file builds mrw once
// and drives it the way a caller does, with the root, the harness, the plan
// and the flags drawn at random, and compares every run against an oracle
// written from the Decision rather than from the code:
//
//   --check --no-check                → 2, nothing written
//   --check --dry-run                 → 2, nothing written
//   --dry-run                         → 0, nothing written, no check
//   --no-check                        → 0, written, no check, tally applied
//   --check   (a demand)              → no command 2 (check_not_run)
//                                       / passing 0 (applied) / failing 3 (failed_check)
//   default                           → runs only when some written path is
//                                       not prose AND a command exists;
//                                       otherwise 0, no check, applied
//
// A seed is printed on failure and honoured from MRW_SEED for a replay.

var (
	binOnce sync.Once
	binPath string
	binErr  error
)

func mrwBinary(t *testing.T) string {
	t.Helper()
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "mrw-adv-")
		if err != nil {
			binErr = err
			return
		}
		name := "mrw"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		binPath = filepath.Join(dir, name)
		out, err := exec.Command("go", "build", "-o", binPath, "../../cmd/mrw").CombinedOutput()
		if err != nil {
			binErr = fmt.Errorf("building mrw: %v\n%s", err, out)
		}
	})
	if binErr != nil {
		t.Fatal(binErr)
	}
	return binPath
}

// run executes mrw with its own state directory and returns combined output
// and the exit code. Stdout and stderr are merged on purpose: the oracle asks
// "did the word check appear anywhere the caller can see".
func run(t *testing.T, state, root string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(mrwBinary(t), append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running mrw: %v\n%s", err, out)
		}
		code = ee.ExitCode()
	}
	return string(out), code
}

type harness int

const (
	harnessNone harness = iota // no .quality-harness.json, no go.mod
	harnessPass                // {"check":"exit 0"}
	harnessFail                // {"check":"exit 3"}
)

func (h harness) String() string {
	return [...]string{"none", "pass", "fail"}[h]
}

var matrixPaths = []string{
	"a.go", "b.rs", "Makefile", "data.jsonl", "Cargo.toml", "x.yml",
	"notes.md", "README.MD", "doc.txt", "spec.rst", "page.adoc", "weird.md.go",
}

var matrixFlags = [][]string{
	nil,
	{"--check"},
	{"--no-check"},
	{"--dry-run"},
	{"--check", "--no-check"},
	{"--check", "--dry-run"},
	{"--no-check", "--dry-run"},
	{"--json"},
	{"--json", "--check"},
	{"--quiet"},
	{"--quiet", "--no-check"},
}

var checkLine = regexp.MustCompile(`(?m)^check (\(declared\)|\(inferred\)|PASS|FAIL|SKIPPED)|"check":\s*\{`)

func has(flags []string, f string) bool {
	for _, x := range flags {
		if x == f {
			return true
		}
	}
	return false
}

func TestRandomisedWriteMatrixMatchesTheExitCodeOracle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the declared checks are sh -c scripts")
	}
	seed := int64(54)
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	r := rand.New(rand.NewSource(seed))
	const iterations = 120

	for i := 0; i < iterations; i++ {
		// A root with 1–3 distinct files, a harness, and a plan touching
		// a random non-empty subset of them.
		n := 1 + r.Intn(3)
		perm := r.Perm(len(matrixPaths))[:n]
		files := map[string]string{}
		var names []string
		for _, k := range perm {
			p := matrixPaths[k]
			files[p] = "line one\nline two {\nline three\n"
			names = append(names, p)
		}
		h := harness(r.Intn(3))
		switch h {
		case harnessPass:
			files[".quality-harness.json"] = `{"check":"exit 0"}`
		case harnessFail:
			files[".quality-harness.json"] = `{"check":"exit 3"}`
		}
		root := tree(t, files)
		state := t.TempDir()
		flags := matrixFlags[r.Intn(len(matrixFlags))]

		touched := names[:1+r.Intn(len(names))]
		var plan strings.Builder
		for _, p := range touched {
			// Random op, single line 2, body of random brace balance.
			switch r.Intn(3) {
			case 0:
				fmt.Fprintf(&plan, "@@ %s 2 replace anchor=\"line two\"\nline two }\n", p)
			case 1:
				fmt.Fprintf(&plan, "@@ %s 2 insert-after\n{ inserted\n", p)
			default:
				fmt.Fprintf(&plan, "@@ %s 2 delete\n", p)
			}
		}
		planFile := filepath.Join(t.TempDir(), "p.mrw")
		if err := os.WriteFile(planFile, []byte(plan.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, code := run(t, state, root, append([]string{"read"}, touched...)...); code != 0 {
			t.Fatalf("seed=%d iter=%d: read failed with %d", seed, i, code)
		}
		before := map[string]string{}
		for _, p := range touched {
			before[p] = readFile(t, root, p)
		}

		out, code := run(t, state, root, append(append([]string{"write"}, flags...), planFile)...)
		label := fmt.Sprintf("seed=%d iter=%d harness=%s flags=%v touched=%v\n%s", seed, i, h, flags, touched, out)

		// ── the oracle ──
		demand, optOut, dry := has(flags, "--check"), has(flags, "--no-check"), has(flags, "--dry-run")
		code0 := false
		for _, p := range touched {
			code0 = code0 || !apply.IsProse(p)
		}
		hasCommand := h != harnessNone

		var wantExit int
		wantWritten, wantCheck := true, false
		wantTally := "applied"
		switch {
		case demand && optOut, demand && dry:
			wantExit, wantWritten, wantTally = 2, false, ""
		case dry:
			wantExit, wantWritten = 0, false
			wantTally = "" // a dry run's tally row is not this record's promise
		case optOut:
			wantExit = 0
		case demand:
			wantCheck = true
			switch h {
			case harnessNone:
				wantExit, wantTally = 2, "check_not_run"
			case harnessPass:
				wantExit = 0
			case harnessFail:
				wantExit, wantTally = 3, "failed_check"
			}
		default:
			if code0 && hasCommand {
				wantCheck = true
				if h == harnessFail {
					wantExit, wantTally = 3, "failed_check"
				}
			}
		}

		if code != wantExit {
			t.Fatalf("%s\nexit %d, oracle says %d", label, code, wantExit)
		}
		if got := checkLine.MatchString(out); got != wantCheck {
			t.Fatalf("%s\ncheck line present=%v, oracle says %v", label, got, wantCheck)
		}
		for _, p := range touched {
			changed := readFile(t, root, p) != before[p]
			if changed != wantWritten {
				t.Fatalf("%s\n%s changed=%v, oracle says written=%v", label, p, changed, wantWritten)
			}
		}
		if wantTally != "" {
			js, scode := run(t, state, root, "stats", "--json")
			if scode != 0 {
				t.Fatalf("%s\nstats exit %d", label, scode)
			}
			var got struct {
				Counts map[string]int `json:"counts"`
				Landed int            `json:"landed"`
				Failed int            `json:"failed_check_of_landed"`
			}
			if err := json.Unmarshal([]byte(js), &got); err != nil {
				t.Fatalf("%s\nstats --json: %v\n%s", label, err, js)
			}
			if got.Counts[wantTally] != 1 {
				t.Fatalf("%s\ntally %v, oracle says one %s", label, got.Counts, wantTally)
			}
			if got.Landed != 1 {
				t.Fatalf("%s\nlanded=%d after one landed write (%s)", label, got.Landed, wantTally)
			}
			if (got.Failed == 1) != (wantTally == "failed_check") {
				t.Fatalf("%s\nfailed_check_of_landed=%d for tally %s", label, got.Failed, wantTally)
			}
			for _, k := range []string{"applied", "refused_parse", "refused_apply", "check_not_run", "failed_check"} {
				if _, ok := got.Counts[k]; !ok {
					t.Fatalf("%s\nstats --json omits %q", label, k)
				}
			}
		}
	}
}

// ── stats arithmetic over a random outcome sequence ────────────────────────
//
// One root, one failing harness, a random walk over the four ways a write
// can land or be refused there, then the landed line is parsed back and
// checked against the JSON counts: N = applied + failed_check +
// check_not_run, F = failed_check, and the percentage is F/N.
var landedLine = regexp.MustCompile(`landed writes: (\d+); failed_check (\d+) of those \(([\d.]+)%\)`)

func TestStatsLandedLineIsTheArithmeticItClaims(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the declared check is an sh -c script")
	}
	seed := int64(54)
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	r := rand.New(rand.NewSource(seed))

	// Two roots share one state directory but have separate tallies, so
	// check_not_run needs its own harness-less root; the walk alternates.
	failRoot := tree(t, map[string]string{
		"a.go": "l1\nl2\nl3\n", "n.md": "l1\nl2\nl3\n", ".quality-harness.json": `{"check":"exit 3"}`,
	})
	state := t.TempDir()
	run(t, state, failRoot, "read", "a.go", "n.md")

	want := map[string]int{}
	steps := 6 + r.Intn(10)
	for s := 0; s < steps; s++ {
		plan := filepath.Join(t.TempDir(), "p.mrw")
		var doc, outcome string
		var flags []string
		switch r.Intn(5) {
		case 0: // --no-check on code → applied
			doc, flags, outcome = "@@ a.go 2 insert-after\nx\n", []string{"--no-check"}, "applied"
		case 1: // prose skip → applied
			doc, outcome = "@@ n.md 2 insert-after\nx\n", "applied"
		case 2: // default on code, failing check → failed_check
			doc, outcome = "@@ a.go 2 insert-after\nx\n", "failed_check"
		case 3: // demand on prose, failing check → failed_check
			doc, flags, outcome = "@@ n.md 2 insert-after\nx\n", []string{"--check"}, "failed_check"
		default: // a plan that does not parse → refused_parse (not landed)
			doc, outcome = "@@ a.go 2 frobnicate\nx\n", "refused_parse"
		}
		if err := os.WriteFile(plan, []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		run(t, state, failRoot, append(append([]string{"write"}, flags...), plan)...)
		want[outcome]++
	}

	js, _ := run(t, state, failRoot, "stats", "--json")
	var got struct {
		Plans  int            `json:"plans"`
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
		Failed int            `json:"failed_check_of_landed"`
	}
	if err := json.Unmarshal([]byte(js), &got); err != nil {
		t.Fatalf("seed=%d: %v\n%s", seed, err, js)
	}
	for k, v := range want {
		if got.Counts[k] != v {
			t.Errorf("seed=%d: counts[%s]=%d, walk recorded %d\n%s", seed, k, got.Counts[k], v, js)
		}
	}
	landed := got.Counts["applied"] + got.Counts["failed_check"] + got.Counts["check_not_run"]
	if got.Landed != landed || got.Failed != got.Counts["failed_check"] || got.Plans != steps {
		t.Errorf("seed=%d: landed=%d failed=%d plans=%d; want %d/%d/%d\n%s", seed, got.Landed, got.Failed, got.Plans, landed, got.Counts["failed_check"], steps, js)
	}

	human, _ := run(t, state, failRoot, "stats")
	m := landedLine.FindStringSubmatch(human)
	if m == nil {
		t.Fatalf("seed=%d: no landed line:\n%s", seed, human)
	}
	n, _ := strconv.Atoi(m[1])
	f, _ := strconv.Atoi(m[2])
	pct, _ := strconv.ParseFloat(m[3], 64)
	if n != landed || f != got.Counts["failed_check"] {
		t.Errorf("seed=%d: human line %s vs json landed=%d failed=%d", seed, m[0], landed, got.Counts["failed_check"])
	}
	if wantPct := 100 * float64(f) / float64(n); fmt.Sprintf("%.1f", wantPct) != m[3] {
		t.Errorf("seed=%d: percentage %.1f, want %.1f", seed, pct, wantPct)
	}
	for _, name := range []string{"applied", "refused_parse", "refused_apply", "check_not_run", "failed_check"} {
		if !strings.Contains(human, name) {
			t.Errorf("seed=%d: human stats omits %q:\n%s", seed, name, human)
		}
	}
}

// ── the Zeus case the record says arm 2 cannot see, and arm 1 can ──────────
//
// A balanced insert-before of complete test functions, landing inside an
// existing function body. Delimiter arithmetic sees 0 → 0 and says nothing;
// the record says so. The compiler sees one function unclosed and a stray
// closer at the end. With the check inferred from go.mod, the write must
// exit 3 — the tree is changed and unverified — and the receipt must carry
// no balance row, because a row there would be a lie about what the arm saw.
func TestABalancedInsertInTheWrongPlaceIsCaughtByTheCheckNotTheBalance(t *testing.T) {
	root := tree(t, map[string]string{
		"go.mod": "module zeus\n\ngo 1.26\n",
		"a.go":   "package zeus\n\nfunc A() int {\n\tx := 1\n\treturn x\n}\n",
	})
	state := t.TempDir()
	run(t, state, root, "read", "a.go")
	plan := filepath.Join(t.TempDir(), "p.mrw")
	body := "@@ a.go 5 insert-before\nfunc TestOne() {\n\t_ = 1\n}\nfunc TestTwo() {\n\t_ = 2\n}\n"
	if err := os.WriteFile(plan, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", plan)
	if code != 3 {
		t.Fatalf("a write that broke the build exited %d, want 3:\n%s", code, out)
	}
	if strings.Contains(out, "balance") {
		t.Errorf("the balance row fired on a net-0 insert; the record says this case is invisible to it:\n%s", out)
	}
	if !strings.Contains(out, "check FAIL") {
		t.Errorf("the inferred check did not report FAIL:\n%s", out)
	}
	if !strings.Contains(readFile(t, root, "a.go"), "func TestTwo()") {
		t.Error("exit 3 promised a changed tree and the insert is not there")
	}
}

// ── the receipt's balance row can never move an exit code ──────────────────
//
// Same plan, same file, two extensions. The .go copy prints a row; the .md
// copy does not; both exit 0 with --no-check, and both files hold the same
// bytes afterwards. Visibility only.
func TestTheBalanceRowNeverChangesTheVerdictOrTheBytes(t *testing.T) {
	root := tree(t, map[string]string{
		"f.go": "a {\nb\nc }\n",
		"f.md": "a {\nb\nc }\n",
	})
	state := t.TempDir()
	run(t, state, root, "read", "f.go", "f.md")
	plan := filepath.Join(t.TempDir(), "p.mrw")
	doc := "@@ f.go 1 replace anchor=\"a {\"\na\n@@ f.md 1 replace anchor=\"a {\"\na\n"
	if err := os.WriteFile(plan, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", "--no-check", plan)
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	rows := 0
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "balance {") {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("balance rows = %d, want exactly one (the .go hunk):\n%s", rows, out)
	}
	if readFile(t, root, "f.go") != readFile(t, root, "f.md") {
		t.Error("the same plan produced different bytes on .go and .md")
	}
}
