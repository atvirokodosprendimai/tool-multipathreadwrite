// Command nativebench runs the native performance baseline that
// docs/bench/native-baseline/PREREGISTRATION.md registers: the same workloads
// against two mrw binaries, interleaved, under a load gate and a wall budget,
// every timing kept. It measures processes, not agents: nothing here says what
// a task costs an agent.
//
//	nativebench -a RELEASE_BIN -b SOURCE_BIN -tree TAGGED_TREE -out DIR
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// seed is the fixtures' seed: every fixture is a function of it and its size.
const seed = 20261029

func main() {
	a := flag.String("a", "", "the release binary")
	b := flag.String("b", "", "the source build under test")
	tree := flag.String("tree", "", "this repository at the tagged tree, for R3")
	out := flag.String("out", "", "the result directory; raw/ is written inside it")
	runs := flag.Int("runs", 30, "timed runs per workload, size and binary")
	warm := flag.Int("warm", 5, "discarded warm-up runs per workload, size and binary")
	budget := flag.Duration("budget", 15*time.Minute, "the wall budget for the whole baseline")
	maxLoad := flag.Float64("max-load", 2.0, "a workload starts only below this 1-minute load average")
	loadWait := flag.Duration("load-wait", 10*time.Minute, "how long to wait for the load gate before stopping")
	flag.Parse()
	if *a == "" || *b == "" || *tree == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	binOf = func(i int) string { return [2]string{*a, *b}[i] }
	r := &runner{bins: [2]string{*a, *b}, runs: *runs, warm: *warm, maxLoad: *maxLoad, loadWait: *loadWait, deadline: time.Now().Add(*budget)}
	if err := r.all(*tree, *out); err != nil {
		fmt.Fprintln(os.Stderr, "nativebench:", err)
		os.Exit(1)
	}
}

// sample is one timed run: the raw record kept for every run, failures included.
type sample struct {
	Workload string  `json:"workload"`
	Size     string  `json:"size"`
	Binary   string  `json:"binary"`
	Run      int     `json:"run"`
	WallNS   int64   `json:"wall_ns"`
	CPUNS    int64   `json:"cpu_ns"`
	MaxRSS   int64   `json:"max_rss_bytes"`
	Stdout   int     `json:"stdout_bytes"`
	Exit     int     `json:"exit"`
	Load     float64 `json:"load_before"`
}

// workload is one (workload, size) cell: prepare runs untimed before each
// timed run, cmd is what is timed, both in dir.
type workload struct {
	name, size string
	dir        string
	prepare    func(bin int) error
	args       func(bin int) []string
	state      func(bin int) string
}

type runner struct {
	bins     [2]string
	runs     int
	warm     int
	maxLoad  float64
	loadWait time.Duration
	deadline time.Time
	samples  []sample
	stopped  string
}

func (r *runner) all(tree, out string) error {
	work, err := os.MkdirTemp("", "nativebench-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	ws, err := workloads(work, tree)
	if err != nil {
		return err
	}
	for _, w := range ws {
		if r.stopped != "" {
			break
		}
		if time.Now().After(r.deadline) {
			r.stopped = "budget"
			break
		}
		if err := r.cell(w); err != nil {
			return fmt.Errorf("%s %s: %w", w.name, w.size, err)
		}
	}
	return r.write(out)
}

// cell runs one workload: the load gate, the warm-ups, then the timed runs
// interleaved A, B, A, B so drift falls on both binaries.
func (r *runner) cell(w workload) error {
	load, ok := r.gate()
	if !ok {
		r.stopped = fmt.Sprintf("load %.2f at %s %s", load, w.name, w.size)
		return nil
	}
	for i := -r.warm; i < r.runs; i++ {
		for bin := 0; bin < 2; bin++ {
			if err := w.prepare(bin); err != nil {
				return err
			}
			s, err := r.time(w, bin)
			if err != nil {
				return err
			}
			if i >= 0 {
				s.Run, s.Load = i, load
				r.samples = append(r.samples, s)
			}
		}
	}
	return nil
}

func (r *runner) time(w workload, bin int) (sample, error) {
	c := exec.Command(r.bins[bin], w.args(bin)...)
	c.Dir = w.dir
	c.Env = append(os.Environ(), "XDG_STATE_HOME="+w.state(bin))
	var stdout counter
	c.Stdout, c.Stderr = &stdout, nil
	start := time.Now()
	err := c.Run()
	wall := time.Since(start)
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		return sample{}, err
	}
	s := sample{Workload: w.name, Size: w.size, Binary: []string{"A", "B"}[bin], WallNS: wall.Nanoseconds(), Stdout: stdout.n, Exit: code}
	if ps := c.ProcessState; ps != nil {
		s.CPUNS = (ps.UserTime() + ps.SystemTime()).Nanoseconds()
		s.MaxRSS = maxRSS(ps)
	}
	return s, nil
}

