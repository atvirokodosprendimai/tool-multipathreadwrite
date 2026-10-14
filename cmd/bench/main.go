// Command bench measures whether mrw pays for itself on agent runs (ADR-156).
// It is a third binary on purpose: it starts an agent CLI, and mrw itself
// never does.
//
//	bench run     --stage S --out DIR --model M [--model M2] --claude PATH --mrw PATH
//	              --config-dir DIR --containment bypass --canary FILE [--task ID ...]
//	              [--configs A,B] [--repeats N] [--seed N] [--budget 20m]
//	bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass
//	bench verify  --stage S --tasks N --repeats R DIR
//	bench analyze DIR
//	bench hook    --log FILE            (configuration B's PreToolUse hook)
//	bench swebench-select [--n 20]      (instance ids on stdin)
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
  bench run     --stage S --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass --canary FILE [--task ID ...] [--configs A,B] [--repeats N] [--seed N] [--budget 20m]
  bench canary  --out DIR --model M --claude PATH --mrw PATH --config-dir DIR --containment bypass
  bench verify  --stage S --tasks N --repeats R DIR
  bench analyze DIR
  bench hook    --log FILE
  bench swebench-select [--n 20] < ids
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
	case "analyze":
		err = analyze(os.Args[2:], os.Stdout)
	case "hook":
		fs := flag.NewFlagSet("hook", flag.ExitOnError)
		log := fs.String("log", "", "where refusals and counted reads are appended")
		_ = fs.Parse(os.Args[2:])
		os.Exit(bench.Hook(os.Stdin, os.Stderr, *log))
	case "swebench-select":
		err = swebenchSelect(os.Args[2:], os.Stdin, os.Stdout)
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
	return fs, o, models, tasks
}

func run(args []string) error {
	fs, o, models, tasksDir := stageFlags("run")
	var only multi
	fs.StringVar(&o.Stage, "stage", "", "the stage's name")
	fs.StringVar(&o.Canary, "canary", "", "a passing canary.json for this CLI version")
	fs.Var(&only, "task", "run only this task (repeatable); default every measured task")
	configs := fs.String("configs", "A,B", "the configurations")
	fs.IntVar(&o.Repeats, "repeats", 2, "attempts per task, configuration and model")
	fs.Int64Var(&o.Seed, "seed", 156, "the order's seed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	o.Models, o.Configs = *models, strings.Split(*configs, ",")
	all, err := bench.LoadTasks(*tasksDir)
	if err != nil {
		return err
	}
	for _, t := range all {
		if t.Shape == bench.ShapeFixture {
			continue
		}
		if len(only) == 0 || contains(only, t.ID) {
			o.Tasks = append(o.Tasks, t)
		}
	}
	if len(only) > 0 && len(o.Tasks) != len(only) {
		return fmt.Errorf("--task names %d task(s), %d found", len(only), len(o.Tasks))
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

func analyze(args []string, w io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("analyze takes one directory")
	}
	var rs []bench.Receipt
	err := filepath.Walk(args[0], func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".receipt.json") || strings.Contains(p, string(filepath.Separator)+"pilot"+string(filepath.Separator)) {
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
	results, err := bench.Analyze(rs)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "| model | source | verdict | tasks | attempts | excluded | cost (B−A)/A [95%] | solved B−A [95%] | silent mis-edits A / B |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|")
	for _, r := range results {
		fmt.Fprintf(w, "| %s | %s | %s | %d | %d | %d | %+.3f [%+.3f, %+.3f] | %+.3f [%+.3f, %+.3f] | %d / %d |\n",
			r.Model, r.Source, r.Verdict, r.Tasks, r.Attempts, r.Excluded, r.RelCost, r.RelLo, r.RelHi, r.SolvedDiff, r.SolvedLo, r.SolvedHi, r.SilentA, r.SilentB)
	}
	return nil
}

func swebenchSelect(args []string, r io.Reader, w io.Writer) error {
	fs := flag.NewFlagSet("swebench-select", flag.ContinueOnError)
	n := fs.Int("n", 20, "how many instances")
	if err := fs.Parse(args); err != nil {
		return err
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
	for _, id := range bench.SelectSWEBench(ids, *n) {
		fmt.Fprintln(w, id)
	}
	return nil
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
