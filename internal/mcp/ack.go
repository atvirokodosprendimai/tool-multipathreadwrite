// Package mcp — checkpoints and the pending store (ADR-031).
//
// A page mrw serves is not a page the caller received. Measured 2026-09-05 on
// Claude Code 2.1.261: mrw returned lines 1-2727, the HOST cut the middle out,
// the model saw lines 1-90 and 2644-2727, and mrw recorded the whole page — so
// a write to line 1500, inside the discarded middle, applied and said ok. mrw
// cannot detect that from inside the server: a cut result and a delivered one
// are identical to it.
//
// So a served span is held PENDING here, and reaches the read-before-modify
// ledger only when the caller echoes the checkpoint that covers it.
//
// ⚠ Checkpoints are interleaved THROUGH the page, not appended to it, and the
// measurement is why: the observed truncation kept both ends and removed the
// middle, so one token at the end survives a cut that destroyed 2,553 lines.
// Proving receipt of a page requires proving it per region.
//
// ⚠ AND A MARKER BRACKETS ITS SPAN RATHER THAN FOLLOWING IT, which is the
// second half of the same lesson and one this file got wrong first. A single
// marker AFTER its lines is the page-level flaw at a smaller scale: a cut that
// begins inside the span and leaves the trailing marker licenses everything the
// caller did not receive. Under the recorded 1-90 / 2644-2727 cut that
// over-licensed lines 2601-2643. So each span opens with a marker naming its
// range and COUNT and closes with the same id, and a caller acknowledges only
// when it holds both ends and counted the lines between.
//
// ⚠ WHAT THIS DOES AND DOES NOT PROVE. mrw cannot verify any of it from inside
// the server, and does not pretend to: the mechanism makes an honest caller ABLE
// TO TELL that it was cut, which it previously could not. A caller that echoes
// markers without checking the count is trusting itself, and no server-side
// design can stop that.
package mcp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// AckRule is THE sentence that states the acknowledgement contract, and every
// caller-facing surface embeds this one value rather than paraphrasing it.
//
// ⚠ It is a constant because paraphrases drifted three times. The mechanism was
// bracketed while the footer, the instructions, the README and AGENTS.md all
// still described the old single-marker rule; and twice a mutant gutted the
// clause while gates that checked for a TOKEN, and then for a HEADING, stayed
// green. A gate can assert this exact string on every surface, which is a
// question about substance rather than about vocabulary.
const AckRule = "Send an id in ack only if you hold BOTH its open and close markers AND counted the N numbered lines the open marker says follow: one marker is not enough, because a cut starting inside a span leaves the other end."

// ckEvery is how many served content lines one checkpoint covers. At 200 a
// 2,727-line page pays twenty-eight marker lines — an open and a close each —
// which is 1.03% of its size.
const ckEvery = 200

// pendingName is the file under the state directory holding spans that have
// been served but not acknowledged. It sits beside the ledger rather than in
// it, because an unacknowledged span is not a permission (ADR-004: nothing in
// the working tree). It is versioned with the ledger: a hold in pending.json
// may span a sparse read's gaps (up to v1.37.1), so it is not read (ADR-108).
const pendingName = "pending-v3.json"

// pendingLock serializes every change to the store across processes (ADR-085):
// two servers on one checkout each rewrote it from a stale copy, and the later
// save dropped the earlier's spans. It is taken before any ledger lock, never
// inside one.
const pendingLock = pendingName + ".lock"

// maxPending bounds the store. A caller that never acknowledges anything would
// otherwise grow it without limit; the oldest go first, and losing one costs a
// re-read rather than a wrong write.
const maxPending = 512

