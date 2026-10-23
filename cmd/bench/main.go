// Command bench measures whether mrw pays for itself on agent runs (ADR-156).
// It is a third binary on purpose: it starts an agent CLI, and mrw itself
// never does.
//
//	bench run     --stage S --out DIR --model M [--model M2] --claude PATH --mrw PATH
//	              --config-dir DIR --containment bypass --canary FILE [--task ID ...]
//	              [--configs A,B[,P,W[,G,C]]] [--repeats N] [--seed N] [--budget 20m] [--max-turns N]
//	              [--profile stage5|stage6 [--plugin-dir DIR] [--node PATH] [--work-line FILE] [--checks FILE] [--side-arms FILE [--qh-binary FILE --qh-sums FILE]]] [--status-line]
//	              [--source swe-bench --dataset FILE --swebench-python PATH [--selection FILE] [--docker PATH]]
//	              [--block K [--configs-for MODEL=A,B,P ...]] [--disk-path PATH --disk-floor GiB]
//	bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass
//	              [--profile stage5|stage6 [--plugin-dir DIR] [--node PATH] [--work-line FILE] [--side-arms FILE [--qh-binary FILE --qh-sums FILE]]]
//	bench baseline --stage S --out DIR --selection FILE --dataset FILE --checks FILE [--surviving FILE] [--docker PATH] [--budget 20m]
//	bench export-paired --pair X,Y --model M --scorer-sha SHA --selection FILE --canary FILE DIR...
//	bench verify  --stage S --tasks N --repeats R DIR
//	bench look    DIR                                     (a Stage 5 directory: exit 0 complete, 1 a block due)
//	bench analyze [--tasks-dir DIR] [--selection FILE ...] DIR
//	bench side    --control P --arm G|C DIR...             (a side arm read against P, per model: ADR-163 Decision 9)
//	bench hook    --log FILE --tree DIR (configuration B's PreToolUse hook)
//	bench shim    --docker PATH --container NAME --tree DIR --state DIR --base SHA --name CMD -- ARGS...   (the shims of stage6; not run by hand)
//	bench swebench-select [--n 20]                         (instance ids on stdin; Stage 3's rule)
//	bench swebench-select --dataset FILE --min-hunks N [--exclude FILE ...] [--salt S] [--n 20] [--subset-min-hunks N] [--no-unnamed]
//	bench mine    --kind wide-rename|check-coupled --repo DIR --commit SHA [--n N] [--min-occurrences 15 --min-files 8] [--min-test-files 1]
//
// Nothing runs an agent until Zy has chosen how an attempt is contained and
// logged in once in the dedicated config directory (ADR-156 Decision 2).
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench"
)

