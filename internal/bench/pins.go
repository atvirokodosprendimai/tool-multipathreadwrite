package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
		return os.WriteFile(dst, o.mrwBytes, 0o555)
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

// mrwLimit bounds what is hashed to check an attempt's bin/mrw afterwards: the
// binary's own size, with room.
const mrwLimit = 256 << 20

// mrwAfter is what the attempt's own mrw is when it is not the pin: "" when it
// is, or when the run pins nothing, else the hash it has now or what it is
// instead (a link, a directory, a file over the bound). An agent that rewrote
// bin/mrw ran bytes the receipt's mrw_sha256 does not describe.
func (o RunOptions) mrwAfter(scratch string) string {
	if o.mrwSHA == "" {
		return ""
	}
	p := filepath.Join(scratch, "bin", "mrw")
	fi, err := os.Lstat(p)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	if !fi.Mode().IsRegular() {
		return "not a regular file: " + fi.Mode().String()
	}
	f, err := os.Open(p)
	if err != nil {
		return "unreadable: " + err.Error()
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, mrwLimit+1))
	if err != nil {
		return "unreadable: " + err.Error()
	}
	if n > mrwLimit {
		return fmt.Sprintf("over the %d bound", mrwLimit)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != o.mrwSHA {
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
	case micros(sum) >= micros(o.MaxSpend):
		return fmt.Errorf("%w: $%.4f is recorded and the figure is $%.2f; a stopped run is not resumed", ErrSpendStop, sum, o.MaxSpend)
	case missing >= 3:
		return fmt.Errorf("%w: %d receipts carry no cost, so the spend is not known ($%.4f recorded, the figure is $%.2f)", ErrSpendStop, missing, sum, o.MaxSpend)
	}
	return nil
}

// micros is a USD amount in millionths, so a cost and a figure are compared as
// integers: 0.10 and 0.70 reach 0.80 although their float sum does not.
func micros(x float64) int64 { return int64(math.Round(x * 1e6)) }

// keptLimit bounds each file kept beside a receipt: one over it is not kept,
// and the receipt's kept_note says so.
const keptLimit = 16 << 20

// outputCap bounds what is kept of a git command's output, which only an error
// message uses.
const outputCap = 64 << 10

// capped is a writer that keeps the first outputCap bytes and drops the rest
// without failing, so a noisy command can neither fill memory nor block.
type capped struct{ b bytes.Buffer }

func (c *capped) Write(p []byte) (int, error) {
	if room := outputCap - c.b.Len(); room > 0 {
		c.b.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}

func (c *capped) String() string { return c.b.String() }

// hasNote says whether one of the semicolon-separated notes of a receipt's
// kept_note starts with prefix: "state:" or "diff:", the artifact it is about.
func hasNote(note, prefix string) bool {
	for _, n := range strings.Split(note, "; ") {
		if strings.HasPrefix(n, prefix) {
			return true
		}
	}
	return false
}

// keepStateFile copies rel, a path under the attempt's state home, to
// <id>.kept beside the receipt and returns its SHA-256; "" with no note says the
// attempt left no such file. The agent has run by now, so nothing in the state
// home is trusted: the home itself must be a directory and not a link, the root
// opened on it must be that directory, the copy goes through the root, the file
// must be regular, and the file opened must be the one that was looked at. What
// is refused is a note, not an error.
func keepStateFile(stateHome, rel, base string) (sha, note string, err error) {
	return keepFileUnder(stateHome, rel, base, "state", "state home")
}

// keepScratchFile is keepStateFile for one file of the attempt's scratch
// directory, which is the agent's TMPDIR and holds the log the bench build of
// mrw writes beside the state home (ADR-211 Decision 4). Its argument is a file
// name: one with a separator is a note, as a link or a directory is.
func keepScratchFile(scratch, name, base string) (sha, note string, err error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Sprintf("scratch: %q not kept: it is not a file name of the scratch directory", name), nil
	}
	return keepFileUnder(scratch, name, base, "scratch", "scratch directory")
}

