// Package seen is mrw's record of what it last observed each file to be.
//
// It exists to make one guarantee: mrw will not edit a file whose current
// contents it has never seen. That is the property the harness's own Write tool
// has — it refuses to overwrite a file you have not Read — and the reason is
// that an edit is written against a picture of the file, so a picture that is
// out of date silently writes the wrong thing into the wrong place.
//
// A range edit needs it more than a whole-file write does, not less: an address
// like "replace 42-58" means nothing without the version of the file those line
// numbers were counted in.
//
// So the record is not "mrw hashed this file" but "mrw SERVED these lines of
// it". A read that rendered no content — `--stat` — observes nothing, and a
// read of lines 1-5 licenses an edit to lines 1-5 and not to line 40. Whoever
// counted the line numbers is the caller, and the caller has only seen what was
// printed to them.
//
// The ledger is updated on READ and on WRITE, recording what the file is after
// each. So a chain of edits flows without re-reading — mrw already knows what it
// just produced, and knows all of it — while anything that changed the file
// BEHIND mrw's back leaves the recorded sha and the real one disagreeing, and
// the next write is refused.
package seen

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// Name is the ledger's filename inside the state directory. The DIRECTORY is
// resolved by internal/state and is deliberately outside the working tree —
// mrw used to write this beside your source, where nothing ignored it and one
// copy was committed by accident. See ADR-004.
const Name = "seen"

// Observation is what mrw last saw one file to be: the whole-file SHA-256, and
// the line spans it actually served to the caller.
//
// A nil Spans means the WHOLE file — what a plain `mrw read path` observes, and
// what a write observes about the file it just produced. An empty-but-non-nil
// Spans means the file was hashed and nothing was shown, which licenses no
// edit at all.
type Observation struct {
	SHA   string
	Spans [][2]int
}

// Whole reports whether this observation covers the entire file.
func (o Observation) Whole() bool { return o.Spans == nil }

// Covers reports whether every line in [start,end] was served. A whole-file
// observation covers everything, and an empty span set covers nothing.
func (o Observation) Covers(start, end int) bool {
	if o.Whole() {
		return true
	}
	// A range whose end precedes its start iterates zero times below and is
	// covered trivially. That is left to the loop rather than special-cased,
	// so nothing here has to claim which caller produces such a range.
	for line := start; line <= end; line++ {
		if !o.coversLine(line) {
			return false
		}
	}
	return true
}

func (o Observation) coversLine(line int) bool {
	for _, s := range o.Spans {
		if line >= s[0] && line <= s[1] {
			return true
		}
	}
	return false
}

// Served renders the spans the way a diagnostic should quote them.
func (o Observation) Served() string {
	if o.Whole() {
		return "the whole file"
	}
	if len(o.Spans) == 0 {
		return "no lines"
	}
	parts := make([]string, 0, len(o.Spans))
	for _, s := range o.Spans {
		if s[0] == s[1] {
			parts = append(parts, strconv.Itoa(s[0]))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d-%d", s[0], s[1]))
	}
	return "lines " + strings.Join(parts, ",")
}

// Ledger maps a path to the observation mrw last made of it.
type Ledger map[string]Observation

// Load reads the ledger. A missing file is an empty ledger, not an error: no
// observations yet is the normal starting state.
// header is the ledger's first line. Its VERSION is bumped whenever an older
// file's contents can no longer be trusted — not merely when the format
// changes — because the whole point is to discard a ledger whose entries mean
// something different from what they say. v2 exists because of the
// served-nothing bug described in Load.
const header = "#mrw-seen v2"

func Load(root string) (Ledger, error) {
	l := Ledger{}
	path, err := ReadPath(root)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return l, sc.Err()
	}
	// A ledger without the current header is DISCARDED, not parsed.
	//
	// Up to v0.0.11 a ranged read that served nothing recorded the whole file,
	// and on disk that entry is byte-identical to a legitimate whole-file
	// read: `<sha>  -  <path>` either way. No parse-time rule can tell a
	// poisoned entry from a good one, so the fixed binary would honour the
	// poisoned licence and silently permit an edit to lines the caller was
	// never shown — for the people who upgraded, which is the moment they
	// would assume they were safe.
	//
	// Dropping it is cheap and cannot lose data: the ledger is a cache of
	// observations, so the worst it costs is one re-read, and the failure
	// direction is a refusal rather than a wrong write.
	if sc.Text() != header {
		// EMPTY and nil, not an error. A stale ledger is a recoverable
		// condition: Record calls Load first, so returning an error here would
		// stop save from ever running, the v2 header would never be written,
		// and every later run would hit the same stale file. The ledger could
		// never heal itself. Staleness travels out of band instead — see
		// IsStale, which the CLI consults to say so once.
		return Ledger{}, nil
	}
	for sc.Scan() {
		if p, obs, ok := parseLine(sc.Text()); ok {
			l[p] = obs
		}
	}
	return l, sc.Err()
}

// IsStale reports whether a ledger exists on disk that this mrw will not
// trust. It is separate from Load because the two answers have different
// consequences: Load must hand back an empty ledger and no error so the next
// write can heal the file, while the CLI wants to SAY so once — a refusal the
// caller cannot account for looks like a bug, and "read it again" is the whole
// remedy.
func IsStale(root string) (bool, error) {
	path, err := ReadPath(root)
	if err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return false, sc.Err() // an empty file is not a stale one
	}
	return sc.Text() != header, nil
}

// StaleNotice is what the CLI prints when IsStale reports true.
const StaleNotice = "mrw: the read ledger was written by an older mrw and has been discarded — " +
	"up to v0.0.11 a read that served nothing recorded the whole file, and such an entry cannot be " +
	"told from a real one. Read the files you mean to edit again."

