package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// testbed is where a SWE-bench image keeps the repository under test.
const testbed = "/testbed"

// containerTmp holds a shim own files inside the container, outside the
// repository, so a sync never sees them.
const containerTmp = "/opt/bench"

// activateTestbed puts the image testbed environment on PATH. T1 S1 read it from
// swebench/sweb.eval.x86_64.django_1776_django-16642 (2026-10-21): bash takes it
// and puts /opt/miniconda3/envs/testbed/bin first. It needs bash: the image's
// sh is dash, which has no source, and dash's . drops the argument, so a sh
// run would activate the base environment, not testbed. The shims run bash.
const activateTestbed = "source /opt/miniconda3/bin/activate testbed"

// evalArtifacts are the paths, or globs, removed from every image before the
// first command of the agent: a test patch, an evaluation script, a list of
// test names (ADR-163 Decision 1). T1 S1 found none in the image it read. A
// search outside /testbed, /opt/miniconda3, /usr, /etc and /var/lib for patch,
// diff, eval*, sh, json and txt files found the miniconda installer, a pip cache
// record and conda's environments list, none of them about a test. The image's
// extra "SWE-bench" commit on top of the base is empty and goes with the history
// replaceTree drops.
var evalArtifacts = []string{}

// settled says whether T1 S1 has looked in a real image and filled evalArtifacts
// and activateTestbed from what it found. While it is false, gates refuses
// bench run under stage6; bench canary still runs. T1 S1 recorded the
// inspection on 2026-10-21 and flipped it.
const settled = true

// syncExcludes are what a sync and a write-back leave out beside what
// .gitignore covers (ADR-163 Decision 1), as git pathspecs.
var syncExcludes = []string{":(exclude,glob)**/__pycache__/**", ":(exclude,glob)**/*.pyc", ":(exclude,glob)**/.pytest_cache/**"}

// snapshotHead builds the tree of /testbed as it is now in a temporary index
// of its own: the container real index is never the record of what a command
// changed.
var snapshotHead = "export GIT_INDEX_FILE=" + containerTmp + "/index; git read-tree HEAD; git add -A -- . " + quotedExcludes()

// snapshotShell prints that tree object.
var snapshotShell = snapshotHead + "; git write-tree"

// snapshotAfter prints the tree object once a command has run, given the tree
// of the sync as its first argument: every path that tree held and that is
// still there stays in, whatever the ignore rules say now. A file that becomes
// ignored in mid-command is not a deletion.
var snapshotAfter = snapshotHead + "; git ls-tree -r --name-only -z " + "\"$1\"" + " | git update-index --add --remove -z --stdin; git write-tree"

func quotedExcludes() string {
	var q []string
	for _, e := range syncExcludes {
		q = append(q, shellQuote(e))
	}
	return strings.Join(q, " ")
}

// container is the attempt own container of the instance image.
type container struct{ docker, name string }

// containerName is a docker-legal name for an attempt, unique to this start.
func containerName(id string) string {
	clean := regexp.MustCompile(`[^A-Za-z0-9_.-]`).ReplaceAllString(id, "_")
	return fmt.Sprintf("bench-%s-%d", clean, time.Now().UnixNano())
}

// dockerRun runs docker with stdin, and returns its standard output; a failure
// carries its standard error.
func dockerRun(docker string, stdin io.Reader, args ...string) ([]byte, error) {
	var out, errb bytes.Buffer
	cmd := exec.Command(docker, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker %s: %w: %s%s", strings.Join(args[:min(len(args), 4)], " "), err, errb.String(), out.String())
	}
	return out.Bytes(), nil
}

// containerLabel marks every container bench starts, so a sweep can find what
// a killed run left behind.
const containerLabel = "mrw-bench-attempt"

