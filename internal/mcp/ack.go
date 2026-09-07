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
// middle, so one token at the end survives a cut that destroyed 2,554 lines.
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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
// which is under one percent of its size.
const ckEvery = 200

// pendingName is the file under the state directory holding spans that have
// been served but not acknowledged. It sits beside the ledger rather than in
// it, because an unacknowledged span is not a permission (ADR-004: nothing in
// the working tree).
const pendingName = "pending.json"

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
func interleave(text string) (string, map[string][2]int) {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	// Group the SERVED content lines into runs of ckEvery, remembering where
	// each run begins and ends in the slice. Two passes are needed because an
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
	for i, line := range lines {
		ln, ok := servedLineNumber(line)
		if !ok {
			continue
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
	return out.String(), spans
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

// hold files served spans as pending, keyed by checkpoint. It writes no ledger
// entry: that is the whole point, and TestAPendingRecordReachesNoLedger pins it.
func hold(root string, obs map[string]seen.Observation, spans map[string][2]int) error {
	if len(spans) == 0 {
		return nil
	}
	// One observation per path here — a paged read serves one file.
	var path, sha string
	for p, o := range obs {
		path, sha = p, o.SHA
		break
	}
	if path == "" {
		return nil
	}
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

// promote turns each acknowledged checkpoint into a ledger record for its own
// span and nothing else.
//
// ⚠ It records the PENDING span, never the requested one. Recording what was
// asked for is the defect this file exists to close, wearing a new name.
//
// An ack matching nothing is ignored rather than refused: it is a stale caller
// from an earlier session, and failing the whole call would punish the spans
// that were acknowledged honestly.
func promote(root string, acks []string) error {
	if len(acks) == 0 {
		return nil
	}
	store, err := loadPending(root)
	if err != nil || len(store) == 0 {
		return err
	}
	// ⚠ KEYED BY PATH **AND SHA**, not by path alone. An earlier cut appended
	// every acknowledged span to one observation per path and let the last
	// checkpoint's SHA win, so acknowledging a stale page and a current one in
	// the same call produced spans from the OLD file recorded under the NEW
	// file's SHA — a licence to write lines the caller never saw in the version
	// on disk. Found by the review of PR #132.
	type key struct{ path, sha string }
	byVersion := map[key][][2]int{}
	changed := false
	for _, ck := range acks {
		p, ok := store[ck]
		if !ok {
			continue
		}
		k := key{p.Path, p.SHA}
		byVersion[k] = append(byVersion[k], [2]int{p.Start, p.End})
		delete(store, ck)
		changed = true
	}
	if !changed {
		return nil
	}
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
	for k, spans := range byVersion {
		sha, err := currentSHA(root, k.path)
		if err != nil || sha != k.sha {
			continue
		}
		o := seen.Observation{SHA: k.sha, Spans: spans}
		if err := seen.Record(root, map[string]seen.Observation{k.path: o}); err != nil {
			return err
		}
	}
	return savePending(root, store)
}

// evict drops the oldest entries once the store passes maxPending.
func evict(store map[string]pending) {
	if len(store) <= maxPending {
		return
	}
	keys := make([]string, 0, len(store))
	for k := range store {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return store[keys[i]].Seq < store[keys[j]].Seq })
	for _, k := range keys[:len(store)-maxPending] {
		delete(store, k)
	}
}

func pendingPath(root string) (string, error) { return state.Path(root, pendingName) }

func loadPending(root string) (map[string]pending, error) {
	p, err := pendingPath(root)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]pending{}, nil
		}
		return nil, err
	}
	store := map[string]pending{}
	if err := json.Unmarshal(b, &store); err != nil {
		// A pending store that will not parse licenses nothing, which is the
		// safe direction: the caller re-reads. Not an error to the caller.
		return map[string]pending{}, nil
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
	return os.WriteFile(p, b, 0o644)
}

// currentSHA is the digest of the file as it stands, in the same form the
// ledger records, so promotion can tell a live acknowledgement from a stale one.
func currentSHA(root, path string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
