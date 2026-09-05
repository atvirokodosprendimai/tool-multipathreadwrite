// Package curve is the benchmark harness ADR-020 describes. It generates a
// trial whose correct answer is PLANTED, hands a client only what mrw itself
// would serve, and scores the plan the client authored by applying it with the
// real engine and reading which line changed.
//
// It calls no model and opens no network connection: a manifest goes out, a
// result comes back, and the scorer refuses a result that does not echo the
// trial it was generated for. That is what lets one harness measure any client.
package curve

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
)

// Position is the stratum a trial's target sits in within the served window.
// It is a stratum and not a nuisance: position is the best-documented effect
// in the long-context literature, and randomising it away would average a
// real effect into nothing.
type Position string

// The three strata. Early puts the target in the first block, Late in the
// last, Middle halfway; the distractors fill the remaining block slots.
const (
	Early  Position = "early"
	Middle Position = "middle"
	Late   Position = "late"
)

// Selector says how the instruction identifies the target block. It is a
// stratum of DIFFICULTY, not a nuisance: the first reading returned 42 correct
// addresses in 45 trials against ByName and flat across a hundredfold change in
// served bytes, which is a ceiling rather than a curve, and a curve cannot bend
// against a task nobody fails.
type Selector string

// The two selectors. ByName is the zero value so every caller written before it
// existed keeps the fixture it had — which is what lets the two readings be
// compared rather than merely both reported.
const (
	// ByName gives the client the target's unique service name. It measures
	// whether a caller can MATCH a string.
	ByName Selector = ""
	// ByOddRetries names no service and describes a relation instead: one
	// block's retry budget differs from every other. It defeats a lookup by
	// unique name and forces a comparison ACROSS blocks. It does not prove the
	// caller read everything: `retries` appears in every block, so one search
	// returns them all. What it removes is the single-match shortcut, which is
	// what put the first reading at a ceiling.
	ByOddRetries Selector = "odd-retries"
)

// Params fixes one trial. ServedBytes is the independent variable. Distractors
// is held constant across size cells by the caller, so that "more context"
// stays separable from "more candidates" — the second variable the
// pre-registration found hiding inside the first.
type Params struct {
	ServedBytes int
	Position    Position
	Distractors int
	Seed        int64
	// Selector is how the instruction points at the target. The zero value is
	// ByName, the fixture T1 shipped.
	Selector Selector
	// ServeFrom is the first line the client is served; zero means the whole
	// file. Every cell before T4 served from line 1, where a row count in the
	// rendering — whose first two rows carry no line number — and the line
	// number plus two coincide. A window from line N separates them by N-1.
	ServeFrom int
}

// Manifest is what the client receives. It carries NO ground truth: not the
// planted line, and not the stratum or distractor count either — with those two
// the target's block index follows by arithmetic (early is the first block,
// late the last), so a client could localise by counting rather than by
// reading, which is exactly the confound the fixture exists to remove.
type Manifest struct {
	TrialID string `json:"trial_id"`
	// Cell is the REQUESTED size, and it is what a tally groups by. Padding
	// overshoots by a seed-dependent amount, so grouping by the measured size
	// would give nearly every trial a cell of its own.
	Cell int `json:"cell"`
	// ServedBytes is what this trial actually served. It is echoed back with
	// the result so a result from another trial can be refused.
	ServedBytes int `json:"served_bytes"`
	// Tree is the absolute fixture root the client's mrw should be pointed at.
	Tree string `json:"tree"`
	// File is the target's path relative to Tree — the path a plan names.
	File string `json:"file"`
	// StateHome is an empty, absolute directory the runner exports as
	// XDG_STATE_HOME while driving mrw, so no trial inherits another's ledger.
	StateHome   string `json:"state_home"`
	Instruction string `json:"instruction"`
}

// Answer is the planted ground truth. It is written beside the manifest and
// never handed to the client. The stratum and the distractor count live here
// rather than in the manifest for the reason given above: together they name
// the target's block.
type Answer struct {
	TrialID     string   `json:"trial_id"`
	Line        int      `json:"line"`
	Position    Position `json:"position"`
	Distractors int      `json:"distractors"`
}

