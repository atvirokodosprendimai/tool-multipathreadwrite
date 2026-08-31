// Package read serves line ranges from many files in one pass.
//
// The economics it exists for: a 400-line file read whole costs roughly 6k
// tokens, while the 30 lines that answer the question cost about 400. Reading
// ranges is only half of it though — the other half is that N independent
// questions should cost ONE round trip, which is why a request is a list of
// specs rather than a single path.
//
// The output deliberately echoes the mrw plan format: a range printed as
// "@@ 3-6" is the address you write back in a plan, so a read and the edit it
// informs share one vocabulary.
package read

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Spec is one request: a path plus the ranges wanted from it. No ranges means
// the whole file.
type Spec struct {
	Path   string
	Ranges []Range
	Raw    string
}

// Range is one span of a file. Exactly one of the numeric or regexp form is
// populated: Re is nil for a numeric range.
type Range struct {
	Start int // 1-based inclusive; 0 means "from the first line"
	End   int // 1-based inclusive; 0 means "to the last line"
	Re    *regexp.Regexp
	ReEnd *regexp.Regexp // nil for a single-pattern match
	Text  string         // the range as written, for diagnostics
}

// Options control how much of the answer is rendered. They exist so a caller
// can ask the cheapest question that still decides what to do next.
type Options struct {
	// Numbers prefixes each line with its number. On by default because a line
	// number read here is the address written back in a plan.
	Numbers bool
	// Stat prints only the per-file header: lines, bytes and SHA-256 prefix.
	// This is "ask for the fact, not the artifact" — it answers "did this file
	// change" and "how long is it" without paying for the content.
	Stat bool
	// Context adds N lines either side of a single-pattern match.
	Context int
	// MaxLines caps the lines emitted per file. A cap that fires is always
	// reported: a silent truncation reads as "that was the whole file".
	MaxLines int
}

// ParseSpec reads "path", "path:3-6", "path:1-8,100-130" or
// "path:/func Foo/,/^}/". The path is everything before the LAST colon that is
// followed by something range-shaped, so a path containing a colon still works.
func ParseSpec(s string) (Spec, error) {
	spec := Spec{Path: s, Raw: s}
	i := strings.LastIndex(s, ":")
	if i <= 0 {
		return spec, nil
	}
	path, rest := s[:i], s[i+1:]
	if rest == "" {
		return Spec{}, fmt.Errorf("%q: empty range after ':'", s)
	}
	// A colon inside a regexp range is part of the pattern, not a separator.
	if j := strings.Index(s, ":/"); j > 0 && j < i {
		path, rest = s[:j], s[j+1:]
	}
	spec.Path = path
	for _, part := range splitRanges(rest) {
		r, err := parseRange(part)
		if err != nil {
			return Spec{}, fmt.Errorf("%q: %w", s, err)
		}
		spec.Ranges = append(spec.Ranges, r)
	}
	return spec, nil
}