const usageText = `usage:
  bench run     --stage S --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass --canary FILE [--task ID ...] [--configs A,B[,P,W[,G,C]]] [--repeats N] [--seed N] [--budget 20m] [--max-turns N] [--profile stage5|stage6 [--plugin-dir DIR] [--node PATH] [--work-line FILE] [--checks FILE] [--side-arms FILE [--qh-binary FILE --qh-sums FILE]]] [--status-line] [--source swe-bench --dataset FILE --swebench-python PATH [--selection FILE] [--docker PATH]]
  bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass [--profile stage5|stage6 [--plugin-dir DIR] [--node PATH] [--work-line FILE] [--side-arms FILE [--qh-binary FILE --qh-sums FILE]]]
  bench verify  --stage S --tasks N --repeats R DIR
  bench run     … [--block K [--configs-for MODEL=A,B,P ...]] [--disk-path PATH --disk-floor GiB]
  bench baseline --stage S --out DIR --selection FILE --dataset FILE --checks FILE [--surviving FILE] [--docker PATH] [--budget 20m]   (the declared check once per instance on the unmodified tree; no model)
  bench look    DIR
  bench export-paired --pair X,Y --model M --scorer-sha SHA --selection FILE --canary FILE DIR...
  bench analyze [--tasks-dir DIR] [--selection FILE ...] DIR
  bench side    --control P --arm G|C DIR...   (side arm G or C read against P alone, per model, paired by task; DIR is each block's directory)
  bench hook    --log FILE --tree DIR
  bench shim    --docker PATH --container NAME --tree DIR --state DIR --base SHA --name CMD -- ARGS...   (the shims of stage6; not run by hand)
  bench swebench-select [--n 20] < ids
  bench swebench-select --dataset FILE --min-hunks N [--exclude FILE ...] [--salt S] [--n 20] [--subset-min-hunks N] [--no-unnamed]
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
	case "baseline":
		err = baseline(os.Args[2:])
	case "export-paired":
		err = exportPaired(os.Args[2:], os.Stdout)
	case "analyze":
		err = analyze(os.Args[2:], os.Stdout)
	case "side":
		err = side(os.Args[2:], os.Stdout)
	case "look":
		os.Exit(look(os.Args[2:], os.Stdout, os.Stderr))
	case "hook":
		fs := flag.NewFlagSet("hook", flag.ExitOnError)
		log := fs.String("log", "", "where refusals and counted reads are appended")
		tree := fs.String("tree", "", "the attempt's task tree: a write whose target lands outside it passes")
		_ = fs.Parse(os.Args[2:])
		os.Exit(bench.Hook(os.Stdin, os.Stderr, *log, *tree))
	case "shim":
		cwd, _ := os.Getwd()
		os.Exit(bench.Shim(os.Args[2:], cwd, os.Stdin, os.Stdout, os.Stderr))
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
		// A run stopped at a boundary is resumable, not failed (ADR-162).
		if errors.Is(err, bench.ErrStartErrors) || errors.Is(err, bench.ErrDiskFloor) {
			os.Exit(2)
		}
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
	fs.StringVar(&o.Profile, "profile", "", "stage5: the profile of ADR-162, for every configuration (skills on, Skill offered, node on PATH, no external model); stage6: the profile of ADR-163, stage5 plus a container of the instance image for python, python3, pytest and py.test")
	fs.StringVar(&o.PluginDir, "plugin-dir", "", "the plugin configuration P loads: its directory in a tagged checkout (ADR-162)")
	fs.StringVar(&o.Node, "node", "", "the node the profile links on every attempt's PATH; found on PATH when empty")
	fs.StringVar(&o.Docker, "docker", "docker", "the docker CLI: stage6 starts the container of each attempt with it, and a SWE-bench run pulls and removes images with it")
	fs.StringVar(&o.CanaryImage, "canary-image", "", "stage6: the image a task with none of its own (the canary fixture) runs its container in")
	fs.StringVar(&o.Checks, "checks", "", "stage6: the declared-check table, bench/swe-bench/checks.json: one whole runner per repository, written as .quality-harness.json into every arm's tree before the base commit (ADR-163 Decision 2); required with a SWE-bench source")
	fs.StringVar(&o.WorkLine, "work-line", "", "stage6: the file holding the one line configuration W adds to its instruction, the text quality-harness's ADR-112 pre-registers; its SHA-256 goes in run.json (ADR-163 Decision 3)")
	fs.IntVar(&o.MaxTurns, "max-turns", 0, "the stage's turn cap, written in run.json and the rows; 0 for the registered 60 (ADR-163 Decision 3)")
	fs.StringVar(&o.SideArms, "side-arms", "", "stage6: the file that registers side arms G (Go hooks) and C (turnEnd notify) at the plugin's tag: C's setting, G's asset sha256, stage path, runtime setting and hook-output marker; the arms run beside P and are read against it by bench side (ADR-163 Decision 9)")
	fs.StringVar(&o.QhBinary, "qh-binary", "", "stage6, G: the release asset (qh-darwin-arm64) bench verifies against --qh-sums and the registration, then stages into each G attempt's own plugin data directory")
	fs.StringVar(&o.QhSums, "qh-sums", "", "stage6, G: the release's SHA256SUMS that names the asset")
	return fs, o, models, tasks
}

func run(args []string) error {
	fs, o, models, tasksDir := stageFlags("run")
	var only multi
	fs.StringVar(&o.Stage, "stage", "", "the stage's name")
	fs.StringVar(&o.Canary, "canary", "", "a passing canary.json for this CLI version")
	fs.Var(&only, "task", "run only this task (repeatable); default every measured task")
	configs := fs.String("configs", "A,B", "the configurations: A, B, P (the quality-harness plugin, under --profile stage5 with --plugin-dir), W (P plus --work-line's line, under --profile stage6), and the side arms G (P with its hooks run by the qh binary) and C (P with turnEnd notify), under --profile stage6 with --side-arms and beside P")
	fs.IntVar(&o.Repeats, "repeats", 2, "attempts per task, configuration and model")
	fs.Int64Var(&o.Seed, "seed", 156, "the order's seed")
	fs.BoolVar(&o.StatusLine, "status-line", false, "append the registered STATUS sentence to every instruction and record the claim (ADR-162)")
	fs.IntVar(&o.Block, "block", 0, "ADR-162: run block K of the selection (instances 40(K−1)+1 to 40K, the fifth the rest) into --out, named block-K")
	var configsFor multi
	fs.Var(&configsFor, "configs-for", "MODEL=A,B,P: a model's own configurations, as bench look printed (repeatable)")
	fs.StringVar(&o.DiskPath, "disk-path", "", "the volume the disk floor is checked on: the one holding Docker's disk image")
	fs.Float64Var(&o.DiskFloorGiB, "disk-floor", 0, "GiB free below which no attempt starts (ADR-162 Decision 9); 0 for none")
	source := fs.String("source", "own", "the task source: own (the corpus), or swe-bench (the registered slice, ADR-156 T5)")
	selection := fs.String("selection", filepath.Join("bench", "swe-bench", "selection.json"), "the SWE-bench slice")
	dataset := fs.String("dataset", "", "the SWE-bench dataset file the harness reads, at the revision T5 recorded")
	python := fs.String("swebench-python", "", "an interpreter with the SWE-bench harness installed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	o.Models, o.Configs = *models, strings.Split(*configs, ",")
	for _, s := range configsFor {
		m, list, ok := strings.Cut(s, "=")
		if !ok || m == "" || list == "" {
			return fmt.Errorf("--configs-for %q: MODEL=A,B,P", s)
		}
		if o.ConfigsFor == nil {
			o.ConfigsFor = map[string][]string{}
		}
		o.ConfigsFor[m] = strings.Split(list, ",")
	}
	if o.DiskFloorGiB > 0 && o.DiskPath == "" {
		return fmt.Errorf("--disk-floor needs --disk-path, the volume to check")
	}
	var all []bench.Task
	var err error
	switch *source {
	case "own":
		all, err = bench.LoadTasks(*tasksDir)
	case "swe-bench":
		if *dataset == "" || *python == "" {
			return fmt.Errorf("--source swe-bench needs --dataset and --swebench-python")
		}
		g := &bench.SWEGrader{Docker: o.Docker}
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
	if o.Block > 0 {
		// Stage6 draws its blocks from the surviving pool, whose block count is its own (ADR-163 Decision 4).
		blockOf := bench.BlockTasks
		if o.Profile == bench.ProfileStage6 {
			blockOf = bench.SurvivingBlock
		}
		if o.Tasks, err = blockOf(o.Tasks, o.Block); err != nil {
			return err
		}
	}
	if o.Bench, err = os.Executable(); err != nil {
		return err
	}
	return bench.Run(*o)
}

// baseline runs each selected instance's declared check once on the
// unmodified tree in its container and records green or red (ADR-163
// Decision 4). It starts no model.
func baseline(args []string) error {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	o := bench.BaselineOptions{}
	fs.StringVar(&o.Stage, "stage", "", "the stage's name; its containers carry it as a label")
	fs.StringVar(&o.Out, "out", "", "the directory baseline.json is written to; one holding a baseline is refused")
	fs.StringVar(&o.Selection, "selection", "", "the selection whose instances are run, in its order")
	dataset := fs.String("dataset", "", "the SWE-bench dataset file the instances are read from")
	fs.StringVar(&o.Checks, "checks", "", "the declared-check table, bench/swe-bench/checks.json")
	fs.StringVar(&o.Surviving, "surviving", "", "write the green instances, in the selection's order, as a selection: the surviving pool")
	fs.StringVar(&o.Docker, "docker", "docker", "the docker CLI")
	fs.StringVar(&o.Scratch, "scratch", os.TempDir(), "where host trees are built")
	fs.DurationVar(&o.Budget, "budget", bench.DefaultBudget, "bounds a container's life, so a check that hangs ends with it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if o.Selection == "" || *dataset == "" {
		return fmt.Errorf("baseline needs --selection and --dataset")
	}
	o.SWE = &bench.SWEGrader{Docker: o.Docker}
	var err error
	if o.Tasks, err = bench.LoadSWEBench(o.Selection, *dataset); err != nil {
		return err
	}
	return bench.Baseline(o)
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
	if o.WorkLine != "" {
		o.Configs = append(o.Configs, "W")
	}
	if o.SideArms != "" {
		arms, err := bench.SideNamed(o.SideArms)
		if err != nil {
			return err
		}
		o.Configs = append(o.Configs, arms...)
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
	// A Stage 5 block is read by its stage's look, which applies the stage's
	// rulings; its raw receipts are not a slice of the 95% table.
	if abs, err := filepath.Abs(fs.Arg(0)); err == nil && bench.LookStage(filepath.Dir(abs)) {
		return fmt.Errorf("%s is a block of a Stage 5 stage: read it with bench look %s", fs.Arg(0), filepath.Dir(abs))
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
	// A Stage 5 directory (ADR-162) is read by its looks, not pooled here:
	// its receipts stay out of the 95% table and get a table of their own.
	var lookDirs []string
	err = filepath.Walk(fs.Arg(0), func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if bench.LookStage(p) {
				lookDirs = append(lookDirs, p)
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".receipt.json") {
			return nil
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
			Unnamed []string       `json:"unnamed"`
			Subset  *bench.HunkSet `json:"min_hunks_subset"`
		}
		if err := json.Unmarshal(b, &sel); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		if len(sel.Unnamed) > 0 {
			subsets = append(subsets, bench.Subset{Label: bench.OnlyUnnamedFiles, Tasks: sel.Unnamed})
		}
		if sel.Subset != nil {
			subsets = append(subsets, bench.Subset{Label: bench.OnlyHunks(sel.Subset.MinHunks), Tasks: sel.Subset.Selected})
		}
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
	for _, dir := range lookDirs {
		rep, err := bench.Look(dir)
		if err != nil {
			return err
		}
		desc, err := bench.LookDescriptive(dir, rep, subsets)
		if err != nil {
			return err
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "| stage | model | source | verdict | look | tasks | attempts | excluded | cost (X−A)/A [99%] | solved X−A [99%, Tango] | B shell reads |")
		fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|")
		for _, row := range rep.Rows {
			look := lookLabel(rep, row)
			r := row.Result
			fmt.Fprintf(w, "| %s | %s | swe-bench, %s against A | %s | %s | %d | %d | %d | %+.3f [%+.3f, %+.3f] | %+.3f [%+.3f, %+.3f] | %d |\n",
				r.Stage, r.Model, row.Comparison, row.Verdict, look, r.Tasks, r.Attempts, r.Excluded, r.RelCost, r.RelLo, r.RelHi, r.SolvedDiff, r.SolvedLo, r.SolvedHi, r.ReadsB)
		}
		for _, r := range desc {
			fmt.Fprintf(w, "| %s | %s | swe-bench, %s against A (%s) | (%s) | look %d | %d | %d | %d | %+.3f [%+.3f, %+.3f] | %+.3f [%+.3f, %+.3f] | %d |\n",
				r.Stage, r.Model, r.Comparison, r.Descriptive, r.Verdict, r.Look, r.Tasks, r.Attempts, r.Excluded, r.RelCost, r.RelLo, r.RelHi, r.SolvedDiff, r.SolvedLo, r.SolvedHi, r.ReadsB)
		}
	}
	return nil
}

// side reads a side arm against P, per model and paired by task, over the
// blocks given (ADR-163 Decision 9). It reads nothing of ADR-112's rows or stop
// rule.
func side(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("side", flag.ContinueOnError)
	control := fs.String("control", "", "the control: P, the only one a side arm is read against")
	arm := fs.String("arm", "", "the side arm: G or C")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *control == "" || *arm == "" || fs.NArg() == 0 {
		return fmt.Errorf("side needs --control P, --arm G|C and the stage's block directories")
	}
	rows, err := bench.Side(fs.Args(), *control, *arm)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "| stage | model | arm | verdict | paired | excluded | cost (arm−P)/P [99%] | solved arm−P [99%, Tango] | turns P / arm | wall s P / arm |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|")
	for _, r := range rows {
		x := r.Result
		fmt.Fprintf(w, "| %s | %s | %s against %s | %s | %d | %d | %+.3f [%+.3f, %+.3f] | %+.3f [%+.3f, %+.3f] | %.1f / %.1f | %.1f / %.1f |\n",
			r.Stage, r.Model, r.Arm, r.Control, r.Verdict, x.Tasks, x.Excluded, x.RelCost, x.RelLo, x.RelHi, x.SolvedDiff, x.SolvedLo, x.SolvedHi,
			r.TurnsControl, r.TurnsArm, r.WallControl, r.WallArm)
	}
	for _, r := range rows {
		fmt.Fprintf(w, "\n%s, %s: tests that %s. ", r.Model, r.Verdict, r.Claim)
		if r.Verdict == bench.SideDescriptive {
			fmt.Fprintf(w, "Fewer than %d paired tasks: the numbers are published and no claim is read from them. ", bench.SideFloor)
		}
		if len(r.Result.Unpaired) > 0 {
			fmt.Fprintf(w, "Dropped, unpaired: %s. ", strings.Join(r.Result.Unpaired, ", "))
		}
		if len(r.Result.CostUnpaired) > 0 {
			fmt.Fprintf(w, "Paired for solving and out of the cost comparison: %s.", strings.Join(r.Result.CostUnpaired, ", "))
		}
		fmt.Fprintln(w)
	}
	return nil
}

// lookLabel says where a pair stands: closed at the look that closed it, or
// still open, either awaiting the next look or left open when the stage ended.
func lookLabel(rep bench.LookReport, row bench.LookRow) string {
	switch {
	case row.ClosedAt != 0:
		return fmt.Sprintf("closed at look %d", row.ClosedAt)
	case rep.Complete:
		return fmt.Sprintf("open when the stage ended at look %d", rep.Blocks)
	default:
		return fmt.Sprintf("open after look %d", rep.Blocks)
	}
}

func swebenchSelect(args []string, r io.Reader, w io.Writer) error {
	fs := flag.NewFlagSet("swebench-select", flag.ContinueOnError)
	n := fs.Int("n", 20, "how many instances")
	salt := fs.String("salt", "adr-156", "with --dataset, the salt hashed before each id (adr-160 for Stage 4)")
	dataset := fs.String("dataset", "", "with --min-hunks, select from this dataset file instead of ids on stdin (ADR-160)")
	minHunks := fs.Int("min-hunks", 0, "keep instances whose gold patch has at least this many hunks")
	var excludes multi
	fs.Var(&excludes, "exclude", "a selection file whose selected ids are left out (repeatable)")
	subset := fs.Int("subset-min-hunks", 0, "with --dataset, also write the selected instances with at least this many hunks (ADR-162's descriptive row)")
	noUnnamed := fs.Bool("no-unnamed", false, "with --dataset, leave out ADR-160's unnamed-files subset")
	name := fs.String("name", "", "with --dataset, the dataset's name to record (ADR-106 reads it)")
	revision := fs.String("revision", "", "with --dataset, the dataset revision to record")
	split := fs.String("split", "", "with --dataset, the dataset split to record")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dataset != "" {
		var ex []string
		for _, f := range excludes {
			b, err := os.ReadFile(f)
			if err != nil {
				return err
			}
			var sel struct {
				Selected []string `json:"selected"`
			}
			if err := json.Unmarshal(b, &sel); err != nil {
				return fmt.Errorf("%s: %w", f, err)
			}
			ex = append(ex, sel.Selected...)
		}
		got, err := bench.SelectMultiHunk(*dataset, *salt+":", *minHunks, ex, *n)
		if err != nil {
			return err
		}
		if *noUnnamed {
			got.Unnamed = nil
		}
		if *subset > 0 {
			ids, err := bench.HunkSubset(*dataset, got.Selected, *subset)
			if err != nil {
				return err
			}
			got.Subset = &bench.HunkSet{MinHunks: *subset, Selected: ids}
		}
		out := struct {
			Rule     string `json:"rule"`
			Dataset  string `json:"dataset,omitempty"`
			Revision string `json:"revision,omitempty"`
			Split    string `json:"split,omitempty"`
			bench.MultiHunk
		}{fmt.Sprintf("the %d instance ids with the lowest sha256(%q + instance_id) among those whose gold patch has at least %d hunk headers, less the ids selected in %s (BACKLOG, Pre-registered for %s; bench swebench-select)",
			*n, *salt+":", *minHunks, strings.Join(excludes, " and "), strings.ToUpper(*salt)), *name, *revision, *split, got}
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

// look applies ADR-162's stop rule to a Stage 5 directory and prints what it
// found: exit 0 when the stage is complete, 1 when a block is due, 2 when the
// directory cannot be read as one.
func look(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "bench look takes one Stage 5 directory")
		return 2
	}
	rep, err := bench.Look(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "bench:", err)
		return 2
	}
	fmt.Fprint(stdout, rep.Text())
	if rep.Complete {
		return 0
	}
	return 1
}
