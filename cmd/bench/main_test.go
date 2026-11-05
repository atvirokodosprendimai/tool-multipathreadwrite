package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench"
)

// TestTheMineFloorIsAFlag: ADR-160 Decision 2 raises the wide-rename floor to
// 25 occurrences across 12 files when A is at ceiling, and the redraw is made
// by bench mine itself (T6). Without the flags the floor is T2's, 15 and 8.
func TestTheMineFloorIsAFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	repo := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.21\n",
		"a/a.go": "package a\n\n// Thing does nothing.\nfunc Thing() {}\n",
		"b/b.go": "package b\n\nimport \"example.com/m/a\"\n\n// B calls a.Thing twice.\nfunc B() {\n\ta.Thing()\n\ta.Thing()\n}\n",
		// check-coupled: "active" is held by one test file beside it, "ready" by two
		"c/c.go":       "package c\n\n// S is a status.\nfunc S() string { return \"active\" }\n",
		"c/c_test.go":  "package c\n\nimport \"testing\"\n\nfunc TestS(t *testing.T) {\n\tif S() != \"active\" {\n\t\tt.Fatal(S())\n\t}\n}\n",
		"d/d.go":       "package d\n\n// R is a state.\nfunc R() string { return \"ready\" }\n",
		"d/d1_test.go": "package d\n\nimport \"testing\"\n\nfunc TestR(t *testing.T) {\n\tif R() != \"ready\" {\n\t\tt.Fatal(R())\n\t}\n}\n",
		"d/d2_test.go": "package d\n\nimport \"testing\"\n\nfunc TestRAgain(t *testing.T) {\n\tif R() != \"ready\" {\n\t\tt.Fatal(R())\n\t}\n}\n",
	}
	for name, body := range files {
		p := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=b", "-c", "user.email=b@b"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "base")
	sha := git("rev-parse", "HEAD")
	list := func(extra ...string) string {
		var out bytes.Buffer
		if err := mine(append([]string{"--kind", "wide-rename", "--repo", repo, "--commit", sha}, extra...), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	if got := list(); strings.Contains(got, "Thing") {
		t.Errorf("under the default floor (15, 8) Thing is listed: %q", got)
	}
	if got := list("--min-occurrences", "3", "--min-files", "2"); !strings.Contains(got, "Thing\t") {
		t.Errorf("with --min-occurrences 3 --min-files 2 Thing is not listed: %q", got)
	}
	if got := list("--min-occurrences", "3", "--min-files", "3"); strings.Contains(got, "Thing") {
		t.Errorf("with --min-files 3 Thing, in 2 files, is listed: %q", got)
	}
	coupled := func(extra ...string) string {
		var out bytes.Buffer
		if err := mine(append([]string{"--kind", "check-coupled", "--repo", repo, "--commit", sha}, extra...), &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	if got := coupled(); !strings.Contains(got, `"active"`) || !strings.Contains(got, `"ready"`) {
		t.Errorf("under the default floor (one test file) both literals are not listed: %q", got)
	}
	if got := coupled("--min-test-files", "2"); strings.Contains(got, `"active"`) || !strings.Contains(got, `"ready"`) {
		t.Errorf("with --min-test-files 2 only the literal two test files hold is listed: %q", got)
	}
}

// TestTheVerbsReachTheirCommands drives the built binary: analyze prints its
// table, and look a Stage 5 directory's report with its exit code. A case of
// the verb switch that falls through to another prints nothing and exits 0,
// which no unit test of the commands themselves can see (found while
// executing ADR-162 T3: a look case inserted inside analyze's).
func TestTheVerbsReachTheirCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine")
	}
	bin := filepath.Join(t.TempDir(), "bench")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v: %s", err, out)
	}
	empty := t.TempDir()
	out, err := exec.Command(bin, "analyze", "--tasks-dir", filepath.Join("..", "..", "bench", "tasks"), empty).Output()
	if err != nil || !strings.HasPrefix(string(out), "| stage | model | source | verdict |") {
		t.Errorf("bench analyze printed %q, %v; want its table", out, err)
	}
	stage := t.TempDir()
	block := filepath.Join(stage, "block-1")
	if err := os.MkdirAll(block, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(block, "run.json"), []byte(`{"stage":"stage5","tasks":[],"models":["m"],"configs":["A","B"],"repeats":1,"block":1,"looks":5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "look", stage)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 || !strings.Contains(stdout.String(), "block 2 is due") {
		t.Errorf("bench look on one open block: %q, %v; want exit 1 and block 2 due", stdout.String(), err)
	}
}

// TestAnalyzeRefusesAStage5Block: a Stage 5 block is read by its stage's look,
// which applies the stage's rulings; analyze on one block would read its raw
// receipts at the ordinary 95% bar (the Codex review of 80e50bd2).
func TestAnalyzeRefusesAStage5Block(t *testing.T) {
	stage := t.TempDir()
	block := filepath.Join(stage, "block-2")
	for _, d := range []string{block, filepath.Join(stage, "block-1")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(stage, "block-1", "run.json"), []byte(`{"stage":"stage5","block":1,"looks":5}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := analyze([]string{block + string(filepath.Separator)}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "bench look") {
		t.Errorf("analyze over a Stage 5 block: %v, want a refusal naming bench look", err)
	}
}

// TestAnalyzeNamesEachRowsBaseline: bench analyze prints every Stage 5 row
// against the configuration it was read against: a stage7 directory has H
// against B beside B and H against A, the cost column naming its baseline, and
// a Stage 5 directory prints what it always printed.
func TestAnalyzeNamesEachRowsBaseline(t *testing.T) {
	stage := func(profile string, cost map[string]float64) string {
		dir := t.TempDir()
		block := filepath.Join(dir, "block-1")
		if err := os.MkdirAll(block, 0o755); err != nil {
			t.Fatal(err)
		}
		put := func(name string, v any) {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(block, name), b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		var tasks, configs []string
		for i := range 10 {
			tasks = append(tasks, fmt.Sprintf("t%02d", i))
		}
		for _, c := range []string{"A", "B", "H"} {
			if _, ok := cost[c]; ok {
				configs = append(configs, c)
			}
		}
		run := map[string]any{"stage": profile, "tasks": tasks, "models": []string{"m"}, "configs": configs, "repeats": 1, "seed": 156,
			"cli_version": "9.9.9", "containment": "bypass", "profile": profile, "block": 1, "looks": 5}
		if profile == bench.ProfileStage7 {
			run["h_sentence_sha256"] = strings.Repeat("a", 64)
		}
		put("run.json", run)
		for _, task := range tasks {
			for _, c := range configs {
				cost := cost[c]
				put(task+"."+c+".m.1.receipt.json", bench.Receipt{Attempt: task + "." + c + ".m.1", Stage: profile, Task: task, Source: "swe-bench",
					Model: "m", Config: c, Repeat: 1, CostUSD: &cost, Oracle: bench.Verdict{Solved: true}, HostReads: 1, HookReads: map[string]int{"B": 7, "H": 3}[c]})
			}
		}
		return dir
	}
	analyzed := func(dir string) string {
		var out bytes.Buffer
		if err := analyze([]string{"--tasks-dir", filepath.Join("..", "..", "bench", "tasks"), dir}, &out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}

	seven := analyzed(stage(bench.ProfileStage7, map[string]float64{"A": 1, "B": 1.4, "H": 1.2}))
	for _, want := range []string{"swe-bench, B against A", "swe-bench, H against A", "swe-bench, H against B", "cost (X−A)/A [99%]", "cost (X−B)/B [99%]", "solved X−B [99%, Tango]"} {
		if !strings.Contains(seven, want) {
			t.Errorf("a stage7 analysis does not say %q:\n%s", want, seven)
		}
	}
	if h := rowLine(seven, "H against B"); !strings.Contains(h, "-0.143") {
		t.Errorf("H against B reads H's cost against B's, (1.2−1.4)/1.4 = −0.143: %q", h)
	}
	// The last column counts the compared configuration's reads alone: B's 7
	// or H's 3 over ten tasks, never the baseline's.
	for row, want := range map[string]string{"B against A": "70", "H against A": "30", "H against B": "30"} {
		if got := lastCell(rowLine(seven, row)); got != want {
			t.Errorf("%s: shell reads %q, want %s", row, got, want)
		}
	}
	if !strings.Contains(seven, "| X shell reads |") || strings.Contains(seven, "B or H shell reads") {
		t.Errorf("a table with H rows does not name the compared configuration's reads as X:\n%s", seven)
	}
	five := analyzed(stage(bench.ProfileStage5, map[string]float64{"A": 1, "B": 1.4}))
	if !strings.Contains(five, "swe-bench, B against A") || !strings.Contains(five, "cost (X−A)/A [99%]") || !strings.Contains(five, "| B shell reads |") || strings.Contains(five, "(X−B)/B") || strings.Contains(five, "H against") {
		t.Errorf("a Stage 5 analysis changed:\n%s", five)
	}
}

// rowLine is the table row of out that holds substr.
func rowLine(out, substr string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "|") && strings.Contains(l, substr) {
			return l
		}
	}
	return ""
}

// lastCell is the last cell of a table row.
func lastCell(row string) string {
	cells := strings.Split(strings.TrimSuffix(strings.TrimSpace(row), "|"), "|")
	return strings.TrimSpace(cells[len(cells)-1])
}

// TestTheCanaryRunsHOnlyUnderStage7: the canary's configurations are A and B,
// with H under stage7 and the plugin's, the work line's after them, chosen
// by the command and not only by bench.CanaryBase (ADR-192 Decision 1).
func TestTheCanaryRunsHOnlyUnderStage7(t *testing.T) {
	for _, c := range []struct {
		o    bench.RunOptions
		want string
	}{
		{bench.RunOptions{Profile: bench.ProfileStage7}, "A,B,H"},
		{bench.RunOptions{Profile: bench.ProfileStage5}, "A,B"},
		{bench.RunOptions{}, "A,B"},
		{bench.RunOptions{Profile: bench.ProfileStage5, PluginDir: "/p", WorkLine: "w"}, "A,B,P,W"},
	} {
		got, err := canaryConfigs(c.o)
		if err != nil || strings.Join(got, ",") != c.want {
			t.Errorf("the canary under profile %q runs %v (%v), want %s", c.o.Profile, got, err, c.want)
		}
	}
}

// TestTheStage8CanaryRunsBAndS: the canary command's configurations are B and
// S under stage8, chosen by the command and not only by bench.CanaryBase, and
// what they were under every other profile (ADR-193 Decision 2).
func TestTheStage8CanaryRunsBAndS(t *testing.T) {
	for _, c := range []struct {
		o    bench.RunOptions
		want string
	}{
		{bench.RunOptions{Profile: bench.ProfileStage8}, "B,S"},
		{bench.RunOptions{Profile: bench.ProfileStage7}, "A,B,H"},
		{bench.RunOptions{Profile: bench.ProfileStage6}, "A,B"},
		{bench.RunOptions{Profile: bench.ProfileStage5}, "A,B"},
		{bench.RunOptions{}, "A,B"},
	} {
		got, err := canaryConfigs(c.o)
		if err != nil || strings.Join(got, ",") != c.want {
			t.Errorf("the canary under profile %q runs %v (%v), want %s", c.o.Profile, got, err, c.want)
		}
	}
}

// TestBenchHookReadToolFlagNamesTheReadTool: bench hook refuses a shell write
// either way, and says the Read tool reads only with --read-tool, the flag
// configuration H's Bash hook carries.
func TestBenchHookReadToolFlagNamesTheReadTool(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"rm x"}}`
	log := filepath.Join(t.TempDir(), "hook.log")
	for _, c := range []struct {
		args     []string
		readTool bool
	}{{[]string{"--log", log}, false}, {[]string{"--read-tool", "--log", log}, true}} {
		var stderr bytes.Buffer
		code := hookCmd(c.args, strings.NewReader(payload), &stderr)
		if code != 2 || strings.Contains(stderr.String(), "the Read tool") != c.readTool || !strings.Contains(stderr.String(), "mrw read") {
			t.Errorf("bench hook %v: exit %d %q; want a refusal naming the Read tool: %v", c.args, code, stderr.String(), c.readTool)
		}
	}
}