// pending is one served span awaiting acknowledgement. Seq orders eviction.
type pending struct {
	Path  string `json:"path"`
	SHA   string `json:"sha"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Seq   int64  `json:"seq"`
}

// checkpoint returns a 16-hex-digit (64-bit) marker from crypto/rand.
//
// ⚠ Random, NOT a hash of the served text and not a counter. Both of those can
// be DERIVED by a caller that received nothing. Randomness buys exactly that —
// it does not stop an id being recalled while it is still pending, which is why
// a pending entry is consumed on promotion.
//
// 8 hex digits (32 bits) was the first cut and is too few: against a 512-entry
// store with nothing rate-limiting, hitting any live id is roughly 8.4 million
// attempts (review of PR #132).
func checkpoint() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failing is not something to paper over with a weaker
		// source: a predictable checkpoint is worse than none, because it
		// looks like proof. Refuse by returning empty and let the caller
		// serve the page unmarked, which licenses nothing.
		return ""
	}
	return hex.EncodeToString(b[:])
}

// interleave inserts a checkpoint marker after every ckEvery served content
// lines of text, and returns the marked text with each checkpoint's span.
//
// It reads the line numbers out of the served text's own `NNN|` prefixes rather
// than taking the requested range, so a span is what was SERVED. A page that
// served 1-2727 of 3619 yields checkpoints inside 1-2727 and none beyond.
//
// recorded says which served lines the read recorded as read; nil means every
// one. A line it did not record — served cut by max_cols (ADR-154) — ends the
// checkpoint as a gap does and starts none, so no ack can license it. It is
// asked of the read's observation, never of the `[cols A-B of L]` marker,
// which is text a file can hold itself.
func interleave(text string, recorded func(line int) bool) (string, map[string][2]int) {
	// ADR-108: JSON replaces bytes that are not valid UTF-8, so the host is
	// shown text that is not the file, while a checkpoint would license the
	// file. Such a slice is served as it is, with no checkpoint, and says why.
	if !utf8.ValidString(text) {
		return text + "-- not licensed: these lines hold bytes that are not valid UTF-8, which the JSON answer replaces, so this read licenses none of them; read and edit the file with the CLI (`mrw read`, `mrw write`)\n", map[string][2]int{}
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	// Group the SERVED content lines into runs of ckEvery, remembering where
	// each run begins and ends in the slice. Each run is BRACKETED, never
	// followed: a marker after its lines is the page-level defect at span scale. Two passes are needed because an
	// opening marker names the span it introduces, and the span is not known
	// until its lines have been counted — the one-pass alternative is a marker
	// that trails its lines, which is the defect this design exists to avoid.
	type group struct {
		from, to   int // indices into lines
		start, end int // served line numbers
		n          int
	}
	var groups []group
	cur := group{from: -1}
	var cut []int
	for i, line := range lines {
		ln, ok := servedLineNumber(line)
		if !ok {
			continue
		}
		if recorded != nil && !recorded(ln) {
			if cur.from >= 0 {
				groups = append(groups, cur)
				cur = group{from: -1}
			}
			cut = append(cut, ln)
			continue
		}
		// ADR-108: a gap ends the checkpoint. Its span is start..end, so a run
		// carried across a gap licensed every line in the gap once acknowledged
		// — lines 1 and 100 served together licensed 1-100.
		if cur.from >= 0 && ln != cur.end+1 {
			groups = append(groups, cur)
			cur = group{from: -1}
		}
		if cur.from < 0 {
			cur = group{from: i, start: ln}
		}
		cur.to, cur.end, cur.n = i, ln, cur.n+1
		if cur.n >= ckEvery {
			groups = append(groups, cur)
			cur = group{from: -1}
		}
	}
	if cur.from >= 0 {
		groups = append(groups, cur)
	}

	spans := map[string][2]int{}
	ids := make([]string, len(groups))
	for i := range groups {
		ids[i] = checkpoint()
		if ids[i] != "" {
			spans[ids[i]] = [2]int{groups[i].start, groups[i].end}
		}
	}

	var out strings.Builder
	g := 0
	for i, line := range lines {
		if g < len(groups) && i == groups[g].from && ids[g] != "" {
			fmt.Fprintf(&out, "-- ck %s open lines %d-%d (%d lines follow)\n",
				ids[g], groups[g].start, groups[g].end, groups[g].n)
		}
		out.WriteString(line)
		out.WriteByte('\n')
		if g < len(groups) && i == groups[g].to {
			if ids[g] != "" {
				fmt.Fprintf(&out, "-- ck %s close\n", ids[g])
			}
			g++
		}
	}
	if len(cut) > 0 {
		out.WriteString(cutNote(cut))
	}
	return out.String(), spans
}

// cutNote names the lines a read served cut and left out of every checkpoint
// (ADR-154): at most ten numbers, then how many more.
func cutNote(cut []int) string {
	names := make([]string, 0, min(len(cut), 10))
	for _, ln := range cut[:min(len(cut), 10)] {
		names = append(names, strconv.Itoa(ln))
	}
	list := strings.Join(names, ", ")
	if len(cut) > 10 {
		list += fmt.Sprintf(" and %d more", len(cut)-10)
	}
	return "-- not licensed: line(s) " + list + " were served cut by max_cols and are in no checkpoint; read them without max_cols to edit them\n"
}

// servedSlice is one ==> header and the lines that belong to it, so a
// multi-file fitting serve can be interleaved per file (ADR-039). sha is the
// prefix the header printed: read.Run reads a file once per spec, so two
// slices of one file can come from two versions of it.
type servedSlice struct {
	path string
	sha  string
	text string
}

// splitServed cuts a read.Run report on `==> path` headers. A slice with no
// numbered lines produces no checkpoints — UNREADABLE and REFUSED stay unmarked.
// servedHeader is the header read.Run prints above a served file —
// `==> <path>  <N>L  <B>B  sha <hex>` — anchored at its END, because the path
// itself may hold spaces, even two in a row. Cutting at the first space turned
// `x y.txt` into `x`, which has no observation, so an MCP read of such a file
// that fit on one page failed with -32603 (found by the chaos harness, 2026-09-24).
var servedHeader = regexp.MustCompile(`^(.*)  \d+L  \d+B  sha ([0-9a-f]+)$`)

func splitServed(report string) []servedSlice {
	if report == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(report, "\n"), "\n")
	var out []servedSlice
	var cur servedSlice
	var b strings.Builder
	flush := func() {
		if cur.path == "" && b.Len() == 0 {
			return
		}
		cur.text = b.String()
		out = append(out, cur)
		b.Reset()
		cur = servedSlice{}
	}
	for _, line := range lines {
		if rest, ok := strings.CutPrefix(line, "==> "); ok {
			flush()
			// A REFUSED or UNREADABLE header serves no lines, so its path is
			// only a label; a served header's path is exact.
			path := rest
			if m := servedHeader.FindStringSubmatch(rest); m != nil {
				path, cur.sha = m[1], m[2]
			} else if i := strings.Index(rest, "  "); i >= 0 {
				path = rest[:i]
			}
			cur.path = path
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	flush()
	return out
}

// unrecordableNameNote is said under a served header whose name holds a control
// byte: read.Run prints such a name quoted and escaped (read.ShownPath), a spelling
// no observation carries, so the file gets no checkpoint, and the ledger cannot
// hold its name either (#702, #716).
const unrecordableNameNote = "-- not licensed: this file's name holds a control byte (a line feed, for one), which no checkpoint and no ledger record can name, so this read licenses none of its lines; edit it with --force or after a rename\n"

// shownKeys maps each observation key to the spelling read.Run prints it in, the
// other way round, so a served header is matched to the file it names by what the
// header says and not by what it looks like when unquoted. read.ShownPath is
// injective, so one spelling names one key.
func shownKeys(observed map[string]seen.Observation) map[string]string {
	m := make(map[string]string, len(observed))
	for k := range observed {
		m[read.ShownPath(k)] = k
	}
	return m
}

// servedKey is the observation key a served header names: its path as printed,
// else the typed spelling cleaned the way read.Run keys it (a header prints the
// spec as typed, the map is cleaned). A quoted header is unquoted first, so
// `"./a\nb"` finds the observation of `a\nb`.
func servedKey(path string, shown map[string]string) (string, bool) {
	if k, ok := shown[path]; ok {
		return k, true
	}
	name := path
	if strings.HasPrefix(path, `"`) {
		if u, err := strconv.Unquote(path); err == nil {
			name = u
		}
	}
	for _, p := range []string{filepath.Clean(name), name} {
		if k, ok := shown[read.ShownPath(p)]; ok {
			return k, true
		}
	}
	return "", false
}

