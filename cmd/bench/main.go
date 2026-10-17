// Command bench measures whether mrw pays for itself on agent runs (ADR-156).
// It is a third binary on purpose: it starts an agent CLI, and mrw itself
// never does.
//
//	bench run     --stage S --out DIR --model M [--model M2] --claude PATH --mrw PATH
//	              --config-dir DIR --containment bypass --canary FILE [--task ID ...]
//	              [--configs A,B[,P]] [--repeats N] [--seed N] [--budget 20m]
//	              [--profile stage5 [--plugin-dir DIR] [--node PATH]] [--status-line]
//	              [--source swe-bench --dataset FILE --swebench-python PATH [--selection FILE] [--docker PATH]]
//	bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass
//	              [--profile stage5 [--plugin-dir DIR] [--node PATH]]
//	bench export-paired --pair X,Y --model M --scorer-sha SHA --selection FILE --canary FILE DIR...
//	bench verify  --stage S --tasks N --repeats R DIR
//	bench analyze [--tasks-dir DIR] [--selection FILE ...] DIR
//	bench hook    --log FILE --tree DIR (configuration B's PreToolUse hook)
//	bench swebench-select [--n 20]                         (instance ids on stdin; Stage 3's rule)
//	bench swebench-select --dataset FILE --min-hunks N [--exclude FILE] [--salt S] [--n 20]
//	bench mine    --kind wide-rename|check-coupled --repo DIR --commit SHA [--n N] [--min-occurrences 15 --min-files 8] [--min-test-files 1]
//
// Nothing runs an agent until Zy has chosen how an attempt is contained and
// logged in once in the dedicated config directory (ADR-156 Decision 2).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench"
)

