package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// ErrSpendStop ends a run at an attempt boundary because the cost its receipts
// record has reached --max-spend, or because three of them record none
// (ADR-206 Decision 6). A stopped run is not resumed.
var ErrSpendStop = errors.New("recorded spend has reached --max-spend")

// ExitSpendStop is bench's exit status for ErrSpendStop; the disk floor and
// three start errors share 2, and every other failure is 1.
const ExitSpendStop = 3

// ExitCode is the exit status bench ends with for err.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrSpendStop):
		return ExitSpendStop
	case errors.Is(err, ErrStartErrors), errors.Is(err, ErrDiskFloor):
		return 2
	}
	return 1
}

func hashHex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// resolvePins checks the run's pins before anything runs (ADR-206 Decisions 2,
// 3, 5 and 6): the file at --mrw must have the SHA-256 --mrw-sha256 names, and
// its bytes are held, so what an attempt runs is what was hashed; the
// instruction file is read once and held; the kept path must stay inside the
// attempt's state home.
func resolvePins(o *RunOptions) error {
	o.mrwBytes, o.mrwSHA, o.instr, o.instrSHA = nil, "", nil, ""
	if o.MaxSpend < 0 {
		return fmt.Errorf("--max-spend %v is negative", o.MaxSpend)
	}
	if k := o.KeepStateFile; k != "" && (strings.Contains(k, `\`) || !filepath.IsLocal(filepath.FromSlash(k))) {
		return fmt.Errorf("--keep-state-file %q is not a path inside the attempt's state home", k)
	}
	if o.MrwSHA256 != "" {
		if !sha256Hx.MatchString(o.MrwSHA256) {
			return fmt.Errorf("--mrw-sha256 %q is not 64 lower-case hex digits", o.MrwSHA256)
		}
		data, err := os.ReadFile(o.Mrw)
		if err != nil {
			return fmt.Errorf("--mrw: %w", err)
		}
		if got := hashHex(data); got != o.MrwSHA256 {
			return fmt.Errorf("--mrw %s has sha256 %s, not the pinned %s", o.Mrw, got, o.MrwSHA256)
		}
		o.mrwBytes, o.mrwSHA = data, o.MrwSHA256
	}
	if o.MrwInstructions != "" {
		data, err := os.ReadFile(o.MrwInstructions)
		if err != nil {
			return fmt.Errorf("--mrw-instructions: %w", err)
		}
		if len(data) == 0 {
			return fmt.Errorf("--mrw-instructions %s is empty", o.MrwInstructions)
		}
		o.instr, o.instrSHA = data, hashHex(data)
	}
	return nil
}

// mrwVersion is the first line of `mrw version`, from the pinned bytes when
// the run holds them, so the version is the hashed build's and not whatever the
// file at --mrw has become.
func (o RunOptions) mrwVersion() (string, error) {
	if o.mrwBytes == nil {
		return firstLine(o.Mrw, "version")
	}
	if err := os.MkdirAll(o.Scratch, 0o755); err != nil && o.Scratch != "" {
		return "", err
	}
	dir, err := os.MkdirTemp(o.Scratch, "mrw-pin-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	p := filepath.Join(dir, "mrw")
	if err := os.WriteFile(p, o.mrwBytes, 0o755); err != nil {
		return "", err
	}
	return firstLine(p, "version")
}

// placeMrw makes dst the mrw an attempt runs: a copy of the pinned bytes, or a
// link to --mrw when the run pins nothing.
func (o RunOptions) placeMrw(dst string) error {
	if o.mrwBytes != nil {
		return os.WriteFile(dst, o.mrwBytes, 0o755)
	}
	return os.Symlink(o.Mrw, dst)
}

// instructions is the text appended to configuration B's prompt: the file
// --mrw-instructions named, or what `instructions` prints when the attempt's own
// mrw is asked, so a run that pins the bytes never executes the file at --mrw.
func (o RunOptions) instructions(mrw string) ([]byte, error) {
	if o.instrSHA != "" {
		return slices.Clone(o.instr), nil
	}
	text, err := exec.Command(mrw, "instructions").Output()
	if err != nil {
		return nil, fmt.Errorf("mrw instructions: %w", err)
	}
	return text, nil
}

// mrwAfter is what the attempt's own mrw hashes to when it is not the pin: ""
// when it is, or when the run pins nothing. An agent that rewrote bin/mrw ran
// bytes the receipt's mrw_sha256 does not describe.
func (o RunOptions) mrwAfter(scratch string) string {
	if o.mrwSHA == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(scratch, "bin", "mrw"))
	if err != nil {
		return "unreadable: " + err.Error()
	}
	if got := hashHex(b); got != o.mrwSHA {
		return got
	}
	return ""
}

// spendStop is the check made before each attempt that --max-spend asks for:
// the cost every receipt in the run's directory records, replacements and
// excluded attempts included. A receipt with no cost is not a cost of zero,
// and three of them end the run, since the sum no longer says what was spent.
func spendStop(o RunOptions) error {
	if o.MaxSpend <= 0 {
		return nil
	}
	var sum float64
	missing := 0
	for _, dir := range []string{o.Out, filepath.Join(o.Out, sideDir)} {
		ents, err := os.ReadDir(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, e := range ents {
			if !strings.HasSuffix(e.Name(), ".receipt.json") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				return err
			}
			var r struct {
				Cost *float64 `json:"total_cost_usd"`
			}
			if err := json.Unmarshal(b, &r); err != nil {
				return fmt.Errorf("%s cannot be read back: %w", e.Name(), err)
			}
			if r.Cost == nil {
				missing++
			} else {
				sum += *r.Cost
			}
		}
	}
	switch {
	case sum >= o.MaxSpend-1e-9: // the figure is cents: 0.10 and 0.70 reach 0.80
		return fmt.Errorf("%w: $%.4f is recorded and the figure is $%.2f; a stopped run is not resumed", ErrSpendStop, sum, o.MaxSpend)
	case missing >= 3:
		return fmt.Errorf("%w: %d receipts carry no cost, so the spend is not known ($%.4f recorded, the figure is $%.2f)", ErrSpendStop, missing, sum, o.MaxSpend)
	}
	return nil
}

// keptLimit bounds each file kept beside a receipt: one over it is not kept,
// and the receipt's kept_note says so.
const keptLimit = 16 << 20

// keepStateFile copies rel, a path under the attempt's state home, to
// <id>.kept beside the receipt and returns its SHA-256; "" with no note says the
// attempt left no such file. The agent has run by now, so nothing in the state
// home is trusted: the home itself must be a directory and not a link, the copy
// goes through a root on it, the file must be regular, and the file opened must
// be the one that was looked at. What is refused is a note, not an error.
func keepStateFile(stateHome, rel, base string) (sha, note string, err error) {
	fi, err := os.Lstat(stateHome)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if !fi.IsDir() {
		return "", "state file not kept: the state home is a link or a file, not a directory", nil
	}
	root, err := os.OpenRoot(stateHome)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = root.Close() }()
	name := filepath.FromSlash(rel)
	st, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", fmt.Sprintf("state file %s not kept: %v", rel, err), nil
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Sprintf("state file %s not kept: it is not a regular file", rel), nil
	}
	if st.Size() > keptLimit {
		return "", fmt.Sprintf("state file %s not kept: it is %d bytes, over the %d bound", rel, st.Size(), keptLimit), nil
	}
	f, err := root.Open(name)
	if err != nil {
		return "", fmt.Sprintf("state file %s not kept: %v", rel, err), nil
	}
	defer func() { _ = f.Close() }()
	if fst, err := f.Stat(); err != nil || !os.SameFile(st, fst) {
		return "", fmt.Sprintf("state file %s not kept: it changed while it was opened", rel), nil
	}
	b, err := io.ReadAll(io.LimitReader(f, keptLimit+1))
	if err != nil {
		return "", "", err
	}
	if len(b) > keptLimit {
		return "", fmt.Sprintf("state file %s not kept: it grew over the %d bound", rel, keptLimit), nil
	}
	if err := writeOnce(base+".kept", b); err != nil {
		return "", "", err
	}
	return hashHex(b), "", nil
}

// keepDiff saves what the attempt's tree holds against its base commit, files
// it created included, as <id>.diff beside the receipt, and returns its
// SHA-256. It is taken before grading and builds its snapshot in an index of
// its own under scratch, so the tree's index, which the oracle reads, is left as
// the agent left it. The diff includes edits the agent committed.
func keepDiff(scratch, tree, baseSHA, base string) (sha, note string, err error) {
	idx := filepath.Join(scratch, "kept-diff.index")
	defer func() { _ = os.Remove(idx) }()
	env := append(os.Environ(), "GIT_INDEX_FILE="+idx)
	git := func(args ...string) *exec.Cmd {
		c := exec.Command("git", append([]string{"-C", tree, "-c", "core.quotepath=off", "-c", "core.autocrlf=false"}, args...)...)
		c.Env = env
		return c
	}
	for _, args := range [][]string{{"read-tree", baseSHA}, {"add", "-A"}} {
		if out, err := git(args...).CombinedOutput(); err != nil {
			return "", "", fmt.Errorf("git %s: %w: %s", args[0], err, out)
		}
	}
	cmd := git("diff", "--cached", "--binary", "--no-color", "--no-ext-diff", "--no-renames", baseSHA)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}
	if err := cmd.Start(); err != nil {
		return "", "", err
	}
	out, rerr := io.ReadAll(io.LimitReader(pipe, keptLimit+1))
	if len(out) > keptLimit {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return "", fmt.Sprintf("diff not kept: it is over the %d bound", keptLimit), nil
	}
	if err := cmd.Wait(); err != nil || rerr != nil {
		return "", "", fmt.Errorf("git diff: %v %v: %s", err, rerr, stderr.String())
	}
	if err := writeOnce(base+".diff", out); err != nil {
		return "", "", err
	}
	return hashHex(out), "", nil
}

// keep takes what the run was asked to keep, before grading and wherever the
// tree and the state home still exist; what it could not keep is the receipt's
// kept_note, and an error of bench's own is a setup exclusion.
func (o RunOptions) keep(r *Receipt, scratch, tree, baseSHA, base string) {
	var notes []string
	if o.KeepStateFile != "" {
		sha, note, err := keepStateFile(filepath.Join(scratch, "state"), o.KeepStateFile, base)
		r.KeptStateFileSHA256 = sha
		if note != "" {
			notes = append(notes, note)
		}
		if err != nil {
			r.Exclusion, r.Note = ExclusionSetup, "kept state file: "+err.Error()
		}
	}
	if o.KeepDiff {
		sha, note, err := keepDiff(scratch, tree, baseSHA, base)
		r.KeptDiffSHA256 = sha
		if note != "" {
			notes = append(notes, note)
		}
		if err != nil {
			r.Exclusion, r.Note = ExclusionSetup, "kept diff: "+err.Error()
		}
	}
	r.KeptNote = strings.Join(notes, "; ")
}

// wantMrwSHA is the hash a configuration's receipts carry: run.json's for a
// configuration that runs mrw, none for one that does not.
func wantMrwSHA(p plan, config string) string {
	if Configs[config].Mrw {
		return p.MrwSHA256
	}
	return ""
}

// hashNote names the hash of a receipt that is not run.json's, for the line
// that says the receipt does not read back.
func hashNote(p plan, r Receipt, config string) string {
	if want := wantMrwSHA(p, config); r.MrwSHA256 != want {
		return fmt.Sprintf(": its mrw_sha256 is %q and run.json's is %q", r.MrwSHA256, want)
	}
	return ""
}

// keptProblem says what is wrong with the files an attempt kept beside its
// receipt, or "": each kept file's SHA-256 must be the receipt's and within the
// bound, a file the receipt does not name must not be there, and with
// keep_diff on every attempt that got as far as having a tree has a diff or
// says why not. A kept state file may be absent: the attempt may not have made it.
func keptProblem(p plan, dir string, r Receipt) string {
	for _, k := range []struct {
		ext, want, setting string
		on                 bool
	}{
		{".kept", r.KeptStateFileSHA256, "keep_state_file", p.KeepStateFile != ""},
		{".diff", r.KeptDiffSHA256, "keep_diff", p.KeepDiff},
	} {
		path := filepath.Join(dir, r.Attempt+k.ext)
		_, statErr := os.Lstat(path)
		switch {
		case k.want != "" && !k.on:
			return fmt.Sprintf("its receipt names a %s file and run.json has no %s", k.ext, k.setting)
		case k.want == "" && statErr == nil:
			return fmt.Sprintf("it has a %s file its receipt does not name", k.ext)
		case k.want == "" && k.ext == ".diff" && k.on && r.KeptNote == "" && r.Exclusion != ExclusionSetup:
			return "its receipt names no kept diff and no reason for none"
		case k.want == "":
			continue
		}
		got, err := readBounded(path)
		switch {
		case err != nil:
			return fmt.Sprintf("its %s file cannot be read: %v", k.ext, err)
		case hashHex(got) != k.want:
			return fmt.Sprintf("its %s file has sha256 %s, its receipt says %s", k.ext, hashHex(got), k.want)
		}
	}
	return ""
}

// readBounded reads a kept file, refusing one over the bound it was kept under.
func readBounded(path string) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.Size() > keptLimit {
		return nil, fmt.Errorf("it is %d bytes, over the %d bound", fi.Size(), keptLimit)
	}
	return os.ReadFile(path)
}

// receiptProblem is what bench verify adds to keptProblem: the mrw the
// attempt ran must still have been the pin when it ended.
func receiptProblem(p plan, dir string, r Receipt) string {
	if r.MrwSHA256After != "" {
		return fmt.Sprintf("its bin/mrw hashed to %s after the attempt, not the pinned %s", r.MrwSHA256After, r.MrwSHA256)
	}
	return keptProblem(p, dir, r)
}