// unrecordableName says whether a served header names a file this read cannot
// checkpoint: its key is one the ledger cannot hold, or no observation answers to
// the header and the name it prints holds a control byte (read.ShownPath quotes
// exactly those, and begins every escaped name with a quote).
func unrecordableName(sl servedSlice, key string, found bool) bool {
	if sl.sha == "" || sl.path == "" {
		return false
	}
	if found {
		return !seen.Recordable(key)
	}
	u, err := strconv.Unquote(sl.path)
	return err == nil && strings.ContainsFunc(u, func(r rune) bool { return r < 0x20 || r == 0x7f })
}

// markServed interleaves checkpoints through each file of a fitting report
// and returns the marked text plus per-path spans. A report with no numbered
// lines is returned unchanged and holds nothing. observed is what the read
// recorded: a slice of the version it names is bracketed only where it
// recorded a line (ADR-154); a slice of another version is held by nobody
// (heldSpans), so its grouping licenses nothing either way.
func markServed(report string, observed map[string]seen.Observation) (string, map[string]map[string]map[string][2]int) {
	slices := splitServed(report)
	if len(slices) == 0 {
		return report, nil
	}
	var out strings.Builder
	byPath := map[string]map[string]map[string][2]int{}
	shown := shownKeys(observed)
	for _, sl := range slices {
		key, found := servedKey(sl.path, shown)
		// #716, #740: a file whose name the ledger cannot hold is served as it is,
		// and says so; an ack cannot name it.
		if unrecordableName(sl, key, found) {
			out.WriteString(sl.text)
			out.WriteString(unrecordableNameNote)
			continue
		}
		var o seen.Observation
		if found {
			o = observed[key]
		}
		marked, spans := interleave(sl.text, recordedIn(o, found, sl.sha))
		out.WriteString(marked)
		// Kept per file AND per version: two specs naming one file — a list
		// split into ranges, each with its own max_lines budget — are two
		// slices, and the second must not drop the first one's checkpoints
		// (review of #318). But read.Run reads the file once per spec, so a
		// writer between them leaves slices of two versions; heldSpans keeps
		// only those of the version the observation names (the review of the
		// merge, which licensed a line served from an older version).
		if sl.path != "" && len(spans) > 0 {
			p := filepath.Clean(sl.path)
			if found {
				p = filepath.Clean(key)
			}
			if byPath[p] == nil {
				byPath[p] = map[string]map[string][2]int{}
			}
			if byPath[p][sl.sha] == nil {
				byPath[p][sl.sha] = map[string][2]int{}
			}
			maps.Copy(byPath[p][sl.sha], spans)
		}
	}
	// A slice interleave declined to mark (ADR-108: bytes that are not valid
	// UTF-8) still carries its notice, so the text is served marked whenever any
	// slice changed, and the footer is added only when something can be acked.
	if out.String() == report {
		return report, nil
	}
	if len(byPath) > 0 {
		fmt.Fprintf(&out, "-- This serve licenses NOTHING until you acknowledge it.\n-- %s\n", AckRule)
	}
	return out.String(), byPath
}

