package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// DefaultMaxTurns is the registered turn cap (BACKLOG, "Pre-registered for
// ADR-156"): an attempt past it is unsolved, its cost counted. --max-turns
// sets the stage's own (ADR-163 Decision 3).
const DefaultMaxTurns = 60

// checkFile is the file Materialize writes under stage6, and its one key.
const checkFile = ".quality-harness.json"

// Check is one entry of bench/swe-bench/checks.json: the whole runner a
// repository's own files document, and where in them it was read.
type Check struct {
	Repo    string `json:"repo"`
	Command string `json:"command"`
	Source  string `json:"source"` // file:line@commit
	Note    string `json:"note,omitempty"`
}

// Checks is the declared-check table (ADR-163 Decision 2) and the SHA-256 of
// the file it was read from, which BACKLOG registers and run.json records.
type Checks struct {
	SHA256 string
	byRepo map[string]Check
}

var (
	checkSource  = regexp.MustCompile(`^[^:]+:[0-9]+@[0-9a-f]{40}$`)
	commandSplit = regexp.MustCompile(`&&|\|\||;`)
	// envAssign is a variable set for a command: NAME=value.
	envAssign = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	// selectors are pytest's options that run a subset of the collected
	// tests, matched on the option name before any `=`.
	selectors = map[string]bool{"--keyword": true, "--deselect": true, "--ignore": true, "--ignore-glob": true,
		"--lf": true, "--last-failed": true, "--sw": true, "--stepwise": true}
)

// LoadChecks reads the table and refuses it unless every entry has a source
// and a whole-runner command, and every repository in repos is named.
func LoadChecks(path string, repos []string) (*Checks, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []Check
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	sum := sha256.Sum256(b)
	c := &Checks{SHA256: hex.EncodeToString(sum[:]), byRepo: map[string]Check{}}
	for _, e := range entries {
		if _, dup := c.byRepo[e.Repo]; dup {
			return nil, fmt.Errorf("%s names %s twice", path, e.Repo)
		}
		if !checkSource.MatchString(e.Source) {
			return nil, fmt.Errorf("%s: %s has source %q, want file:line@commit: a declared check comes from the repository's own files", path, e.Repo, e.Source)
		}
		if err := wholeRunner(e.Command); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, e.Repo, err)
		}
		c.byRepo[e.Repo] = e
	}
	for _, r := range repos {
		if _, ok := c.byRepo[r]; !ok {
			return nil, fmt.Errorf("%s does not name %s, a repository of the pool", path, r)
		}
	}
	return c, nil
}

// wholeRunner refuses a command that selects a subset: an argument holding a
// path, a test id or a .py file, -k, pytest's other selection options, or -m
// other than `-m pytest`; or a variable set anywhere in it, which can carry
// options (PYTEST_ADDOPTS). Words are read with quotes and backslashes removed,
// so `"-k"` and `\-k` are -k. The first word of each simple command is the
// program, so `bin/test` is one, and `python runtests.py` names its runner
// script. It catches the selectors a runner documents, not every way a shell
// can hide one: the table is also read by a person, and its sha is registered.
func wholeRunner(command string) error {
	segments := 0
	for _, seg := range commandSplit.Split(command, -1) {
		words := strings.Fields(strings.NewReplacer(`"`, "", `'`, "", `\`, "").Replace(seg))
		if len(words) == 0 {
			continue
		}
		segments++
		for _, w := range words {
			if envAssign.MatchString(w) {
				return fmt.Errorf("command %q sets %q: the declared check is the runner alone", command, w)
			}
		}
		for i, w := range words[1:] {
			script := i == 0 && (words[0] == "python" || words[0] == "python3") && strings.HasSuffix(w, ".py")
			name, _, _ := strings.Cut(w, "=")
			marker := strings.HasPrefix(w, "-m") && !(w == "-m" && i+2 < len(words) && words[i+2] == "pytest")
			if strings.Contains(w, "/") || strings.Contains(w, "::") || strings.HasPrefix(w, "-k") || marker || selectors[name] ||
				strings.HasSuffix(w, ".py") && !script {
				return fmt.Errorf("command %q selects a subset by %q: the declared check is the whole runner", command, w)
			}
		}
	}
	if segments == 0 {
		return fmt.Errorf("command %q is empty", command)
	}
	return nil
}

// resolveDeclared reads what stage6 declares from files: the checks table,
// which must name every SWE-bench repository of the stage's tasks, and the
// work line W adds to its instruction.
func resolveDeclared(o *RunOptions) error {
	if o.Checks != "" {
		var repos []string
		for _, t := range o.Tasks {
			if t.swe != nil && !slices.Contains(repos, t.Repo) {
				repos = append(repos, t.Repo)
			}
		}
		c, err := LoadChecks(o.Checks, repos)
		if err != nil {
			return err
		}
		o.checks = c
	}
	if o.WorkLine != "" {
		b, err := os.ReadFile(o.WorkLine)
		if err != nil {
			return err
		}
		line := strings.TrimRight(string(b), "\n")
		if strings.TrimSpace(line) == "" {
			return fmt.Errorf("--work-line %s is empty", o.WorkLine)
		}
		sum := sha256.Sum256(b)
		o.workLine, o.workLineSHA = line, hex.EncodeToString(sum[:])
	}
	return nil
}

// declaredFor is the check .quality-harness.json carries in t's tree: the
// table's command for its repository, under stage6 alone. A task whose
// repository the table does not name (the canary fixture) gets no file.
func (o RunOptions) declaredFor(t Task) string {
	if o.Profile != ProfileStage6 || o.checks == nil {
		return ""
	}
	return o.checks.byRepo[t.Repo].Command
}

// maxTurns is the stage's turn cap: --max-turns, else the registered 60.
func (o RunOptions) maxTurns() int {
	if o.MaxTurns == 0 {
		return DefaultMaxTurns
	}
	return o.MaxTurns
}

// writeCheckFile puts the declared check in dst, before the base commit.
func writeCheckFile(dst, command string) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false) // `&&` stays as typed in the file the plugin reads
	if err := enc.Encode(map[string]string{"check": command}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, checkFile), b.Bytes(), 0o644)
}

// instructionFor is the instruction one configuration is given for a task:
// W's carries the work line in the task's text, ahead of the status sentence
// every configuration shares, so PromptTemplate and its digest are unchanged.
func instructionFor(o RunOptions, c Config, t Task) string {
	instruction := t.Instruction
	if c.Work {
		instruction += "\n\n" + o.workLine
	}
	if o.StatusLine {
		instruction = strings.Replace(PromptTemplate, "{instruction}", instruction, 1)
	}
	return instruction
}

// CheckWorkLine is the canary's question of the line (ADR-163 Decision 3): it
// is in the instruction of W and of no other configuration.
func CheckWorkLine(c Config, instruction, line string) []string {
	if has := strings.Contains(instruction, line); has != c.Work {
		return []string{fmt.Sprintf("configuration %s: the work line is in its instruction: %v, want %v", c.Name, has, c.Work)}
	}
	return nil
}
