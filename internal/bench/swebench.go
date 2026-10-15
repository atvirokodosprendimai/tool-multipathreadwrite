package bench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// SelectSWEBench is Stage 3's registered selection (BACKLOG, "Pre-registered
// for ADR-156"): the n instance ids with the lowest sha256("adr-156:" + id),
// whatever order the dataset lists them in, returned in that hash order.
func SelectSWEBench(ids []string, n int) []string {
	return SelectSWEBenchSalted(ids, n, "adr-156:")
}

// SelectSWEBenchSalted is SelectSWEBench with the salt named: ADR-160 draws
// Stage 4's slice with "adr-160:".
func SelectSWEBenchSalted(ids []string, n int, salt string) []string {
	type h struct{ id, hash string }
	hs := make([]h, 0, len(ids))
	for _, id := range ids {
		hs = append(hs, h{id, saltedHash(salt, id)})
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].hash < hs[j].hash })
	if n > len(hs) {
		n = len(hs)
	}
	out := make([]string, n)
	for i := range out {
		out[i] = hs[i].id
	}
	return out
}

// MultiHunk is ADR-160's swe-multi-hunk slice and its registered subset.
type MultiHunk struct {
	Selected []string `json:"selected"`
	// Unnamed are the selected instances whose problem statement contains
	// none of the patched files' paths and none of their basenames but
	// __init__.py: the agent has to find the code first (ADR-160 Decision 5).
	Unnamed []string `json:"unnamed"`
}

var patchHunk = regexp.MustCompile(`(?m)^@@ -`)
var patchedFile = regexp.MustCompile(`(?m)^diff --git .*$`)

// SelectMultiHunk draws ADR-160's slice from the dataset file: the instances
// whose gold patch has at least minHunks hunk headers, less exclude, ordered by
// sha256(salt + id), the first n. Selection reads the gold patch to count its
// hunks and files; nothing it reads reaches a task, which LoadSWEBench builds
// from sweRow alone.
func SelectMultiHunk(dataset, salt string, minHunks int, exclude []string, n int) (MultiHunk, error) {
	b, err := os.ReadFile(dataset)
	if err != nil {
		return MultiHunk{}, err
	}
	var rows []struct {
		ID      string `json:"instance_id"`
		Patch   string `json:"patch"`
		Problem string `json:"problem_statement"`
	}
	if err := json.Unmarshal(b, &rows); err != nil {
		return MultiHunk{}, fmt.Errorf("%s: %w", dataset, err)
	}
	var ids []string
	unnamed := map[string]bool{}
	for _, r := range rows {
		if len(patchHunk.FindAllStringIndex(r.Patch, -1)) < minHunks || slices.Contains(exclude, r.ID) {
			continue
		}
		ids = append(ids, r.ID)
		named := false
		for _, header := range patchedFile.FindAllString(r.Patch, -1) {
			path, same := diffPath(header)
			if !same { // a rename: judge it by its old path
				path, _, _ = strings.Cut(strings.TrimPrefix(header, "diff --git a/"), " b/")
			}
			base := filepath.Base(path)
			if strings.Contains(r.Problem, path) || base != "__init__.py" && strings.Contains(r.Problem, base) {
				named = true
			}
		}
		unnamed[r.ID] = !named
	}
	out := MultiHunk{Selected: SelectSWEBenchSalted(ids, n, salt), Unnamed: []string{}}
	for _, id := range out.Selected {
		if unnamed[id] {
			out.Unnamed = append(out.Unnamed, id)
		}
	}
	return out, nil
}

// ShapeSWEBench marks a task built from the SWE-bench Verified slice (ADR-156
// T5): it has no allowed regions, so it has no silent mis-edit.
const ShapeSWEBench = "swe-bench"

// sweInstance is what a SWE-bench task carries beyond the corpus's fields: the
// image its attempts are graded in.
type sweInstance struct{ image string }

// sweRow is the part of a dataset row a task is built from. The rest of the
// row — the hints, the test patch, the gold patch, the tests that grade it — is
// never read into a task, so none of it can reach the agent (T5's invariant).
type sweRow struct {
	InstanceID       string `json:"instance_id"`
	Repo             string `json:"repo"`
	BaseCommit       string `json:"base_commit"`
	ProblemStatement string `json:"problem_statement"`
	Image            string `json:"image"`
}

