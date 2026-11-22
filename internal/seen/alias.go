package seen

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// AliasIndex looks up ledger entries by filesystem identity. The first lookup
// that needs an alias stats each key once for the entire read or write.
type AliasIndex struct {
	stat   func(string) (os.FileInfo, error)
	root   string
	ledger map[string]Observation
	once   sync.Once
	keys   []aliasKey
	byID   map[[2]uint64][]aliasKey
}

type aliasKey struct {
	key  string
	info os.FileInfo
}

// NewAliasIndex looks up aliases in one snapshot of a checkout's read ledger.
// The stat function captures each ledger key's file identity on first use.
func NewAliasIndex(root string, ledger Ledger, stat func(string) (os.FileInfo, error)) *AliasIndex {
	return &AliasIndex{root: root, ledger: ledger, stat: stat}
}

// FileEvidence keeps the exact-spelling fast path and combines the other
// spellings once if the caller needs them. ADR-203 keeps an exact current
// observation for its numbered-line rule.
type FileEvidence struct {
	index           *AliasIndex
	full, path, sha string
	exact           Observation
	known           bool
	loaded          bool
	alsoCurrent     bool // another spelling has a current record
	combined        Observation
	found           bool
}

// File binds a ledger spelling and full path to the digest observed on disk.
func (a *AliasIndex) File(full, path, sha string) FileEvidence {
	e := FileEvidence{index: a, full: full, path: path, sha: sha}
	if a != nil {
		e.exact, e.known = a.ledger[path]
	}
	return e
}

// Primary returns the current exact record when available. Otherwise it
// consults aliases, preserving a stale record for the changed-file diagnostic.
func (e *FileEvidence) Primary() (Observation, bool) {
	if e.known && e.exact.SHA == e.sha {
		return e.exact, true
	}
	return e.All()
}

// Licence keeps a current exact record that covers the range, as ADR-203
// requires for numeric addresses. If only an alias licenses the range, its
// written/whole status must also reach the stale-number gate.
func (e *FileEvidence) Licence(from, to int) (Observation, bool) {
	return e.satisfying(func(o Observation) bool { return o.Covers(from, to) })
}

// Shows reports whether all lines of the range were shown in the current version.
func (e *FileEvidence) Shows(start, end int) bool {
	_, shown := e.satisfying(func(o Observation) bool { return o.FirstUnshown(start, end) == 0 })
	return shown
}

// satisfying tries the exact licence before consulting the combined evidence.
// If no alias contributed, the same predicate has already judged this record.
func (e *FileEvidence) satisfying(accept func(Observation) bool) (Observation, bool) {
	if e.known && e.exact.SHA == e.sha && accept(e.exact) {
		return e.exact, true
	}
	combined, found := e.All()
	if !found || combined.SHA != e.sha {
		return Observation{Spans: [][2]int{}}, false
	}
	return combined, e.alsoCurrent && accept(combined)
}

// All returns the union of current records when any exist. Callers checking a
// permission must check the digest: a stale result explains a refusal only.
func (e *FileEvidence) All() (Observation, bool) {
	if !e.loaded {
		e.loaded = true
		e.combined, e.found = e.combine()
	}
	return e.combined, e.found
}

func (e *FileEvidence) combine() (Observation, bool) {
	a := e.index
	if a == nil || len(a.ledger) == 0 || (e.known && len(a.ledger) == 1) {
		return e.exact, e.known
	}
	want, err := os.Stat(e.full)
	if err != nil {
		return e.exact, e.known
	}
	id, keyed := rooted.SnapshotID(want)
	a.once.Do(func() {
		if keyed {
			a.byID = make(map[[2]uint64][]aliasKey)
		}
		for key := range a.ledger {
			if info, err := a.stat(filepath.Join(a.root, filepath.FromSlash(key))); err == nil {
				k := aliasKey{key, info}
				if a.byID != nil {
					if id, ok := rooted.SnapshotID(info); ok {
						a.byID[id] = append(a.byID[id], k)
						continue
					}
				}
				a.keys = append(a.keys, k)
			}
		}
	})
	var records []Observation
	if e.known {
		records = append(records, e.exact)
	}
	// Prefer the exact stale record for a changed-file refusal, otherwise
	// report a consistent spelling when every alias is stale.
	staleKey, stale, found := e.path, e.exact, e.known
	collect := func(k aliasKey) {
		if k.key == e.path || !os.SameFile(want, k.info) {
			return
		}
		r := a.ledger[k.key]
		if r.SHA == e.sha {
			e.alsoCurrent = true
			records = append(records, r)
		} else if !e.known && (!found || k.key < staleKey) {
			staleKey, stale, found = k.key, r, true
		}
	}
	// The public Unix snapshot identifies a bucket directly. On Windows the
	// file index stays private to os.SameFile, so compare the saved snapshots.
	for _, k := range a.keys {
		collect(k)
	}
	if keyed {
		for _, k := range a.byID[id] {
			collect(k)
		}
	}
	if current, ok := unionVersion(records, e.sha); ok {
		return current, true
	}
	return stale, found
}
