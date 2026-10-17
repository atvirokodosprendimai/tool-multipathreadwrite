package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestTheSWEBenchSliceIsTheLowestTwentyHashes holds the selection rule BACKLOG
// registered for Stage 3 before any run: the 20 instances with the lowest
// sha256("adr-156:" + instance_id), whatever order the dataset lists them in.
func TestTheSWEBenchSliceIsTheLowestTwentyHashes(t *testing.T) {
	var ids []string
	for i := range 60 {
		ids = append(ids, fmt.Sprintf("org__repo-%d", 1000+i*7))
	}
	hash := func(id string) string {
		s := sha256.Sum256([]byte("adr-156:" + id))
		return hex.EncodeToString(s[:])
	}
	got := SelectSWEBench(ids, 20)
	if len(got) != 20 {
		t.Fatalf("selected %d, want 20", len(got))
	}
	worst := ""
	for _, id := range got {
		worst = max(worst, hash(id))
	}
	for _, id := range ids {
		if !slices.Contains(got, id) && hash(id) < worst {
			t.Errorf("%s was left out though its hash is lower than a selected one", id)
		}
	}
	rev := slices.Clone(ids)
	slices.Reverse(rev)
	if !slices.Equal(SelectSWEBench(rev, 20), got) {
		t.Error("the selection depends on the dataset's order")
	}
	if n := len(SelectSWEBench(ids[:5], 20)); n != 5 {
		t.Errorf("fewer instances than the slice: selected %d, want all 5", n)
	}
}