// LoadSWEBench builds the registered slice's tasks: one for each id selection
// lists, in its order, from that id's row in dataset, a JSON array as the
// harness reads it. The agent is told the problem statement and nothing else.
// A task's directory is named under the selection's, so its repository is
// found in bench/repos.json as a corpus task's is.
func LoadSWEBench(selection, dataset string) ([]Task, error) {
	var sel struct {
		Selected []string `json:"selected"`
	}
	b, err := os.ReadFile(selection)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &sel); err != nil {
		return nil, fmt.Errorf("%s: %w", selection, err)
	}
	var rows []sweRow
	if b, err = os.ReadFile(dataset); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("%s: %w", dataset, err)
	}
	by := map[string]sweRow{}
	for _, r := range rows {
		by[r.InstanceID] = r
	}
	var ts []Task
	for _, id := range sel.Selected {
		r, ok := by[id]
		if !ok {
			return nil, fmt.Errorf("%s selects %s, which %s does not hold", selection, id, dataset)
		}
		if r.Repo == "" || len(r.BaseCommit) != 40 || r.ProblemStatement == "" || r.Image == "" {
			return nil, fmt.Errorf("%s: %s needs repo, base_commit, problem_statement and image", dataset, id)
		}
		ts = append(ts, Task{ID: id, Shape: ShapeSWEBench, Repo: r.Repo, Commit: r.BaseCommit, Instruction: r.ProblemStatement,
			Dir: filepath.Join(filepath.Dir(selection), id), swe: &sweInstance{image: r.Image}})
	}
	if len(ts) == 0 {
		return nil, fmt.Errorf("%s selects no instance", selection)
	}
	return ts, nil
}

// errGradingInterrupted is a grading run stopped by a signal to bench: the
// attempt is not graded, and Run stops the stage on it as on an interrupted
// attempt.
var errGradingInterrupted = errors.New("interrupted while the harness graded it")

// SWEGrader grades a SWE-bench attempt with the official harness, in the
// instance's own image, one container at a time (ADR-156 Decision 4).
type SWEGrader struct {
	Python  string        // an interpreter with the harness installed, by absolute path
	Dataset string        // the dataset file the tasks came from, by absolute path
	Docker  string        // the docker CLI
	Budget  time.Duration // one grading run's bound; harnessBudget when zero
}

// harnessBudget bounds one grading run: the harness's own per-instance test
// timeout is 30 minutes (-t 1800), and the rest is the container's start.
const harnessBudget = 40 * time.Minute

// present says whether an image is here. Only docker's own "No such image"
// answer means it is not: a daemon it cannot reach, or a docker it cannot
// start, is an error, never an absence (the re-review of #422).
func (g SWEGrader) present(image string) (bool, error) {
	out, err := exec.Command(g.Docker, "image", "inspect", image).CombinedOutput()
	if err == nil {
		return true, nil
	}
	if strings.Contains(string(out), "No such image") || strings.Contains(string(out), "No such object") {
		return false, nil
	}
	return false, fmt.Errorf("docker image inspect %s: %w: %s", image, err, out)
}

// ensure makes an instance's image present before an attempt, pulled for
// linux/amd64: the slice's images have no arm64 manifest (T5 S2).
func (g SWEGrader) ensure(image string) error {
	if here, err := g.present(image); here || err != nil {
		return err
	}
	if out, err := exec.Command(g.Docker, "pull", "--platform", "linux/amd64", image).CombinedOutput(); err != nil {
		return fmt.Errorf("docker pull %s: %w: %s", image, err, out)
	}
	return nil
}

// drop removes an instance's image, if it is here, once its attempts are
// graded: twenty of them do not fit on this disk together (T5 S2).
func (g SWEGrader) drop(image string) error {
	if here, err := g.present(image); !here || err != nil {
		return err
	}
	if out, err := exec.Command(g.Docker, "image", "rm", image).CombinedOutput(); err != nil {
		return fmt.Errorf("docker image rm %s: %w: %s", image, err, out)
	}
	return nil
}

