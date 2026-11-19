package bench

import (
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
// --mrw-instructions named, or what the binary's `instructions` prints.
func (o RunOptions) instructions() ([]byte, error) {
	if o.instrSHA != "" {
		return slices.Clone(o.instr), nil
	}
	text, err := exec.Command(o.Mrw, "instructions").Output()
	if err != nil {
		return nil, fmt.Errorf("mrw instructions: %w", err)
	}
	return text, nil
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
	case sum >= o.MaxSpend:
		return fmt.Errorf("%w: $%.4f is recorded and the figure is $%.2f; a stopped run is not resumed", ErrSpendStop, sum, o.MaxSpend)
	case missing >= 3:
		return fmt.Errorf("%w: %d receipts carry no cost, so the spend is not known ($%.4f recorded, the figure is $%.2f)", ErrSpendStop, missing, sum, o.MaxSpend)
	}
	return nil
}

// keepStateFile copies rel, a path under the attempt's state home, to
// <id>.kept beside the receipt and returns its SHA-256; "" says the attempt
// left no such file. The copy goes through a root on the state home, so a link
// the agent made there cannot read a file outside it.
func keepStateFile(stateHome, rel, base string) (string, error) {
	root, err := os.OpenRoot(stateHome)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = root.Close() }()
	fi, err := root.Lstat(filepath.FromSlash(rel))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", rel)
	}
	f, err := root.Open(filepath.FromSlash(rel))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	if err := writeOnce(base+".kept", b); err != nil {
		return "", err
	}
	return hashHex(b), nil
}

// keepDiff saves what the attempt changed against its base commit, files it
// created included, as <id>.diff beside the receipt, and returns its SHA-256.
func keepDiff(tree, baseSHA, base string) (string, error) {
	if err := git(tree, "add", "-A"); err != nil {
		return "", err
	}
	out, err := gitOut(tree, "diff", "--cached", "--binary", "--no-color", "--no-ext-diff", "--no-renames", baseSHA)
	if err != nil {
		return "", err
	}
	if err := writeOnce(base+".diff", []byte(out)); err != nil {
		return "", err
	}
	return hashHex([]byte(out)), nil
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
// receipt, or "": each kept file's SHA-256 must be the receipt's, and a file
// the receipt does not name must not be there.
func keptProblem(p plan, dir string, r Receipt) string {
	for _, k := range []struct {
		ext, want, setting string
		on                 bool
	}{
		{".kept", r.KeptStateFileSHA256, "keep_state_file", p.KeepStateFile != ""},
		{".diff", r.KeptDiffSHA256, "keep_diff", p.KeepDiff},
	} {
		got, err := os.ReadFile(filepath.Join(dir, r.Attempt+k.ext))
		switch {
		case k.want != "" && !k.on:
			return fmt.Sprintf("its receipt names a %s file and run.json has no %s", k.ext, k.setting)
		case k.want != "" && err != nil:
			return fmt.Sprintf("its %s file cannot be read: %v", k.ext, err)
		case k.want != "" && hashHex(got) != k.want:
			return fmt.Sprintf("its %s file has sha256 %s, its receipt says %s", k.ext, hashHex(got), k.want)
		case k.want == "" && err == nil:
			return fmt.Sprintf("it has a %s file its receipt does not name", k.ext)
		}
	}
	return ""
}