// TestTheStage4SWESliceIsTheLowestSaltedHashesWithFiveHunks holds ADR-160's
// swe-multi-hunk rule: of the instances whose gold patch has at least 5 `@@ -`
// hunk headers, less an exclusion list, the n with the lowest
// sha256("adr-160:" + id); and the subset whose problem statement contains
// none of the patched paths and none of their basenames but __init__.py.
func TestTheStage4SWESliceIsTheLowestSaltedHashesWithFiveHunks(t *testing.T) {
	patch := func(file string, hunks int) string {
		s := "diff --git a/" + file + " b/" + file + "\n--- a/" + file + "\n+++ b/" + file + "\n"
		for i := range hunks {
			s += fmt.Sprintf("@@ -%d,1 +%d,1 @@\n-x\n+y\n", i*10+1, i*10+1)
		}
		return s
	}
	type row struct {
		ID      string `json:"instance_id"`
		Patch   string `json:"patch"`
		Problem string `json:"problem_statement"`
	}
	var rows []row
	for i := range 40 {
		hunks := 5 + i%3
		if i%4 == 0 {
			hunks = 4 // under the floor
		}
		file := fmt.Sprintf("pkg/mod%d.py", i)
		problem := "something is wrong"
		switch i % 5 {
		case 1:
			problem = "the bug is in " + file
		case 2:
			problem = "see mod" + fmt.Sprint(i) + ".py"
		case 3:
			file = "pkg/__init__.py"
			problem = "fails on import of __init__.py"
		}
		rows = append(rows, row{fmt.Sprintf("org__repo-%d", 2000+i), patch(file, hunks), problem})
	}
	ds := filepath.Join(t.TempDir(), "ds.json")
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ds, b, 0o644); err != nil {
		t.Fatal(err)
	}
	exclude := []string{"org__repo-2001", "org__repo-2002"}
	got, err := SelectMultiHunk(ds, "adr-160:", 5, exclude, 8)
	if err != nil {
		t.Fatal(err)
	}
	var eligible []string
	for i, r := range rows {
		if i%4 != 0 && !slices.Contains(exclude, r.ID) {
			eligible = append(eligible, r.ID)
		}
	}
	if want := SelectSWEBenchSalted(eligible, 8, "adr-160:"); !slices.Equal(got.Selected, want) {
		t.Fatalf("selected %v, want the 8 lowest salted hashes among the eligible: %v", got.Selected, want)
	}
	for _, id := range got.Selected {
		var i int
		fmt.Sscanf(id, "org__repo-%d", &i)
		i -= 2000
		named := i%5 == 1 || i%5 == 2
		if slices.Contains(got.Unnamed, id) == named {
			t.Errorf("%s (problem %q): in the unnamed subset %v, want %v", id, rows[i].Problem, !named, !named)
		}
	}
	if slices.Equal(SelectSWEBenchSalted(eligible, 8, "adr-156:"), got.Selected) {
		t.Error("the salt does not change the order")
	}
	// A rename whose new path git quotes and whose old path it does not: the
	// old path is read alone (the third review of #426).
	mixed := filepath.Join(t.TempDir(), "mixed.json")
	mb, err := json.Marshal([]row{{"org__repo-9", strings.Replace(patch("pkg/old.py", 5), "diff --git a/pkg/old.py b/pkg/old.py", `diff --git a/pkg/old.py "b/pkg/new\"name.py"`, 1), "fails in old.py"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mixed, mb, 0o644); err != nil {
		t.Fatal(err)
	}
	if m, err := SelectMultiHunk(mixed, "adr-160:", 5, nil, 1); err != nil || len(m.Selected) != 1 || len(m.Unnamed) != 0 {
		t.Errorf("a problem naming the renamed file's old path: %+v, %v; want it selected and named", m, err)
	}
}

// sweFixture lays out what a SWE-bench stage reads: a local repository at one
// commit, bench/repos.json naming it, a dataset whose rows carry everything the
// harness's rows carry — hints, the test patch, the gold patch, the tests that
// grade it, each a sentinel — and a selection of two instances on that commit.
func sweFixture(t *testing.T, work string) (selection, dataset string) {
	t.Helper()
	repo := filepath.Join(work, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "calc.py"), []byte("def add(a, b):\n    return a - b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@localhost", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "base"}} {
		if err := git(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	commit, err := gitOut(repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	commit = strings.TrimSpace(commit)
	if err := os.MkdirAll(filepath.Join(work, "bench", "swe-bench"), 0o755); err != nil {
		t.Fatal(err)
	}
	repos, _ := json.Marshal(map[string]Repo{"org/calc": {URL: "https://example.invalid/org/calc", Path: repo}})
	if err := os.WriteFile(filepath.Join(work, "bench", "repos.json"), repos, 0o644); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]string
	for _, id := range []string{"org__calc-1", "org__calc-2", "org__calc-3"} {
		rows = append(rows, map[string]string{
			"instance_id": id, "repo": "org/calc", "base_commit": commit,
			"problem_statement": "add() subtracts; make it add. (" + id + ")",
			"hints_text":        "SENTINEL hint", "test_patch": "SENTINEL test patch", "patch": "SENTINEL gold patch",
			"FAIL_TO_PASS": `["SENTINEL fail to pass"]`, "PASS_TO_PASS": `["SENTINEL pass to pass"]`,
			"image": "img/" + id,
		})
	}
	b, _ := json.Marshal(rows)
	dataset = filepath.Join(work, "dataset.json")
	if err := os.WriteFile(dataset, b, 0o644); err != nil {
		t.Fatal(err)
	}
	selection = filepath.Join(work, "bench", "swe-bench", "selection.json")
	if err := os.WriteFile(selection, []byte(`{"selected":["org__calc-2","org__calc-1"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return selection, dataset
}

// TestTheSWEBenchTaskIsTheProblemStatementAlone holds T5's invariant: no gold
// patch, hint or grading test reaches the agent. A task is built from the
// problem statement and the base commit, in the selection's order, and nothing
// else of its row is kept in it.
func TestTheSWEBenchTaskIsTheProblemStatementAlone(t *testing.T) {
	needGit(t)
	sel, ds := sweFixture(t, t.TempDir())
	ts, err := LoadSWEBench(sel, ds)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 || ts[0].ID != "org__calc-2" || ts[1].ID != "org__calc-1" {
		t.Fatalf("tasks %v, want the selection's two, in its order", ts)
	}
	for _, tk := range ts {
		if tk.Instruction != "add() subtracts; make it add. ("+tk.ID+")" || tk.Shape != ShapeSWEBench || tk.Repo != "org/calc" || len(tk.Commit) != 40 {
			t.Errorf("%s: %+v", tk.ID, tk)
		}
		if tk.swe == nil || tk.swe.image != "img/"+tk.ID {
			t.Errorf("%s: the task does not carry its image", tk.ID)
			continue
		}
		if dump := fmt.Sprintf("%#v %#v", tk, *tk.swe); strings.Contains(dump, "SENTINEL") {
			t.Errorf("%s carries more of its row than the problem statement: %s", tk.ID, dump)
		}
	}
	if err := os.WriteFile(sel, []byte(`{"selected":["org__calc-9"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSWEBench(sel, ds); err == nil || !strings.Contains(err.Error(), "org__calc-9") {
		t.Errorf("a selected id the dataset lacks: %v, want an error naming it", err)
	}
}

// sweStage is a stage over the slice with a stub CLI that applies a fix, a stub
// harness that resolves a prediction holding FIXED, and a stub docker that
// keeps which images are here in a directory and logs every call.
func sweStage(t *testing.T) (RunOptions, string) {
	t.Helper()
	needGit(t)
	work := t.TempDir()
	sel, ds := sweFixture(t, work)
	tasks, err := LoadSWEBench(sel, ds)
	if err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(work, "fix.patch")
	if err := os.WriteFile(patch, []byte("diff --git a/calc.py b/calc.py\n--- a/calc.py\n+++ b/calc.py\n@@ -1,2 +1,2 @@\n def add(a, b):\n-    return a - b\n+    return a + b  # FIXED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	claude, _ := stubClaude(t, work, patch, "done")
	harness := filepath.Join(work, "python")
	if err := os.WriteFile(harness, []byte(`#!/bin/sh
preds=""; id=""; run=""; prev=""
for a in "$@"; do
	case "$prev" in -p) preds="$a";; -i) id="$a";; -id) run="$a";; esac
	prev="$a"
done
echo "$id" >> "`+filepath.Join(work, "harness.log")+`"
d="logs/run_evaluation/$run/$run/$id"; mkdir -p "$d"
if grep -q FIXED "$preds"; then r=true; else r=false; fi
printf '{"%s": {"resolved": %s}}\n' "$id" "$r" > "$d/report.json"
`), 0o755); err != nil {
		t.Fatal(err)
	}
	images := filepath.Join(work, "images")
	if err := os.Mkdir(images, 0o755); err != nil {
		t.Fatal(err)
	}
	docker := filepath.Join(work, "docker")
	if err := os.WriteFile(docker, []byte(`#!/bin/sh
echo "$*" >> "`+filepath.Join(work, "docker.log")+`"
key() { printf '%s' "$1" | tr / _; }
case "$1" in
image)
	[ "$2" = inspect ] && { [ -e "`+images+`/BROKEN" ] && { echo "Cannot connect to the Docker daemon" >&2; exit 1; }; [ -e "`+images+`/$(key "$3")" ] && exit 0; echo "Error: No such image: $3" >&2; exit 1; }
	[ "$2" = rm ] && { rm -f "`+images+`/$(key "$3")"; exit 0; } ;;
pull) touch "`+images+`/$(key "$4")" ;;
ps) printf '%s\n' 'cid1 sweb.eval.org__calc-2.run.1' 'cid2 sweb.eval.org__calc-2.run.10' 'cid3 sweb.eval.org__calc-2.run.1.r1' 'cid4 sweb.eval.org__calc-2.run.1.1712345678' 'cid5 sweb.eval.org__calc-2xrun.1' 'cid6 sweb.eval.org__calc-1.run.1' ;;
esac
exit 0
`), 0o755); err != nil {
		t.Fatal(err)
	}
	canary := filepath.Join(work, "canary.json")
	if err := os.WriteFile(canary, []byte(`{"pass":true,"cli_version":"9.9.9 (Claude Code)"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return RunOptions{Tasks: tasks, Stage: "s3", Out: filepath.Join(work, "out"), Models: []string{"stub-model"}, Configs: []string{"A", "B"},
		Repeats: 1, Seed: 156, Claude: claude, Mrw: stubMrw(t, work), Bench: "/bin/true", ConfigDir: filepath.Join(work, "config"),
		Containment: "bypass", Canary: canary, Budget: time.Minute, Scratch: filepath.Join(work, "scratch"),
		SWE: &SWEGrader{Python: harness, Dataset: ds, Docker: docker}}, work
}

// dockerActions are the stub docker's calls that change something: every one
// but the inspections.
func dockerActions(t *testing.T, work string) string {
	t.Helper()
	b, _ := os.ReadFile(filepath.Join(work, "docker.log"))
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l != "" && !strings.HasPrefix(l, "image inspect ") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime drives Run over the
// slice: every attempt is graded by the harness's report and its receipt says
// so; each instance's attempts run together, its image pulled once before them
// and removed after; and a resume over a finished stage drops an image an
// interrupted run left behind, though none of that instance's attempts runs
// again (the review of #422).
func TestASWEBenchStageIsGradedByTheHarnessOneImageAtATime(t *testing.T) {
	o, work := sweStage(t)
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	got := receipts(t, o.Out)
	if len(got) != 4 {
		t.Fatalf("receipts: %d, want 4", len(got))
	}
	for name, b := range got {
		var r Receipt
		if err := json.Unmarshal(b, &r); err != nil {
			t.Fatal(err)
		}
		if r.Source != "swe-bench" || r.Shape != ShapeSWEBench || !r.Oracle.Solved || r.SilentMisEdit || r.Exclusion != "" {
			t.Errorf("%s: %+v", name, r)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(work, "harness.log")); strings.Count(string(b), "\n") != 4 {
		t.Errorf("the harness graded %q, want each of the 4 attempts", b)
	}
	want := "pull --platform linux/amd64 img/org__calc-2\nimage rm img/org__calc-2\npull --platform linux/amd64 img/org__calc-1\nimage rm img/org__calc-1"
	if got := dockerActions(t, work); got != want {
		t.Errorf("docker did:\n%s\nwant each image pulled once before its instance's attempts and removed after:\n%s", got, want)
	}
	for _, img := range []string{"img_org__calc-2", "img_org__calc-1"} {
		if err := os.WriteFile(filepath.Join(work, "images", img), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Run(o); err != nil {
		t.Fatal(err)
	}
	want += "\nimage rm img/org__calc-2\nimage rm img/org__calc-1"
	if got := dockerActions(t, work); got != want {
		t.Errorf("a resume over a finished stage did:\n%s\nwant both images left behind dropped and none pulled:\n%s", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(work, "harness.log")); strings.Count(string(b), "\n") != 4 {
		t.Errorf("the resume graded again: %q", b)
	}
	// A docker that cannot be asked is not an image that is gone: the
	// stage stops, naming the error and the image (the re-review of #422).
	for _, f := range []string{"img_org__calc-2", "BROKEN"} {
		if err := os.WriteFile(filepath.Join(work, "images", f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Run(o); err == nil || !strings.Contains(err.Error(), "Cannot connect") || !strings.Contains(err.Error(), "img/org__calc-2") {
		t.Errorf("a docker that cannot be asked: %v, want the stage stopped, naming the error and the image", err)
	}
}

// TestAnInterruptedSWEBenchAttemptIsNotGraded: an attempt stopped by a signal
// after it changed a file stops the stage without starting the harness, and the
// stage's error names the image it leaves (the review of #422).
func TestAnInterruptedSWEBenchAttemptIsNotGraded(t *testing.T) {
	o, work := sweStage(t)
	o.Claude = filepath.Join(work, "claude-int")
	script := "#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = \"--version\" ] && { echo \"9.9.9 (Claude Code)\"; exit 0; }; done\n" +
		"git apply \"" + filepath.Join(work, "fix.patch") + "\"\nkill -INT " + strconv.Itoa(os.Getpid()) + "\nsleep 30\n"
	if err := os.WriteFile(o.Claude, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Run(o)
	if err == nil || !strings.Contains(err.Error(), "interrupted") || !strings.Contains(err.Error(), "img/org__calc-2") {
		t.Fatalf("an interrupted attempt: %v, want the stage stopped, naming the image it leaves", err)
	}
	if _, err := os.Stat(filepath.Join(work, "harness.log")); err == nil {
		t.Error("the harness graded an interrupted attempt")
	}
}

// TestAnInterruptWhileGradingStopsTheHarnessAndItsContainer: a signal to bench
// while the harness grades an attempt reaches the harness, removes the
// containers it started, writes no receipt and stops the stage. Before this,
// grading ran under no signal handler, so the interrupt ended bench and left
// the harness and its container running (the Stage 3 pilot, 2026-10-15).
func TestAnInterruptWhileGradingStopsTheHarnessAndItsContainer(t *testing.T) {
	// The test binary must outlive the signal whatever bench does with it.
	held := make(chan os.Signal, 4)
	signal.Notify(held, os.Interrupt)
	defer signal.Stop(held)
	kill := "kill -INT " + strconv.Itoa(os.Getpid())
	for _, c := range []struct {
		name, body, want string
		budget           time.Duration
		brokenPs         bool
	}{
		{name: "a harness the signal stops", body: kill + "\nexec sleep 5\n"},
		{name: "a harness that exits 0 on it", body: "trap 'exit 0' INT\n" + kill + "\nsleep 5 &\nwait\n"},
		// The budget runs out while the harness shuts down: still a signal,
		// not a setup failure with a replacement (the review of #425).
		{name: "a shutdown that outlasts the budget", body: "trap 'sleep 2; exit 1' INT\n" + kill + "\nsleep 5 &\nwait\n", budget: time.Second},
		// A cleanup that failed is said, since a container may be left.
		{name: "a cleanup that fails", body: kill + "\nexec sleep 5\n", brokenPs: true, want: "docker ps"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o, work := sweStage(t)
			slow := filepath.Join(work, "interrupting-python")
			script := "#!/bin/sh\necho started >> \"" + filepath.Join(work, "harness.log") + "\"\n" + c.body
			if err := os.WriteFile(slow, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			o.SWE.Python, o.SWE.Budget = slow, c.budget
			if c.brokenPs {
				broken := filepath.Join(work, "broken-docker")
				if err := os.WriteFile(broken, []byte("#!/bin/sh\n[ \"$1\" = ps ] && { echo 'daemon gone' >&2; exit 1; }\nexec \""+o.SWE.Docker+"\" \"$@\"\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				o.SWE.Docker = broken
			}
			err := Run(o)
			if err == nil || !strings.Contains(err.Error(), "interrupted") || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("an interrupt while grading: %v, want the stage stopped as interrupted, saying %q", err, c.want)
			}
			if b, _ := os.ReadFile(filepath.Join(work, "harness.log")); strings.Count(string(b), "started") != 1 {
				t.Errorf("the harness started %d times, want once: the stage goes no further", strings.Count(string(b), "started"))
			}
			receipts, _ := filepath.Glob(filepath.Join(o.Out, "*.receipt.json"))
			if len(receipts) != 0 {
				t.Errorf("an attempt whose grading was interrupted has a receipt: %v", receipts)
			}
			if got := dockerActions(t, work); !c.brokenPs && !strings.Contains(got, "ps -a --filter name=sweb.eval.") {
				t.Errorf("docker did:\n%s\nwant the interrupted run's containers looked up and removed by name", got)
			}
		})
	}
}

// TestATimedOutHarnessRemovesItsContainer: a harness run past its budget is
// stopped, and the containers it started for that instance and run are
// removed by name, since a killed harness skips its own cleanup (the review
// of #422).
func TestATimedOutHarnessRemovesItsContainer(t *testing.T) {
	o, work := sweStage(t)
	tree := filepath.Join(work, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(o.Tasks[0], tree, nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPatch(tree, filepath.Join(work, "fix.patch")); err != nil {
		t.Fatal(err)
	}
	slow := filepath.Join(work, "slow-python")
	if err := os.WriteFile(slow, []byte("#!/bin/sh\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	g := *o.SWE
	g.Python, g.Budget = slow, time.Second
	if _, err := g.grade(o.Tasks[0], tree, t.TempDir(), "run.1"); err == nil {
		t.Fatal("a harness past its budget graded the attempt")
	}
	// Only the run's own container and the harness's numbered retry of it
	// go: never .10 beside .1, a replacement's .1.r1, or another instance's.
	if got, want := dockerActions(t, work), "ps -a --filter name=sweb.eval. --format {{.ID}} {{.Names}}\nrm -f cid1\nrm -f cid4"; got != want {
		t.Errorf("docker did:\n%s\nwant the run's containers found by name and removed:\n%s", got, want)
	}
}

// TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness: an attempt that changed
// nothing is unsolved, and the harness is not started for it.
func TestAnEmptySWEBenchDiffIsUnsolvedWithoutTheHarness(t *testing.T) {
	o, work := sweStage(t)
	tree := filepath.Join(work, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(o.Tasks[0], tree, nil); err != nil {
		t.Fatal(err)
	}
	g := *o.SWE
	g.Python = filepath.Join(work, "no-such-python")
	v, err := g.grade(o.Tasks[0], tree, t.TempDir(), "run")
	if err != nil || v.Solved {
		t.Errorf("an empty diff: %+v, %v; want unsolved, and no harness started", v, err)
	}
}

// TestABrokenPassToPassTestIsRecorded: the harness's report (swebench 5.0.2,
// harness/grading.py: tests_status.PASS_TO_PASS.failure) sets broke_tests
// beside resolved (ADR-162 Decision 3).
func TestABrokenPassToPassTestIsRecorded(t *testing.T) {
	report := func(resolved bool, p2pFail string) []byte {
		return []byte(`{"x__y-1":{"patch_is_None":false,"patch_exists":true,"patch_successfully_applied":true,"resolved":` + strconv.FormatBool(resolved) +
			`,"tests_status":{"FAIL_TO_PASS":{"success":["t1"],"failure":[]},"PASS_TO_PASS":{"success":["t2"],"failure":[` + p2pFail + `]}}}}`)
	}
	v, err := readReport(report(false, `"test_old (m.T)"`), "x__y-1")
	if err != nil || v.Solved || v.BrokeTests == nil || !*v.BrokeTests {
		t.Errorf("a failing PASS_TO_PASS test: %+v %v", v, err)
	}
	v, err = readReport(report(true, ``), "x__y-1")
	if err != nil || !v.Solved || v.BrokeTests == nil || *v.BrokeTests {
		t.Errorf("no failing PASS_TO_PASS test: %+v %v", v, err)
	}
	v, err = readReport([]byte(`{"x__y-1":{"patch_successfully_applied":false,"resolved":false}}`), "x__y-1")
	if err != nil || v.BrokeTests == nil || *v.BrokeTests {
		t.Errorf("a report with no tests_status: %+v %v", v, err)
	}
	if _, err := readReport(report(true, ``), "other__z-2"); err == nil {
		t.Error("a report that does not name the instance was read")
	}
}

// TestTheStage5PoolIsTheSaltedTwoHunkSetLessEarlierSlices holds ADR-162
// Decision 4: every instance with 2 or more hunks, less two exclusion files'
// ids, ordered by sha256("adr-162:" + id), with the 5-or-more-hunk subset
// beside it; Stages 3's and 4's rules are unchanged by it.
func TestTheStage5PoolIsTheSaltedTwoHunkSetLessEarlierSlices(t *testing.T) {
	patch := func(hunks int) string {
		s := "diff --git a/m.py b/m.py\n--- a/m.py\n+++ b/m.py\n"
		for i := range hunks {
			s += fmt.Sprintf("@@ -%d,1 +%d,1 @@\n-x\n+y\n", i*10+1, i*10+1)
		}
		return s
	}
	type row struct {
		ID      string `json:"instance_id"`
		Patch   string `json:"patch"`
		Problem string `json:"problem_statement"`
	}
	var rows []row
	hunksOf := map[string]int{}
	for i := range 30 {
		id := fmt.Sprintf("org__repo-%d", 3000+i)
		hunksOf[id] = 1 + i%6
		rows = append(rows, row{id, patch(hunksOf[id]), "something is wrong"})
	}
	ds := filepath.Join(t.TempDir(), "ds.json")
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ds, b, 0o644); err != nil {
		t.Fatal(err)
	}
	first, second := []string{"org__repo-3001", "org__repo-3003"}, []string{"org__repo-3005"}
	got, err := SelectMultiHunk(ds, "adr-162:", 2, append(slices.Clone(first), second...), 100)
	if err != nil {
		t.Fatal(err)
	}
	var eligible []string
	for _, r := range rows {
		if hunksOf[r.ID] >= 2 && !slices.Contains(first, r.ID) && !slices.Contains(second, r.ID) {
			eligible = append(eligible, r.ID)
		}
	}
	if want := SelectSWEBenchSalted(eligible, 100, "adr-162:"); !slices.Equal(got.Selected, want) {
		t.Fatalf("selected %v, want every eligible id in adr-162: order %v", got.Selected, want)
	}
	five, err := HunkSubset(ds, got.Selected, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range got.Selected {
		if slices.Contains(five, id) != (hunksOf[id] >= 5) {
			t.Errorf("%s with %d hunks: in the 5+ subset %v", id, hunksOf[id], slices.Contains(five, id))
		}
	}
	if !slices.IsSortedFunc(five, func(a, b string) int { return slices.Index(got.Selected, a) - slices.Index(got.Selected, b) }) {
		t.Errorf("the subset %v is not in the pool's order", five)
	}
}