// startContainer starts the attempt container with no network and makes
// /testbed the host tree base commit and nothing else: the tracked files of
// the image go, those of the base come, an evaluation artifact the table
// names goes, and /testbed/.git becomes a fresh repository of that one
// commit, so no history, reflog or upstream object survives (ADR-163
// Decision 1). Ignored build output stays. The tree of that commit must be the
// tree of the base on the host, or the start fails. The container removes
// itself after life, and is returned even when a step fails, the start
// included, for the caller to remove.
func startContainer(docker, image, name, label string, life time.Duration, tree, base string) (*container, error) {
	c := &container{docker: docker, name: name}
	if _, err := dockerRun(docker, nil, "run", "-d", "--rm", "--name", name, "--label", containerLabel+"="+label, "--network", "none", "--platform", "linux/amd64", image, "sleep", strconv.Itoa(int(life.Seconds()))); err != nil {
		return c, err
	}
	inBox := func(stdin io.Reader, args ...string) ([]byte, error) {
		return dockerRun(docker, stdin, append([]string{"exec", "-i", name}, args...)...)
	}
	// The untracked files go first: once the tracked .gitignore is removed, the
	// build output it covers would no longer be ignored, and would go with them.
	if _, err := inBox(nil, "sh", "-c", "set -e; cd "+testbed+"; git clean -fdq; git ls-files -z | xargs -0 rm -f"); err != nil {
		return c, err
	}
	var archive, stderr bytes.Buffer
	ar := exec.Command("git", "-C", tree, "archive", "--format=tar", base)
	ar.Stdout, ar.Stderr = &archive, &stderr
	if err := ar.Run(); err != nil {
		return c, fmt.Errorf("git archive %s: %w: %s", base, err, stderr.String())
	}
	if _, err := inBox(&archive, "tar", "--no-same-owner", "-x", "-C", testbed); err != nil {
		return c, err
	}
	if len(evalArtifacts) > 0 {
		if _, err := inBox(nil, append([]string{"sh", "-c", "for p; do rm -rf -- $p; done", "sh"}, evalArtifacts...)...); err != nil {
			return c, err
		}
	}
	fresh := "set -e; cd " + testbed + "; rm -rf .git; git init -q; git add -A; git -c user.name=bench -c user.email=bench@localhost -c commit.gpgsign=false commit -q --no-verify -m base; git rev-parse HEAD^{tree}"
	out, err := inBox(nil, "sh", "-c", fresh)
	if err != nil {
		return c, err
	}
	want, err := exec.Command("git", "-C", tree, "rev-parse", base+"^{tree}").Output()
	if err != nil {
		return c, fmt.Errorf("git rev-parse %s^{tree}: %w", base, err)
	}
	if got := strings.TrimSpace(string(out)); got != strings.TrimSpace(string(want)) {
		return c, fmt.Errorf("the tree of the container commit is %s, the host base tree is %s: it does not start from the base", got, strings.TrimSpace(string(want)))
	}
	return c, nil
}

// sweepContainers removes every container carrying the label of bench for this
// stage: what a run killed before its cleanup left behind. The containers of
// another stage are not touched.
func sweepContainers(docker, stage string) error {
	out, err := dockerRun(docker, nil, "ps", "-aq", "--filter", "label="+containerLabel+"="+stage)
	if err != nil {
		return err
	}
	for _, id := range strings.Fields(string(out)) {
		if _, err := dockerRun(docker, nil, "rm", "-f", id); err != nil && !strings.Contains(err.Error(), "No such container") {
			return err
		}
	}
	return nil
}

// remove removes the container, running or not; one already gone is no error.
func (c *container) remove() error {
	out, err := exec.Command(c.docker, "rm", "-f", c.name).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "No such container") {
		return fmt.Errorf("docker rm -f %s: %w: %s", c.name, err, out)
	}
	return nil
}

// shimConfig is what the shims of one attempt are told: bench itself, which
// they run; the docker CLI by absolute path (the PATH of an attempt holds
// none); the container; the host tree and the base commit of bench; and the
// directory holding the lock and the log.
type shimConfig struct{ Bench, Docker, Container, Tree, State, Base string }

func (c shimConfig) logPath() string { return filepath.Join(c.State, "shim.log") }

// args are the flags of `bench shim` for the shim of this name, ending at `--`.
func (c shimConfig) args(name string) []string {
	return []string{"--docker", c.Docker, "--container", c.Container, "--tree", c.Tree, "--state", c.State, "--base", c.Base, "--name", name, "--"}
}

// shimNames are the commands an attempt finds as shims.
var shimNames = []string{"python", "python3", "pytest", "py.test"}