// heldSpans is what a read may hold for one file: the checkpoints of every
// slice served from the version whose sha the observation names, merged, and
// none from a version a writer replaced between specs — those lines are not
// what the file holds now. bySHA is keyed by the header's sha prefix.
func heldSpans(bySHA map[string]map[string][2]int, sha string) map[string][2]int {
	held := map[string][2]int{}
	for prefix, spans := range bySHA {
		if prefix != "" && strings.HasPrefix(sha, prefix) {
			maps.Copy(held, spans)
		}
	}
	return held
}

// recordedIn is which lines of a served slice the read recorded, or nil — every
// line — when the observation is of another version or there is none.
func recordedIn(o seen.Observation, found bool, sha string) func(int) bool {
	if !found || sha == "" || !strings.HasPrefix(o.SHA, sha) {
		return nil
	}
	return func(line int) bool { return o.Covers(line, line) }
}

// servedLineNumber reads the line number off a served content line, which
// read.Run renders as spaces, digits, "|", then the file's own text.
func servedLineNumber(s string) (int, bool) {
	i := strings.IndexByte(s, '|')
	if i <= 0 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(s[:i]))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// observationOf returns the observation for path under the same key
// read.Run uses (filepath.Clean), then the typed spelling. The ==> header
// prints the spec as typed; the map is cleaned. Looking up only the header
// spelling holds an empty SHA, and promote then drops the ack as stale —
// which is how a fitting nested-path read licensed nothing after a correct
// ack on Windows (PR #157).
func observationOf(observed map[string]seen.Observation, path string) (seen.Observation, bool) {
	if o, ok := observed[filepath.Clean(path)]; ok {
		return o, true
	}
	if o, ok := observed[path]; ok {
		return o, true
	}
	return seen.Observation{}, false
}

