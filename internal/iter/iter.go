// Package iter keeps the working set: the handful of files and ranges the
// current piece of work actually concerns.
//
// It exists for one reason, and the reason is arithmetic. Naming six paths on
// every call costs those tokens on every call, and output tokens are the
// expensive direction. Writing them down once and referring to them by nothing
// at all costs them once. Write one, use many.
//
// Entries are read SPECS, not bare paths — "internal/apply/apply.go:100-140" is
// a legal entry — so a bare `mrw read` returns exactly the ranges the work is
// in, not the whole of every file it touches. The same list scopes `mrw check`,
// which is what makes "run the tests for what I am working on" a call with no
// arguments.
package iter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// Name is the working set's filename inside the state directory. The DIRECTORY
// is resolved by internal/state and sits outside the working tree — this used
// to be written beside your source, where nothing ignored it. See ADR-004.
// The file itself is still plain text: reviewable, and editable by hand.
const Name = "iteration"

// Set is an ordered, de-duplicated list of read specs. Order is the order the
// caller added them, because that is usually the order they think in.
type Set struct {
	Entries []string
	Note    string // free-text first-line comment, e.g. what this iteration is
}

// Load reads the working set. A missing file yields an empty set and no error:
// having no iteration is the normal starting state, not a fault.
func Load(root string) (Set, error) {
	var s Set
	path, err := ReadPath(root)
	if err != nil {
		return s, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	defer f.Close()

	seen := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if s.Note == "" {
				s.Note = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			}
			continue
		}
		if !seen[line] {
			seen[line] = true
			s.Entries = append(s.Entries, line)
		}
	}
	return s, sc.Err()
}

// Save writes the working set back, creating .mrw/ if needed.
func Save(root string, s Set) error {
	path, err := state.Path(root, Name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	if s.Note != "" {
		fmt.Fprintf(&b, "# %s\n", s.Note)
	}
	for _, e := range s.Entries {
		b.WriteString(e)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// Add appends entries that are not already present and reports how many were
// new. Re-adding an entry is a no-op rather than an error: the caller is
// usually a loop that does not track what it has already said.
func (s *Set) Add(entries ...string) int {
	have := map[string]bool{}
	for _, e := range s.Entries {
		have[e] = true
	}
	n := 0
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" || have[e] {
			continue
		}
		have[e] = true
		s.Entries = append(s.Entries, e)
		n++
	}
	return n
}

// Remove drops entries. A bare path also removes every ranged entry for that
// path, so "stop working on this file" does not require repeating the ranges.
func (s *Set) Remove(entries ...string) int {
	drop := map[string]bool{}
	for _, e := range entries {
		drop[strings.TrimSpace(e)] = true
	}
	kept := s.Entries[:0]
	n := 0
	for _, e := range s.Entries {
		if drop[e] || drop[Path(e)] {
			n++
			continue
		}
		kept = append(kept, e)
	}
	s.Entries = kept
	return n
}

// Paths returns the distinct file paths in the set, with any range suffix
// stripped. This is what scopes a check: the tests care which files changed,
// not which lines were being read.
func (s Set) Paths() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range s.Entries {
		p := Path(e)
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// Sigil introduces a pointer into the working set. A path never starts with it,
// so "@3" can only ever mean "the third entry" — a number on its own would be a
// legal filename and would resolve silently to the wrong thing.
const Sigil = '@'

// IsPointer reports whether tok is a pointer rather than a literal spec.
func IsPointer(tok string) bool { return len(tok) > 1 && tok[0] == Sigil }

// Resolve expands one token. A literal spec passes through untouched; a pointer
// expands against the working set:
//
//	@2      the second entry, whole
//	@2-4    entries two through four
//	@*      every entry
//	@2:8-20 the second entry's PATH, with this range instead of its own
//
// Pointers are 1-based to match what `mrw iter` prints, and an out-of-range one
// is an error rather than an empty result: a pointer that quietly resolves to
// nothing is how a batch silently does less than it was asked to.
func (s Set) Resolve(tok string) ([]string, error) {
	if !IsPointer(tok) {
		return []string{tok}, nil
	}
	body := tok[1:]

	// An override range binds to the pointer's path: "@2:8-20".
	var override string
	if i := strings.Index(body, ":"); i >= 0 {
		body, override = body[:i], body[i+1:]
	}

	if body == "*" {
		if len(s.Entries) == 0 {
			return nil, fmt.Errorf("%s: the working set is empty", tok)
		}
		return append([]string(nil), s.Entries...), nil
	}

	lo, hi, err := s.pointerRange(tok, body)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		e := s.Entries[i-1]
		if override != "" {
			e = Path(e) + ":" + override
		}
		out = append(out, e)
	}
	return out, nil
}

// pointerRange parses the numeric part of a pointer and bounds-checks it.
func (s Set) pointerRange(tok, body string) (lo, hi int, err error) {
	num := func(t string) (int, error) {
		n, err := strconv.Atoi(t)
		if err != nil || n < 1 {
			return 0, fmt.Errorf("%s: %q is not a positive entry number", tok, t)
		}
		if n > len(s.Entries) {
			return 0, fmt.Errorf("%s: the working set has %d entr(ies)", tok, len(s.Entries))
		}
		return n, nil
	}
	a, b, ranged := strings.Cut(body, "-")
	if lo, err = num(a); err != nil {
		return 0, 0, err
	}
	if !ranged {
		return lo, lo, nil
	}
	if hi, err = num(b); err != nil {
		return 0, 0, err
	}
	if hi < lo {
		return 0, 0, fmt.Errorf("%s: range ends before it starts", tok)
	}
	return lo, hi, nil
}

// ResolveAll expands every token, flattening pointer ranges in place.
func (s Set) ResolveAll(tokens []string) ([]string, error) {
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		got, err := s.Resolve(t)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	return out, nil
}

// Path strips a spec's range suffix. It mirrors the reader's rule: the path is
// everything before the last colon, unless the colon introduces a /pattern/.
func Path(spec string) string {
	i := strings.LastIndex(spec, ":")
	if i <= 0 {
		return spec
	}
	if j := strings.Index(spec, ":/"); j > 0 && j < i {
		return spec[:j]
	}
	return spec[:i]
}

// ReadPath is where the working set is READ from: the state directory, falling
// back to a legacy in-tree file when the state directory holds none. Writes
// always go to the state directory — see Save.
func ReadPath(root string) (string, error) {
	p, err := state.Path(root, Name)
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return p, nil
	}
	legacy := state.LegacyPath(root, Name)
	if fi, err := os.Stat(legacy); err == nil && !fi.IsDir() {
		return legacy, nil
	}
	return p, nil
}