// writeShims writes the shims into dir; each one hands its command to
// `bench shim`, where the lock, the sync and the log are.
func writeShims(dir string, c shimConfig) error {
	if err := os.MkdirAll(c.State, 0o755); err != nil {
		return err
	}
	for _, name := range shimNames {
		quoted := []string{shellQuote(c.Bench), "shim"}
		for _, a := range c.args(name) {
			quoted = append(quoted, shellQuote(a))
		}
		script := "#!/bin/sh\nexec " + strings.Join(quoted, " ") + " " + "\"$@\"" + "\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// shimCall is one line of the shim log.
type shimCall struct {
	Name            string   `json:"name"`
	Args            []string `json:"args"`
	Cwd             string   `json:"cwd"`
	WaitMS          int64    `json:"wait_ms"`
	Exit            int      `json:"exit"`
	Refused         bool     `json:"refused,omitempty"`
	SyncFailed      bool     `json:"sync_failed,omitempty"`
	WriteBacks      int      `json:"write_backs,omitempty"`
	WriteBackFailed bool     `json:"write_back_failed,omitempty"`
	Conflicts       []string `json:"conflicts,omitempty"`
}

// Shim is `bench shim`: one python, python3, pytest or py.test command of an
// attempt, run in the attempt container (ADR-163 Decision 1). It takes the
// per-container lock, resets /testbed to its one commit, applies the change
// of the tree since the base commit of bench, runs the command there under the
// `testbed` environment with the working directory and the paths of the tree
// in its arguments mapped, writes back what the command changed, and logs the
// call. Exit 2: the working directory is outside the tree. A change that does
// not apply, or a write-back that conflicts or fails, fails the command and is
// counted. A signal to the shim kills the command in the container too, since
// a docker exec client that dies leaves its command running outside the lock.
func Shim(args []string, cwd string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
	var c shimConfig
	var name string
	fs := flag.NewFlagSet("shim", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&c.Docker, "docker", "", "the docker CLI")
	fs.StringVar(&c.Container, "container", "", "the attempt container")
	fs.StringVar(&c.Tree, "tree", "", "the attempt tree on the host")
	fs.StringVar(&c.State, "state", "", "where the lock and the log are")
	fs.StringVar(&c.Base, "base", "", "the base commit of bench in the tree")
	fs.StringVar(&name, "name", "", "the command the shim stands for")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	argv := fs.Args()
	call := shimCall{Name: name, Args: argv, Cwd: cwd}
	defer func() {
		call.Exit = code
		if c.State != "" {
			appendShimLog(c.logPath(), call)
		}
	}()
	rel, ok := within(c.Tree, cwd)
	if !ok {
		call.Refused = true
		fmt.Fprintf(stderr, "bench shim: the working directory %s is outside the attempt tree %s: a shim maps the tree and nothing outside it\n", cwd, c.Tree)
		return 2
	}
	start := time.Now()
	unlock, err := lockFile(filepath.Join(c.State, "lock"))
	if err != nil {
		fmt.Fprintf(stderr, "bench shim: the container lock: %v\n", err)
		return 1
	}
	defer unlock()
	call.WaitMS = time.Since(start).Milliseconds()
	// A signal from here on kills what the shim has in the container, the sync
	// and the write-back included: their shells record a pid of their own.
	sig := make(chan os.Signal, 1)
	if sigs := subproc.Signals(); len(sigs) > 0 {
		signal.Notify(sig, sigs...)
	}
	stop, gone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(gone)
		select {
		case <-sig:
			_ = c.killOrphan()
		case <-stop:
		}
	}()
	defer func() {
		close(stop)
		<-gone
		signal.Stop(sig)
	}()
	synced, err := c.syncIn()
	if err != nil {
		call.SyncFailed = true
		fmt.Fprintf(stderr, "bench shim: the change of the tree did not reach the container, so nothing ran: %v\n", err)
		return 1
	}
	pre := append([]string{"exec", "-i"}, c.forwardedEnv(os.Environ())...)
	run := append(pre, append([]string{"-w", joinBox(testbed, rel), c.Container, "bash", "-c", activateScript, "bash", name}, c.mapArgs(argv)...)...)
	cmd := exec.Command(c.Docker, run...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	runErr := cmd.Run()
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			code = ee.ExitCode()
		} else {
			fmt.Fprintf(stderr, "bench shim: docker exec: %v\n", runErr)
			code = 1
		}
	}
	written, conflicts, err := c.writeBack(synced)
	call.WriteBacks, call.Conflicts = written, conflicts
	switch {
	case err != nil:
		call.WriteBackFailed = true
		fmt.Fprintf(stderr, "bench shim: what the command wrote did not reach the tree: %v\n", err)
	case len(conflicts) > 0:
		fmt.Fprintf(stderr, "bench shim: write-back conflict, the tree changed these since the sync and nothing was written for them: %s\n", strings.Join(conflicts, ", "))
	}
	if code == 0 && (err != nil || len(conflicts) > 0) {
		code = 1
	}
	return code
}

// killOrphan runs killOrphanShell in the container.
func (c shimConfig) killOrphan() error {
	_, err := dockerRun(c.Docker, nil, "exec", c.Container, "sh", "-c", killOrphanShell)
	return err
}

// activateScript runs the command of the shim, handed to bash as arguments,
// once the testbed environment is active, after recording its pid for
// killOrphanShell; a failed activation fails the call.
var activateScript = "echo $$ > " + containerTmp + "/pid; if ! " + activateTestbed + " >/dev/null 2>&1; then echo bench shim: the testbed environment did not activate 1>&2; exit 126; fi; exec \"$@\""