const (
	manifestName = "manifest.json"
	answerName   = "answer.json"
	servedName   = "served.txt"
	treeName     = "tree"
	stateName    = "state"
	targetFile   = "services.conf"
	targetLine   = "timeout = 30"
	wantLine     = "timeout = 45"
	// commonRetries is what every block carries; oddRetries is what exactly one
	// carries under ByOddRetries. The values are arbitrary and only their
	// INEQUALITY is load-bearing, which is why the relation is what the
	// instruction describes.
	commonRetries = "retries = 3"
	oddRetries    = "retries = 5"
	// padBytes is a rough size of one rendered padding line, used only to
	// choose how many to add per fitting step. The loop measures the truth.
	padBytes = 70
	fitSteps = 64
)

// Generate writes one trial into dir: the fixture tree, the served rendering,
// the manifest, the answer, and an empty state directory for the client's mrw.
// The same Params regenerate the same trial.
func Generate(dir string, p Params) (Manifest, error) {
	if err := p.check(); err != nil {
		return Manifest{}, err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Manifest{}, err
	}
	tree := filepath.Join(abs, treeName)
	state := filepath.Join(abs, stateName)
	for _, d := range []string{tree, state} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return Manifest{}, err
		}
	}
	// A state directory that already holds something is refused rather than
	// reused: the runner exports it as XDG_STATE_HOME, so an inherited ledger
	// would license reads this trial never made. Regenerating into a used -out
	// is the way that happens, and it looks like success.
	if entries, err := os.ReadDir(state); err != nil {
		return Manifest{}, err
	} else if len(entries) > 0 {
		return Manifest{}, fmt.Errorf("state directory %s is not empty (%d entries); generate into a fresh -out so no trial inherits another's ledger", state, len(entries))
	}
	names := serviceNames(p)
	target := targetIndex(p)

	// Fit the padding to the size cell by measuring what read.Run renders:
	// the header and the number prefix are part of what a caller pays for,
	// and only the renderer knows their width.
	var served []byte
	var line int
	pad := 0
	for step := 0; ; step++ {
		var content string
		content, line = render(p, names, target, pad)
		if err := os.WriteFile(filepath.Join(tree, targetFile), []byte(content), 0o600); err != nil {
			return Manifest{}, err
		}
		// A window past the end of a still-short file is not a serve failure,
		// it is a fixture that has not grown enough yet: serve nothing and let
		// the loop add padding until the window exists.
		if p.ServeFrom > strings.Count(content, "\n") {
			served = nil
		} else if served, err = serve(tree, p.ServeFrom); err != nil {
			return Manifest{}, err
		}
		if len(served) >= p.ServedBytes {
			break
		}
		if step >= fitSteps {
			if p.ServeFrom > 0 && served == nil {
				// The window never came into existence: the file would need
				// more lines than this size cell produces. Say that, rather
				// than the byte count, because the byte count is a symptom.
				return Manifest{}, fmt.Errorf("a window served from line %d excludes the target: the fixture has only %d lines at %d served bytes, so the window never exists", p.ServeFrom, strings.Count(content, "\n"), p.ServedBytes)
			}
			return Manifest{}, fmt.Errorf("could not reach %d served bytes in %d steps (at %d)", p.ServedBytes, fitSteps, len(served))
		}
		pad += (p.ServedBytes-len(served))/padBytes + 1
	}

	m := Manifest{
		TrialID:     trialID(p),
		Cell:        p.ServedBytes,
		ServedBytes: len(served),
		Tree:        tree,
		File:        targetFile,
		StateHome:   state,
		Instruction: instruction(p, names[target]),
	}
	// A window that excludes the target is unanswerable rather than hard, and
	// it can only be checked here, once the fit has settled the target's line.
	if p.ServeFrom > line {
		return Manifest{}, fmt.Errorf("a window served from line %d excludes the target at line %d; the client would be served no answer", p.ServeFrom, line)
	}
	a := Answer{TrialID: m.TrialID, Line: line, Position: p.Position, Distractors: p.Distractors}
	if err := writeJSON(filepath.Join(abs, manifestName), m); err != nil {
		return Manifest{}, err
	}
	if err := writeJSON(filepath.Join(abs, answerName), a); err != nil {
		return Manifest{}, err
	}
	if err := os.WriteFile(filepath.Join(abs, servedName), served, 0o600); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (p Params) check() error {
	switch p.Position {
	case Early, Middle, Late:
	default:
		return fmt.Errorf("position %q is not early, middle or late", p.Position)
	}
	if p.ServedBytes <= 0 {
		return fmt.Errorf("served bytes must be positive, got %d", p.ServedBytes)
	}
	// Two is the minimum that makes the three strata distinct: with one
	// distractor there are two blocks, and early and middle both name the
	// first — two advertised strata that are the same trial. It is also what
	// makes ByOddRetries meaningful: with two blocks each differs from the
	// other, so "the one that differs from every other" picks out nothing.
	if p.Distractors < 2 {
		return fmt.Errorf("a trial needs at least two distractors so the strata differ, got %d", p.Distractors)
	}
	if p.ServeFrom < 0 {
		return fmt.Errorf("serve-from must be a line number or zero for the whole file, got %d", p.ServeFrom)
	}
	switch p.Selector {
	case ByName, ByOddRetries:
	default:
		return fmt.Errorf("selector %q is not the named one or %q", p.Selector, ByOddRetries)
	}
	return nil
}

