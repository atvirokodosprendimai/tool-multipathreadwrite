package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The shim is bench itself run as `bench shim`: a test binary stands in for
// it, so a shim script reaches the same code a stage runs.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "shim" {
		cwd, _ := os.Getwd()
		os.Exit(Shim(os.Args[2:], cwd, os.Stdin, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

const fakeImage = "swebench/sweb.eval.x86_64.fake:latest"

// stage6Rig is a stage under the ADR-163 profile whose docker is a script that
// records its calls and runs what it is told in a directory on this machine
// standing for the container: /testbed and /opt are rewritten to it. No image
// is pulled and no container started.
type stage6Rig struct {
	t         *testing.T
	opts      RunOptions
	work      string
	root      string // the fake container filesystem
	dockerLog string
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func put(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

// seedImage builds what a real image has and the container must not keep: a
// repository with history, a stale tracked file, ignored build output, an
// untracked file, and an evaluation artifact outside the tree.
func seedImage(t *testing.T, root string) {
	tb := filepath.Join(root, "testbed")
	put(t, filepath.Join(tb, ".gitignore"), "build/\n", 0o644)
	put(t, filepath.Join(tb, "notes.txt"), "OLD\n", 0o644)
	put(t, filepath.Join(tb, "stale.txt"), "stale\n", 0o644)
	gitIn(t, tb, "init", "-q")
	gitIn(t, tb, "add", "-A")
	gitIn(t, tb, "commit", "-q", "-m", "upstream one")
	put(t, filepath.Join(tb, "notes.txt"), "OLDER\n", 0o644)
	gitIn(t, tb, "commit", "-q", "-am", "the upstream fix")
	put(t, filepath.Join(tb, "build", "ext.so"), "compiled\n", 0o644)
	put(t, filepath.Join(tb, "scratch.txt"), "untracked\n", 0o644)
	put(t, filepath.Join(root, "opt", "swe-eval", "test_patch.diff"), "the hidden tests\n", 0o644)
	// The testbed environment: activating it puts a python on PATH that runs
	// the -c code as shell, answers the canary probes, and echoes anything else.
	activate := "export PATH=" + shellQuote(filepath.Join(root, "fakebin")) + ":$PATH\n"
	put(t, filepath.Join(root, "opt", "miniconda3", "bin", "activate"), activate, 0o644)
	py := "#!/bin/sh\nROOT=" + shellQuote(root) + "\ncase \"$*\" in\n" +
		"*sys.platform*) cat \"$ROOT/platform\" 2>/dev/null || echo linux; exit 0;;\n" +
		"*create_connection*) if [ -e \"$ROOT/net-open\" ]; then echo open; else echo blocked; fi; exit 0;;\n" +
		"esac\nif [ \"$1\" = -c ]; then exec sh -c \"$2\"; fi\n" +
		"echo \"$(basename \"$0\") $*\"\n"
	for _, n := range shimNames {
		put(t, filepath.Join(root, "fakebin", n), py, 0o755)
	}
}

func fakeDocker(t *testing.T, dir, root, log string) string {
	script := `#!/bin/bash
ROOT=` + shellQuote(root) + `; LOG=` + shellQuote(log) + `
line="$*"; line="${line//$'\n'/ }"; printf '%s\n' "$line" >> "$LOG"
sub() { REPLY="$1"; REPLY="${REPLY//\/testbed/$ROOT/testbed}"; REPLY="${REPLY//\/opt\//$ROOT/opt/}"; }
case "$1" in
run) if [ -e "$ROOT/fail-run" ]; then echo "docker: boom" 1>&2; exit 1; fi; echo fakeid; exit 0;;
ps) [ -e "$ROOT/ps.out" ] || exit 0; want=""; for a in "$@"; do case "$a" in label=mrw-bench-attempt=*) want="${a#label=mrw-bench-attempt=}";; esac; done; while read -r id lab; do if [ "$lab" = "$want" ]; then echo "$id"; fi; done < "$ROOT/ps.out"; exit 0;;
exec)
	shift; dir=""
	while [ $# -gt 0 ]; do
		case "$1" in -i) shift;; -e) sub "$2"; export "$REPLY"; shift 2;; -w) sub "$2"; dir="$REPLY"; shift 2;; *) break;; esac
	done
	shift
	args=(); for a in "$@"; do sub "$a"; args+=("$REPLY"); done
	case "$*" in *"git apply"*) if [ -e "$ROOT/break-apply" ]; then echo "error: patch failed: notes.txt:1" >&2; exit 1; fi;; esac
	case "$*" in *"git init -q"*) if [ -e "$ROOT/skew" ]; then echo extra > "$ROOT/testbed/skew.txt"; fi;; esac
	if [ -n "$dir" ]; then cd "$dir"; fi
	if [ -e "$ROOT/leftover" ]; then if [ -e "$ROOT/leader" ]; then set -m; fi; "${args[@]}" <&0 & wait $!; exit $?; fi
	exec "${args[@]}";;
*) exit 0;;
esac
`
	p := filepath.Join(dir, "docker")
	put(t, p, script, 0o755)
	return p
}

func newStage6Rig(t *testing.T) *stage6Rig {
	t.Helper()
	needGit(t)
	for _, tool := range []string{"bash", "tar", "xargs"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " is not on PATH")
		}
	}
	o := profiledOptions(t, "", "A")
	work := filepath.Dir(o.Out)
	// The fixture ignores build output as the repository of the image does, or
	// the build output of the image would join the base commit of the container.
	put(t, filepath.Join(o.Tasks[0].Dir, "base", ".gitignore"), "build/\n", 0o644)
	root := filepath.Join(work, "container")
	seedImage(t, root)
	log := filepath.Join(work, "docker.log")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	o.Profile, o.Docker, o.CanaryImage, o.Bench = ProfileStage6, fakeDocker(t, work, root, log), fakeImage, exe
	if err := resolveProfile(&o); err != nil {
		t.Fatal(err)
	}
	put(t, log, "", 0o644)
	if err := os.MkdirAll(o.Out, 0o755); err != nil {
		t.Fatal(err)
	}
	return &stage6Rig{t: t, opts: o, work: work, root: root, dockerLog: log}
}

func (g *stage6Rig) dockerCalls() []string {
	b, _ := os.ReadFile(g.dockerLog)
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

// callsMatching are the docker calls whose line holds every part.
func (g *stage6Rig) callsMatching(parts ...string) []string {
	var out []string
	for _, l := range g.dockerCalls() {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(l, p)
		}
		if ok {
			out = append(out, l)
		}
	}
	return out
}

// start builds a host tree at its base commit, with the declared check file in
// it as T2 will write it, and starts the container from it.
func (g *stage6Rig) start() *shimConfig {
	t := g.t
	t.Helper()
	tree := filepath.Join(g.work, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(g.opts.Tasks[0], tree, nil); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(tree, ".quality-harness.json"), `{"check":"python -m pytest"}`+"\n", 0o644)
	put(t, filepath.Join(tree, ".gitignore"), "build/\n", 0o644)
	gitIn(t, tree, "add", "-A")
	gitIn(t, tree, "commit", "-q", "-m", "base with the check")
	cfg := &shimConfig{Bench: g.opts.Bench, Docker: g.opts.Docker, Container: containerName("rig"), Tree: tree,
		State: filepath.Join(g.work, "shim"), Base: gitIn(t, tree, "rev-parse", "HEAD")}
	if err := os.MkdirAll(cfg.State, 0o755); err != nil {
		t.Fatal(err)
	}
	box, err := startContainer(cfg.Docker, fakeImage, cfg.Container, "test", time.Minute, tree, cfg.Base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = box.remove() })
	return cfg
}

// shim runs one shim command in-process and returns what the agent would see.
func (g *stage6Rig) shim(cfg *shimConfig, cwd, name string, argv ...string) (stdout, stderr string, code int) {
	var out, errb bytes.Buffer
	code = Shim(append(cfg.args(name), argv...), cwd, nil, &out, &errb)
	return out.String(), errb.String(), code
}

func (g *stage6Rig) applyFix() string {
	return "git apply " + shellQuote(filepath.Join(g.opts.Tasks[0].Dir, "reference.patch")) + " >/dev/null 2>&1"
}

func (g *stage6Rig) attempt(id string) attempt {
	return attempt{ID: id, Task: g.opts.Tasks[0], Model: "stub-model", Config: Configs["A"], Repeat: 1}
}

func (g *stage6Rig) run(id string) Receipt {
	return runAttempt(g.opts, g.attempt(id), 0, "9.9.9 (Claude Code)", "")
}

// claude writes a stand-in CLI that runs pre in the attempt directory, then
// prints a session whose Bash tool calls are the commands given.
func (g *stage6Rig) claude(pre string, commands ...string) {
	t := g.t
	lines := []string{`{"type":"system","subtype":"init","model":"stub-model","tools":["Bash"],"mcp_servers":[]}`}
	for _, c := range commands {
		b, _ := json.Marshal(map[string]any{"type": "assistant", "message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "name": "Bash", "input": map[string]string{"command": c}}}}})
		lines = append(lines, string(b))
	}
	lines = append(lines, `{"type":"result","subtype":"success","is_error":false,"num_turns":2,"total_cost_usd":0.1,"usage":{}}`)
	session := filepath.Join(g.work, "session.jsonl")
	put(t, session, strings.Join(lines, "\n")+"\n", 0o644)
	script := "#!/bin/sh\nfor a in " + "\"$@\"; do [ \"$a\" = \"--version\" ] && { echo 9.9.9; exit 0; }; done\n" + pre + "\ncat " + shellQuote(session) + "\n"
	g.opts.Claude = filepath.Join(g.work, "claude-s6")
	put(t, g.opts.Claude, script, 0o755)
}

// TestStage6ShimsRunInTheInstanceContainer: under stage6 python, python3,
// pytest and py.test are shims that call `docker exec` on the attempt
// container with the mapped working directory and arguments, after the reset
// and the diff; output and exit code pass through; a diff that does not apply
// exits non-zero, counts in sync_failures, and the attempt is still graded.
// Without stage6, and under stage5, toolPath links the host python3 and no
// container starts.
func TestStage6ShimsRunInTheInstanceContainer(t *testing.T) {
	t.Run("the variables of the command are forwarded", theCommandVariablesAreForwarded)
	for _, profile := range []string{"", ProfileStage5} {
		g := newStage6Rig(t)
		o := g.opts
		o.Profile = profile
		scratch := t.TempDir()
		if _, _, err := invocation(o, g.attempt("x"), scratch); err != nil {
			t.Fatal(err)
		}
		tools := filepath.Join(scratch, "tools")
		if _, err := exec.LookPath("python3"); err == nil {
			if fi, err := os.Lstat(filepath.Join(tools, "python3")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
				t.Errorf("profile %q: python3 is not the host one, linked: %v %v", profile, fi, err)
			}
		}
		for _, n := range []string{"python", "pytest", "py.test"} {
			if exists(filepath.Join(tools, n)) {
				t.Errorf("profile %q: %s is in the tools directory", profile, n)
			}
		}
		if calls := g.dockerCalls(); len(calls) != 1 || calls[0] != "" {
			t.Errorf("profile %q: docker was called: %q", profile, calls)
		}
	}

	g := newStage6Rig(t)
	cfg := g.start()
	a := g.attempt("shimmed")
	a.shim = cfg
	scratch := t.TempDir()
	if _, _, err := invocation(g.opts, a, scratch); err != nil {
		t.Fatal(err)
	}
	tools := filepath.Join(scratch, "tools")
	for _, n := range shimNames {
		fi, err := os.Lstat(filepath.Join(tools, n))
		if err != nil || fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&0o111 == 0 {
			t.Errorf("%s is not an executable shim: %v %v", n, fi, err)
		}
	}
	if !exists(filepath.Join(tools, "git")) {
		t.Error("git is no longer linked beside the shims")
	}
	sub := filepath.Join(cfg.Tree, "sub")
	put(t, filepath.Join(sub, "test_x.py"), "def test_x(): pass\n", 0o644)
	runShim := func(name, dir string, args ...string) (string, string, int) {
		cmd := exec.Command(filepath.Join(tools, name), args...)
		cmd.Dir, cmd.Env = dir, []string{"PATH=" + tools + ":/usr/bin:/bin", "HOME=" + t.TempDir()}
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		code := 0
		if err := cmd.Run(); err != nil {
			ee, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			code = ee.ExitCode()
		}
		return out.String(), errb.String(), code
	}
	out, _, code := runShim("pytest", sub, filepath.Join(sub, "test_x.py"))
	if want := "pytest " + filepath.Join(g.root, "testbed", "sub", "test_x.py") + "\n"; out != want || code != 0 {
		t.Errorf("pytest through its shim printed %q exit %d, want %q exit 0", out, code, want)
	}
	if out, _, code := runShim("python", cfg.Tree, "-c", "echo hi; exit 3"); out != "hi\n" || code != 3 {
		t.Errorf("python through its shim printed %q exit %d, want hi and exit 3", out, code)
	}
	calls := g.dockerCalls()
	reset, run := -1, -1
	for i, l := range calls {
		if strings.Contains(l, "git reset") && strings.Contains(l, "git apply") && reset < 0 {
			reset = i
		}
		if strings.Contains(l, "-w /testbed/sub") && strings.Contains(l, "/testbed/sub/test_x.py") && run < 0 {
			run = i
		}
		if strings.Contains(l, cfg.Tree) {
			t.Errorf("the host tree path reached the container: %s", l)
		}
	}
	if reset < 0 || run < 0 || reset > run {
		t.Errorf("the command did not follow the reset and the diff: reset at %d, exec at %d in %q", reset, run, calls)
	}

	// A diff that does not apply: the shim fails and says so, nothing runs, the
	// count goes in the receipt, and the attempt is graded as any other.
	broken := newStage6Rig(t)
	put(t, filepath.Join(broken.root, "break-apply"), "", 0o644)
	bc := broken.start()
	put(t, filepath.Join(bc.Tree, "notes.txt"), "changed on the host\n", 0o644)
	out, stderr, code := broken.shim(bc, bc.Tree, "python", "-c", "echo ran")
	if code == 0 || out != "" || !strings.Contains(stderr, "did not reach the container") {
		t.Errorf("an unapplied diff: exit %d, stdout %q, stderr %q", code, out, stderr)
	}
	if got := readLog(bc.logPath()); len(got) != 1 || !got[0].SyncFailed {
		t.Errorf("the log does not record the failed sync: %+v", got)
	}
	graded := newStage6Rig(t)
	put(t, filepath.Join(graded.root, "break-apply"), "", 0o644)
	graded.claude(graded.applyFix()+"\npython -c 'echo ran' >/dev/null 2>&1", "python -c 'echo ran'")
	r := graded.run("fix-one.A.stub-model.1")
	if r.Exclusion != "" || !r.Oracle.Solved {
		t.Errorf("an attempt whose sync failed was not graded: %+v", r)
	}
	if r.ShimCounts == nil || r.SyncFailures != 1 || r.ShimCalls != 1 {
		t.Errorf("the receipt shim counts are %+v, want 1 call and 1 sync failure", r.ShimCounts)
	}
}

// TestTheContainerHasNoNetworkAndNoHistory: the container starts with
// --network none; /testbed/.git is replaced and a planted evaluation artifact
// is removed before the agent starts; the container is removed on a clean end,
// an error, the budget and an interrupt.
func TestTheContainerHasNoNetworkAndNoHistory(t *testing.T) {
	t.Run("the start is labelled, checked and swept", theStartIsLabelledCheckedAndSwept)
	old := evalArtifacts
	evalArtifacts = []string{"/opt/swe-eval/*"}
	t.Cleanup(func() { evalArtifacts = old })

	g := newStage6Rig(t)
	tb := filepath.Join(g.root, "testbed")
	upstream := gitIn(t, tb, "rev-parse", "HEAD")
	artifact := filepath.Join(g.root, "opt", "swe-eval", "test_patch.diff")
	if !exists(artifact) {
		t.Fatal("the fixture did not plant an artifact")
	}
	seen := filepath.Join(g.work, "seen")
	probe := "{ git -C " + shellQuote(tb) + " log --all --oneline | wc -l; git -C " + shellQuote(tb) + " reflog | wc -l; if [ -e " + shellQuote(artifact) + " ]; then echo present; else echo gone; fi; } > " + shellQuote(seen)
	g.claude(probe, "ls")
	r := g.run("fix-one.A.stub-model.1")
	if r.Exclusion != "" {
		t.Fatalf("the attempt was excluded: %+v", r)
	}
	if runs := g.callsMatching("run -d"); len(runs) != 1 || !strings.Contains(runs[0], "--network none") || !strings.Contains(runs[0], "--platform linux/amd64") {
		t.Errorf("the container did not start with no network on linux/amd64: %q", runs)
	}
	var fields []string
	if b, err := os.ReadFile(seen); err == nil {
		fields = strings.Fields(string(b))
	}
	if len(fields) != 3 || fields[0] != "1" || fields[1] != "1" || fields[2] != "gone" {
		t.Errorf("when the agent started, /testbed showed commits, reflog entries and the artifact as %v, want 1, 1 and gone", fields)
	}
	if err := exec.Command("git", "-C", tb, "cat-file", "-e", upstream).Run(); err == nil {
		t.Error("the upstream fix commit is still in the /testbed object store")
	}
	if !exists(filepath.Join(tb, "build", "ext.so")) {
		t.Error("the image ignored build output was removed")
	}

	runName := func(g *stage6Rig) string {
		runs := g.callsMatching("run -d")
		if len(runs) != 1 {
			t.Fatalf("containers started: %q", runs)
		}
		f := strings.Fields(runs[0])
		for i, w := range f {
			if w == "--name" {
				return f[i+1]
			}
		}
		t.Fatalf("no --name in %q", runs[0])
		return ""
	}
	removed := func(label string, g *stage6Rig) {
		t.Helper()
		if rm := g.callsMatching("rm -f " + runName(g)); len(rm) != 1 {
			t.Errorf("%s: the container was removed %d times, want once: %q", label, len(rm), g.dockerCalls())
		}
	}
	removed("a clean end", g)

	e := newStage6Rig(t)
	e.claude("", "ls")
	// Bench cannot open its transcript: an error after the container started.
	put(t, filepath.Join(e.opts.Out, "fix-one.A.stub-model.1.transcript.jsonl"), "", 0o644)
	if r := e.run("fix-one.A.stub-model.1"); r.Exclusion != ExclusionSetup {
		t.Fatalf("an attempt that could not open its transcript: %+v", r)
	}
	removed("an error", e)

	b := newStage6Rig(t)
	b.opts.Budget = 400 * time.Millisecond
	b.claude("sleep 30", "ls")
	if r := b.run("fix-one.A.stub-model.1"); !r.TimedOut {
		t.Fatalf("the budget did not end the attempt: %+v", r)
	}
	removed("the budget", b)

	i := newStage6Rig(t)
	i.claude("kill -INT "+strconv.Itoa(os.Getpid())+"\nsleep 30", "ls")
	if r := i.run("fix-one.A.stub-model.1"); !r.interrupted {
		t.Fatalf("a signal did not interrupt the attempt: %+v", r)
	}
	removed("an interrupt", i)
}

// TestAShimStartsFromTheHostBaseAndKeepsWhatPythonWrote: after start, the
// container tracked tree equals the host prepared base, the check file
// included; a shim command that writes a source file and a new test leaves
// both in the host tree and in its diff, __pycache__ and .pytest_cache not; a
// file the host also changed is a named conflict, not written; an edit to the
// check file syncs.
func TestAShimStartsFromTheHostBaseAndKeepsWhatPythonWrote(t *testing.T) {
	t.Run("a failed write-back is counted apart", aFailedWriteBackIsCountedApart)
	g := newStage6Rig(t)
	cfg := g.start()
	tb := filepath.Join(g.root, "testbed")
	tracked := func(dir string) string { return gitIn(t, dir, "ls-files") }
	if got, want := tracked(tb), tracked(cfg.Tree); got != want {
		t.Fatalf("the container tracks %q, the host base %q", got, want)
	}
	for _, f := range strings.Fields(tracked(cfg.Tree)) {
		a, _ := os.ReadFile(filepath.Join(tb, f))
		b, _ := os.ReadFile(filepath.Join(cfg.Tree, f))
		if !bytes.Equal(a, b) {
			t.Errorf("%s differs between the container and the host base", f)
		}
	}
	if !exists(filepath.Join(tb, ".quality-harness.json")) {
		t.Error("the declared check file is not in the container")
	}
	if !exists(filepath.Join(tb, "build", "ext.so")) || exists(filepath.Join(tb, "stale.txt")) || exists(filepath.Join(tb, "scratch.txt")) {
		t.Error("the container kept stale or untracked files, or lost ignored build output")
	}

	// What python writes reaches the host tree and its diff.
	write := `echo 'x = 2' > notes.txt; echo 'def test_a(): pass' > test_new.py; mkdir -p __pycache__ .pytest_cache pkg/__pycache__; echo z > __pycache__/a.pyc; echo z > pkg/__pycache__/b.cpython-311.pyc; echo z > .pytest_cache/v; echo z > pkg/c.pyc`
	if _, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", write); code != 0 {
		t.Fatalf("the write failed: exit %d %s", code, stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(cfg.Tree, "notes.txt")); string(got) != "x = 2\n" {
		t.Errorf("notes.txt on the host is %q", got)
	}
	if !exists(filepath.Join(cfg.Tree, "test_new.py")) {
		t.Error("the new test did not reach the host tree")
	}
	for _, p := range []string{"__pycache__", ".pytest_cache", "pkg"} {
		if exists(filepath.Join(cfg.Tree, p)) {
			t.Errorf("%s reached the host tree", p)
		}
	}
	gitIn(t, cfg.Tree, "add", "-A")
	if diff := gitIn(t, cfg.Tree, "diff", "--cached", "--name-only", "HEAD"); diff != "notes.txt\ntest_new.py" {
		t.Errorf("the graded diff names %q, want notes.txt and test_new.py", diff)
	}
	gitIn(t, cfg.Tree, "reset", "-q")

	// An edit to the check file syncs.
	put(t, filepath.Join(cfg.Tree, ".quality-harness.json"), `{"check":"edited"}`+"\n", 0o644)
	if out, _, _ := g.shim(cfg, cfg.Tree, "python", "-c", "cat .quality-harness.json"); out != `{"check":"edited"}`+"\n" {
		t.Errorf("the container check file after the edit is %q", out)
	}

	// A file the host changed while the command ran is a conflict; a sibling
	// the host left alone is still written. The host path reaches the command
	// through the environment, since a path in an argument is mapped.
	t.Setenv("HOSTPARENT", filepath.Dir(cfg.Tree))
	conflict := `echo box > conflict.txt; echo box > fine.txt; echo host > "$HOSTPARENT/tree/conflict.txt"`
	_, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", conflict)
	if code == 0 || !strings.Contains(stderr, "conflict.txt") || strings.Contains(stderr, "fine.txt") {
		t.Errorf("a conflict: exit %d, stderr %q, want non-zero naming conflict.txt only", code, stderr)
	}
	if got, _ := os.ReadFile(filepath.Join(cfg.Tree, "conflict.txt")); string(got) != "host\n" {
		t.Errorf("the conflicting file on the host is %q, want the host own", got)
	}
	if got, _ := os.ReadFile(filepath.Join(cfg.Tree, "fine.txt")); string(got) != "box\n" {
		t.Errorf("the sibling was not written back: %q", got)
	}
	last := readLog(cfg.logPath())
	if c := last[len(last)-1]; len(c.Conflicts) != 1 || c.WriteBacks != 1 {
		t.Errorf("the log records %+v, want one conflict and one write-back", c)
	}
}

// TestShimCallsTakeTurnsAndNeverTouchTheIndex: two concurrent shim calls run
// one after the other, the second wait logged; after a sync the agent
// git diff, git diff --cached and index are unchanged.
func TestShimCallsTakeTurnsAndNeverTouchTheIndex(t *testing.T) {
	t.Run("an orphaned command is killed", anOrphanedCommandIsKilled)
	t.Run("a signal kills the container command", aSignalKillsTheContainerCommand)
	t.Run("a command group is killed with its leader", aCommandGroupIsKilledWithItsLeader)
	g := newStage6Rig(t)
	cfg := g.start()
	order := filepath.Join(g.work, "order")
	body := "echo start >> " + shellQuote(order) + "; sleep 0.4; echo end >> " + shellQuote(order)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", body); code != 0 {
				t.Errorf("a concurrent call failed: %d %s", code, stderr)
			}
		}()
	}
	wg.Wait()
	if b, _ := os.ReadFile(order); string(b) != "start\nend\nstart\nend\n" {
		t.Errorf("the two commands overlapped: %q", b)
	}
	calls := readLog(cfg.logPath())
	if len(calls) != 2 {
		t.Fatalf("the log holds %d calls, want 2", len(calls))
	}
	if w := max(calls[0].WaitMS, calls[1].WaitMS); w < 300 {
		t.Errorf("the second call wait is %d ms, want most of the first command 400 ms", w)
	}
	if s, _ := shimCounts(cfg.logPath(), nil, "", "", ""); s.ShimCalls != 2 || s.ShimWaitMS < 300 {
		t.Errorf("the receipt counts are %+v", s)
	}

	// The agent index: staged, unstaged and untracked changes all stay as they
	// were, byte for byte.
	put(t, filepath.Join(cfg.Tree, "notes.txt"), "staged\n", 0o644)
	gitIn(t, cfg.Tree, "add", "notes.txt")
	put(t, filepath.Join(cfg.Tree, "notes.txt"), "staged\nand unstaged\n", 0o644)
	put(t, filepath.Join(cfg.Tree, "new.txt"), "untracked\n", 0o644)
	gitIn(t, cfg.Tree, "status", "--porcelain")
	snap := func() [4]string {
		idx, _ := os.ReadFile(filepath.Join(cfg.Tree, ".git", "index"))
		return [4]string{string(idx), gitIn(t, cfg.Tree, "diff"), gitIn(t, cfg.Tree, "diff", "--cached"), gitIn(t, cfg.Tree, "status", "--porcelain")}
	}
	before := snap()
	if before[2] == "" || before[1] == "" {
		t.Fatal("the fixture staged nothing")
	}
	if _, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", "true"); code != 0 {
		t.Fatalf("the sync failed: %d %s", code, stderr)
	}
	after := snap()
	for i, what := range []string{"index", "git diff", "git diff --cached", "git status"} {
		if before[i] != after[i] {
			t.Errorf("the sync changed the agent %s", what)
		}
	}
	if out, _, _ := g.shim(cfg, cfg.Tree, "python", "-c", "cat notes.txt new.txt"); out != "staged\nand unstaged\nuntracked\n" {
		t.Errorf("the container saw %q, want the staged, unstaged and untracked content", out)
	}
}