// killOrphanShell kills what a shim left in the container, if it is still
// there, and waits for it to go. Every container shell of a transaction records
// its pid in a file of its own under /opt/bench (pid for the command, pid.sync,
// pid.post and pid.diff for the others), since a docker exec client that dies
// does not stop its shell. Each pid goes with its group, which it leads when
// the image starts an exec process as one (the group first, then the pid: ps
// may be absent).
var killOrphanShell = `for f in ` + containerTmp + `/pid ` + containerTmp + `/pid.*; do if [ -s "$f" ]; then p=$(cat "$f"); if [ "$p" != "$$" ]; then kill -KILL -$p 2>/dev/null || true; kill -KILL $p 2>/dev/null || true; i=0; while kill -0 $p 2>/dev/null && [ $i -lt 50 ]; do sleep 0.1; i=$((i+1)); done; rm -f "$f"; fi; else rm -f "$f"; fi; done`

// neverForward names the variables a shim does not hand to the container: the
// PATH, HOME and TMPDIR of the container stay its own, and so do the variables
// of bench itself and those the agent harness sets for itself.
var neverForward = regexp.MustCompile(`^(PATH|HOME|TMPDIR|USER|PWD|OLDPWD|SHLVL|_|TERM|SHELL|MRW_.*|BENCH_.*|CLAUDE.*|ANTHROPIC_.*|CODEX_.*|XDG_.*|GIT_.*|BASH_.*)$`)

// writeStartEnv records the environment the agent starts with, for the shims
// to tell what a command set or changed since.
func writeStartEnv(state string, env []string) error {
	return os.WriteFile(filepath.Join(state, "env0"), []byte(strings.Join(env, "\x00")), 0o644)
}

// forwardedEnv is the docker exec flags for every variable in now that the
// agent command set or changed, absent or different in the starting
// environment, with the paths of the tree in its value mapped to /testbed.
func (c shimConfig) forwardedEnv(now []string) []string {
	start := map[string]string{}
	b, _ := os.ReadFile(filepath.Join(c.State, "env0"))
	for _, kv := range strings.Split(string(b), "\x00") {
		if k, v, ok := strings.Cut(kv, "="); ok {
			start[k] = v
		}
	}
	var pairs []string
	for _, kv := range now {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || neverForward.MatchString(k) {
			continue
		}
		if old, had := start[k]; had && old == v {
			continue
		}
		pairs = append(pairs, k+"="+c.mapArgs([]string{v})[0])
	}
	slices.Sort(pairs)
	var out []string
	for _, p := range pairs {
		out = append(out, "-e", p)
	}
	return out
}

// joinBox joins a container path and a slash-separated relative one.
func joinBox(root, rel string) string {
	if rel == "" {
		return root
	}
	return root + "/" + rel
}

// within says where p is below tree, slash-separated, resolving links on both
// sides: macOS spells a temporary directory two ways.
func within(tree, p string) (string, bool) {
	real := func(s string) string {
		if r, err := filepath.EvalSymlinks(s); err == nil {
			return r
		}
		return filepath.Clean(s)
	}
	rel, err := filepath.Rel(real(tree), real(p))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if rel == "." {
		return "", true
	}
	return filepath.ToSlash(rel), true
}

// mapArgs rewrites the path of the tree, however it is spelled, to /testbed in
// each argument; a relative path is already right under the mapped working
// directory, and a path outside the tree is left alone.
func (c shimConfig) mapArgs(argv []string) []string {
	spellings := []string{c.Tree}
	if real, err := filepath.EvalSymlinks(c.Tree); err == nil && real != c.Tree {
		spellings = append(spellings, real)
	}
	// The longest spelling first: the resolved path may contain the other.
	slices.SortFunc(spellings, func(a, b string) int { return len(b) - len(a) })
	out := make([]string, len(argv))
	for i, a := range argv {
		for _, s := range spellings {
			a = replaceTree(a, s)
		}
		out[i] = a
	}
	return out
}

// replaceTree rewrites the path of the tree to /testbed where it ends: at the
// end of the argument, before a slash, or before a character that cannot
// continue a path name. /tmp/a/tree-backup is another directory than
// /tmp/a/tree and stays as it is.
func replaceTree(a, tree string) string {
	var b strings.Builder
	pos := 0
	for {
		i := strings.Index(a[pos:], tree)
		if i < 0 {
			b.WriteString(a[pos:])
			return b.String()
		}
		i += pos
		end := i + len(tree)
		b.WriteString(a[pos:i])
		startsHere := i == 0 || !(namePart(a[i-1]) || a[i-1] == 0x2f)
		endsHere := end == len(a) || !namePart(a[end])
		if startsHere && endsHere {
			b.WriteString(testbed)
		} else {
			b.WriteString(tree)
		}
		pos = end
	}
}