// gate waits, bounded, for the 1-minute load average to fall below maxLoad.
func (r *runner) gate() (float64, bool) {
	until := time.Now().Add(r.loadWait)
	for {
		l := loadAvg()
		if l < r.maxLoad {
			return l, true
		}
		if time.Now().After(until) {
			return l, false
		}
		time.Sleep(15 * time.Second)
	}
}

func loadAvg() float64 {
	b, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 99
	}
	f := strings.Fields(strings.Trim(strings.TrimSpace(string(b)), "{}"))
	if len(f) == 0 {
		return 99
	}
	v, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 99
	}
	return v
}

type counter struct{ n int }

func (c *counter) Write(p []byte) (int, error) { c.n += len(p); return len(p), nil }

// write keeps every sample in raw/samples.jsonl and a summary per cell:
// p25, p50, p75, p90 and max of wall time, the median CPU and the largest RSS.
func (r *runner) write(out string) error {
	if err := os.MkdirAll(filepath.Join(out, "raw"), 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(out, "raw", "samples.jsonl"))
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, s := range r.samples {
		if err := enc.Encode(s); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	type key struct{ w, size, bin string }
	cells := map[key][]sample{}
	var order []key
	for _, s := range r.samples {
		k := key{s.Workload, s.Size, s.Binary}
		if _, ok := cells[k]; !ok {
			order = append(order, k)
		}
		cells[k] = append(cells[k], s)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "| workload | size | binary | n | failed | p25 ms | p50 ms | p75 ms | p90 ms | max ms | cpu p50 ms | max rss MB | stdout B |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, k := range order {
		ss := cells[k]
		wall, cpu := make([]float64, 0, len(ss)), make([]float64, 0, len(ss))
		var rss int64
		failed := 0
		for _, s := range ss {
			wall = append(wall, float64(s.WallNS)/1e6)
			cpu = append(cpu, float64(s.CPUNS)/1e6)
			rss = max(rss, s.MaxRSS)
			if s.Exit != 0 {
				failed++
			}
		}
		sort.Float64s(wall)
		sort.Float64s(cpu)
		fmt.Fprintf(&b, "| %s | %s | %s | %d | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.1f | %d |\n",
			k.w, k.size, k.bin, len(ss), failed, pct(wall, 25), pct(wall, 50), pct(wall, 75), pct(wall, 90), wall[len(wall)-1], pct(cpu, 50), float64(rss)/1e6, ss[0].Stdout)
	}
	if r.stopped != "" {
		fmt.Fprintf(&b, "\nstopped: %s\n", r.stopped)
	}
	return os.WriteFile(filepath.Join(out, "summary.md"), []byte(b.String()), 0o644)
}

// pct is the nearest-rank percentile of sorted values.
func pct(v []float64, p int) float64 {
	i := (p*len(v)+99)/100 - 1
	return v[max(0, min(i, len(v)-1))]
}

// lines is n generated lines, the same for every seed and n.
func lines(n int, rng *rand.Rand) []string {
	words := []string{"alpha", "beta", "gamma", "delta", "needle", "omega"}
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("line %07d %s %d", i+1, words[rng.IntN(len(words)-1)], rng.IntN(1000))
	}
	return out
}

func writeLines(path string, ls []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(ls, "\n")+"\n"), 0o644)
}