// parseLine reads one ledger line. Two shapes are accepted: the current
// "<sha>  <spans>  <path>" and the pre-span "<sha>  <path>", which is read as a
// whole-file observation — a ledger written by an older mrw stays usable rather
// than reading as "never seen".
func parseLine(text string) (string, Observation, bool) {
	sha, rest, ok := strings.Cut(strings.TrimSpace(text), "  ")
	if !ok || sha == "" || rest == "" {
		return "", Observation{}, false
	}
	spans, path, ok := strings.Cut(rest, "  ")
	// Two shapes have to be told apart, and a legacy PATH may itself contain a
	// double space: "<sha>  my  file.go" is one path, not spans plus a path.
	// The middle field is only spans when it is shaped like spans.
	if !ok || !spanShaped(spans) {
		return rest, Observation{SHA: sha}, true // legacy: whole file
	}
	if path == "" {
		return "", Observation{}, false
	}
	return path, Observation{SHA: sha, Spans: parseSpans(spans)}, true
}

// spanShaped reports whether s is the spans field rather than the first half of
// a path: "-" for whole, "" for nothing served, else digits, commas and dashes.
func spanShaped(s string) bool {
	if s == "-" || s == "" {
		return true
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != ',' && r != '-' {
			return false
		}
	}
	return true
}

// parseSpans reads "-" (whole file), "" (nothing served) or "1-5,20-30".
func parseSpans(s string) [][2]int {
	if s == "-" {
		return nil
	}
	out := [][2]int{}
	for _, part := range strings.Split(s, ",") {
		lo, hi, ok := strings.Cut(part, "-")
		if !ok {
			hi = lo
		}
		a, errA := strconv.Atoi(lo)
		b, errB := strconv.Atoi(hi)
		if errA != nil || errB != nil {
			continue
		}
		out = append(out, [2]int{a, b})
	}
	return out
}

func formatSpans(o Observation) string {
	if o.Whole() {
		return "-"
	}
	parts := make([]string, 0, len(o.Spans))
	for _, s := range o.Spans {
		parts = append(parts, fmt.Sprintf("%d-%d", s[0], s[1]))
	}
	return strings.Join(parts, ",")
}

// Record merges observations into the ledger on disk and saves it. Paths absent
// from obs keep whatever was recorded for them: one command observing two files
// must not erase what another observed about a third.
//
// Within one path the observations ACCUMULATE while the file is unchanged: two
// reads of different ranges leave the caller having seen both. A different SHA
// replaces the record outright, because spans counted in one version of a file
// say nothing about another.
func Record(root string, obs map[string]Observation) error {
	if len(obs) == 0 {
		return nil
	}
	l, err := Load(root)
	if err != nil {
		return err
	}
	for path, o := range obs {
		l[path] = merge(l[path], o)
	}
	return save(root, l)
}

// merge combines a new observation with what was already recorded for the same
// path. Anything about a different version of the file is discarded.
func merge(old, new Observation) Observation {
	if old.SHA != new.SHA || new.Whole() {
		return new
	}
	if old.Whole() {
		return old
	}
	return Observation{SHA: new.SHA, Spans: mergeSpans(append(append([][2]int{}, old.Spans...), new.Spans...))}
}

// mergeSpans sorts and coalesces overlapping or touching spans, so a ledger
// entry cannot grow without bound as a file is read range by range.
func mergeSpans(in [][2]int) [][2]int {
	if len(in) == 0 {
		return [][2]int{}
	}
	sort.Slice(in, func(i, j int) bool { return in[i][0] < in[j][0] })
	out := [][2]int{in[0]}
	for _, s := range in[1:] {
		last := &out[len(out)-1]
		if s[0] <= last[1]+1 {
			if s[1] > last[1] {
				last[1] = s[1]
			}
			continue
		}
		out = append(out, s)
	}
	return out
}

// Forget drops paths from the ledger, so the next write must observe them
// afresh. Used by `mrw forget` when a caller knows their picture is stale.
func Forget(root string, paths []string) (int, error) {
	l, err := Load(root)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, p := range paths {
		if _, ok := l[p]; ok {
			delete(l, p)
			n++
		}
	}
	if n == 0 {
		return 0, nil
	}
	// Record cannot express a deletion, so rewrite the whole file here.
	return n, save(root, l)
}

// save writes the whole ledger, sorted so the file diffs cleanly and two runs
// produce the same bytes.
func save(root string, l Ledger) error {
	path, err := writePath(root)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	paths := make([]string, 0, len(l))
	for p := range l {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	b.WriteString(header + "\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "%s  %s  %s\n", l[p].SHA, formatSpans(l[p]), p)
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// SHA is the ledger's hash of a byte slice, and the one every other package
// must use — two hashes of the same bytes disagreeing would make every write
// look stale.
func SHA(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ReadPath is where this root's ledger is READ from: the state directory,
// falling back to a legacy in-tree file when the state directory holds none, so
// a caller who has not run a migrating command still sees data they already
// have. Writes never use this — see writePath.
func ReadPath(root string) (string, error) {
	p, err := state.Path(root, Name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	if legacy := state.LegacyPath(root, Name); fileExists(legacy) {
		return legacy, nil
	}
	return p, nil
}

// writePath is where the ledger is WRITTEN: always the state directory, never
// the legacy in-tree file. Reading a file a caller already has is compatibility;
// writing it again would be re-creating the bug ADR-004 fixes.
func writePath(root string) (string, error) {
	return state.Path(root, Name)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
