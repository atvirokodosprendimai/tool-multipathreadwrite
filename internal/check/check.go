// Package check runs a project's own verification immediately after an edit,
// scoped to what the edit touched.
//
// The point is round trips. Edit, run the tests, read the output is three calls
// and three result blocks; chaining the check to the write makes it one. The
// same habit that makes `gofmt -l x && go vet ./... && git commit` one call
// applies here — the verification and the thing it gates belong in one output.
//
// Two rules constrain the implementation, and both come from checks that lied:
//
//  1. The exit status is never inferred from what was printed. Output goes to a
//     file, a bounded tail is shown, and the process's real code is reported. A
//     tail in the pipeline would make the pipeline's status the tail's, so a
//     failing suite would surface as a pass.
//  2. A failing check never triggers a revert. The caller is told, loudly and
//     with a distinct status; undoing their edit for them can destroy work they
//     wanted to keep and inspect.
package check

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Config is the project's declared verification, read from
// .quality-harness.json at the repository root.
type Config struct {
	// Check is the whole-project command, used when no narrower scope can be
	// derived from the edited paths.
	Check string `json:"check"`
	// ScopedCheck is the narrow command. {packages} expands to the Go packages
	// containing the edited files, {files} to the edited paths themselves.
	ScopedCheck string `json:"scoped_check"`
	// TimeoutSeconds bounds the run. Zero means the built-in default.
	TimeoutSeconds int `json:"timeout_seconds"`
	// TailLines is how many trailing lines of output to show. Zero means the
	// built-in default; the full output is always kept in a file.
	TailLines int `json:"tail_lines"`

	declared bool
}

// Declared reports whether the project actually stated its check, as opposed to
// this package inferring one. A caller should say which it used: a command
// inferred from the repository shape can be red on an untouched tree, and that
// finding is about the machine rather than about the edit.
func (c Config) Declared() bool { return c.declared }

const (
	defaultTimeout = 5 * time.Minute
	defaultTail    = 30
)

// Load reads .quality-harness.json from root. A missing file is not an error:
// it falls back to a Go-shaped default when the root has a go.mod, and to no
// check at all otherwise.
func Load(root string) (Config, error) {
	var c Config
	b, err := os.ReadFile(filepath.Join(root, ".quality-harness.json"))
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf(".quality-harness.json: %w", err)
		}
		c.declared = c.Check != "" || c.ScopedCheck != ""
	case !os.IsNotExist(err):
		return c, err
	}
	if !c.declared {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			c.Check, c.ScopedCheck = "go test ./...", "go test {packages}"
		}
	}
	return c, nil
}

// Result is what one check run produced.
type Result struct {
	Ran        bool     `json:"ran"`
	Declared   bool     `json:"declared"`
	Skipped    string   `json:"skipped,omitempty"`
	Command    string   `json:"command,omitempty"`
	ExitCode   int      `json:"exit_code"`
	DurationMS int64    `json:"duration_ms,omitempty"`
	OutputFile string   `json:"output_file,omitempty"`
	Tail       []string `json:"tail,omitempty"`
	Truncated  int      `json:"truncated_lines,omitempty"`
}

// OK reports whether the check ran and passed. A check that did not run is not
// a pass — the caller has no evidence either way, and saying so is the whole
// job of this type.
func (r Result) OK() bool { return r.Ran && r.ExitCode == 0 }

// Run picks a command for the edited paths and executes it in root.
func Run(ctx context.Context, root string, cfg Config, editedPaths []string) (Result, error) {
	cmdline, scoped := command(root, cfg, editedPaths)
	if cmdline == "" {
		return Result{Declared: cfg.declared, Skipped: "no check declared and no go.mod found"}, nil
	}
	res := Result{Ran: true, Declared: cfg.declared, Command: cmdline}
	_ = scoped

	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// The output goes to a file first and is read back from there. Nothing
	// between the process and its exit code may be a pipe.
	f, err := os.CreateTemp("", "mrw-check-*.log")
	if err != nil {
		return res, err
	}
	res.OutputFile = f.Name()

	c := exec.CommandContext(ctx, "sh", "-c", cmdline)
	c.Dir = root
	c.Stdout, c.Stderr = f, f

	start := time.Now()
	runErr := c.Run()
	res.DurationMS = time.Since(start).Milliseconds()
	f.Close()

	switch {
	case runErr == nil:
		res.ExitCode = 0
	case c.ProcessState != nil:
		res.ExitCode = c.ProcessState.ExitCode()
	default:
		res.ExitCode = -1
	}
	if ctx.Err() == context.DeadlineExceeded {
		res.ExitCode = -1
		res.Skipped = fmt.Sprintf("timed out after %s", timeout)
	}

	tail := cfg.TailLines
	if tail <= 0 {
		tail = defaultTail
	}
	res.Tail, res.Truncated = lastLines(res.OutputFile, tail)
	return res, nil
}

// command chooses between the scoped and whole-project forms. It returns the
// scoped one only when every path maps to a Go package, because a scoped run
// that quietly omits a changed file is worse than a slow complete one.
//
// root is needed to tell a directory from a typo; see packages.
func command(root string, cfg Config, paths []string) (cmdline string, scoped bool) {
	if cfg.ScopedCheck != "" {
		if pkgs := packages(root, paths); pkgs != "" {
			r := strings.NewReplacer("{packages}", pkgs, "{files}", strings.Join(paths, " "))
			return r.Replace(cfg.ScopedCheck), true
		}
	}
	return cfg.Check, false
}

// packages maps paths to ./dir form, or returns "" if any of them is neither a
// Go file nor a directory in the tree — in which case the caller must fall back
// to the full check.
//
// A DIRECTORY counts because naming one is how a caller asks for a package
// directly: `mrw check internal/apply` is the documented spelling, and ./dir is
// the form this function itself emits, so a scope mrw printed can be handed
// straight back to it. Paths reaching here from a write are always files, so
// that path is unaffected.
//
// Only an EXISTING directory qualifies, and that is the whole reason root is a
// parameter. A mistyped path has no extension either, and scoping to it would
// run a check that covers nothing and then report PASS — falling back to the
// full command is the honest reading of a path we cannot place.
func packages(root string, paths []string) string {
	seen := map[string]bool{}
	for _, p := range paths {
		var dir string
		switch {
		case filepath.Ext(p) == ".go":
			dir = filepath.Dir(p)
		case isDir(root, p):
			dir = p
		default:
			return ""
		}
		dir = filepath.ToSlash(filepath.Clean(dir))
		if dir == "." {
			dir = ""
		}
		seen["./"+dir] = true
	}
	if len(seen) == 0 {
		return ""
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, strings.TrimSuffix(d, "/"))
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}

// isDir reports whether p names a directory under root. It is the only question
// this package asks the filesystem about a path, and it is asked only to tell a
// package from a typo.
func isDir(root, p string) bool {
	fi, err := os.Stat(filepath.Join(root, p))
	return err == nil && fi.IsDir()
}

// lastLines returns the final n lines of a file plus how many it left out, so a
// trimmed report never reads as the whole output.
func lastLines(path string, n int) ([]string, int) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, 0
	}
	if len(lines) <= n {
		return lines, 0
	}
	return lines[len(lines)-n:], len(lines) - n
}