const usageText = `usage:
  bench run     --stage S --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass --canary FILE [--task ID ...] [--configs A,B[,P]] [--repeats N] [--seed N] [--budget 20m] [--profile stage5 [--plugin-dir DIR] [--node PATH]] [--status-line] [--source swe-bench --dataset FILE --swebench-python PATH [--selection FILE] [--docker PATH]]
  bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass [--profile stage5 [--plugin-dir DIR] [--node PATH]]
  bench verify  --stage S --tasks N --repeats R DIR
  bench export-paired --pair X,Y --model M --scorer-sha SHA --selection FILE --canary FILE DIR...
  bench analyze [--tasks-dir DIR] [--selection FILE ...] DIR
  bench hook    --log FILE --tree DIR
  bench swebench-select [--n 20] < ids
  bench swebench-select --dataset FILE --min-hunks N [--exclude FILE] [--salt S] [--n 20]
  bench mine    --kind wide-rename|check-coupled --repo DIR --commit SHA [--n N] [--min-occurrences 15 --min-files 8] [--min-test-files 1]
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "canary":
		err = canary(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	case "export-paired":
		err = exportPaired(os.Args[2:], os.Stdout)
	case "analyze":
		err = analyze(os.Args[2:], os.Stdout)
	case "hook":
		fs := flag.NewFlagSet("hook", flag.ExitOnError)
		log := fs.String("log", "", "where refusals and counted reads are appended")
		tree := fs.String("tree", "", "the attempt's task tree: a write whose target lands outside it passes")
		_ = fs.Parse(os.Args[2:])
		os.Exit(bench.Hook(os.Stdin, os.Stderr, *log, *tree))
	case "swebench-select":
		err = swebenchSelect(os.Args[2:], os.Stdin, os.Stdout)
	case "mine":
		err = mine(os.Args[2:], os.Stdout)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usageText)
		return
	default:
		fmt.Fprintf(os.Stderr, "bench: unknown verb %q\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		os.Exit(1)
	}
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

// stageFlags are the flags run and canary share.
func stageFlags(name string) (*flag.FlagSet, *bench.RunOptions, *multi, *string) {
	o := &bench.RunOptions{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	models := &multi{}
	tasks := fs.String("tasks-dir", filepath.Join("bench", "tasks"), "the corpus")
	fs.StringVar(&o.Out, "out", "", "the receipt directory, docs/bench/<date>-<stage>/")
	fs.Var(models, "model", "a model id (repeatable)")
	fs.StringVar(&o.Claude, "claude", "", "the CLI, by absolute path")
	fs.StringVar(&o.Mrw, "mrw", "", "the pinned mrw build configuration B gets")
	fs.StringVar(&o.ConfigDir, "config-dir", "", "the dedicated CLAUDE_CONFIG_DIR")
	fs.StringVar(&o.Containment, "containment", "", "bypass, or user:<name> once Zy chooses it")
	fs.DurationVar(&o.Budget, "budget", bench.DefaultBudget, "an attempt's wall-clock budget")
	fs.StringVar(&o.Scratch, "scratch", os.TempDir(), "where attempt trees are built")
	fs.StringVar(&o.Profile, "profile", "", "stage5: ADR-162's profile, for every configuration (skills on, Skill offered, node on PATH, no external model)")
	fs.StringVar(&o.PluginDir, "plugin-dir", "", "the plugin configuration P loads: its directory in a tagged checkout (ADR-162)")
	fs.StringVar(&o.Node, "node", "", "the node the profile links on every attempt's PATH; found on PATH when empty")
	return fs, o, models, tasks
}

func run(args []string) error {
	fs, o, models, tasksDir := stageFlags("run")
	var only multi
	fs.StringVar(&o.Stage, "stage", "", "the stage's name")
	fs.StringVar(&o.Canary, "canary", "", "a passing canary.json for this CLI version")
	fs.Var(&only, "task", "run only this task (repeatable); default every measured task")
	configs := fs.String("configs", "A,B", "the configurations: A, B, and P (the quality-harness plugin, under --profile stage5 with --plugin-dir)")
	fs.IntVar(&o.Repeats, "repeats", 2, "attempts per task, configuration and model")
	fs.Int64Var(&o.Seed, "seed", 156, "the order's seed")
	fs.BoolVar(&o.StatusLine, "status-line", false, "append the registered STATUS sentence to every instruction and record the claim (ADR-162)")
	source := fs.String("source", "own", "the task source: own (the corpus), or swe-bench (the registered slice, ADR-156 T5)")
	selection := fs.String("selection", filepath.Join("bench", "swe-bench", "selection.json"), "the SWE-bench slice")
	dataset := fs.String("dataset", "", "the SWE-bench dataset file the harness reads, at the revision T5 recorded")
	python := fs.String("swebench-python", "", "an interpreter with the SWE-bench harness installed")
	docker := fs.String("docker", "docker", "the docker CLI the harness's images are pulled and removed with")
	if err := fs.Parse(args); err != nil {
		return err
	}
	o.Models, o.Configs = *models, strings.Split(*configs, ",")
	var all []bench.Task
	var err error
	switch *source {
	case "own":
		all, err = bench.LoadTasks(*tasksDir)
	case "swe-bench":
		if *dataset == "" || *python == "" {
			return fmt.Errorf("--source swe-bench needs --dataset and --swebench-python")
		}
		g := &bench.SWEGrader{Docker: *docker}
		if g.Dataset, err = filepath.Abs(*dataset); err != nil {
			return err
		}
		if g.Python, err = filepath.Abs(*python); err != nil {
			return err
		}
		o.SWE = g
		all, err = bench.LoadSWEBench(*selection, g.Dataset)
	default:
		return fmt.Errorf("--source %q: own or swe-bench", *source)
	}
	if err != nil {
		return err
	}
	if o.Tasks, err = bench.SelectTasks(all, only); err != nil {
		return err
	}
	if o.Bench, err = os.Executable(); err != nil {
		return err
	}
	return bench.Run(*o)
}

func canary(args []string) error {
	fs, o, models, tasksDir := stageFlags("canary")
	if err := fs.Parse(args); err != nil {
		return err
	}
	o.Models, o.Stage, o.Repeats, o.Configs = *models, "canary", 1, []string{"A", "B"}
	if o.PluginDir != "" {
		o.Configs = append(o.Configs, "P")
	}
	task, err := bench.LoadTask(filepath.Join(*tasksDir, "canary"))
	if err != nil {
		return err
	}
	o.Tasks = []bench.Task{task}
	if o.Bench, err = os.Executable(); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	md, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	c, err := bench.RunCanary(*o, task, string(md))
	if err != nil {
		return err
	}
	if !c.Pass {
		b, _ := json.MarshalIndent(c.Problems, "", "  ")
		return fmt.Errorf("the canary failed; no stage may start:\n%s", b)
	}
	fmt.Println("canary passed for", c.CLIVersion)
	return nil
}

func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	stage := fs.String("stage", "", "the stage's name")
	tasks := fs.Int("tasks", 0, "the tasks the stage planned")
	repeats := fs.Int("repeats", 0, "the repeats the stage planned")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("verify takes one directory")
	}
	return bench.Verify(fs.Arg(0), *stage, *tasks, *repeats)
}

// exportPaired writes quality-harness ADR-106's paired rows for one model and
// one pair of configurations (ADR-162 Decision 3).
func exportPaired(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("export-paired", flag.ContinueOnError)
	pair := fs.String("pair", "A,P", "the two configurations, the control first")
	o := bench.ExportOptions{}
	fs.StringVar(&o.Model, "model", "", "the model whose rows are written")
	fs.StringVar(&o.ScorerSHA, "scorer-sha", "", "the git blob sha of ADR-106's scorer, from its pre-registration")
	fs.StringVar(&o.Selection, "selection", "", "the selection file the stage drew from")
	fs.StringVar(&o.Canary, "canary", "", "the canary.json the stage ran on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	x, y, ok := strings.Cut(*pair, ",")
	if !ok || x == "" || y == "" || strings.Contains(y, ",") {
		return fmt.Errorf("--pair %q: two configurations, X,Y", *pair)
	}
	o.X, o.Y, o.Dirs = x, y, fs.Args()
	return bench.ExportPaired(o, w)
}

func analyze(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	tasksDir := fs.String("tasks-dir", filepath.Join("bench", "tasks"), "the corpus, read for the tasks superseded since a stage ran (ADR-159)")
	var selections multi
	fs.Var(&selections, "selection", "a slice's selection file whose unnamed-files subset gets a descriptive row (ADR-160); repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("analyze takes one directory")
	}
	tasks, err := bench.LoadTasks(*tasksDir)
	if err != nil {
		return err
	}
	var superseded []string
	for _, t := range tasks {
		if t.SupersededBy != "" {
			superseded = append(superseded, t.ID)
		}
	}
	var rs []bench.Receipt
	err = filepath.Walk(fs.Arg(0), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".receipt.json") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var r bench.Receipt
		if err := json.Unmarshal(b, &r); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		rs = append(rs, r)
		return nil
	})
	if err != nil {
		return err
	}
	var subsets []bench.Subset
	for _, f := range selections {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		var sel struct {
			Unnamed []string `json:"unnamed"`
		}
		if err := json.Unmarshal(b, &sel); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		subsets = append(subsets, bench.Subset{Label: bench.OnlyUnnamedFiles, Tasks: sel.Unnamed})
	}
	results, err := bench.Analyze(rs, superseded...)
	if len(subsets) > 0 {
		results, err = bench.AnalyzeSubsets(rs, superseded, subsets)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "| stage | model | source | verdict | tasks | attempts | excluded | cost (B−A)/A [95%] | solved B−A [95%] | silent mis-edits A / B | B shell reads | B slipped writes |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, r := range results {
		source, verdict := r.Source, r.Verdict
		if r.Descriptive != "" {
			source, verdict = r.Source+" ("+r.Descriptive+")", "("+r.Verdict+")"
		}
		// A SWE-bench task has no allowed regions, so a silent mis-edit is
		// not measured there (ADR-156 Decision 4), which is not a count of 0.
		silent := fmt.Sprintf("%d / %d", r.SilentA, r.SilentB)
		if r.Source == "swe-bench" {
			silent = "not measured"
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %d | %d | %d | %+.3f [%+.3f, %+.3f] | %+.3f [%+.3f, %+.3f] | %s | %d | %d |\n",
			r.Stage, r.Model, source, verdict, r.Tasks, r.Attempts, r.Excluded, r.RelCost, r.RelLo, r.RelHi, r.SolvedDiff, r.SolvedLo, r.SolvedHi, silent, r.ReadsB, r.SlippedB)
	}
	return nil
}

func swebenchSelect(args []string, r io.Reader, w io.Writer) error {
	fs := flag.NewFlagSet("swebench-select", flag.ContinueOnError)
	n := fs.Int("n", 20, "how many instances")
	salt := fs.String("salt", "adr-156", "with --dataset, the salt hashed before each id (adr-160 for Stage 4)")
	dataset := fs.String("dataset", "", "with --min-hunks, select from this dataset file instead of ids on stdin (ADR-160)")
	minHunks := fs.Int("min-hunks", 0, "keep instances whose gold patch has at least this many hunks")
	exclude := fs.String("exclude", "", "a selection file whose selected ids are left out")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dataset != "" {
		var ex []string
		if *exclude != "" {
			b, err := os.ReadFile(*exclude)
			if err != nil {
				return err
			}
			var sel struct {
				Selected []string `json:"selected"`
			}
			if err := json.Unmarshal(b, &sel); err != nil {
				return fmt.Errorf("%s: %w", *exclude, err)
			}
			ex = sel.Selected
		}
		got, err := bench.SelectMultiHunk(*dataset, *salt+":", *minHunks, ex, *n)
		if err != nil {
			return err
		}
		out := struct {
			Rule string `json:"rule"`
			bench.MultiHunk
		}{fmt.Sprintf("the %d instance ids with the lowest sha256(%q + instance_id) among those whose gold patch has at least %d hunk headers, less the ids selected in %s (BACKLOG, Pre-registered for ADR-160; bench swebench-select)", *n, *salt+":", *minHunks, *exclude), got}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	var ids []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if id := strings.TrimSpace(sc.Text()); id != "" {
			ids = append(ids, id)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	// Ids on stdin are Stage 3's rule, unsalted by --salt (ADR-156 T5).
	for _, id := range bench.SelectSWEBench(ids, *n) {
		fmt.Fprintln(w, id)
	}
	return nil
}

// mine lists ADR-160's candidates of one kind, in registered order: for a
// wide rename the commit's tree is read from git, never the working tree.
func mine(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("mine", flag.ContinueOnError)
	kind := fs.String("kind", "", "wide-rename or check-coupled")
	repo := fs.String("repo", "", "the repository")
	commit := fs.String("commit", "", "the pinned commit")
	n := fs.Int("n", 0, "how many candidates to print; 0 prints every one")
	minOcc := fs.Int("min-occurrences", 15, "wide-rename: the fewest occurrences, comments included (25 when Decision 2 raises the floor)")
	minFiles := fs.Int("min-files", 8, "wide-rename: the fewest files they span (12 when Decision 2 raises the floor)")
	minTests := fs.Int("min-test-files", 1, "check-coupled: the fewest test files beside the source that hold the literal (2 when Decision 2 raises the floor)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *commit == "" {
		return fmt.Errorf("mine needs --repo and --commit")
	}
	var lines []string
	if *kind != "wide-rename" && *kind != "check-coupled" {
		return fmt.Errorf("--kind is wide-rename or check-coupled, not %q", *kind)
	}
	dir, err := os.MkdirTemp("", "bench-mine-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	if err := bench.ExportTree(*repo, *commit, dir); err != nil {
		return err
	}
	switch *kind {
	case "wide-rename":
		cs, err := bench.MineWideRenames(dir, *minOcc, *minFiles)
		if err != nil {
			return err
		}
		for _, c := range cs {
			lines = append(lines, fmt.Sprintf("%s\t%d occurrences\t%d files", c.Name, c.Occurrences, c.Files))
		}
	case "check-coupled":
		cs, err := bench.MineCoupledLiterals(dir)
		if err != nil {
			return err
		}
		for _, c := range cs {
			if len(c.Tests) < *minTests {
				continue
			}
			lines = append(lines, fmt.Sprintf("%s\t%q -> %q\t%s", c.File, c.Old, c.New, strings.Join(c.Tests, " ")))
		}
	default:
		return fmt.Errorf("--kind is wide-rename or check-coupled, not %q", *kind)
	}
	if *n > 0 && *n < len(lines) {
		lines = lines[:*n]
	}
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	return nil
}