// splitRanges splits on commas that separate ranges, keeping both a single
// /pattern/ and the paired /start/,/end/ form intact. The comma inside the
// paired form is part of the range, not a separator, so the scanner has to look
// ahead one character after a pattern closes.
func splitRanges(s string) []string {
	var (
		out []string
		cur strings.Builder
	)
	for i := 0; i < len(s); {
		switch s[i] {
		case '/':
			j := i + 1
			for j < len(s) && (s[j] != '/' || s[j-1] == '\\') {
				j++
			}
			if j < len(s) {
				j++ // include the closing slash
			}
			cur.WriteString(s[i:j])
			i = j
			// "/a/,/b/" is one range: swallow the comma and keep scanning.
			if i+1 < len(s) && s[i] == ',' && s[i+1] == '/' {
				cur.WriteByte(',')
				i++
			}
		case ',':
			out = append(out, cur.String())
			cur.Reset()
			i++
		default:
			cur.WriteByte(s[i])
			i++
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func parseRange(s string) (Range, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Range{}, fmt.Errorf("empty range")
	}
	if strings.HasPrefix(s, "/") {
		// "/a/,/b/" arrives as one part because splitRanges kept it together.
		pats := strings.Split(strings.TrimSuffix(strings.TrimPrefix(s, "/"), "/"), "/,/")
		re, err := regexp.Compile(pats[0])
		if err != nil {
			return Range{}, fmt.Errorf("bad pattern %q: %w", pats[0], err)
		}
		r := Range{Re: re, Text: s}
		if len(pats) == 2 {
			if r.ReEnd, err = regexp.Compile(pats[1]); err != nil {
				return Range{}, fmt.Errorf("bad end pattern %q: %w", pats[1], err)
			}
		}
		return r, nil
	}
	num := func(t string) (int, error) {
		if t == "$" || t == "" {
			return 0, nil
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 1 {
			return 0, fmt.Errorf("bad line number %q", t)
		}
		return n, nil
	}
	lo, hi, ranged := strings.Cut(s, "-")
	start, err := num(lo)
	if err != nil {
		return Range{}, err
	}
	if !ranged {
		return Range{Start: start, End: start, Text: s}, nil
	}
	end, err := num(hi)
	if err != nil {
		return Range{}, err
	}
	if start == 0 {
		start = 1
	}
	if end != 0 && end < start {
		return Range{}, fmt.Errorf("range %q ends before it starts", s)
	}
	return Range{Start: start, End: end, Text: s}, nil
}

// Run renders every spec to w. It returns the number of specs that could not be
// served; a missing file is reported in the output rather than aborting the
// batch, because the other N-1 answers are still worth having.
func Run(w io.Writer, root string, specs []Spec, opt Options) (problems int) {
	for _, sp := range specs {
		b, err := os.ReadFile(filepath.Join(root, sp.Path))
		if err != nil {
			fmt.Fprintf(w, "==> %s  UNREADABLE  %v\n", sp.Path, err)
			problems++
			continue
		}
		lines, sha := split(b)
		fmt.Fprintf(w, "==> %s  %dL  %dB  sha %s\n", sp.Path, len(lines), len(b), sha[:8])
		if opt.Stat {
			continue
		}
		spans, missed := resolve(sp.Ranges, lines, opt.Context)
		for _, m := range missed {
			fmt.Fprintf(w, "!! no match for %s\n", m)
			problems++
		}
		budget := opt.MaxLines
		for _, sn := range spans {
			n := sn.end - sn.start + 1
			if opt.MaxLines > 0 && budget <= 0 {
				fmt.Fprintf(w, "@@ %d-%d  WITHHELD %d line(s): --max-lines reached\n", sn.start, sn.end, n)
				problems++
				continue
			}
			cut := 0
			if opt.MaxLines > 0 && n > budget {
				cut, n = n-budget, budget
			}
			fmt.Fprintf(w, "@@ %d-%d\n", sn.start, sn.start+n-1)
			for i := 0; i < n; i++ {
				if opt.Numbers {
					fmt.Fprintf(w, "%5d| %s\n", sn.start+i, lines[sn.start+i-1])
				} else {
					fmt.Fprintln(w, lines[sn.start+i-1])
				}
			}
			if cut > 0 {
				fmt.Fprintf(w, "!! %d more line(s) withheld: --max-lines reached\n", cut)
				problems++
			}
			budget -= n
		}
	}
	return problems
}

type span struct{ start, end int }

// resolve turns ranges into concrete, merged, ascending spans and reports which
// patterns matched nothing. Merging matters: overlapping context windows would
// otherwise print the same lines twice and pay for them twice.
func resolve(ranges []Range, lines []string, ctx int) ([]span, []string) {
	total := len(lines)
	if total == 0 {
		return nil, nil
	}
	if len(ranges) == 0 {
		return []span{{1, total}}, nil
	}

	var (
		spans  []span
		missed []string
	)
	for _, r := range ranges {
		switch {
		case r.Re != nil && r.ReEnd != nil:
			found := false
			for i := 0; i < total; i++ {
				if !r.Re.MatchString(lines[i]) {
					continue
				}
				end := total
				for j := i; j < total; j++ {
					if j > i && r.ReEnd.MatchString(lines[j]) {
						end = j + 1
						break
					}
				}
				spans = append(spans, span{i + 1, end})
				found = true
				i = end - 1
			}
			if !found {
				missed = append(missed, r.Text)
			}
		case r.Re != nil:
			found := false
			for i, l := range lines {
				if !r.Re.MatchString(l) {
					continue
				}
				spans = append(spans, span{max(1, i+1-ctx), min(total, i+1+ctx)})
				found = true
			}
			if !found {
				missed = append(missed, r.Text)
			}
		default:
			start, end := r.Start, r.End
			if start == 0 {
				start = 1
			}
			if end == 0 || end > total {
				end = total
			}
			if start > total {
				missed = append(missed, fmt.Sprintf("%s (file has %d lines)", r.Text, total))
				continue
			}
			spans = append(spans, span{start, end})
		}
	}
	return merge(spans), missed
}

func merge(in []span) []span {
	if len(in) < 2 {
		return in
	}
	sorted := append([]span(nil), in...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].start < sorted[j-1].start; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	out := []span{sorted[0]}
	for _, s := range sorted[1:] {
		last := &out[len(out)-1]
		if s.start <= last.end+1 {
			last.end = max(last.end, s.end)
			continue
		}
		out = append(out, s)
	}
	return out
}

func split(b []byte) ([]string, string) {
	sum := sha256.Sum256(b)
	sha := hex.EncodeToString(sum[:])
	if len(b) == 0 {
		return nil, sha
	}
	s := strings.TrimSuffix(string(b), "\n")
	return strings.Split(s, "\n"), sha
}
