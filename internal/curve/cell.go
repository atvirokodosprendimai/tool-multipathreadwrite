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

// Params fixes one trial. ServedBytes is the independent variable. Distractors
// is held constant across size cells by the caller, so that "more context"
// stays separable from "more candidates" — the second variable the
// pre-registration found hiding inside the first.
type Params struct {
	ServedBytes int
	Position    Position
	Distractors int
	Seed        int64
}

// Manifest is what the client receives. It carries no ground truth.
type Manifest struct {
	TrialID     string   `json:"trial_id"`
	ServedBytes int      `json:"served_bytes"`
	Position    Position `json:"position"`
	Distractors int      `json:"distractors"`
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
// never handed to the client.
type Answer struct {
	TrialID string `json:"trial_id"`
	Line    int    `json:"line"`
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
		served, err = serve(tree)
		if err != nil {
			return Manifest{}, err
		}
		if len(served) >= p.ServedBytes {
			break
		}
		if step >= fitSteps {
			return Manifest{}, fmt.Errorf("could not reach %d served bytes in %d steps (at %d)", p.ServedBytes, fitSteps, len(served))
		}
		pad += (p.ServedBytes-len(served))/padBytes + 1
	}

	m := Manifest{
		TrialID:     trialID(p),
		ServedBytes: len(served),
		Position:    p.Position,
		Distractors: p.Distractors,
		Tree:        tree,
		File:        targetFile,
		StateHome:   state,
		Instruction: instruction(names[target]),
	}
	a := Answer{TrialID: m.TrialID, Line: line}
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
	if p.Distractors < 1 {
		return fmt.Errorf("a trial needs at least one distractor, got %d", p.Distractors)
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
func trialID(p Params) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("curve/v1|%d|%s|%d|%d", p.ServedBytes, p.Position, p.Distractors, p.Seed)))
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
		emit("retries = 3")
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
func serve(tree string) ([]byte, error) {
	var buf bytes.Buffer
	_, problems := read.Run(&buf, tree, []read.Spec{{Path: targetFile, Raw: targetFile}}, read.Options{Numbers: true})
	if problems != 0 {
		return nil, fmt.Errorf("mrw could not serve %s: %d problem(s)", targetFile, problems)
	}
	return buf.Bytes(), nil
}

// instruction is identical across cells for one target name: only the served
// size and the address the client must produce vary with the cell.
func instruction(name string) string {
	return fmt.Sprintf("In %s, service %q has the wrong timeout. Change its `%s` line to `%s`. "+
		"Author one mrw write plan containing exactly one `replace` hunk that addresses that line and nothing else; "+
		"every other service keeps its timeout. Reply with the plan text and nothing else.",
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