// TestAShimRefusesAPathOutsideTheTreeAndCountsHostCalls: a working directory
// outside the tree exits 2 naming it; commands naming /usr/bin/python3,
// /opt/homebrew/bin/pytest, env -i python3, command -p python3, xcrun python3
// count in host_interpreter_calls; a pytest with no shim log entry counts; a
// command naming docker is counted for the stop of T5.
func TestAShimRefusesAPathOutsideTheTreeAndCountsHostCalls(t *testing.T) {
	t.Run("the longest spelling of the tree maps first", theLongestSpellingMapsFirst)
	t.Run("an unlogged call matches no log entry", anUnloggedCallMatchesNoLogEntry)
	g := newStage6Rig(t)
	cfg := g.start()
	outside := t.TempDir()
	_, stderr, code := g.shim(cfg, outside, "python", "-c", "echo no")
	if code != 2 || !strings.Contains(stderr, outside) || !strings.Contains(stderr, "outside") {
		t.Errorf("a working directory outside the tree: exit %d, stderr %q, want 2 naming it", code, stderr)
	}
	if got := g.callsMatching("-w"); len(got) != 0 {
		t.Errorf("a command ran in the container: %q", got)
	}
	if got := readLog(cfg.logPath()); len(got) != 1 || !got[0].Refused {
		t.Errorf("the refusal is not logged as a call: %+v", got)
	}
	// The tree through a link is the tree.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(cfg.Tree, link); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := g.shim(cfg, link, "python", "-c", "true"); code != 0 {
		t.Errorf("the tree through a link was refused: %d %s", code, stderr)
	}

	cases := []struct {
		command            string
		host, unlogged, dk int
	}{
		{"/usr/bin/python3 -c 1", 1, 0, 0},
		{"cd x && /opt/homebrew/bin/pytest -q", 1, 0, 0},
		{"env -i python3 t.py", 1, 0, 0},
		{"command -p python3 t.py", 1, 0, 0},
		{"xcrun python3 t.py", 1, 0, 0},
		{"pytest -q tests/", 0, 1, 0},
		{"FOO=1 python3 -m pytest", 0, 1, 0},
		{"cd sub; python t.py && pytest", 0, 2, 0},
		{"docker run -it ubuntu bash", 0, 0, 1},
		{"echo " + fakeImage, 0, 0, 1},
		{"docker exec bench-rig-1 ls", 0, 0, 1},
		{"grep -r python3 .; ls; git status; node t.js; cat pytest.ini", 0, 0, 0},
	}
	for _, c := range cases {
		h, _, u, d := countCommands([]string{c.command}, nil, fakeImage, "bench-rig-1", "")
		if h != c.host || u != c.unlogged || d != c.dk {
			t.Errorf("%q counted host %d, unlogged %d, docker %d; want %d, %d, %d", c.command, h, u, d, c.host, c.unlogged, c.dk)
		}
	}
	if _, _, u, _ := countCommands([]string{"pytest -q"}, []shimCall{{Name: "pytest", Args: []string{"-q"}}}, "", "", ""); u != 0 {
		t.Errorf("a pytest with a shim log entry counted as unlogged: %d", u)
	}

	// In a receipt: explicit zeros under stage6, none of the keys without it.
	g.claude("", "pytest -q", "/usr/bin/python3 -V", "docker ps")
	r := g.run("fix-one.A.stub-model.2")
	if r.ShimCounts == nil || r.UnloggedPythonCalls != 1 || r.HostInterpreterCalls != 1 || r.DockerCalls != 1 || r.ShimCalls != 0 {
		t.Fatalf("the receipt counts are %+v, want 1 unlogged, 1 host, 1 docker, 0 shim calls", r.ShimCounts)
	}
	b, _ := json.Marshal(r)
	for _, k := range []string{"host_interpreter_calls", "shim_calls", "shim_wait_ms", "sync_failures", "write_backs", "write_back_conflicts", "write_back_failures", "bare_python_calls", "unlogged_python_calls", "docker_calls"} {
		if !strings.Contains(string(b), fmt.Sprintf("%q:", k)) {
			t.Errorf("the receipt lacks %s: %s", k, b)
		}
	}
	plain := newStage6Rig(t)
	plain.opts.Profile = ""
	plain.claude("", "pytest -q")
	pb, _ := json.Marshal(plain.run("fix-one.A.stub-model.1"))
	if strings.Contains(string(pb), "shim_calls") || strings.Contains(string(pb), "host_interpreter_calls") {
		t.Errorf("a receipt without the profile carries the container keys: %s", pb)
	}
	if calls := plain.dockerCalls(); len(calls) != 1 || calls[0] != "" {
		t.Errorf("a stage without the profile called docker: %q", calls)
	}
}