// keepFileUnder is what both share: label prefixes the note, and home names the
// directory in it.
func keepFileUnder(stateHome, rel, base, label, home string) (sha, note string, err error) {
	fi, err := os.Lstat(stateHome)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if !fi.IsDir() {
		return "", label + ": not kept: the " + home + " is a link or a file, not a directory", nil
	}
	root, err := os.OpenRoot(stateHome)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = root.Close() }()
	if rst, err := root.Stat("."); err != nil || !os.SameFile(fi, rst) {
		return "", label + ": not kept: the " + home + " changed while it was opened", nil
	}
	name := filepath.FromSlash(rel)
	st, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return "", "", nil
	}
	if err != nil {
		return "", fmt.Sprintf("%s: %s not kept: %v", label, rel, err), nil
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Sprintf("%s: %s not kept: it is not a regular file", label, rel), nil
	}
	if st.Size() > keptLimit {
		return "", fmt.Sprintf("%s: %s not kept: it is %d bytes, over the %d bound", label, rel, st.Size(), keptLimit), nil
	}
	f, err := root.Open(name)
	if err != nil {
		return "", fmt.Sprintf("%s: %s not kept: %v", label, rel, err), nil
	}
	defer func() { _ = f.Close() }()
	if fst, err := f.Stat(); err != nil || !os.SameFile(st, fst) {
		return "", fmt.Sprintf("%s: %s not kept: it changed while it was opened", label, rel), nil
	}
	b, err := io.ReadAll(io.LimitReader(f, keptLimit+1))
	if err != nil {
		return "", "", err
	}
	if len(b) > keptLimit {
		return "", fmt.Sprintf("%s: %s not kept: it grew over the %d bound", label, rel, keptLimit), nil
	}
	if err := writeOnce(base+".kept", b); err != nil {
		return "", "", err
	}
	return hashHex(b), "", nil
}

// copyIndex copies the tree's own index to dst, bounded, and gives the copy the
// original's modification time. git reads the content of an entry no older than
// the index file ("racily clean") and trusts the stat of any other; a copy with a
// fresh mtime made such an entry trusted, so a same-size edit in the second of
// the checkout was left out of the diff (#816).
func copyIndex(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	fi, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, io.LimitReader(in, mrwLimit+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > mrwLimit {
		err = fmt.Errorf("the index is over the %d bound", mrwLimit)
	}
	if err == nil {
		err = os.Chtimes(dst, fi.ModTime(), fi.ModTime())
	}
	return err
}

// privateIndex makes an index of its own under scratch, a copy of the tree's (the
// base's tree when the tree has none), with everything the tree holds added to it,
// and returns a git command that uses it and the function that removes it. The
// tree's own index, which the oracle reads, is left as the agent left it, and a
// path the agent added by force or committed stays though it is ignored; `add -A`
// then takes in what the copy lacks (ADR-206 Decision 5, ADR-211 Decision 11).
func privateIndex(scratch, tree, baseSHA, name string) (git func(args ...string) *exec.Cmd, cleanup func(), err error) {
	idx := filepath.Join(scratch, name)
	cleanup = func() { _ = os.Remove(idx) }
	env := append(os.Environ(), "GIT_INDEX_FILE="+idx)
	git = func(args ...string) *exec.Cmd {
		c := exec.Command("git", append([]string{"-C", tree, "-c", "core.quotepath=off", "-c", "core.autocrlf=false"}, args...)...)
		c.Env = env
		return c
	}
	steps := [][]string{{"add", "-A"}}
	if err := copyIndex(filepath.Join(tree, ".git", "index"), idx); errors.Is(err, os.ErrNotExist) {
		steps = [][]string{{"read-tree", baseSHA}, {"add", "-A"}}
	} else if err != nil {
		cleanup()
		return nil, nil, err
	}
	for _, args := range steps {
		c := git(args...)
		var out capped
		c.Stdout, c.Stderr = &out, &out
		if err := c.Run(); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("git %s: %w: %s", args[0], err, out.String())
		}
	}
	return git, cleanup, nil
}

// keepDiff saves what the attempt's tree holds against its base commit, files
// it created included, as <id>.diff beside the receipt, and returns its
// SHA-256. It is taken before grading and works in a private index. The diff
// includes edits the agent committed.
func keepDiff(scratch, tree, baseSHA, base string) (sha, note string, err error) {
	git, cleanup, err := privateIndex(scratch, tree, baseSHA, "kept-diff.index")
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	cmd := git("diff", "--cached", "--binary", "--no-color", "--no-ext-diff", "--no-renames", baseSHA)
	var stderr capped
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
		return "", fmt.Sprintf("diff: not kept: it is over the %d bound", keptLimit), nil
	}
	if err := cmd.Wait(); err != nil || rerr != nil {
		return "", "", fmt.Errorf("git diff: %w: %s", errors.Join(err, rerr), stderr.String())
	}
	if err := writeOnce(base+".diff", out); err != nil {
		return "", "", err
	}
	return hashHex(out), "", nil
}