// hold files served spans as pending, keyed by checkpoint, against the file
// whose lines they bracketed (ADR-039). It writes no ledger entry: that is
// the whole point, and TestAPendingRecordReachesNoLedger pins it.
func hold(root, path, sha string, spans map[string][2]int) error {
	if len(spans) == 0 || path == "" {
		return nil
	}
	if sha == "" {
		return fmt.Errorf("empty sha for %s", path)
	}
	release, err := state.Hold(root, pendingLock)
	if err != nil {
		return err
	}
	defer release()
	store, err := loadPending(root)
	if err != nil {
		return err
	}
	var seq int64
	for _, p := range store {
		if p.Seq > seq {
			seq = p.Seq
		}
	}
	for ck, sp := range spans {
		seq++
		store[ck] = pending{Path: path, SHA: sha, Start: sp[0], End: sp[1], Seq: seq}
	}
	evict(store)
	return savePending(root, store)
}

// keptAck is an acknowledged checkpoint promoteAcks did not consume because the
// file it names could not be read just now: it stays pending, and the caller can
// send it again (#733). cause says why, with the path relative to the root.
type keptAck struct {
	id    string
	cause string
}

// transientRead says whether a read failed for a reason that says nothing about
// the file's version, and may clear on its own: a permission, a descriptor limit,
// an interrupt, a busy file, or on Windows a sharing violation or a handle or
// memory limit. Every other error (the file is gone, it is a directory now, a
// path component is a file, a link loops) is a fact about the version, so the
// span is dropped as it always was.
func transientRead(err error) bool { return transientReadOn(runtime.GOOS, err) }

// windowsTransient are the native Win32 codes a Windows read can fail with that
// clear on their own: syscall.EMFILE and its kin do not match them, since Go's
// Windows errno values for those are its own invented ones.
var windowsTransient = []syscall.Errno{
	4,  // ERROR_TOO_MANY_OPEN_FILES
	8,  // ERROR_NOT_ENOUGH_MEMORY
	32, // ERROR_SHARING_VIOLATION
	33, // ERROR_LOCK_VIOLATION
}

// transientReadOn is transientRead for the named OS, so the Windows codes are
// tested wherever the tests run: on Unix the same numbers mean other errors
// (4 is EINTR, 8 ENOEXEC, 32 EPIPE) and are not asked about.
func transientReadOn(goos string, err error) bool {
	for _, e := range []error{os.ErrPermission, syscall.EMFILE, syscall.ENFILE, syscall.EINTR, syscall.EAGAIN, syscall.EBUSY} {
		if errors.Is(err, e) {
			return true
		}
	}
	if goos == "windows" {
		for _, e := range windowsTransient {
			if errors.Is(err, e) {
				return true
			}
		}
	}
	return false
}

// readCause names a failed read by the path the checkpoint stored, relative to
// the checkout, whatever way the root was spelled: the open error's own path is
// the absolute one.
func readCause(path string, err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Op + " " + path + ": " + pe.Err.Error()
	}
	return err.Error()
}

// ackKey is a version of a file: acknowledged spans are kept per path AND sha.
type ackKey struct{ path, sha string }

// ackGroup is the acknowledged spans of one version of one file and the ids that
// named them.
type ackGroup struct {
	spans [][2]int
	ids   []string
	// keep and record are what prepare saw of the file, and what commit does
	// without looking again: a file whose read failed for a passing reason keeps
	// its ids pending; one still at the acknowledged version has its span
	// recorded; any other version is gone and its ids are consumed.
	keep, record bool
}

// ackBatch is what a call's acknowledgements amount to, taken from the pending
// store BEFORE the call serves anything: a serve holds new checkpoints, and a
// full store drops its oldest, which may be the very one acknowledged. Nothing is
// changed until commit, which runs only once the call has been answered without
// an error (#741). kept is known at once, so the answer can name it (#733).
type ackBatch struct {
	byVersion map[ackKey]*ackGroup
	unknown   []string  // ids that matched nothing
	kept      []keptAck // ids whose file could not be read just now: they stay pending
}

