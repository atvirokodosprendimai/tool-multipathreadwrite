package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// BaselineOptions is a baseline: the instances, the table of declared checks,
// and the docker that starts each instance's container. It names no model
// because a baseline starts none (ADR-163 Decision 4).
type BaselineOptions struct {
	Tasks       []Task
	Stage       string
	Out         string        // holds baseline.json
	Scratch     string        // where the host trees are built; os.TempDir() when empty
	Docker      string        // the docker CLI
	CanaryImage string        // the image a task with none of its own runs in
	Checks      string        // bench/swe-bench/checks.json
	Budget      time.Duration // bounds a container's life, so a check that hangs ends with it
	SWE         *SWEGrader    // pulls and drops the instances' images; nil for none
	Selection   string        // the selection Tasks were drawn from, for Surviving
	Surviving   string        // where the green ids are written as a selection; "" for none
}

// BaselineRow is what one instance's check did on the unmodified tree.
type BaselineRow struct {
	Instance string `json:"instance"`
	Green    bool   `json:"green"`
	ExitCode int    `json:"exit_code"` // -1 when the check has no exit status
	WallMS   int64  `json:"wall_ms"`
	Reason   string `json:"reason,omitempty"` // why a check that could not run is red
}

// Baseline runs each instance's declared check once on the tree its attempts
// start from, inside its own container, and writes baseline.json, and with
// Surviving the green ids as a selection. One run decides: a red is never
// rerun, and a directory that holds a baseline is refused.
func Baseline(o BaselineOptions) error {
	if o.Stage == "" || o.Out == "" || o.Docker == "" || o.Checks == "" || len(o.Tasks) == 0 {
		return errors.New("baseline needs --stage, --out, --checks and the instances of --selection")
	}
	var repos []string
	for _, t := range o.Tasks {
		if !slices.Contains(repos, t.Repo) {
			repos = append(repos, t.Repo)
		}
	}
	checks, err := LoadChecks(o.Checks, repos)
	if err != nil {
		return err
	}
	for _, dir := range []string{o.Out, o.Scratch} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	path := filepath.Join(o.Out, "baseline.json")
	if exists(path) {
		return fmt.Errorf("%s holds a baseline already: one run decides, so it is not run again", path)
	}
	// The rows of a run that stopped are kept as they finish, and a rerun
	// skips those instances: a check, once run, is never run again.
	partial := filepath.Join(o.Out, "baseline.partial.json")
	done, err := loadPartial(partial)
	if err != nil {
		return err
	}
	if err := sweepContainers(o.Docker, o.Stage); err != nil {
		return err
	}
	var rows []BaselineRow
	for _, t := range o.Tasks {
		if i := slices.IndexFunc(done, func(r BaselineRow) bool { return r.Instance == t.ID }); i >= 0 {
			rows = append(rows, done[i])
			continue
		}
		row, err := baselineOne(o, t, checks.byRepo[t.Repo].Command)
		var dropped dropError
		if err != nil && !errors.As(err, &dropped) {
			return err
		}
		rows = append(rows, row)
		if serr := savePartial(partial, rows); serr != nil {
			return serr
		}
		if err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	if err := writeOnce(path, append(b, '\n')); err != nil {
		return err
	}
	if err := os.Remove(partial); err != nil && !os.IsNotExist(err) {
		return err
	}
	if o.Surviving == "" {
		return nil
	}
	return writeSurviving(o, rows)
}

// dropError is an image that would not drop after its instance's row was
// complete: the row is kept, the stage stops.
type dropError struct{ error }

// loadPartial reads the rows an earlier run of this directory finished.
func loadPartial(path string) ([]BaselineRow, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []BaselineRow
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return rows, nil
}

// savePartial rewrites the finished rows whole, through a rename, so a stop
// leaves the earlier file or the new one and never half of either.
func savePartial(path string, rows []BaselineRow) error {
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// baselineOne runs one instance's check. A failure of the instance itself is a
// red row with its reason; an error is a failure of the stage.
func baselineOne(o BaselineOptions, t Task, command string) (row BaselineRow, err error) {
	row = BaselineRow{Instance: t.ID, ExitCode: -1}
	image := imageFor(RunOptions{CanaryImage: o.CanaryImage}, t)
	if t.swe != nil && o.SWE != nil {
		if err := o.SWE.ensure(image); err != nil {
			row.Reason = err.Error()
			return row, nil
		}
		defer func() {
			if derr := o.SWE.drop(image); derr != nil && err == nil {
				err = dropError{derr}
			}
		}()
	}
	scratch, err := os.MkdirTemp(o.Scratch, "baseline-")
	if err != nil {
		return row, err
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	tree := filepath.Join(scratch, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		return row, err
	}
	base, err := materialize(t, tree, nil, command)
	if err != nil {
		row.Reason = err.Error()
		return row, nil
	}
	box, err := startContainer(o.Docker, image, containerName("baseline."+t.ID), o.Stage, o.Budget+300*time.Second, tree, base)
	if box != nil {
		defer func() {
			if rerr := box.remove(); rerr != nil {
				row.Reason += " (" + rerr.Error() + ")"
			}
		}()
	}
	if err != nil {
		row.Reason = err.Error()
		return row, nil
	}
	script := "if ! " + activateTestbed + " >/dev/null 2>&1; then echo bench baseline: the testbed environment did not activate 1>&2; exit 126; fi; " + command
	start := time.Now()
	var stderr bytes.Buffer
	cmd := exec.Command(o.Docker, "exec", "-w", testbed, box.name, "bash", "-c", script)
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	row.WallMS = time.Since(start).Milliseconds()
	var ee *exec.ExitError
	switch {
	case runErr == nil:
		row.Green, row.ExitCode = true, 0
	case errors.As(runErr, &ee):
		row.ExitCode = ee.ExitCode()
		// docker's own 125, a failed activation's 126 and a missing command's
		// 127 mean the check never ran: the reason is what stderr said last.
		if slices.Contains([]int{125, 126, 127}, row.ExitCode) {
			row.Reason = fmt.Sprintf("exit %d: %s", row.ExitCode, lastLine(stderr.String()))
		}
	default:
		row.Reason = runErr.Error()
	}
	return row, nil
}

// lastLine is the last non-empty line of s.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// writeSurviving writes the green instances, in the selection's order, as a
// selection: the source's rule, dataset, revision and split, so an export reads
// the salt from it, plus the rule that thinned it, the source's sha and how
// many were red. The red ids are never written (ADR-163 Decision 4).
func writeSurviving(o BaselineOptions, rows []BaselineRow) error {
	src, err := os.ReadFile(o.Selection)
	if err != nil {
		return err
	}
	var sel struct {
		Rule, Dataset, Revision, Split string
		Selected                       []string
	}
	if err := json.Unmarshal(src, &sel); err != nil {
		return fmt.Errorf("%s: %w", o.Selection, err)
	}
	if len(sel.Selected) != len(rows) {
		return fmt.Errorf("%s selects %d instances, the baseline ran %d", o.Selection, len(sel.Selected), len(rows))
	}
	green, red := []string{}, 0
	for i, r := range rows {
		if sel.Selected[i] != r.Instance {
			return fmt.Errorf("%s selects %s at place %d, the baseline ran %s", o.Selection, sel.Selected[i], i+1, r.Instance)
		}
		if r.Green {
			green = append(green, r.Instance)
		} else {
			red++
		}
	}
	sum := sha256.Sum256(src)
	b, err := json.MarshalIndent(map[string]any{
		"rule": sel.Rule, "dataset": sel.Dataset, "revision": sel.Revision, "split": sel.Split,
		"surviving_rule":          "the instances of the source selection whose declared check is green on the unmodified tree, in its order (ADR-163 Decision 4)",
		"source_selection_sha256": hex.EncodeToString(sum[:]),
		"red_count":               red,
		"selected":                green,
	}, "", "  ")
	if err != nil {
		return err
	}
	return writeOnce(o.Surviving, append(b, '\n'))
}

// SurvivingBlock is block k of the surviving pool: instances 40(k−1)+1 to 40k,
// the last block the rest of it (ADR-163 Decision 4). Unlike BlockTasks it has
// no fixed number of blocks: L_max is the pool's own.
func SurvivingBlock(ts []Task, k int) ([]Task, error) {
	lo := (k - 1) * BlockSize
	if k < 1 || lo >= len(ts) {
		return nil, fmt.Errorf("block %d: the surviving pool of %d instances has blocks 1 to %d", k, len(ts), (len(ts)+BlockSize-1)/BlockSize)
	}
	return ts[lo:min(lo+BlockSize, len(ts))], nil
}