// keep takes what the run was asked to keep, before grading and wherever the
// tree and the state home still exist. What it could not keep is the receipt's
// kept_note, one note per artifact, "state:" or "diff:"; an error of bench's own
// is a setup exclusion and a note as well.
func (o RunOptions) keep(r *Receipt, scratch, tree, baseSHA, base string) {
	var notes []string
	if o.KeepStateFile != "" {
		sha, note, err := keepStateFile(filepath.Join(scratch, "state"), o.KeepStateFile, base)
		r.KeptStateFileSHA256 = sha
		if note != "" {
			notes = append(notes, note)
		}
		if err != nil {
			notes = append(notes, "state: failed: "+err.Error())
			r.diagnose(ExclusionSetup, "kept state file: "+err.Error())
		}
	}
	if o.KeepScratchFile != "" {
		sha, note, err := keepScratchFile(scratch, o.KeepScratchFile, base)
		r.KeptScratchFileSHA256 = sha
		if note != "" {
			notes = append(notes, note)
		}
		if err != nil {
			notes = append(notes, "scratch: failed: "+err.Error())
			r.diagnose(ExclusionSetup, "kept scratch file: "+err.Error())
		}
	}
	if o.KeepDiff {
		sha, note, err := keepDiff(scratch, tree, baseSHA, base)
		r.KeptDiffSHA256 = sha
		if note != "" {
			notes = append(notes, note)
		}
		if err != nil {
			notes = append(notes, "diff: failed: "+err.Error())
			r.diagnose(ExclusionSetup, "kept diff: "+err.Error())
		}
	}
	r.KeptNote = strings.Join(notes, "; ")
}

// noDiffNote is for the way out of runAttempt: a receipt of a run that keeps
// diffs and holds neither a diff nor a note about one says the attempt ended
// before its tree was captured, so every receipt verifies on its own account.
func (o RunOptions) noDiffNote(r *Receipt) {
	if !o.KeepDiff || r.KeptDiffSHA256 != "" || hasNote(r.KeptNote, "diff:") {
		return
	}
	if r.KeptNote != "" {
		r.KeptNote += "; "
	}
	r.KeptNote += "diff: not taken: the attempt ended before its tree was captured"
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
// keep_diff on every receipt has a diff or a note about the diff. A kept state
// file may be absent: the attempt may not have made it.
func keptProblem(p plan, dir string, r Receipt) string {
	for _, k := range []struct {
		ext, want, setting string
		on                 bool
	}{
		{".kept", r.KeptStateFileSHA256 + r.KeptScratchFileSHA256, "keep_state_file or keep_scratch_file", p.KeepStateFile != "" || p.KeepScratchFile != ""},
		{".diff", r.KeptDiffSHA256, "keep_diff", p.KeepDiff},
	} {
		path := filepath.Join(dir, r.Attempt+k.ext)
		_, statErr := os.Lstat(path)
		switch {
		case k.want != "" && !k.on:
			return fmt.Sprintf("its receipt names a %s file and run.json has no %s", k.ext, k.setting)
		case k.want == "" && statErr == nil:
			return fmt.Sprintf("it has a %s file its receipt does not name", k.ext)
		case k.want == "" && k.ext == ".diff" && k.on && !hasNote(r.KeptNote, "diff:"):
			return "its receipt names no kept diff and no note about one"
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, keptLimit+1))
	if err != nil {
		return nil, err
	}
	if len(b) > keptLimit {
		return nil, fmt.Errorf("it is over the %d bound", keptLimit)
	}
	return b, nil
}

// receiptProblem is what bench verify adds to keptProblem: the mrw the
// attempt ran must still have been the pin when it ended.
func receiptProblem(p plan, dir string, r Receipt) string {
	if r.MrwSHA256After != "" {
		return fmt.Sprintf("its bin/mrw hashed to %s after the attempt, not the pinned %s", r.MrwSHA256After, r.MrwSHA256)
	}
	return keptProblem(p, dir, r)
}