// namePart says whether c can continue a file name.
func namePart(c byte) bool {
	return c >= 0x30 && c <= 0x39 || c >= 0x41 && c <= 0x5a || c >= 0x61 && c <= 0x7a || c >= 0x80 || strings.IndexByte("._-+~@%", c) >= 0
}

// hostGit runs git in the tree with the extra environment and standard input.
func (c shimConfig) hostGit(env []string, stdin []byte, args ...string) ([]byte, error) {
	var out, errb bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = c.Tree, &out, &errb
	cmd.Env = append(os.Environ(), env...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args[:min(len(args), 2)], " "), err, errb.String())
	}
	return out.Bytes(), nil
}

// hostTree is the tree object of the host tree as it is now, built in a
// temporary index of the state directory: the index and the staging of the
// agent are never touched.
func (c shimConfig) hostTree() (string, error) {
	index := filepath.Join(c.State, "host-index")
	defer func() { _ = os.Remove(index) }()
	env := []string{"GIT_INDEX_FILE=" + index}
	if _, err := c.hostGit(env, nil, "read-tree", c.Base); err != nil {
		return "", err
	}
	if _, err := c.hostGit(env, nil, append([]string{"add", "-A", "--", "."}, syncExcludes...)...); err != nil {
		return "", err
	}
	out, err := c.hostGit(env, nil, "write-tree")
	return strings.TrimSpace(string(out)), err
}

// synced is what a sync left: the host tree object it sent and the container
// tree object once the change was applied.
type synced struct{ host, box string }

// syncIn puts the change of the tree since the base into the container: reset
// to the one commit keeping ignored output, apply the binary diff, snapshot.
func (c shimConfig) syncIn() (synced, error) {
	h, err := c.hostTree()
	if err != nil {
		return synced{}, err
	}
	patch, err := c.hostGit(nil, nil, "diff", "--binary", "--full-index", "--no-ext-diff", c.Base, h)
	if err != nil {
		return synced{}, err
	}
	patchFile := containerTmp + "/in.patch"
	script := "set -e; mkdir -p " + containerTmp + "; echo $$ > " + containerTmp + "/pid.sync.$$; " + killOrphanShell + "; cat > " + patchFile + "; cd " + testbed +
		"; git reset -q --hard; git clean -fdq; if [ -s " + patchFile + " ]; then git apply --binary --whitespace=nowarn " + patchFile + "; fi; t1=$(" + snapshotShell + "); rm -f " + containerTmp + "/pid.sync.$$; printf '%s\n' " + "\"$t1\""
	out, err := dockerRun(c.Docker, bytes.NewReader(patch), "exec", "-i", c.Container, "sh", "-c", script)
	if err != nil {
		return synced{}, err
	}
	return synced{host: h, box: strings.TrimSpace(string(out))}, nil
}

// changedPaths splits the NUL-separated names that `git diff-tree -z` prints.
func changedPaths(b []byte) []string {
	var out []string
	for _, p := range bytes.Split(b, []byte{0}) {
		if len(p) > 0 {
			out = append(out, string(p))
		}
	}
	return out
}

// writeBack carries what the command changed in /testbed into the host tree. A
// path the host changed too since the sync is a conflict: it is not written,
// and the caller names it.
func (c shimConfig) writeBack(s synced) (written int, conflicts []string, err error) {
	post := "set -e; mkdir -p " + containerTmp + "; echo $$ > " + containerTmp + "/pid.post.$$; " + killOrphanShell + "; cd " + testbed + "; t2=$(" + snapshotAfter + "); printf '%s\n' " + "\"$t2\"; git diff-tree -r --name-only -z --no-renames \"$1\" \"$t2\"; rm -f " + containerTmp + "/pid.post.$$"
	out, err := dockerRun(c.Docker, nil, "exec", c.Container, "sh", "-c", post, "sh", s.box)
	if err != nil {
		return 0, nil, err
	}
	t2, names, _ := bytes.Cut(out, []byte("\n"))
	changed := changedPaths(names)
	if len(changed) == 0 {
		return 0, nil, nil
	}
	h2, err := c.hostTree()
	if err != nil {
		return 0, nil, err
	}
	moved, err := c.hostGit(nil, nil, "diff-tree", "-r", "--name-only", "-z", "--no-renames", s.host, h2)
	if err != nil {
		return 0, nil, err
	}
	host := map[string]bool{}
	for _, p := range changedPaths(moved) {
		host[p] = true
	}
	var allowed []string
	for _, p := range changed {
		if host[p] {
			conflicts = append(conflicts, p)
		} else {
			allowed = append(allowed, p)
		}
	}
	if len(allowed) == 0 {
		return 0, conflicts, nil
	}
	diff := "set -e; mkdir -p " + containerTmp + "; echo $$ > " + containerTmp + "/pid.diff.$$; cd " + testbed + "; a=$1; b=$2; shift 2; GIT_LITERAL_PATHSPECS=1 git diff --binary --full-index --no-ext-diff \"$a\" \"$b\" -- \"$@\"; rm -f " + containerTmp + "/pid.diff.$$"
	patch, err := dockerRun(c.Docker, nil, append([]string{"exec", c.Container, "sh", "-c", diff, "sh", s.box, string(t2)}, allowed...)...)
	if err != nil {
		return 0, conflicts, err
	}
	if _, err := c.hostGit(nil, patch, "apply", "--binary", "--whitespace=nowarn", "-"); err != nil {
		return 0, conflicts, err
	}
	return len(allowed), conflicts, nil
}