// prepareAcks reads the pending entries the acknowledgements name, changing
// nothing.
//
// ⚠ It records the PENDING span, never the requested one (commit). Recording what
// was asked for is the defect this file exists to close, wearing a new name.
//
// An ack matching nothing is ignored rather than refused: it is a stale caller
// from an earlier session, and failing the whole call would punish the spans
// that were acknowledged honestly. A store that cannot be loaded is an error,
// before the call records anything.
//
// An id is consumed (by commit) only after its span was recorded or its version
// is known to be gone (changed, removed, no longer a regular file). A read that
// failed for any other reason (a permission, a descriptor limit) says nothing
// about the version: the id is kept, and returned in kept (#733).
func prepareAcks(root string, acks []string) (*ackBatch, error) {
	b := &ackBatch{byVersion: map[ackKey]*ackGroup{}}
	if len(acks) == 0 {
		return b, nil
	}
	release, err := state.Hold(root, pendingLock)
	if err != nil {
		return nil, err
	}
	defer release()
	store, err := loadPending(root)
	if err != nil {
		return nil, err
	}
	// ⚠ KEYED BY PATH **AND SHA**, not by path alone. An earlier cut appended
	// every acknowledged span to one observation per path and let the last
	// checkpoint's SHA win, so acknowledging a stale page and a current one in
	// the same call produced spans from the OLD file recorded under the NEW
	// file's SHA — a licence to write lines the caller never saw in the version
	// on disk. Found by the review of PR #132.
	matched := map[string]bool{}
	for _, ck := range acks {
		p, ok := store[ck]
		if !ok {
			// An id this call already used, or already named, is not named
			// again: the caller sent it twice (the reviews of #343).
			if !matched[ck] {
				b.unknown = append(b.unknown, ck)
				matched[ck] = true
			}
			continue
		}
		if matched[ck] {
			continue
		}
		matched[ck] = true
		k := ackKey{p.Path, p.SHA}
		g := b.byVersion[k]
		if g == nil {
			g = &ackGroup{}
			b.byVersion[k] = g
		}
		g.spans = append(g.spans, [2]int{p.Start, p.End})
		g.ids = append(g.ids, ck)
	}
	for k, g := range b.byVersion {
		sha, serr := currentSHA(root, k.path)
		switch {
		case serr != nil && transientRead(serr):
			g.keep = true
			for _, id := range g.ids {
				b.kept = append(b.kept, keptAck{id: id, cause: readCause(k.path, serr)})
			}
		case serr == nil && sha == k.sha:
			g.record = true
		}
	}
	sort.Slice(b.kept, func(i, j int) bool { return b.kept[i].id < b.kept[j].id })
	return b, nil
}

// consume takes the acknowledged ids out of the pending store, except those prepare
// kept, and saves it. It runs BEFORE any ledger record: a save that fails grants
// nothing (#742). It returns the observations to record, each the acknowledged
// span of a version prepare saw on disk and nothing else.
//
// ⚠ ONLY THE VERSION ON DISK IS RECORDED, and the rest are dropped.
//
// An earlier cut recorded one observation per version by ranging over a map.
// seen.merge REPLACES the observation when the SHA differs, so the last
// iteration won — and map order is random, which meant a STALE
// acknowledgement could overwrite a current, valid licence and consume both
// ids doing it. Found by the review of PR #132.
//
// A span acknowledged against a version that is no longer on disk cannot
// license anything anyway: the ledger's own SHA check refuses it. Dropping
// it is what it already meant, done deterministically.
func (b *ackBatch) consume(root string) (map[string]seen.Observation, error) {
	if len(b.byVersion) == 0 {
		return nil, nil
	}
	release, err := state.Hold(root, pendingLock)
	if err != nil {
		return nil, err
	}
	defer release()
	store, err := loadPending(root)
	if err != nil {
		return nil, err
	}
	obs := map[string]seen.Observation{}
	consumed := false
	for k, g := range b.byVersion {
		if g.keep {
			continue
		}
		if g.record {
			obs[k.path] = seen.Observation{SHA: k.sha, Spans: g.spans}
		}
		for _, id := range g.ids {
			delete(store, id)
		}
		consumed = true
	}
	if !consumed {
		return nil, nil
	}
	if err := savePending(root, store); err != nil {
		return nil, err
	}
	return obs, nil
}

// commitAll is consume, then one ledger record of the acknowledged spans and of
// extra (a quiet read's licences), as one Record when no path is in both, so a
// failure leaves none of them. When a path is in both the two are recorded one
// after the other.
func (b *ackBatch) commitAll(root string, extra map[string]seen.Observation) error {
	obs, err := b.consume(root)
	if err != nil {
		return err
	}
	both := false
	for p := range extra {
		if _, dup := obs[p]; dup {
			both = true
		}
	}
	if both {
		if err := seen.Record(root, obs); err != nil {
			return err
		}
		return seen.Record(root, extra)
	}
	all := map[string]seen.Observation{}
	for p, o := range obs {
		all[p] = o
	}
	for p, o := range extra {
		all[p] = o
	}
	return seen.Record(root, all)
}

