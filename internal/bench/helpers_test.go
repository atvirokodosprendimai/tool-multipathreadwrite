package bench

import (
	"sort"
	"strings"
)

// Helpers only the tests need (the static gate keeps them out of the package
// proper, ADR-088).

// Files are the files the reference fix touches, sorted.
func (t Task) Files() []string {
	var fs []string
	for f := range t.regions {
		fs = append(fs, f)
	}
	sort.Strings(fs)
	return fs
}

// GradedFiles are the files the oracle reads: the touched ones and every file
// a require or forbid names.
func (t Task) GradedFiles() []string {
	seen := map[string]bool{}
	var fs []string
	for _, f := range t.Files() {
		seen[f] = true
		fs = append(fs, f)
	}
	for _, m := range append(append([]Match(nil), t.Require...), t.Forbid...) {
		if !seen[m.File] {
			seen[m.File] = true
			fs = append(fs, m.File)
		}
	}
	sort.Strings(fs)
	return fs
}

// FixedLine maps base line n of file, one the reference fix leaves alone, to
// its number once the fix is applied.
func (t Task) FixedLine(file string, n int) int {
	shift := 0
	for _, h := range t.regions[file] {
		end := h.Start + h.Count - 1
		if h.Count == 0 {
			end = h.Start
		}
		if end < n {
			shift += h.Added - h.Count
		}
	}
	return n + shift
}

// IsInstructionPath says whether any component of a slash-separated path is
// one the runner strips.
func IsInstructionPath(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if instructionNames[part] {
			return true
		}
	}
	return false
}

// ApplyPatch applies a patch to the tree at dir.
func ApplyPatch(dir, patch string) error {
	return git(dir, "apply", "--unidiff-zero", "--whitespace=nowarn", patch)
}

// neighbour is the base line just before the reference fix's first change in
// its first touched file, when the fix itself leaves that line alone. It reads
// the fix's hunks directly, never CoversBaseLine: a test that asked the rule
// under test which line to try would skip the very line a broken rule licenses.
func (t Task) neighbour() (string, int, bool) {
	for _, f := range t.Files() {
		hs := t.regions[f]
		if len(hs) == 0 || t.created[f] {
			continue
		}
		n := hs[0].Start - 1
		if hs[0].Count == 0 {
			n = hs[0].Start
		}
		changed := false
		for _, h := range hs {
			if h.Count > 0 && n >= h.Start && n < h.Start+h.Count {
				changed = true
			}
		}
		if n >= 1 && !changed {
			return f, n, true
		}
	}
	return "", 0, false
}