// appendShimLog adds one call to the log, as one write. A call that cannot be
// written is noted in a file beside the log, which the receipt surfaces: a lost
// line would show a real shim call as one that bypassed the shims.
func appendShimLog(path string, call shimCall) {
	b, err := json.Marshal(call)
	if err == nil {
		err = appendLine(path, append(b, 10))
	}
	if err != nil {
		if f, ferr := os.OpenFile(path+".errors", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); ferr == nil {
			_, _ = fmt.Fprintf(f, "%s: %v\n", call.Name, err)
			_ = f.Close()
		}
	}
}

// appendLine appends one line to a file, and says whether all of it was kept.
func appendLine(path string, line []byte) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(line)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// readShimLog reads the calls the shims logged. A log that was never started
// is no calls; one that cannot be read to its end is an error.
func readShimLog(path string) ([]shimCall, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var calls []shimCall
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var c shimCall
		if json.Unmarshal(sc.Bytes(), &c) == nil {
			calls = append(calls, c)
		}
	}
	return calls, sc.Err()
}

// ShimCounts are what the receipt of a container attempt adds (ADR-163
// Decision 6): the log of the shims, and what the transcript shows that went
// round them. A pointer on the receipt, so a stage without the container has
// none of these keys and a stage with it reports a zero as 0.
type ShimCounts struct {
	ShimCalls            int   `json:"shim_calls"`
	ShimWaitMS           int64 `json:"shim_wait_ms"`
	SyncFailures         int   `json:"sync_failures"`
	WriteBacks           int   `json:"write_backs"`
	WriteBackConflicts   int   `json:"write_back_conflicts"`
	WriteBackFailures    int   `json:"write_back_failures"`
	HostInterpreterCalls int   `json:"host_interpreter_calls"`
	BarePythonCalls      int   `json:"bare_python_calls"`
	UnloggedPythonCalls  int   `json:"unlogged_python_calls"`
	DockerCalls          int   `json:"docker_calls"`
}

// shimCounts reads the shim log and counts the Bash commands of the
// transcript. tools is the tools directory of the attempt, whose own
// interpreters are not host ones. The second result says what could not be
// read or written in the log, for the Note of the receipt.
func shimCounts(log string, commands []string, image, box, tools string) (*ShimCounts, string) {
	s := &ShimCounts{}
	calls, err := readShimLog(log)
	var notes []string
	if err != nil {
		notes = append(notes, err.Error())
	}
	if b, rerr := os.ReadFile(log + ".errors"); rerr == nil {
		notes = append(notes, "calls not logged: "+strings.TrimSpace(string(b)))
	}
	for _, c := range calls {
		s.ShimCalls++
		s.ShimWaitMS += c.WaitMS
		if c.SyncFailed {
			s.SyncFailures++
		}
		if c.WriteBackFailed {
			s.WriteBackFailures++
		}
		s.WriteBacks += c.WriteBacks
		s.WriteBackConflicts += len(c.Conflicts)
	}
	s.HostInterpreterCalls, s.BarePythonCalls, s.UnloggedPythonCalls, s.DockerCalls = countCommands(commands, calls, image, box, tools)
	return s, strings.Join(notes, "; ")
}

var (
	// An absolute path to one of the host interpreters or runners.
	hostRunner = regexp.MustCompile(`(?:^|[\s\x27\x22=;&|(])/[\w.+/-]*/(?:python[\d.]*|pip[\d.]*|pytest|py\.test|uvx?)(?:[\s\x27\x22;&|)]|$)`)
	// A wrapper that skips PATH, or the xcrun of the host, before one.
	hostWrapped = regexp.MustCompile(`\b(?:command\s+-p|xcrun)\s+[^;&|]*?\b(?:python[\d.]*|pip[\d.]*|pytest|py\.test|uvx?)(?:[\s\x27\x22;&|)]|$)`)
	// A python or pytest by bare name where a command starts; the name is the
	// first group, and the character after it is judged in bareCalls.
	pythonCommand = regexp.MustCompile(`(?:^|[;&|(\x60\n]|\$\(|\b(?:then|do|else|exec|time|nohup)\b)\s*(?:[A-Za-z_]\w*=\S*\s+)*(python[\d.]*|pytest|py\.test)`)
	dockerWord    = regexp.MustCompile(`\bdocker\b`)
)