// Load reads a generated trial back. The scorer uses it; a client reads only
// manifest.json and must never be given answer.json.
func Load(dir string) (Manifest, Answer, error) {
	var m Manifest
	var a Answer
	if err := readJSON(filepath.Join(dir, manifestName), &m); err != nil {
		return m, a, err
	}
	if err := readJSON(filepath.Join(dir, answerName), &a); err != nil {
		return m, a, err
	}
	return m, a, nil
}

// trialID is a stable name for one Params, so a result can say which trial it
// answers and the scorer can refuse one that answers a different trial.
//
// The selector is appended ONLY when it is not the zero value. That is not
// cosmetic: the ids of every cell generated before T2 are recorded in
// docs/curve/reading-02-scores, and a formula that appended "|" for ByName too
// would give the same parameters a different id — so regenerating a historical
// cell would refuse the raw result collected against it. Found in review; the
// first version of this function did exactly that.
func trialID(p Params) string {
	s := fmt.Sprintf("curve/v1|%d|%s|%d|%d", p.ServedBytes, p.Position, p.Distractors, p.Seed)
	if p.Selector != ByName {
		s += "|" + string(p.Selector)
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}

func targetIndex(p Params) int {
	switch p.Position {
	case Early:
		return 0
	case Late:
		return p.Distractors
	default:
		return p.Distractors / 2
	}
}

// serviceNames returns Distractors+1 distinct names. The name is the ONE detail
// the instruction gives, so the target is findable only by reading which block
// a line belongs to. The threat to validity the pre-registration names first is
// a target that is a unique string, because that measures matching, not reading.
func serviceNames(p Params) []string {
	rng := rand.New(rand.NewSource(p.Seed))
	taken := map[string]bool{}
	var names []string
	for len(names) < p.Distractors+1 {
		b := make([]byte, 5)
		for i := range b {
			b[i] = 'a' + byte(rng.Intn(26))
		}
		n := "svc-" + string(b)
		if !taken[n] {
			taken[n] = true
			names = append(names, n)
		}
	}
	return names
}

// words feed the inert padding. Comment prose is obviously not a candidate for
// "the timeout line of a service", which is what makes it inert rather than a
// distractor.
var words = strings.Fields(`the registry keeps one block per service and each block carries
its own retry budget and timeout so an operator can tune them apart without
touching the others while the defaults stay visible here for review`)

// render lays out Distractors+1 identical service blocks with the target at
// its stratum and pad comment lines spread evenly through the gaps around
// them. It returns the content and the 1-based line of the target's timeout.
// The padding text is seeded independently of the names, so growing pad
// between fitting steps extends the padding rather than reshuffling it.
func render(p Params, names []string, target, pad int) (string, int) {
	rng := rand.New(rand.NewSource(p.Seed ^ 0x5eed))
	common, odd := retryPair(p.Seed)
	gaps := len(names) + 1
	var b strings.Builder
	line := 0
	emit := func(s string) {
		b.WriteString(s)
		b.WriteByte('\n')
		line++
	}
	padding := func(g int) {
		n := pad / gaps
		if g < pad%gaps {
			n++
		}
		for i := 0; i < n; i++ {
			emit("# " + prose(rng))
		}
	}
	emit("# service registry — every service has the same shape")
	targetLineNo := 0
	for i, name := range names {
		padding(i)
		emit("[service " + name + "]")
		if i == target && p.Selector == ByOddRetries {
			emit(odd)
		} else if p.Selector == ByOddRetries {
			emit(common)
		} else {
			emit(commonRetries)
		}
		emit(targetLine)
		if i == target {
			targetLineNo = line
		}
		emit("")
	}
	padding(len(names))
	return b.String(), targetLineNo
}

func prose(rng *rand.Rand) string {
	n := 6 + rng.Intn(5)
	ws := make([]string, n)
	for i := range ws {
		ws[i] = words[rng.Intn(len(words))]
	}
	return strings.Join(ws, " ")
}

// serve renders the target file exactly as `mrw read` would — header, line
// numbers and all — so served bytes is what a caller actually pays for.
func serve(tree string, from int) ([]byte, error) {
	var buf bytes.Buffer
	spec := read.Spec{Path: targetFile, Raw: targetFile}
	if from > 0 {
		// Start with End 0 is "from this line to the last": the range the
		// engine already offers, so the fixture uses it and never touches
		// internal/read. Raw is what the header prints for the caller.
		spec.Ranges = []read.Range{{Start: from, Text: fmt.Sprintf("%d-", from)}}
		spec.Raw = fmt.Sprintf("%s:%d-", targetFile, from)
	}
	_, problems := read.Run(&buf, tree, []read.Spec{spec}, read.Options{Numbers: true})
	if problems != 0 {
		return nil, fmt.Errorf("mrw could not serve %s: %d problem(s)", targetFile, problems)
	}
	return buf.Bytes(), nil
}

// instruction is identical across cells for one selector: only the served size
// and the address the client must produce vary with the cell. Under ByOddRetries
// it names NO service, because a name is a string to match and matching is what
// the other selector already measures.
func instruction(p Params, name string) string {
	const tail = "Author one mrw write plan containing exactly one `replace` hunk that addresses that line and nothing else; " +
		"every other service keeps its timeout. Reply with the plan text and nothing else."
	if p.Selector == ByOddRetries {
		return fmt.Sprintf("In %s, exactly one service has a retry budget that differs from every other service's. "+
			"That service has the wrong timeout: change ITS `%s` line to `%s`. "+
			"The file does not say which service it is and neither does this instruction. "+tail,
			targetFile, targetLine, wantLine)
	}
	return fmt.Sprintf("In %s, service %q has the wrong timeout. Change its `%s` line to `%s`. "+tail,
		targetFile, name, targetLine, wantLine)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// retryPair draws the common and the odd retry budget for one trial from its
// seed, and guarantees they differ.
//
// They are DRAWN rather than fixed because a constant odd value is a signature:
// T2 removed the target's unique name and left "retries = 5" at every seed, so a
// client that had seen one relational cell could search for it in every other at
// a cost independent of served size — the single-match shortcut the selector
// exists to remove, surviving inside its own fixture. The common value varies
// too, because a fixed common value identifies the odd block by elimination.
//
// It derives from the seed and not from a generator that advances, because
// render is called repeatedly while the padding is fitted to the size cell and
// the budgets must not move between steps.
//
// This removes the CONSTANT, not the alphabet: the values are small integers and
// a determined client could enumerate them. ADR-020-T3 says so rather than
// claiming more.
func retryPair(seed int64) (common, odd string) {
	const alternatives = 7 // budgets 2..8
	rng := rand.New(rand.NewSource(seed ^ 0x0ddba11))
	c := rng.Intn(alternatives)
	// The odd budget is drawn as an index into the alternatives EXCLUDING the
	// common one, then shifted past it. Inequality and termination are
	// structural rather than probabilistic: a rejection loop would be correct
	// too, but deleting it leaves a generator that produces a cell with no odd
	// block at all — and only at some seeds, so a test that exercises a few
	// would still pass. Found by review of PR #94: seeds 13, 18, 23 and 24
	// collide on the first draw, and none of them was exercised.
	o := rng.Intn(alternatives - 1)
	if o >= c {
		o++
	}
	return fmt.Sprintf("retries = %d", c+2), fmt.Sprintf("retries = %d", o+2)
}
