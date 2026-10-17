package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// ExportOptions names what bench export-paired reads: the stage's block
// directories, one pair of configurations (the control first), one model, the
// scorer quality-harness ADR-106 pre-registered, and the selection and canary
// the stage ran on.
type ExportOptions struct {
	Dirs      []string
	X, Y      string
	Model     string
	ScorerSHA string // the git blob sha of ADR-106's scripts/eval-paired.mjs, from its pre-registration
	Selection string
	Canary    string
}

// pairedRow is one row ADR-106's scorer reads: one instance, arm and repeat.
type pairedRow struct {
	Instance   string `json:"instance"`
	Arm        string `json:"arm"`
	Repeat     int    `json:"repeat"`
	Status     string `json:"status"`
	Resolved   bool   `json:"resolved"`
	BrokeTests bool   `json:"brokeTests"`
	Limit      bool   `json:"limit"`
	Unrun      bool   `json:"unrun"`
}

var saltOf = regexp.MustCompile(`sha256\(\\?"([^"\\]*)\\?"`)

// ExportPaired writes quality-harness ADR-106's paired rows for one model
// (ADR-162 Decision 3): a {"config": …} line with every field its Decision 2
// names, then one row per instance, arm and repeat of the pair. A retry chain
// collapses to its last try; a row is unrun only when every try was excluded;
// resolved is the harness's own verdict, kept over the budget. A missing field
// refuses the export rather than writing a config the scorer would refuse.
func ExportPaired(o ExportOptions, w io.Writer) error {
	switch {
	case o.ScorerSHA == "":
		return errors.New("export-paired needs --scorer-sha: ADR-106 Decision 2 pins its scorer's git blob sha, read from its pre-registration")
	case o.Selection == "" || o.Canary == "":
		return errors.New("export-paired needs --selection and --canary: the config line names the pool and the canary's result")
	case o.Model == "" || o.X == "" || o.Y == "" || len(o.Dirs) == 0:
		return errors.New("export-paired needs --model, --pair X,Y and at least one stage directory")
	}
	var sel struct {
		Rule, Dataset, Revision, Split string
		Selected                       []string
	}
	if err := readJSON(o.Selection, &sel); err != nil {
		return err
	}
	m := saltOf.FindStringSubmatch(sel.Rule)
	if sel.Rule == "" || sel.Dataset == "" || sel.Revision == "" || sel.Split == "" || len(sel.Selected) == 0 || m == nil {
		return fmt.Errorf("%s lacks the rule, its salt, the dataset, revision, split or selected ids", o.Selection)
	}
	var canary map[string]any
	if err := readJSON(o.Canary, &canary); err != nil {
		return err
	}
	if _, ok := canary["pass"]; !ok {
		return fmt.Errorf("%s holds no canary verdict", o.Canary)
	}
	var first *plan
	rows := map[string]pairedRow{}
	for _, dir := range o.Dirs {
		var p plan
		if err := readJSON(filepath.Join(dir, "run.json"), &p); err != nil {
			return err
		}
		if first == nil {
			first = &p
		} else if p.Stage != first.Stage || p.CLIVersion != first.CLIVersion || p.Repeats != first.Repeats || p.Profile != first.Profile ||
			p.BudgetSeconds != first.BudgetSeconds || p.StatusLineSHA256 != first.StatusLineSHA256 || p.PromptSHA256 != first.PromptSHA256 || !samePlugin(p.Plugin, first.Plugin) {
			return fmt.Errorf("%s ran another stage, CLI, repeat count, profile, budget, prompt or plugin than %s: one config line cannot describe both", dir, o.Dirs[0])
		}
		if !slices.Contains(p.Models, o.Model) {
			return fmt.Errorf("%s did not run model %s", dir, o.Model)
		}
		paths, err := filepath.Glob(filepath.Join(dir, "*.receipt.json"))
		if err != nil {
			return err
		}
		// A retry chain collapses within the directory that ran it; the same
		// attempt in two directories is refused, never picked by argument
		// order (the review of T2).
		local, tries := map[string]pairedRow{}, map[string]int{}
		for _, path := range paths {
			var r Receipt
			if err := readJSON(path, &r); err != nil {
				return err
			}
			if r.Model != o.Model || (r.Config != o.X && r.Config != o.Y) {
				continue
			}
			key := r.Task + "\x00" + r.Config + "\x00" + strconv.Itoa(r.Repeat)
			try := retryOf(r.Attempt)
			if prev, ok := tries[key]; ok && prev >= try {
				continue
			}
			tries[key] = try
			row := pairedRow{Instance: r.Task, Arm: r.Config, Repeat: r.Repeat, Status: r.Status,
				Limit: r.OverBudget || r.TimedOut, Unrun: r.Exclusion != ""}
			if r.Oracle.HarnessResolved != nil {
				row.Resolved = *r.Oracle.HarnessResolved
			}
			if r.Oracle.BrokeTests != nil {
				row.BrokeTests = *r.Oracle.BrokeTests
			}
			local[key] = row
		}
		for key, row := range local {
			if _, ok := rows[key]; ok {
				return fmt.Errorf("%s holds attempt %s.%s.%d, which another directory holds too", dir, row.Instance, row.Arm, row.Repeat)
			}
			rows[key] = row
		}
	}
	if first.StatusLineSHA256 == "" || first.PromptSHA256 == "" {
		return errors.New("the stage ran without --status-line: ADR-106 reads the STATUS claim and the prompt's digest")
	}
	if first.StatusLineSHA256 == "" {
		return errors.New("the stage ran without --status-line: ADR-106 reads the STATUS claim")
	}
	if (o.X == "P" || o.Y == "P") && first.Plugin == nil {
		return errors.New("the pair holds P and the stage names no plugin")
	}
	list := sha256.Sum256([]byte(strings.Join(sel.Selected, "\n")))
	instances := map[string]bool{}
	for _, r := range rows {
		instances[r.Instance] = true
	}
	config := map[string]any{
		"dataset": sel.Dataset, "split": sel.Split, "revision": sel.Revision,
		"selection_rule": sel.Rule, "selection_salt": m[1],
		"instance_count": len(sel.Selected), "instance_list_sha256": hex.EncodeToString(list[:]),
		"k": first.Repeats, "n": len(instances), "model": o.Model,
		"cli_version": first.CLIVersion, "plugin": first.Plugin,
		"scaffold":        map[string]any{"runner": "bench", "stage": first.Stage, "profile": first.Profile, "configs": first.Configs, "pair": []string{o.X, o.Y}},
		"prompt_template": PromptTemplate,
		"prompt_sha256":   first.PromptSHA256, "status_line_sha256": first.StatusLineSHA256,
		"max_turns": MaxTurns, "budget_seconds": first.BudgetSeconds,
		"scorer_blob_sha": o.ScorerSHA, "exporter": exporterRevision(), "canary": canary,
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(map[string]any{"config": config}); err != nil {
		return err
	}
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := enc.Encode(rows[k]); err != nil {
			return err
		}
	}
	return nil
}

// retryOf is the try an attempt id names: 0 for the first, N for ".rN".
func retryOf(id string) int {
	if i := strings.LastIndex(id, ".r"); i >= 0 {
		if n, err := strconv.Atoi(id[i+2:]); err == nil {
			return n
		}
	}
	return 0
}

func samePlugin(a, b *PluginRef) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// exporterRevision names the bench build that wrote the rows, from the build's
// own version-control stamp.
func exporterRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "bench (no build information)"
	}
	rev, dirty := "", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "bench (no revision stamp)"
	}
	if dirty {
		rev += "+modified"
	}
	return "bench@" + rev
}