func readLog(path string) []shimCall {
	calls, _ := readShimLog(path)
	return calls
}

// anOrphanedCommandIsKilled: a docker exec
// client that dies leaves its command running in the container, outside the
// lock; the next shim call kills it before it syncs.
func anOrphanedCommandIsKilled(t *testing.T) {
	g := newStage6Rig(t)
	cfg := g.start()
	put(t, filepath.Join(g.root, "leftover"), "", 0o644)
	if err := os.MkdirAll(filepath.Join(g.root, "opt", "bench"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(cfg.Docker, "exec", "-i", cfg.Container, "bash", "-c", activateScript, "bash", "sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := 0
	for range 100 {
		b, err := os.ReadFile(filepath.Join(g.root, "opt", "bench", "pid"))
		if n, aerr := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && aerr == nil {
			pid = n
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("the command never recorded its pid")
	}
	p, _ := os.FindProcess(pid)
	t.Cleanup(func() { _ = p.Kill() })
	alive := func() bool { return p.Signal(syscall.Signal(0)) == nil }
	// The client dies; its command does not.
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if !alive() {
		t.Fatal("the fixture did not leave a command running")
	}
	if _, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", "true"); code != 0 {
		t.Fatalf("the shim call failed: %d %s", code, stderr)
	}
	for range 60 {
		if !alive() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if alive() {
		t.Error("the command left in the container survived the next shim call")
	}
}

// theLongestSpellingMapsFirst: the resolved spelling of the
// tree may contain the spelling the shim was given (/private/var/x holds
// /var/x on macOS); replacing the short one first leaves /private/testbed.
func theLongestSpellingMapsFirst(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the path of the fixture joins two drive letters here")
	}
	td := t.TempDir()
	link := filepath.Join(td, "l")
	target := filepath.Join(td, "real") + link
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("no symlinks here")
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil || !strings.Contains(resolved, link) {
		t.Fatalf("the fixture is not nested: %q %q %v", link, resolved, err)
	}
	c := shimConfig{Tree: link}
	got := c.mapArgs([]string{resolved + "/x", "open('" + resolved + "/x')", link + "/y"})
	want := []string{"/testbed/x", "open('/testbed/x')", "/testbed/y"}
	if !slices.Equal(got, want) {
		t.Errorf("mapped %q, want %q", got, want)
	}
}

// theStartIsLabelledCheckedAndSwept: the container removes itself
// after a bounded life and carries the bench label; a start whose commit tree
// is not the host base tree fails; a failed docker run still returns the
// handle; the containers an earlier run left behind are removed.
func theStartIsLabelledCheckedAndSwept(t *testing.T) {
	g := newStage6Rig(t)
	put(t, filepath.Join(g.root, "ps.out"), "id1 test\nid2 other\n", 0o644)
	o := g.opts
	if err := resolveContainer(&o); err != nil {
		t.Fatal(err)
	}
	if got := g.callsMatching("ps -aq --filter label=mrw-bench-attempt=test"); len(got) == 0 {
		t.Error("no sweep looked for the containers of this stage")
	}
	if rm := g.callsMatching("rm -f id1"); len(rm) != 1 {
		t.Errorf("id1 was removed %d times by the sweep", len(rm))
	}
	if rm := g.callsMatching("rm -f id2"); len(rm) != 0 {
		t.Error("the sweep removed a container of another stage")
	}
	cfg := g.start()
	runs := g.callsMatching("run -d", cfg.Container)
	if len(runs) != 1 || !strings.Contains(runs[0], " --rm ") || !strings.Contains(runs[0], "--label mrw-bench-attempt=test") || !strings.HasSuffix(runs[0], " sleep 60") {
		t.Errorf("the container is not labelled, self-removing and bounded: %q", runs)
	}

	put(t, filepath.Join(g.root, "skew"), "", 0o644)
	box, err := startContainer(cfg.Docker, fakeImage, containerName("skew"), "test", time.Minute, cfg.Tree, cfg.Base)
	if box == nil || err == nil || !strings.Contains(err.Error(), "does not start from the base") {
		t.Errorf("a container whose tree is not the base tree: %v %v", box, err)
	}
	if err := os.Remove(filepath.Join(g.root, "skew")); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(g.root, "fail-run"), "", 0o644)
	if box, err := startContainer(cfg.Docker, fakeImage, containerName("fail"), "test", time.Minute, cfg.Tree, cfg.Base); box == nil || err == nil {
		t.Errorf("a failed docker run returned %v %v, want the handle and the error", box, err)
	}
}

// aFailedWriteBackIsCountedApart: a write-back that
// cannot be applied is counted in write_back_failures and not in
// sync_failures; a write-back that works leaves the index, the staging and the
// diff of the agent as they were; a log line that cannot be written is noted.
func aFailedWriteBackIsCountedApart(t *testing.T) {
	g := newStage6Rig(t)
	cfg := g.start()
	put(t, filepath.Join(cfg.Tree, "notes.txt"), "staged\n", 0o644)
	gitIn(t, cfg.Tree, "add", "notes.txt")
	gitIn(t, cfg.Tree, "status", "--porcelain")
	snap := func() [3]string {
		idx, _ := os.ReadFile(filepath.Join(cfg.Tree, ".git", "index"))
		return [3]string{string(idx), gitIn(t, cfg.Tree, "diff"), gitIn(t, cfg.Tree, "diff", "--cached")}
	}
	before := snap()
	if _, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", "echo n > wb.txt"); code != 0 {
		t.Fatalf("the write-back failed: %d %s", code, stderr)
	}
	if !exists(filepath.Join(cfg.Tree, "wb.txt")) || snap() != before {
		t.Error("the write-back was lost, or it changed the index, the staging or the diff of the agent")
	}

	t.Setenv("HOSTPARENT", filepath.Dir(cfg.Tree))
	code := `mkdir -p sub2; echo x > sub2/f.txt; echo h > "$HOSTPARENT/tree/sub2"`
	_, stderr, c := g.shim(cfg, cfg.Tree, "python", "-c", code)
	if c == 0 || !strings.Contains(stderr, "did not reach the tree") {
		t.Errorf("a write-back that cannot be applied: exit %d, stderr %q", c, stderr)
	}
	last := readLog(cfg.logPath())
	if l := last[len(last)-1]; !l.WriteBackFailed || l.SyncFailed {
		t.Errorf("the log records %+v, want a failed write-back and no failed sync", l)
	}
	if s, _ := shimCounts(cfg.logPath(), nil, "", "", ""); s.WriteBackFailures != 1 || s.SyncFailures != 0 {
		t.Errorf("the counts are %+v, want 1 write-back failure and 0 sync failures", s)
	}

	other := *cfg
	other.State = filepath.Join(g.work, "shim2")
	if err := os.MkdirAll(filepath.Join(other.State, "shim.log"), 0o755); err != nil {
		t.Fatal(err)
	}
	g.shim(&other, other.Tree, "python", "-c", "true")
	if _, note := shimCounts(other.logPath(), nil, "", "", ""); !strings.Contains(note, "calls not logged") {
		t.Error("a log line that could not be written left no note")
	}
}

// theCommandVariablesAreForwarded: a variable the
// command set or changed reaches the container with the paths of the tree
// mapped; PATH, HOME and TMPDIR stay the container own.
func theCommandVariablesAreForwarded(t *testing.T) {
	g := newStage6Rig(t)
	cfg := g.start()
	t.Setenv("KEPT_VAR", "same")
	if err := writeStartEnv(cfg.State, os.Environ()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DJANGO_SETTINGS_MODULE", "x")
	t.Setenv("PYTHONPATH", cfg.Tree+"/lib")
	t.Setenv("PATH", os.Getenv("PATH")+":/hostonly")
	t.Setenv("HOME", "/hostonly-home")
	t.Setenv("TMPDIR", "/hostonly-tmp")
	t.Setenv("LC_ALL", "C.UTF-8")
	t.Setenv("GIT_FOO", "1")
	t.Setenv("BASH_FOO", "1")
	out, stderr, code := g.shim(cfg, cfg.Tree, "python", "-c", "echo $DJANGO_SETTINGS_MODULE $PYTHONPATH")
	if want := "x " + filepath.Join(g.root, "testbed", "lib") + "\n"; out != want || code != 0 {
		t.Errorf("the container saw %q exit %d %s, want %q", out, code, stderr, want)
	}
	execs := g.callsMatching("exec -i -e ")
	if len(execs) != 1 {
		t.Fatalf("exec calls with forwarded variables: %q", execs)
	}
	for _, flag := range []string{"-e DJANGO_SETTINGS_MODULE=x", "-e PYTHONPATH=/testbed/lib", "-e LC_ALL=C.UTF-8"} {
		if !strings.Contains(execs[0], flag) {
			t.Errorf("%q lacks %s", execs[0], flag)
		}
	}
	for _, flag := range []string{"-e PATH=", "-e HOME=", "-e TMPDIR=", "KEPT_VAR", "GIT_FOO", "BASH_FOO"} {
		if strings.Contains(execs[0], flag) {
			t.Errorf("%q forwards %s", execs[0], flag)
		}
	}
}

// anUnloggedCallMatchesNoLogEntry: a bare python or pytest
// command matches an unused log entry of the same program and arguments, so a
// shim call cannot hide another that went round the shims.
func anUnloggedCallMatchesNoLogEntry(t *testing.T) {
	calls := []shimCall{{Name: "pytest", Args: []string{"-q", "tests/a.py"}}, {Name: "python", Args: []string{"-c", "print(1)"}}}
	cmds := []string{"pytest -q tests/a.py", "python -c 'print(1)' > out", "pytest -q tests/a.py", "pytest -x"}
	_, bare, unlogged, _ := countCommands(cmds, calls, "", "", "")
	if bare != 4 || unlogged != 2 {
		t.Errorf("bare %d, unlogged %d, want 4 and 2: the second pytest -q tests/a.py has no entry of its own, and pytest -x none", bare, unlogged)
	}
	check := func(cmd string, want ...string) {
		if got := bareCalls(cmd); len(got) != 1 || !slices.Equal(got[0].args, want) {
			t.Errorf("bareCalls(%q) = %+v, want one call with arguments %q", cmd, got, want)
		}
	}
	check("python -m pytest tests/ 2>&1 | tail", "-m", "pytest", "tests/")
	check("python -c \"print(\\\"hi\\\")\"", "-c", "print(\"hi\")")
	check("python a\\ b.py \\\n c", "a b.py", "c")
	check("cd sub\npython t.py", "t.py")
	if _, _, u, _ := countCommands([]string{"pytest -x"}, calls[:1], "", "", ""); u != 1 {
		t.Errorf("a pytest with other arguments than the log entry counted %d unlogged, want 1", u)
	}
	tools := t.TempDir()
	if h, _, _, _ := countCommands([]string{tools + "/python -c 1"}, nil, "", "", tools); h != 0 {
		t.Errorf("the shim in the tools directory of the attempt counted as a host interpreter: %d", h)
	}
}

// aSignalKillsTheContainerCommand: a shim
// that is told to stop takes its command in the container with it.
func aSignalKillsTheContainerCommand(t *testing.T) {
	g := newStage6Rig(t)
	cfg := g.start()
	put(t, filepath.Join(g.root, "leftover"), "", 0o644)
	cmd := exec.Command(cfg.Bench, append([]string{"shim"}, append(cfg.args("python"), "-c", "sleep 120")...)...)
	cmd.Dir = cfg.Tree
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(g.root, "opt", "bench", "pid")
	pid := 0
	for range 200 {
		b, err := os.ReadFile(pidFile)
		if n, aerr := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && aerr == nil {
			pid = n
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if pid == 0 {
		_ = cmd.Process.Kill()
		t.Fatal("the command never recorded its pid")
	}
	p, _ := os.FindProcess(pid)
	t.Cleanup(func() { _ = p.Kill() })
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		t.Error("the shim did not stop after the signal")
	}
	if p.Signal(syscall.Signal(0)) == nil {
		t.Error("the command in the container outlived the signal to its shim")
	}
	if calls := readLog(cfg.logPath()); len(calls) != 1 {
		t.Errorf("the log holds %d calls after a signal, want 1", len(calls))
	}
}

// aCommandGroupIsKilledWithItsLeader: the pid a shim records leads a process
// group, and what the command started in the container goes with it.
func aCommandGroupIsKilledWithItsLeader(t *testing.T) {
	g := newStage6Rig(t)
	cfg := g.start()
	put(t, filepath.Join(g.root, "leftover"), "", 0o644)
	put(t, filepath.Join(g.root, "leader"), "", 0o644)
	if err := os.MkdirAll(filepath.Join(g.root, "opt", "bench"), 0o755); err != nil {
		t.Fatal(err)
	}
	gpid := filepath.Join(g.work, "gpid")
	code := "sleep 120 & echo $! > " + shellQuote(gpid) + "; wait"
	cmd := exec.Command(cfg.Docker, "exec", "-i", cfg.Container, "bash", "-c", activateScript, "bash", "sh", "-c", code)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	child := 0
	for range 100 {
		b, err := os.ReadFile(gpid)
		if n, aerr := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && aerr == nil {
			child = n
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if child == 0 {
		_ = cmd.Process.Kill()
		t.Fatal("the command never started its child")
	}
	p, _ := os.FindProcess(child)
	t.Cleanup(func() { _ = p.Kill() })
	alive := func() bool { return p.Signal(syscall.Signal(0)) == nil }
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if !alive() {
		t.Fatal("the fixture did not leave a child running")
	}
	if _, stderr, c := g.shim(cfg, cfg.Tree, "python", "-c", "true"); c != 0 {
		t.Fatalf("the shim call failed: %d %s", c, stderr)
	}
	for range 60 {
		if !alive() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if alive() {
		t.Error("a child of the command survived: only its leader was killed")
	}
}