// grade turns the tree's change into the harness's prediction, runs the
// harness for that one instance in scratch, and reads back whether it was
// resolved. A tree with no change is unsolved, and the harness is not started.
func (g SWEGrader) grade(t Task, tree, scratch, run string) (Verdict, error) {
	if err := git(tree, "add", "-A"); err != nil {
		return Verdict{}, err
	}
	patch, err := gitOut(tree, "diff", "--cached", "--binary", "HEAD")
	if err != nil {
		return Verdict{}, err
	}
	if strings.TrimSpace(patch) == "" {
		return Verdict{}, nil
	}
	preds := filepath.Join(scratch, "predictions.json")
	b, err := json.Marshal([]map[string]string{{"instance_id": t.ID, "model_name_or_path": run, "model_patch": patch}})
	if err != nil {
		return Verdict{}, err
	}
	if err := os.WriteFile(preds, b, 0o644); err != nil {
		return Verdict{}, err
	}
	budget := g.Budget
	if budget == 0 {
		budget = harnessBudget
	}
	// A signal to bench reaches the harness as the deadline does: without
	// this, bench ended on the signal and left the harness and its container
	// running (the Stage 3 pilot, 2026-10-15). The signal context is the
	// deadline's parent, so a signal is still told apart from the budget when
	// the budget runs out during the harness's shutdown (the review of #425).
	sigCtx, stopSignals := subproc.Interruptible(context.Background())
	defer stopSignals()
	ctx, cancel := context.WithTimeout(sigCtx, budget)
	defer cancel()
	cmd := exec.CommandContext(ctx, g.Python, "-m", "swebench.harness.run_evaluation", "-d", g.Dataset, "-i", t.ID,
		"-p", preds, "--max_workers", "1", "-t", "1800", "-id", run, "--report_dir", scratch)
	cmd.Dir = scratch
	// At the deadline the harness is interrupted, not killed, so its own
	// cleanup removes the container it started; a kill would skip that
	// (the review of #422). Whatever is left is removed below by name.
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = time.Minute
	out, err := cmd.CombinedOutput()
	// A signal ends grading even when the harness exits 0 on it: the attempt
	// is not graded, and its containers go as after a deadline. A signal that
	// arrives during that cleanup counts too, so it is read after it.
	if err != nil || sigCtx.Err() != nil {
		if cerr := g.removeContainers(t.ID, run); cerr != nil {
			err = errors.Join(err, cerr)
		}
		if sigCtx.Err() != nil && err == nil {
			return Verdict{}, errGradingInterrupted
		}
		if sigCtx.Err() != nil {
			return Verdict{}, fmt.Errorf("%w: %w", errGradingInterrupted, err)
		}
		return Verdict{}, fmt.Errorf("the harness: %w: %s", err, out)
	}
	rb, err := os.ReadFile(filepath.Join(scratch, "logs", "run_evaluation", run, run, t.ID, "report.json"))
	if err != nil {
		return Verdict{}, fmt.Errorf("the harness wrote no report for %s: %w", t.ID, err)
	}
	var rep map[string]struct {
		Resolved bool `json:"resolved"`
	}
	if err := json.Unmarshal(rb, &rep); err != nil {
		return Verdict{}, fmt.Errorf("the harness's report for %s: %w", t.ID, err)
	}
	r, ok := rep[t.ID]
	if !ok {
		return Verdict{}, fmt.Errorf("the harness's report does not name %s", t.ID)
	}
	return Verdict{Solved: r.Resolved}, nil
}

// removeContainers removes every container the harness started for this
// instance and run. It names one sweb.eval.<instance id, lower case>.<run>, and
// when that name is taken adds .<unix seconds> (swebench 5.0.2,
// run_evaluation.py). Docker's name filter is an unanchored regular
// expression, so it only narrows the listing; the names are matched here
// exactly, so another attempt's container — .10 beside .1, or .1.r1 — is never
// removed (the re-review of #422).
func (g SWEGrader) removeContainers(id, run string) error {
	out, err := exec.Command(g.Docker, "ps", "-a", "--filter", "name=sweb.eval.", "--format", "{{.ID}} {{.Names}}").Output()
	if err != nil {
		return fmt.Errorf("docker ps: %w", err)
	}
	name := "sweb.eval." + strings.ToLower(id) + "." + run
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		c, n, ok := strings.Cut(line, " ")
		if !ok || !harnessContainer(n, name) {
			continue
		}
		if b, err := exec.Command(g.Docker, "rm", "-f", c).CombinedOutput(); err != nil {
			return fmt.Errorf("docker rm -f %s: %w: %s", c, err, b)
		}
	}
	return nil
}

// harnessContainer says whether n is the container the harness names name, or
// that name with the numeric suffix it adds on a retry.
func harnessContainer(n, name string) bool {
	rest, ok := strings.CutPrefix(n, name)
	if !ok {
		return false
	}
	if rest == "" {
		return true
	}
	digits := strings.TrimPrefix(rest, ".")
	return digits != rest && digits != "" && strings.Trim(digits, "0123456789") == ""
}
