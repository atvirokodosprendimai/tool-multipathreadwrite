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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
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

// Load reads the working set under its lock, so it never sees a file another
// process has emptied to rewrite (ADR-079). A missing file yields an empty set
// and no error: having no iteration is the normal starting state, not a fault.
// A lock that cannot be taken — an unwritable state directory — is read past:
// the set is still worth reading.
func Load(root string) (Set, error) {
	if release, err := state.Hold(root, lockName); err == nil {
		defer release()
	}
	return load(root)
}

// Update reads the working set, lets fn change it, and writes it back, all
// under its lock (ADR-079). Two `mrw iter add` racing each other each read the
// set, added, and wrote it back, and the later write lost the earlier entry —
// or read the file mid-rewrite, found it empty, and wiped the set. fn's error
// is returned as it is and nothing is written.
func Update(root string, fn func(*Set) error) (Set, error) {
	release, err := state.Hold(root, lockName)
	if err != nil {
		return Set{}, err
	}
	defer release()
	s, err := load(root)
	if err != nil {
		return s, err
	}
	if err := fn(&s); err != nil {
		return s, err
	}
	return s, Save(root, s)
}

// lockName is the working set's lock, beside it in the state directory.
const lockName = Name + ".lock"

// load reads the working set without the lock; Load and Update hold it.
func load(root string) (Set, error) {
	var s Set
	path, err := ReadPath(root)
	if err != nil {
		return s, err
	}
	// ADR-109: a legacy working set sits in the checkout; one that is not a
	// regular file holds nothing, and is not waited on.
	f, _, err := regular.Open(path)
	if os.IsNotExist(err) || errors.Is(err, regular.ErrNotRegular) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	defer func() { _ = f.Close() }()

	seen := map[string]bool{}
	r := bufio.NewReader(f)
	for {
		line, skip, err := nextLine(r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return s, err
		}
		if skip {
			continue // ADR-108: past the bound Save keeps, so not mrw's own
		}
		// Trimmed only to recognise a blank line or the note; an entry is kept
		// as written, so "x " does not come back as x (ADR-069).
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "#") {
			if s.Note == "" {
				s.Note = strings.TrimSpace(strings.TrimPrefix(t, "#"))
			}
			continue
		}
		if !seen[line] {
			seen[line] = true
			s.Entries = append(s.Entries, line)
		}
	}
	return s, nil
}

// maxEntryBytes bounds one line of the working set, on save and on load alike
// (ADR-108). bufio.Scanner's default stopped load at ~64 KiB while Save wrote
// any length, so a long note saved and then failed every later load — every
// write, and `mrw iter clear` itself.
const maxEntryBytes = 64 << 10

// nextLine returns r's next line as bufio.ScanLines did: "\n" ends a line, a
// "\r" before it is dropped, and a final line needs no terminator (its trailing
// "\r" is dropped too). A line longer than maxEntryBytes is consumed whole and
// reported as skip, so the lines after it still load; at most the bound plus
// one buffer is held. ReadSlice, not ReadLine: ReadLine's EOF after a final
// fragment that filled the buffer carries no data, and lost that line.
func nextLine(r *bufio.Reader) (string, bool, error) {
	var b []byte
	n, skip := 0, false
	for {
		chunk, err := r.ReadSlice('\n')
		n += len(chunk)
		if !skip {
			b = append(b, chunk...)
			if len(b) > maxEntryBytes+2 { // past the bound even without "\r\n"
				skip, b = true, nil
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err != nil && (!errors.Is(err, io.EOF) || n == 0) {
			return "", false, err
		}
		if skip {
			return "", true, nil
		}
		line := strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
		if len(line) > maxEntryBytes {
			return "", true, nil
		}
		return line, false, nil
	}
}

// Save writes the working set back, creating its directory if needed. It takes
// no lock of its own: Update holds it around the read, the change and this write.
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
	// ADR-108: a record load could not read back is refused, naming it; the
	// caller typed it, so it is not dropped the way a ledger record is.
	if len("# "+s.Note) > maxEntryBytes { // the line Save writes, which load bounds
		return fmt.Errorf("the note is %d bytes, over the %d-byte limit the working set keeps for it", len(s.Note), maxEntryBytes-len("# "))
	}
	for _, e := range s.Entries {
		if len(e) > maxEntryBytes {
			return fmt.Errorf("an entry is %d bytes, over the %d-byte limit the working set keeps", len(e), maxEntryBytes)
		}
	}
	if s.Note != "" {
		fmt.Fprintf(&b, "# %s\n", s.Note)
	}
	for _, e := range s.Entries {
		b.WriteString(e)
		b.WriteByte('\n')
	}
	return state.Write(path, []byte(b.String()), 0o600)
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
		if strings.TrimSpace(e) == "" || have[e] {
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
		drop[e] = true
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