// bareCall is a python or pytest the agent typed by bare name, with the words
// after it up to the end of the command.
type bareCall struct {
	name string
	args []string
}

// bareCalls finds the bare invocations in one Bash command.
func bareCalls(command string) []bareCall {
	var out []bareCall
	for _, m := range pythonCommand.FindAllStringSubmatchIndex(command, -1) {
		if rest := command[m[3]:]; rest != "" && !strings.ContainsAny(rest[:1], " \t\n;&|)") {
			continue
		}
		out = append(out, bareCall{command[m[2]:m[3]], shellWords(command[m[3]:])})
	}
	return out
}

// shellWords splits the start of a command into words as a shell reads them:
// quotes removed; outside quotes a backslash makes the next character plain
// and a backslash-newline vanishes; inside double quotes a backslash does so
// only before a double quote, a backslash, a dollar or a backtick. It stops at
// the first unquoted ; & | ) < > or newline, dropping an all-digit word that a
// redirection takes for its file descriptor (the 2 of 2>&1).
func shellWords(s string) []string {
	var words []string
	var cur strings.Builder
	have, digits := false, true
	var quote rune
	flush := func() {
		if have {
			words = append(words, cur.String())
		}
		cur.Reset()
		have, digits = false, true
	}
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote == 0x27:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case quote == 0x22:
			switch {
			case r == quote:
				quote = 0
			case r == 0x5c && i+1 < len(rs) && strings.ContainsRune("\"\\$\x60", rs[i+1]):
				i++
				cur.WriteRune(rs[i])
			default:
				cur.WriteRune(r)
			}
		case r == 0x27 || r == 0x22:
			quote, have, digits = r, true, false
		case r == 0x5c:
			if i+1 < len(rs) {
				i++
				if rs[i] != 0x0a {
					cur.WriteRune(rs[i])
					have, digits = true, false
				}
			}
		case r == 0x20 || r == 0x09:
			flush()
		case r == 0x3c || r == 0x3e:
			if have && digits {
				cur.Reset()
				have = false
			}
			flush()
			return words
		case strings.ContainsRune(";&|)\n", r):
			flush()
			return words
		default:
			cur.WriteRune(r)
			have = true
			digits = digits && r >= 0x30 && r <= 0x39
		}
	}
	flush()
	return words
}

// countCommands counts, over the Bash commands of an attempt: those naming a
// host interpreter or runner by path, or behind env -i, command -p or xcrun,
// the interpreters in tools, the tools directory of the attempt, being the
// shims; the bare python and pytest invocations, and those among them that
// match no shim call of the log by program and arguments, which ran somewhere
// the log did not see; and those naming docker, the image of the instance or
// the container, which a stage stops on (ADR-163 Decision 6). It misses a
// script whose shebang names a host interpreter, over-counts a directory named
// python3, and sees arguments as typed: a variable or glob in them is no match.
func countCommands(commands []string, calls []shimCall, image, box, tools string) (host, bare, unlogged, docker int) {
	spellings := []string{tools}
	if real, err := filepath.EvalSymlinks(tools); err == nil && real != tools {
		spellings = append(spellings, real)
	}
	used := make([]bool, len(calls))
	for _, c := range commands {
		judged := c
		for _, s := range spellings {
			if s != "" {
				judged = strings.ReplaceAll(judged, s, "TOOLS")
			}
		}
		if hostRunner.MatchString(judged) || hostWrapped.MatchString(judged) || envBypass(judged) {
			host++
		}
		for _, b := range bareCalls(c) {
			bare++
			matched := false
			for i, call := range calls {
				if !used[i] && call.Name == b.name && slices.Equal(call.Args, b.args) {
					used[i], matched = true, true
					break
				}
			}
			if !matched {
				unlogged++
			}
		}
		if dockerWord.MatchString(c) || (image != "" && strings.Contains(c, image)) || (box != "" && strings.Contains(c, box)) {
			docker++
		}
	}
	return host, bare, unlogged, docker
}

// imageFor is the image whose container an attempt of the task runs in: the
// instance own, or for a task with none (the canary fixture) the one named by
// --canary-image.
func imageFor(o RunOptions, t Task) string {
	if t.swe != nil {
		return t.swe.image
	}
	return o.CanaryImage
}