// promoteAcks is prepareAcks and commitAll at once, for a call that has nothing to
// answer first (mrw_write: its acknowledgements license the plan it carries). What
// it reports kept is what it did.
func promoteAcks(root string, acks []string) (unknown []string, kept []keptAck, err error) {
	b, err := prepareAcks(root, acks)
	if err != nil {
		return nil, nil, err
	}
	return b.unknown, b.kept, b.commitAll(root, nil)
}

// evict drops the oldest entries once the store passes maxPending, never one the
// call is keeping for a retry (callKeep, #742): the store may stay over the
// bound by that many until the next call.
func evict(store map[string]pending) {
	if len(store) <= maxPending {
		return
	}
	keys := make([]string, 0, len(store))
	for k := range store {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return store[keys[i]].Seq < store[keys[j]].Seq })
	for _, k := range keys {
		if len(store) <= maxPending {
			break
		}
		if !callKeep[k] {
			delete(store, k)
		}
	}
}

func pendingPath(root string) (string, error) { return state.Path(root, pendingName) }

// readPending loads the store under its lock for a caller that only reads it,
// and reads past a lock it cannot take, as iter.Load does (ADR-085).
func readPending(root string) (map[string]pending, error) {
	if release, err := state.Hold(root, pendingLock); err == nil {
		defer release()
	}
	return loadPending(root)
}

// readPendingFile is regular.ReadFile, as a variable so a test can hold the
// store busy. It never falls back to a blocking open (ADR-201).
var readPendingFile = regular.ReadFile

// loadPending reads the store. A store that is missing, that is not a regular
// file (a FIFO there is read without blocking and is refused) or that will not
// parse is an empty one, which licenses nothing and which the next save
// replaces; a read that failed for another reason is an error, so nothing is
// saved over a store that could not be read (ADR-201).
func loadPending(root string) (map[string]pending, error) {
	p, err := pendingPath(root)
	if err != nil {
		return nil, err
	}
	b, err := readPendingFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, regular.ErrNotRegular) {
			return map[string]pending{}, nil
		}
		return nil, err
	}
	store := map[string]pending{}
	if err := json.Unmarshal(b, &store); err != nil {
		// A pending store that will not parse licenses nothing, which is the
		// safe direction: the caller re-reads. Not an error to the caller.
		return map[string]pending{}, nil //nolint:nilerr // an unparsable store licenses nothing (above)
	}
	return store, nil
}

func savePending(root string, store map[string]pending) error {
	p, err := pendingPath(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return state.WriteSynced(p, b, 0o644)
}

// ForgetPending empties this root's store of served-but-unacknowledged spans
// (ADR-175). A later ack promotes such a span into the ledger when the file's
// sha still matches, with no observation needed first, so leaving the store
// would let a serve from before a compaction license a write after it. Taken
// under the store's own lock, before any ledger lock, as everything here is.
func ForgetPending(root string) error {
	release, err := state.Hold(root, pendingLock)
	if err != nil {
		return err
	}
	defer release()
	store, err := loadPending(root)
	if err != nil || len(store) == 0 {
		return err
	}
	return savePending(root, map[string]pending{})
}

// currentSHA is the digest of the file as it stands, in the same form the
// ledger records, so promotion can tell a live acknowledgement from a stale one.
// The file is streamed into the hash, never held whole (ADR-107): a file grown
// after its read would otherwise be read into memory at any size.
func currentSHA(root, path string) (string, error) {
	// ADR-108: opened without blocking and refused unless it is still a
	// regular file — a FIFO swapped in after the read blocked a plain open,
	// under the pending-store lock, and a device streams for ever. ADR-109 made
	// that one function, regular.Open, which passes a directory back.
	full, err := rooted.Resolve(root, filepath.FromSlash(path))
	if err != nil {
		return "", err
	}
	afterResolveFn()
	f, fi, err := rooted.OpenRead(root, full)
	if errors.Is(err, regular.ErrNotRegular) {
		return "", fmt.Errorf("%s is no longer a regular file", path)
	}
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s is no longer a regular file", path)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