// workloads builds every fixture under work and returns the cells in the
// order the pre-registration lists them.
func workloads(work, tree string) ([]workload, error) {
	rng := rand.New(rand.NewPCG(seed, seed))
	var ws []workload
	stateOf := func(dir string) func(int) string {
		return func(bin int) string { return filepath.Join(dir, fmt.Sprintf("state-%d", bin)) }
	}
	none := func(int) error { return nil }

	// R1: one 40-line range at the middle of a file of 100, 10,000 and 1,000,000 lines.
	for _, n := range []int{100, 10_000, 1_000_000} {
		dir := filepath.Join(work, fmt.Sprintf("r1-%d", n))
		if err := writeLines(filepath.Join(dir, "f.txt"), lines(n, rng)); err != nil {
			return nil, err
		}
		from := max(1, n/2-20)
		spec := fmt.Sprintf("f.txt:%d-%d", from, from+39)
		ws = append(ws, workload{name: "R1", size: strconv.Itoa(n), dir: dir, prepare: none, state: stateOf(dir),
			args: func(int) []string { return []string{"read", spec} }})
	}
	// R2: W specs across W files of 1,000 lines.
	for _, w := range []int{1, 10, 100} {
		dir := filepath.Join(work, fmt.Sprintf("r2-%d", w))
		var specs []string
		for i := 0; i < w; i++ {
			name := fmt.Sprintf("f%03d.txt", i)
			if err := writeLines(filepath.Join(dir, name), lines(1000, rng)); err != nil {
				return nil, err
			}
			specs = append(specs, name+":480-519")
		}
		ws = append(ws, workload{name: "R2", size: strconv.Itoa(w), dir: dir, prepare: none, state: stateOf(dir),
			args: func(int) []string { return append([]string{"read"}, specs...) }})
	}
	// R3: --grep over this repository at the tagged tree, and over a synthetic
	// 3,000-file tree of BenchmarkWalkDeep131's shape.
	ws = append(ws, workload{name: "R3", size: "repo", dir: tree, prepare: none, state: stateOf(work + "/r3-repo"),
		args: func(int) []string { return []string{"read", "--grep", "func Run", "--count", "."} }})
	deep := filepath.Join(work, "r3-deep")
	for br := 0; br < 10; br++ {
		d := deep
		for lvl := 0; lvl < 10; lvl++ {
			d = filepath.Join(d, fmt.Sprintf("l%d_%d", lvl, br))
		}
		for f := 0; f < 300; f++ {
			if err := writeLines(filepath.Join(d, fmt.Sprintf("f%d.txt", f)), []string{"line one", "not here"}); err != nil {
				return nil, err
			}
		}
	}
	ws = append(ws, workload{name: "R3", size: "deep3000", dir: deep, prepare: none, state: stateOf(work + "/r3-deep-state"),
		args: func(int) []string { return []string{"read", "--grep", "needle", "."} }})

	// W1: a plan of H one-line hunks over F files of 1,000 lines, each file put
	// back and read (untimed, --quiet) before every timed write.
	for _, hf := range [][2]int{{1, 1}, {50, 10}, {500, 50}} {
		h, f := hf[0], hf[1]
		dir := filepath.Join(work, fmt.Sprintf("w1-%d-%d", h, f))
		orig := map[string][]string{}
		var names []string
		var plan strings.Builder
		for i := 0; i < f; i++ {
			name := fmt.Sprintf("f%03d.txt", i)
			orig[name] = lines(1000, rng)
			names = append(names, name)
		}
		for j := 0; j < h; j++ {
			name := names[j%f]
			line := 1 + (j/f)*(1000/((h+f-1)/f))
			fmt.Fprintf(&plan, "@@ %s %d replace\nedited %d\n", name, line, j)
		}
		planPath := filepath.Join(work, fmt.Sprintf("w1-%d-%d.plan", h, f))
		if err := os.WriteFile(planPath, []byte(plan.String()), 0o644); err != nil {
			return nil, err
		}
		state := stateOf(dir)
		prepare := func(bin int) error {
			for name, ls := range orig {
				if err := writeLines(filepath.Join(dir, name), ls); err != nil {
					return err
				}
			}
			c := exec.Command(binOf(bin), append([]string{"read", "--quiet"}, names...)...)
			c.Dir, c.Env = dir, append(os.Environ(), "XDG_STATE_HOME="+state(bin))
			// A pipe, not the null device: a read whose answer goes to the null
			// device licenses nothing (ADR-133).
			c.Stdout = io.Discard
			return c.Run()
		}
		ws = append(ws, workload{name: "W1", size: fmt.Sprintf("%dh/%df", h, f), dir: dir, prepare: prepare, state: state,
			args: func(int) []string { return []string{"write", "--no-check", planPath} }})
	}
	// W2: the same edit at 10 and 1,000 sites through --grep --sub.
	for _, sites := range []int{10, 1000} {
		dir := filepath.Join(work, fmt.Sprintf("w2-%d", sites))
		ls := lines(sites*10, rng)
		for i := 0; i < sites; i++ {
			ls[i*10] = fmt.Sprintf("site OldName %d", i)
		}
		prepare := func(int) error { return writeLines(filepath.Join(dir, "f.txt"), ls) }
		if err := prepare(0); err != nil {
			return nil, err
		}
		ws = append(ws, workload{name: "W2", size: strconv.Itoa(sites), dir: dir, prepare: prepare, state: stateOf(dir),
			args: func(int) []string {
				return []string{"write", "--grep", `\bOldName\b`, "--sub", "NewName", "--no-check", "."}
			}})
	}
	// L1: R1 at 10,000 lines with a ledger of E entries, made by reading E
	// other files once before the cell.
	for _, e := range []int{0, 1000, 10_000} {
		dir := filepath.Join(work, fmt.Sprintf("l1-%d", e))
		if err := writeLines(filepath.Join(dir, "f.txt"), lines(10_000, rng)); err != nil {
			return nil, err
		}
		var list strings.Builder
		for i := 0; i < e; i++ {
			name := fmt.Sprintf("e/%05d.txt", i)
			if err := writeLines(filepath.Join(dir, name), []string{"x"}); err != nil {
				return nil, err
			}
			list.WriteString(name + "\n")
		}
		listPath := filepath.Join(work, fmt.Sprintf("l1-%d.list", e))
		if err := os.WriteFile(listPath, []byte(list.String()), 0o644); err != nil {
			return nil, err
		}
		state := stateOf(dir)
		filled := [2]bool{}
		prepare := func(bin int) error {
			if filled[bin] || e == 0 {
				return nil
			}
			filled[bin] = true
			c := exec.Command(binOf(bin), "read", "--quiet", "--files-from", listPath)
			c.Dir, c.Env = dir, append(os.Environ(), "XDG_STATE_HOME="+state(bin))
			c.Stdout = io.Discard
			return c.Run()
		}
		ws = append(ws, workload{name: "L1", size: strconv.Itoa(e), dir: dir, prepare: prepare, state: state,
			args: func(int) []string { return []string{"read", "f.txt:4981-5020"} }})
	}
	// C1: first use (an empty state home before every run) and warm use.
	c1 := filepath.Join(work, "c1")
	if err := writeLines(filepath.Join(c1, "f.txt"), lines(10_000, rng)); err != nil {
		return nil, err
	}
	cold := func(bin int) string { return filepath.Join(work, fmt.Sprintf("c1-cold-%d", bin)) }
	ws = append(ws, workload{name: "C1", size: "first", dir: c1, state: cold,
		prepare: func(bin int) error { return os.RemoveAll(cold(bin)) },
		args:    func(int) []string { return []string{"read", "f.txt:4981-5020"} }})
	ws = append(ws, workload{name: "C1", size: "warm", dir: c1, prepare: none, state: stateOf(c1),
		args: func(int) []string { return []string{"read", "f.txt:4981-5020"} }})
	return ws, nil
}

// binOf is set by main before the fixtures are built, so a prepare step reads
// with the binary the timed run will write with.
var binOf = func(int) string { return "mrw" }
