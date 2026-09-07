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
package mcp

import (
	"bufio"
	"crypto/rand"
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

// ckEvery is how many served content lines one checkpoint covers. At 200 a
// 2,727-line page pays fourteen marker lines, under half a percent of its size.
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

// checkpoint returns an unguessable 8-hex-digit marker.
//
// ⚠ Random, NOT a hash of the served text and not a counter. Both of those can
// be derived by a caller that received nothing, which would make the whole
// mechanism ceremony: the point is that a checkpoint can only be echoed by
// someone it actually reached.
func checkpoint() string {
	var b [4]byte
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
	spans := map[string][2]int{}
	var out strings.Builder
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	first, last, n := 0, 0, 0
	flush := func() {
		if n == 0 || first == 0 {
			return
		}
		if ck := checkpoint(); ck != "" {
			fmt.Fprintf(&out, "-- ck %s\n", ck)
			spans[ck] = [2]int{first, last}
		}
		first, last, n = 0, 0, 0
	}
	for sc.Scan() {
		line := sc.Text()
		out.WriteString(line)
		out.WriteByte('\n')
		if ln, ok := servedLineNumber(line); ok {
			if first == 0 {
				first = ln
			}
			last = ln
			n++
			if n >= ckEvery {
				flush()
			}
		}
	}
	flush()
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
	obs := map[string]seen.Observation{}
	changed := false
	for _, ck := range acks {
		p, ok := store[ck]
		if !ok {
			continue
		}
		o := obs[p.Path]
		o.SHA = p.SHA
		o.Spans = append(o.Spans, [2]int{p.Start, p.End})
		obs[p.Path] = o
		delete(store, ck)
		changed = true
	}
	if !changed {
		return nil
	}
	if err := seen.Record(root, obs); err != nil {
		return err
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