// resolveContainer fills what the container profile needs before anything
// runs: the docker CLI by absolute path, since the PATH of an attempt holds
// none and the shims call it, and bench itself, which the shims run. It also
// removes the containers an earlier run left behind. Windows is refused: the
// lock of the shims and the profile run on macOS and Linux.
func resolveContainer(o *RunOptions) error {
	if runtime.GOOS == "windows" {
		return errors.New("the stage6 profile runs on macOS and Linux only")
	}
	if o.Docker == "" {
		o.Docker = "docker"
	}
	p, err := exec.LookPath(o.Docker)
	if err != nil {
		return fmt.Errorf("the stage6 profile starts a container, and %w", err)
	}
	if o.Docker, err = filepath.Abs(p); err != nil {
		return err
	}
	if o.Bench == "" {
		return errors.New("the stage6 profile needs bench itself, which the shims run")
	}
	return sweepContainers(o.Docker, o.Stage)
}

var interpreter = regexp.MustCompile(`^(?:python[\d.]*|pip[\d.]*|pytest|py\.test|uvx?)$`)

var assignment = regexp.MustCompile(`^[A-Za-z_]\w*=`)

// envBypass says whether a command runs an interpreter or runner through env
// with a PATH of its own, which no shim stands in: env -i, or env with a PATH=
// word, before the program. env with other variables only finds the shims on
// the PATH of the agent, and is no bypass.
func envBypass(command string) bool {
	for _, at := range commandWordsAt(command, "env") {
		if envRunsInterpreter(shellWords(command[at:])) {
			return true
		}
	}
	return false
}

// commandWordsAt says where, in a command, a word of this name stands at a
// command position: the start, or after an unquoted ; & | ( ) or newline, past
// any NAME=VALUE words (a quoted value included), or at the start of a command
// substitution, $( or a backtick, which also counts inside double quotes. Each
// result is the index just after the word. Text in quotes is no command: the
// env inside python -c 'print("env …")' is not one.
func commandWordsAt(command, name string) []int {
	type frame struct {
		quote   byte
		restore bool
		paren   bool
		subst   bool
	}
	var out []int
	var stack []frame
	atStart, restore, assignHeld := true, false, false
	var quote byte
	push := func(paren, subst bool) {
		stack = append(stack, frame{quote, restore, paren, subst})
		quote, restore, atStart = 0, false, true
	}
	pop := func() {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		quote, restore = f.quote, f.restore
		atStart = f.paren && !f.subst && quote == 0
	}
	top := func() (frame, bool) {
		if len(stack) == 0 {
			return frame{}, false
		}
		return stack[len(stack)-1], true
	}
	for i := 0; i < len(command); {
		c := command[i]
		switch {
		case quote == 0x27:
			if c == quote {
				quote, atStart, restore = 0, restore, false
			}
			i++
		case quote == 0x22:
			switch {
			case c == 0x5c:
				i++
			case c == quote:
				quote, atStart, restore = 0, restore, false
			case c == 0x60:
				push(false, false)
			case c == 0x24 && i+1 < len(command) && command[i+1] == 0x28:
				push(true, true)
				i++
			}
			i++
		case c == 0x27 || c == 0x22:
			quote, restore, assignHeld = c, assignHeld, false
			if !restore {
				atStart = false
			}
			i++
		case c == 0x5c:
			atStart = false
			i += 2
		case c == 0x20 || c == 0x09:
			i++
		case c == 0x60:
			if f, ok := top(); ok && !f.paren {
				pop()
			} else {
				push(false, false)
			}
			i++
		case c == 0x28:
			push(true, i > 0 && command[i-1] == 0x24)
			i++
		case c == 0x29:
			if f, ok := top(); ok && f.paren {
				pop()
			} else {
				atStart = true
			}
			i++
		case c == 0x3b || c == 0x26 || c == 0x7c || c == 0x0a:
			atStart = true
			i++
		default:
			j := i
			for j < len(command) && strings.IndexByte(" \t;&|()\n'\"`", command[j]) < 0 {
				j++
			}
			word := command[i:j]
			if atStart && word == name {
				out = append(out, j)
			}
			atStart = atStart && assignment.MatchString(word)
			assignHeld = atStart && j < len(command) && (command[j] == 0x27 || command[j] == 0x22)
			i = j
		}
	}
	return out
}

// envRunsInterpreter reads the words after env: its options and NAME=VALUE
// words, then the program.
func envRunsInterpreter(words []string) bool {
	reset := false
	for i := 0; i < len(words); i++ {
		w := words[i]
		switch {
		case w == "-u" || w == "-C" || w == "-S":
			i++
		case w == "-i" || w == "-" || w == "--ignore-environment":
			reset = true
		case strings.HasPrefix(w, "-"):
		case strings.HasPrefix(w, "PATH="):
			reset = true
		case strings.Contains(w, "="):
		default:
			return reset && interpreter.MatchString(filepath.Base(w))
		}
	}
	return false
}
