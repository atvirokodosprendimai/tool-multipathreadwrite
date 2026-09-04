// Command curve is the benchmark harness ADR-020 describes. It is a second
// binary on purpose: mrw's own surface is for callers, and this is for whoever
// is measuring mrw.
//
//	curve generate -out DIR -bytes N [-position early|middle|late] [-distractors K] [-seed S]
//	curve score    -cell DIR -result FILE
//	curve tally    SCORE.json ...
//
// generate writes a trial; a client authors a plan from DIR/manifest.json and
// DIR/served.txt; score applies that plan with the real engine and says which
// line changed; tally turns scores into cells with their intervals.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/curve"
)

const usageText = `usage:
  curve generate -out DIR -bytes N [-position early|middle|late] [-distractors K] [-seed S]
  curve score    -cell DIR -result FILE
  curve tally    SCORE.json ...

generate writes DIR/manifest.json (what the client sees), DIR/served.txt (what
mrw would serve, measured), DIR/answer.json (the planted line; never show it to
the client) and DIR/state (export as XDG_STATE_HOME while the client runs mrw).

score reads a result {"trial_id","served_bytes","plan"} and refuses one that
does not echo the trial; the verdict is hit, miss, refused_parse or refused_apply.

tally groups scores by served bytes and position and reports each cell as a
rate with a 95% Wilson interval; refusals are counted beside the cell, not in it.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "generate":
		err = generate(os.Args[2:])
	case "score":
		err = score(os.Args[2:])
	case "tally":
		err = tally(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usageText)
		return
	default:
		fmt.Fprintf(os.Stderr, "curve: unknown verb %q\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "curve:", err)
		os.Exit(2)
	}
}

func generate(args []string) error {
	fl := flag.NewFlagSet("generate", flag.ContinueOnError)
	out := fl.String("out", "", "directory to write the trial into (required)")
	size := fl.Int("bytes", 0, "served bytes to reach — the independent variable (required)")
	pos := fl.String("position", string(curve.Middle), "target stratum: early, middle or late")
	k := fl.Int("distractors", 4, "near-identical distractor blocks; hold constant across size cells")
	seed := fl.Int64("seed", 1, "the same params and seed regenerate the same trial")
	if err := fl.Parse(args); err != nil {
		return err
	}
	if *out == "" || *size <= 0 {
		return fmt.Errorf("generate needs -out and a positive -bytes")
	}
	m, err := curve.Generate(*out, curve.Params{ServedBytes: *size, Position: curve.Position(*pos), Distractors: *k, Seed: *seed})
	if err != nil {
		return err
	}
	return emit(m)
}

func score(args []string) error {
	fl := flag.NewFlagSet("score", flag.ContinueOnError)
	cell := fl.String("cell", "", "the directory generate wrote (required)")
	result := fl.String("result", "", `JSON file {"trial_id","served_bytes","plan"} (required)`)
	if err := fl.Parse(args); err != nil {
		return err
	}
	if *cell == "" || *result == "" {
		return fmt.Errorf("score needs -cell and -result")
	}
	b, err := os.ReadFile(*result)
	if err != nil {
		return err
	}
	var r curve.Result
	if err := json.Unmarshal(b, &r); err != nil {
		return fmt.Errorf("%s: %w", *result, err)
	}
	s, err := curve.ScoreTrial(*cell, r)
	if err != nil {
		return err
	}
	return emit(s)
}

func tally(paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("tally needs at least one score file")
	}
	scores := make([]curve.Score, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var s curve.Score
		if err := json.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		scores = append(scores, s)
	}
	cells, err := curve.Tally(scores)
	if err != nil {
		return err
	}
	return emit(cells)
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
