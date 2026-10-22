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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/addr"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// maxFileBytes is the largest file read serves or a --grep walk searches
// (ADR-104). Both read a file whole to answer any range of it, so a larger one
// is refused by its size rather than held in memory. A variable so a test can
// lower it.
var maxFileBytes int64 = 1 << 30

// overFileCap is the refusal of a file of size bytes, over maxFileBytes.
func overFileCap(size int64) string {
	return fmt.Sprintf("%d bytes, over the %d-byte limit mrw reads a file whole to (ADR-104): read a smaller file, or split this one", size, maxFileBytes)
}

// errOverFileCap is readCapped's refusal. Exact says the size came from a stat;
// otherwise the file grew past the limit while it was read, and size is where
// the read stopped.
type errOverFileCap struct {
	size  int64
	exact bool
}

func (e errOverFileCap) Error() string {
	if e.exact {
		return overFileCap(e.size)
	}
	return fmt.Sprintf("more than %d bytes by the time it was read, over the limit mrw reads a file whole to (ADR-104): read a smaller file, or split this one", maxFileBytes)
}

// readCapped reads path whole, up to maxFileBytes. It refuses by the file's
// size first, and then reads through a limit of one byte more, because a size
// taken before the read is outrun by a file that grows (the review of #295).
// It opens through regular.Open (ADR-109): a file swapped for a FIFO after the
// caller's path check is refused rather than blocking the read.
func readCapped(path string) ([]byte, error) {
	f, fi, err := regular.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	if fi.Mode().IsRegular() && fi.Size() > maxFileBytes {
		return nil, errOverFileCap{size: fi.Size(), exact: true}
	}
	b, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxFileBytes {
		return nil, errOverFileCap{size: int64(len(b))}
	}
	return b, nil
}

// Sentinels for an address end that is not a plain line number. They are
// distinct because they resolve differently: an omitted end is unbounded in
// whichever direction it appears, while `$` is one specific line — the last.
// Sharing 0 for both is what made `f.txt:$` serve the whole file.
const (
	unbounded = 0
	lastLine  = -1
)

// Spec is one request: a path plus the ranges wanted from it. No ranges means
// the whole file.
type Spec struct {
	Path   string
	Ranges []Range
	Raw    string
	// Data, when not nil, is what the read serves for Path instead of a file:
	// standard input, read once by the caller (ADR-161). Nothing resolves it on
	// disk, and the caller records no observation of it.
	Data []byte
}

// Range is one span of a file. Exactly one of the numeric or regexp form is
// populated: Re is nil for a numeric range.
type Range struct {
	Start int // 1-based inclusive; 0 means "from the first line"
	End   int // 1-based inclusive; 0 means "to the last line"
	Re    *regexp.Regexp
	ReEnd *regexp.Regexp // nil for a single-pattern match
	Text  string         // the range as written, for diagnostics
	// RelEnd is the `A,+N` form (ADR-026): the range ends N lines after its
	// start. 0 means absent, which is unambiguous because `,+0` is refused —
	// no caller can mean it.
	RelEnd int
	// JSON is a JSON path address (ADR-155): the steps from the document's
	// root to one value, whose lines the range serves. Nil for every other form.
	JSON []jsonStep
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
	// MaxLines caps the lines emitted per SPEC — the budget resets for each one,
	// so two specs naming one file get two budgets — and NIL is how a caller says
	// "no cap" (ADR-033). A pointer, not a sentinel: zero used to mean unlimited
	// because both guards asked `> 0`, so there was no way to say "serve me the
	// header and nothing else" and — worse — nothing was reported withheld,
	// though the README promises whatever is withheld always is. This
	// repository decided the same question the other way for `body=0`
	// (ADR-027) and `lines=0`.
	//
	// ⚠ A numeric sentinel would invert the danger: the struct's zero value
	// would mean "serve nothing", and the three call sites that omit this field
	// would silently stop serving. Nil is what omission already meant.
	//
	// A cap that fires is always
	// reported: a silent truncation reads as "that was the whole file".
	MaxLines *int
	// MaxCols cuts a line longer than this many characters to a window of that
	// width round the first match of the spec's pattern (the start when the range
	// has none), and records the line as NOT read: what was cut was not shown
	// (ADR-137). Zero is no cut, so an Options that omits it serves whole lines.
	MaxCols int
	// Count answers a --grep with how many lines match in each file instead of
	// the lines (ADR-153): each file's header carries its count, a total line
	// ends the answer, and the file is recorded as Stat records it — hashed,
	// none of it shown — so a count licenses no line.
	Count bool
	// Counts, when Count is set and this is not nil, receives each counted
	// file's count, keyed as the MCP receipt keys `observed`: cleaned and
	// spelled with "/".
	Counts map[string]int
	// Stop, when set, is asked before each spec; true ends the read there. A
	// caller that has already decided to refuse the answer stops paying for the
	// rest of it (ADR-078). Nil reads every spec.
	Stop func() bool
	// Sub, when set, makes the read print a plan instead of lines (ADR-161):
	// w then carries only what could not be served, and Sub the plan. Only the
	// lines planned are recorded as read.
	Sub *Sub
	// Quiet records what the same read without it records and prints no line
	// of it (ADR-161): each file's header, problems, and at the end
	// `-- licensed N line(s) in M file(s)`. For lines the caller already knows,
	// without a scratch file and without the null device, which licenses nothing.
	Quiet bool
	// NextLine serves the line after each range that ends before the file does
	// (ADR-161 Decision 10). The CLI read and mrw_read set it. A page of mrw_read
	// does not: its continuation serves the line a page ends on, and a page must
	// not repeat a line.
	NextLine bool
	// Withheld receives, when w is a *Budget (ADR-165), one line per range or
	// file the budget kept from the caller, and every report line — a refusal,
	// a --max-lines notice, a missed pattern, the quiet count — the budget
	// dropped, so nothing is unsaid because it did not fit. It is not charged:
	// a budget that silenced its own report would be ADR-032's elided receipt
	// on the CLI.
	Withheld io.Writer
}

// Budget is a writer with a character budget (ADR-165): a write that fits
// passes to W and is charged; the first that does not is dropped, and every
// later one with it, so W receives a prefix of what was meant, cut at a write.
// Every write Run makes ends a line, so that prefix is cut at a line. A caller
// passes Run a *Budget as w to bound the answer; Run then licenses only what
// reached W, and names on Options.Withheld what did not, without opening a
// file it never reached. A caller that writes its own lines on the same stream
// writes them through the same Budget, and through Reporting where they report.
type Budget struct {
	W    io.Writer
	Left int
	over bool
}

// NewBudget is w with a budget of n characters.
func NewBudget(w io.Writer, n int) *Budget { return &Budget{W: w, Left: n} }

// Write charges p in runes, or drops it once the budget cannot hold it.
func (b *Budget) Write(p []byte) (int, error) {
	if b.over {
		return len(p), nil
	}
	if n := utf8.RuneCount(p); n <= b.Left {
		b.Left -= n
		return b.W.Write(p)
	}
	b.over = true
	return len(p), nil
}

// Over says whether a write has been dropped. A nil Budget never is.
func (b *Budget) Over() bool { return b != nil && b.over }

// exhausted says whether nothing more can be written: a write was dropped, or
// none is left. A file met then is named without being opened, and the budget
// counts as reached, since what it names was kept from the caller.
func (b *Budget) exhausted() bool {
	if b == nil {
		return false
	}
	if b.Left <= 0 {
		b.over = true
	}
	return b.over
}

// Reporting is w for a line that reports — a refusal, a withheld notice, a
// count: when w is a *Budget that drops the line, the line goes to to as well,
// so the report survives the budget. Any other w is returned as it is.
func Reporting(w, to io.Writer) io.Writer {
	if b, ok := w.(*Budget); ok {
		return &spill{b: b, to: to}
	}
	return w
}

type spill struct {
	b  *Budget
	to io.Writer
}

func (s *spill) Write(p []byte) (int, error) {
	if n, err := s.b.Write(p); err != nil {
		return n, err
	}
	if s.b.over {
		return s.to.Write(p)
	}
	return len(p), nil
}

// msysHint recognises MSYS2 argument conversion, which rewrites a spec BEFORE
// mrw is started. In Git Bash `f.go:/^func main/` arrives as
// `f.go;C:\...\Git\^func main\`: MSYS reads `a:b` as a POSIX path LIST and
// converts it, and `/^func main/` looks root-relative so it is expanded against
// the Git installation prefix.
//
// Quoting does not prevent it — the conversion happens in the process-spawn
// layer, after the shell has finished — so the README's own quoted example
// fails verbatim on the shell the docs recommend for Windows, and the parse
// error names a line number the caller never typed (issue #45).
//
// The signature is the PAIR: a ';' where the caller wrote ':', and a Windows
// path. Neither alone is enough — a ';' can sit inside a regex and a drive
// letter is legitimate. A regex containing both could still trip it, and that
// is acceptable: the hint is only ever appended to an error that was already
// going to be returned, so the worst case is one extra sentence on a spec that
// failed anyway.
func msysHint(s string) string {
	if !strings.Contains(s, ";") || !strings.Contains(s, `\`) {
		return ""
	}
	return " — this looks like MSYS2 argument conversion (Git Bash): your ':' became ';' " +
		"and a /pattern/ was expanded against the Git install prefix. Quoting does not stop it. " +
		"Set MSYS_NO_PATHCONV=1 or MSYS2_ARG_CONV_EXCL='*', or use PowerShell or WSL"
}

// readElsewhere is the advice for a file the root does not hold: ": " and the
// command that reads it, the directory it is in as the root and its name, with
// the ranges the caller wrote after path in raw, behind -- so a file called
// --help or " x" is still a file (ADR-069). The refusal that names the flag
// alone sent a caller to sed. It is "" when no --root serves the file: it is not
// a regular file, or it is mrw's own state, which no root lifts (ADR-157).
func readElsewhere(file, raw, path string) string {
	if fi, err := os.Stat(file); err != nil || !fi.Mode().IsRegular() || rooted.InState(file) {
		return ""
	}
	ranges := ""
	if strings.HasPrefix(raw, path) {
		ranges = raw[len(path):]
	}
	win := runtime.GOOS == "windows"
	// ./ keeps the name a file: never a flag (-), standard input (- or -:), a working-set pointer
	// (@) or a Windows drive (a:). A spec that does not parse back to this file (a colon in a name
	// given no range) cannot name it, so no command is printed (Codex re-checks of 59d1cc14, 3da5a009).
	name := "./" + filepath.Base(file)
	if sp, err := ParseSpec(name + ranges); err != nil || sp.Path != name {
		return ""
	}
	return ": mrw --root " + quoteWord(filepath.Dir(file), win) + " read -- " + quoteWord(name+ranges, win)
}

// quoteWord quotes s for the shell a caller pastes it into, only when s holds a
// character that shell would take for syntax: a space, a quote, a backslash
// (Git Bash strips it from a bare path), a pattern's bars. POSIX shells spell a
// single quote inside single quotes by closing, escaping and reopening them;
// PowerShell, which a Windows caller runs, doubles it.
func quoteWord(s string, windows bool) string {
	safe := s != ""
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_-./:,+=@%", r)) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	if windows {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// withheldName is how the withheld report names a spec: the spec as the caller
// wrote it when it carries ranges, so two withheld ranges of one file are two
// different lines, else the path.
func withheldName(sp Spec) string {
	if len(sp.Ranges) > 0 && sp.Raw != "" {
		return sp.Raw
	}
	return sp.Path
}

// ParseSpec reads "path", "path:3-6", "path:1-8,100-130" or
// "path:/func Foo/,/^}/". Everything after the LAST colon is the range, and it
// must parse as one: a path that itself contains a colon is NOT supported, and
// is reported as a bad range rather than read as a filename. That is deliberate
// — reading an unparseable tail as part of the path would turn a typo'd range
// into a missing file, and a typo is the likelier of the two by far.
func ParseSpec(s string) (Spec, error) {
	spec := Spec{Path: s, Raw: s}
	// Skip the VOLUME before looking for the range separator. On Windows a
	// drive letter carries a colon — `C:\dir\f.go` — and LastIndex found it,
	// so the path became "C" and the range became "\dir\f.go": every absolute
	// Windows path was unparseable, with a "bad line number" naming most of
	// the path. Not a --grep problem: plain `mrw read C:\dir\f.go` could not
	// work either, on the platform this project ships a binary for.
	//
	// filepath.VolumeName returns "" on POSIX, so this is a no-op there. Found
	// by the windows CI job, 2026-09-03 — its third real finding.
	vol := len(filepath.VolumeName(s))
	i := strings.LastIndex(s[vol:], ":")
	if i < 0 {
		return spec, nil
	}
	i += vol
	if i == 0 {
		return spec, nil
	}
	path, rest := s[:i], s[i+1:]
	if rest == "" {
		return Spec{}, fmt.Errorf("%q: empty range after ':'%s", s, msysHint(s))
	}
	// A colon inside a regexp range is part of the pattern, not a separator.
	if j := strings.Index(s[vol:], ":/"); j >= 0 && j+vol > 0 && j+vol < i {
		path, rest = s[:j+vol], s[j+vol+1:]
	}
	spec.Path = path
	for _, part := range splitRanges(rest) {
		r, err := parseRange(part)
		if err != nil {
			return Spec{}, fmt.Errorf("%q: %w%s", s, err, msysHint(s))
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
			// Inside a JSON path a slash is part of a quoted name, never a
			// pattern delimiter: `$["x/y"]` is one range (ADR-155).
			if strings.HasPrefix(cur.String(), "$") {
				cur.WriteByte('/')
				i++
				continue
			}
			// One scanner for every pattern delimiter in this repository, so
			// splitRanges and the two parsers cannot disagree about where a
			// pattern ends (ADR-026, third review of #125).
			j := addr.ClosingDelim(s, i)
			if j < 0 {
				j = len(s)
			} else {
				j++ // include the closing slash
			}
			cur.WriteString(s[i:j])
			i = j
			// "/a/,/b/" is one range: swallow the comma and keep scanning.
			//
			// `,+N` is deliberately NOT handled here as well. It reads as the
			// symmetrical case and it is unreachable by difference: when this
			// does not swallow the comma, the loop's next pass reaches the
			// `case ','` below, which continues the range for a `+`. Adding
			// the clause here was mutated on 2026-09-06 and the mutant
			// SURVIVED — nothing can tell the two versions apart, so the
			// clause was removed rather than left as code no test can reach.
			if i+1 < len(s) && s[i] == ',' && s[i+1] == '/' {
				cur.WriteByte(',')
				i++
			}
		case ',':
			// A comma followed by `+` continues the range: `A,+N` is one
			// address carrying a relative end (ADR-026). Every other comma
			// separates two addresses of the same file.
			if i+1 < len(s) && s[i+1] == '+' {
				cur.WriteByte(',')
				i++
				continue
			}
			out = append(out, cur.String())
			cur.Reset()
			i++
			// A comma at the very end leaves an empty component, and dropping
			// it silently turned `5,+2,` into `5,+2` on the read path while the
			// plan path refused the string whole. Emit the empty range so
			// parseRange refuses it, in the same words, on both. Found by the
			// fifth Codex review of PR #125.
			if i == len(s) {
				out = append(out, "")
			}
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
	// `A,+N` is the N lines after A (ADR-026). internal/addr cuts it, so this
	// path and the plan path recognise and REFUSE exactly the same strings —
	// they are one function, not two copies a contract row watches.
	raw := s
	base, rel, err := addr.CutRelative(s)
	if err != nil {
		return Range{}, err
	}
	s = base
	if isJSONPath(s) {
		if rel > 0 {
			return Range{}, fmt.Errorf("%q: a JSON path names its own span, so it takes no relative end", raw)
		}
		steps, err := parseJSONPath(s)
		if err != nil {
			return Range{}, err
		}
		return Range{JSON: steps, Text: raw}, nil
	}
	if strings.HasPrefix(s, "/") {
		// SCANNED, NOT SPLIT. TrimSuffix+Split accepted `/` as an empty regexp
		// matching every line at exit 0, compiled `/a/garbage` as the pattern
		// `a/garbage`, and silently reduced `/a/,/b/,/c/` to its first
		// endpoint. The plan path refused all three, so the two grammars
		// disagreed on exactly the inputs nobody writes on purpose — and the
		// record claimed they could not. Found by the fourth Codex review of
		// PR #125.
		end := addr.ClosingDelim(s, 0)
		if end < 0 {
			return Range{}, fmt.Errorf("pattern %q is never closed: expected a second /", raw)
		}
		body := s[1:end]
		if body == "" {
			return Range{}, fmt.Errorf("empty pattern // matches every line, so it addresses nothing in particular — name the file alone if you want all of it")
		}
		re, err := regexp.Compile(body)
		if err != nil {
			return Range{}, fmt.Errorf("bad pattern %q: %w", body, err)
		}
		r := Range{Re: re, Text: raw, RelEnd: rel}
		rest := s[end+1:]
		switch {
		case rest == "":
			return r, nil
		case strings.HasPrefix(rest, ",/"):
			if rel > 0 {
				return Range{}, fmt.Errorf("%q has both an end pattern and a relative end: write /from/,/to/ or A,+N, not both", raw)
			}
			e2 := addr.ClosingDelim(rest, 1)
			if e2 < 0 {
				return Range{}, fmt.Errorf("end pattern in %q is never closed: expected a second /", raw)
			}
			if tail := rest[e2+1:]; tail != "" {
				return Range{}, fmt.Errorf("%q has %q after the end pattern: an address takes one range, not three", raw, tail)
			}
			body2 := rest[2:e2]
			if body2 == "" {
				return Range{}, fmt.Errorf("empty pattern // matches every line, so it addresses nothing in particular — name the file alone if you want all of it")
			}
			if r.ReEnd, err = regexp.Compile(body2); err != nil {
				return Range{}, fmt.Errorf("bad end pattern %q: %w", body2, err)
			}
			return r, nil
		default:
			return Range{}, fmt.Errorf("%q has %q after the pattern: write /re/, /from/,/to/ or A,+N", raw, rest)
		}
	}
	// `$` and an OMITTED end are different addresses and used to share the
	// sentinel 0. Downstream 0 means "unbounded in whichever direction you
	// find it" — 1 at the start, EOF at the end — so `$` inherited that and a
	// bare `f.txt:$` served the WHOLE file. The README says "`$` is the last
	// line" and `plan.ParseAddr` on the write path agrees: `@@ f.txt $
	// replace` touches one line. read was the only reader of `$` that did not.
	num := func(t string) (int, error) {
		switch t {
		case "":
			return unbounded, nil
		case "$":
			return lastLine, nil
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
		return Range{Start: start, End: start, Text: raw, RelEnd: rel}, nil
	}
	end, err := num(hi)
	if err != nil {
		return Range{}, err
	}
	if start == unbounded {
		start = 1
	}
	// Only a range whose ends are both PLAIN numbers can be judged reversed
	// here. One written with `$` depends on how long the file turns out to be
	// — `$-5` is fine on a five-line file and reversed on a ten-line one — so
	// that verdict waits for resolve, which is the first place the length is
	// known.
	if start > 0 && end > 0 && end < start {
		return Range{}, fmt.Errorf("range %q ends before it starts", raw)
	}
	return Range{Start: start, End: end, Text: raw, RelEnd: rel}, nil
}

// Run renders every spec to w. It returns what it OBSERVED of every file it
// served — the whole-file sha, and the line spans actually printed — plus the
// number of specs it could not serve. A missing file is reported in the output
// rather than aborting the batch, because the other N-1 answers are still worth
// having.
//
// The spans are the point, not a detail. Reading a file is how mrw learns what
// it holds, and that observation is what later authorises an edit — so what is
// recorded has to be what the CALLER saw, not what mrw hashed. A --stat read
// prints no content and therefore observes nothing; a ranged read observes its
// ranges; a withheld span is not observed at all.
func Run(w io.Writer, root string, specs []Spec, opt Options) (observed map[string]seen.Observation, problems int) {
	observed = map[string]seen.Observation{}
	note := func(path, sha string, spans [][2]int) {
		key := filepath.Clean(path)
		// Keyed the same way apply keys its hunks, so "a.go" read and "./a.go"
		// written are one file rather than two ledger entries.
		prev, ok := observed[key]
		if ok && prev.SHA == sha {
			// A whole-file observation is never downgraded by a later partial
			// one: reading a file MORE thoroughly cannot leave the caller
			// having seen less. seen.merge says the same thing about the
			// ledger on disk; this is the same rule one step earlier, before
			// Record ever sees it.
			if prev.Spans == nil || spans == nil {
				observed[key] = seen.Observation{SHA: sha}
				return
			}
			spans = append(prev.Spans, spans...)
		}
		observed[key] = seen.Observation{SHA: sha, Spans: spans}
	}
	englishUnreadables := 0
	// ADR-153's total line.
	// The note is printed once per call, at the first range that needs it
	// (ADR-136), and only where the line after is not served (ADR-161): a page
	// of mrw_read, whose continuation serves it.
	neighbourNoted := false
	countedLines, countedFiles := 0, 0
	fileLines := map[string]int{} // each file's length, for a quiet read's count of whole files
	bud, _ := w.(*Budget)
	withheld := opt.Withheld
	if withheld == nil {
		withheld = io.Discard
	}
	report := Reporting(w, withheld)
	// reportOn writes a report line about path to w, exactly as an unbounded
	// read writes it; when the budget drops it, the line goes to withheld too,
	// prefixed with the path it is about, since on its own it names no file.
	reportOn := func(path, format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		fmt.Fprint(w, line)
		if bud.Over() {
			fmt.Fprintf(withheld, "%s: %s", path, line)
		}
	}
	for _, sp := range specs {
		if opt.Stop != nil && opt.Stop() {
			break
		}
		if bud.exhausted() { // never reached: named, not opened
			fmt.Fprintf(withheld, "%s: WITHHELD: --max-chars reached\n", withheldName(sp))
			problems++
			continue
		}
		var b []byte
		if sp.Data != nil {
			b = sp.Data // ADR-161: standard input, one unnamed file
		} else {
			var englishUnreadable, ok bool
			if b, englishUnreadable, ok = load(report, root, &sp); !ok {
				if englishUnreadable {
					englishUnreadables++
				}
				problems++
				continue
			}
		}
		// ADR-073: a file mrw cannot split into lines (a UTF-16 or UTF-32 BOM,
		// a NUL early on) is served as the bytes it is, with one note saying so:
		// the read did its job, and a write to it is refused.
		foreign := lines.Unsplittable(b)
		lines, sha := split(b)
		fileLines[filepath.Clean(sp.Path)] = len(lines)
		header := fmt.Sprintf("==> %s  %dL  %dB  sha %s", sp.Path, len(lines), len(b), sha[:8])
		// countMissed is the pattern a counted file no longer matches: the walk
		// found it, and another writer changed it before this read. A plain
		// grep reports that as a missed range, and so does a count (the Codex
		// review of #406), rather than a count of 0 at exit 0.
		countMissed := ""
		if opt.Count {
			n := matchingLines(sp.Ranges, lines)
			header += fmt.Sprintf("  %d matching line(s)", n)
			countedLines += n
			countedFiles++
			if opt.Counts != nil {
				opt.Counts[filepath.ToSlash(filepath.Clean(sp.Path))] = n
			}
			for _, r := range sp.Ranges {
				if n == 0 && r.Re != nil {
					countMissed = r.Text
					break
				}
			}
		}
		if opt.Sub == nil {
			fmt.Fprintln(w, header)
		}
		if bud.Over() { // the header did not fit: nothing of this file was shown
			fmt.Fprintf(withheld, "%s: WITHHELD: --max-chars reached\n", withheldName(sp))
			problems++
			continue
		}
		if foreign != "" {
			fmt.Fprintf(w, "-- note: %s %s: served as bytes; a write to it is refused\n", sp.Path, foreign)
		}
		if opt.Stat || opt.Count {
			// The fact, not the artifact — and a fact the caller cannot count
			// lines in. Recording an empty span set says exactly that: mrw has
			// hashed this file and shown you none of it.
			note(sp.Path, sha, [][2]int{})
			if countMissed != "" {
				reportOn(sp.Path, "!! no match for %s\n", countMissed)
				problems++
			}
			continue
		}
		spans, missed := resolve(sp.Ranges, lines, opt.Context, b)
		// served accumulates only what is actually PRINTED below, so a withheld
		// span never counts as seen. whole starts as a fact about the REQUEST
		// and is demoted the moment anything is withheld: --max-lines is the
		// flag a caller reaches for on a big file, which is precisely when they
		// cannot count the lines they were not shown.
		// EMPTY, not nil. seen.Observation reads a nil Spans as "the whole
		// file" and an empty-but-non-nil one as "hashed, and none of it
		// shown" — two different licences. A ranged read that printed nothing
		// left this nil and so recorded the WHOLE FILE as seen:
		// `mrw read f.txt:/nomatch/` served no lines, exited 1, and then
		// licensed `@@ f.txt 3 replace` on a line the caller was never shown.
		// A FAILED read granted strictly more than a successful partial one,
		// which inverts ADR-005.
		served := [][2]int{}
		whole := len(sp.Ranges) == 0
		for _, m := range missed {
			reportOn(sp.Path, "!! no match for %s\n", m)
			problems++
		}
		if opt.Sub != nil {
			planned := opt.Sub.file(sp.Path, sha, lines, spans)
			if len(planned) == 0 {
				note(sp.Path, sha, planned)
			} else {
				// Licensed and not shown (ADR-118): a plan's lines may be edited,
				// but occurrence=N counts only lines a read showed.
				observed[filepath.Clean(sp.Path)] = seen.Observation{SHA: sha, Spans: planned, Written: true, Shown: [][2]int{}}
			}
			continue
		}
		// ADR-161 Decision 10: a ranged read also serves the line after each
		// range that ends before the file does, so a multi-line replace of
		// exactly the lines read has the line below it licensed (ADR-052). It
		// retires ADR-136's note, which only named that line.
		if opt.NextLine && len(sp.Ranges) > 0 {
			spans = withNextLine(spans, len(lines))
		}
		capped := opt.MaxLines != nil
		budget := 0
		if capped {
			budget = *opt.MaxLines
		}
		var res []*regexp.Regexp // the patterns that chose the lines: where a cut window is centred (ADR-137)
		for _, r := range sp.Ranges {
			if r.Re != nil {
				res = append(res, r.Re)
			}
			if r.ReEnd != nil { // a /start/,/end/ range may end on the long line (the Codex review of #367)
				res = append(res, r.ReEnd)
			}
		}
		// withholdFrom names, on the withheld writer, what the character budget
		// kept from the caller: span si from line from to its end, and every
		// span after it (ADR-165).
		withholdFrom := func(si, from int) {
			for j, s := range spans[si:] {
				if j > 0 {
					from = s.start
				}
				fmt.Fprintf(withheld, "%s: lines %d-%d WITHHELD (%d line(s)): --max-chars reached\n", sp.Path, from, s.end, s.end-from+1)
				problems++
			}
			whole = false
		}
		stopped := false
		for si, sn := range spans {
			n := sn.end - sn.start + 1
			if capped && budget <= 0 {
				if opt.Quiet { // a quiet read prints no range line, a withheld one included
					reportOn(sp.Path, "!! lines %d-%d withheld (%d line(s)): --max-lines reached\n", sn.start, sn.end, n)
				} else {
					reportOn(sp.Path, "@@ %d-%d  WITHHELD %d line(s): --max-lines reached\n", sn.start, sn.end, n)
				}
				problems++
				whole = false
				continue
			}
			cut := 0
			if capped && n > budget {
				cut, n = n-budget, budget
				whole = false
			}
			if !opt.Quiet {
				fmt.Fprintf(w, "@@ %d-%d\n", sn.start, sn.start+n-1)
			}
			if bud.Over() { // the range line did not fit
				withholdFrom(si, sn.start)
				break
			}
			if servedEnd := sn.start + n - 1; n >= 2 && servedEnd < len(lines) && !neighbourNoted && !opt.Quiet && !opt.NextLine {
				neighbourNoted = true
				fmt.Fprintf(w, "-- note: a multi-line replace of %d-%d needs a served line after %d\n",
					sn.start, servedEnd, servedEnd)
			}
			// ADR-137: a line cut to a window was not shown whole, so it is left out
			// of what is recorded as read; the lines either side of it, shown whole,
			// are recorded as two spans.
			pieceStart := sn.start
			for i := 0; i < n; i++ {
				ln := sn.start + i
				text := lines[ln-1]
				if opt.MaxCols > 0 && utf8.RuneCountInString(text) > opt.MaxCols {
					if ln > pieceStart {
						served = append(served, [2]int{pieceStart, ln - 1})
					}
					pieceStart = ln + 1
					whole = false
					text = window(text, opt.MaxCols, res)
				}
				switch {
				case opt.Quiet:
				case opt.Numbers:
					fmt.Fprintf(w, "%5d| %s\n", ln, text)
				default:
					fmt.Fprintln(w, text)
				}
				if bud.Over() { // line ln did not fit: it and the rest are withheld
					if ln-1 >= pieceStart {
						served = append(served, [2]int{pieceStart, ln - 1})
					}
					withholdFrom(si, ln)
					stopped = true
					break
				}
			}
			if stopped {
				break
			}
			if end := sn.start + n - 1; end >= pieceStart {
				served = append(served, [2]int{pieceStart, end})
			}
			if cut > 0 {
				reportOn(sp.Path, "!! %d more line(s) withheld: --max-lines reached\n", cut)
				problems++
			}
			budget -= n
		}
		if whole {
			note(sp.Path, sha, nil)
		} else {
			note(sp.Path, sha, served)
		}
	}
	if opt.Count {
		fmt.Fprintf(w, "-- %d matching line(s) in %d file(s)\n", countedLines, countedFiles)
	}
	if opt.Quiet {
		n, files := licensedCount(observed, fileLines)
		fmt.Fprintf(report, "-- licensed %d line(s) in %d file(s)\n", n, files)
	}
	if englishUnreadables >= 2 {
		fmt.Fprint(w, hintSplitRegexAddress())
	}
	if opt.Sub != nil && len(opt.Sub.Refused) > 0 {
		// A path the plan cannot carry makes the whole read void: nothing is
		// planned, so nothing may be licensed.
		opt.Sub.Plan.Reset()
		opt.Sub.Sites, opt.Sub.Files = 0, 0
		observed = map[string]seen.Observation{}
	}
	return observed, problems
}

// load reads the file a spec names, inside root, and reports on w why it could
// not; englishUnreadable says the failure was a path that reads as an English
// word (hintSplitRegexAddress). It may rewrite sp.Path, an absolute path inside
// the root becoming relative to it.
func load(w io.Writer, root string, sp *Spec) (b []byte, englishUnreadable, ok bool) {
	// An ABSOLUTE path is honoured on this surface, not joined onto the
	// root. A plan is a document whose paths are relative by design, and
	// apply refuses an absolute one by name — but a command-line argument
	// is shell-produced, tab completion emits absolute paths, and
	// `mrw read /repo/x.go` is a reasonable thing to type. That is an input
	// CONVENTION differing per surface, not a boundary holding on one path
	// and not another: the result still goes through Resolve below, so
	// containment is decided in exactly one place.
	//
	// Joining it was the bug a user hit: `mrw -C repo read /elsewhere/x.md`
	// looked for `repo/elsewhere/x.md` and reported "no such file" about a
	// path nobody wrote, while the header above echoed the path they did.
	// They read it as a defect in the tool that called mrw. check already
	// pre-screens this way and says its comment expects read to "say so" —
	// this is read saying so.
	argPath := sp.Path
	if rooted.IsRooted(argPath) {
		absRoot, absErr := rooted.Abs(root)
		if absErr != nil {
			fmt.Fprintf(w, "==> %s  REFUSED  %v\n", sp.Path, absErr)
			return nil, false, false
		}
		cleaned := rooted.Real(argPath)
		if !rooted.Contains(absRoot, cleaned) {
			fmt.Fprintf(w, "==> %s  REFUSED  %s is outside the root %s: "+
				"read it with --root pointed where you mean%s\n", sp.Path, sp.Path, absRoot, readElsewhere(cleaned, sp.Raw, argPath))
			return nil, false, false
		}
		if rel, relErr := filepath.Rel(absRoot, cleaned); relErr == nil {
			sp.Path = rel
			// ADR-076: Real cleaned the separator away; it is kept, so the
			// spelling is judged below as the caller wrote it.
			if rooted.SpelledAsDirectory(argPath) {
				sp.Path += string(filepath.Separator)
			}
		}
	}
	full, err := rooted.Resolve(root, sp.Path)
	if rooted.RefusedByName(err) {
		// ADR-081: no --root makes a device name, or a name Windows does not
		// keep as written, a file, so the advice below would send the caller
		// nowhere (the review of #243).
		fmt.Fprintf(w, "==> %s  REFUSED  %v\n", sp.Path, err)
		return nil, false, false
	}
	if errors.Is(err, rooted.ErrNotADirectory) {
		fmt.Fprintf(w, "==> %s  UNREADABLE  %v\n", sp.Path, err)
		return nil, false, false
	}
	if errors.Is(err, rooted.ErrOwnState) {
		// ADR-157: no --root lifts this refusal, so none is advised.
		fmt.Fprintf(w, "==> %s  REFUSED  %v\n", sp.Path, err)
		return nil, false, false
	}
	if err != nil {
		// The same boundary the write path enforces. Serving a file the
		// caller did not scope is a smaller harm than writing one, and it
		// is the same mistake: mrw is pointed at a tree, and ../ and a
		// symlink are the two ways out of it.
		advice := ""
		var oe *rooted.OutsideError
		if errors.As(err, &oe) && oe.Exists {
			advice = readElsewhere(oe.Resolved, sp.Raw, sp.Path)
		}
		fmt.Fprintf(w, "==> %s  REFUSED  %v: read it with --root pointed where you mean%s\n", sp.Path, err, advice)
		return nil, false, false
	}
	// ADR-074: opening a FIFO for reading blocks until something writes to
	// it, and a device can stream without end. The walk has refused a
	// non-regular candidate since ADR-007; a named spec is reported the
	// same way. A directory keeps ReadFile's own "is a directory", and a
	// stat that fails falls through to the error ReadFile gives.
	fi, statErr := os.Stat(full)
	if statErr == nil && !fi.Mode().IsRegular() && !fi.IsDir() {
		fmt.Fprintf(w, "==> %s  UNREADABLE  %s\n", sp.Path, lines.NotRegular)
		return nil, false, false
	}
	// ADR-104: a file is read whole to serve any range of it, so readCapped
	// refuses one over the cap by its size, and bounds the read itself.
	b, err = readCapped(full)
	if err != nil {
		glob := hintUnexpandedGlob(sp.Path)
		fmt.Fprintf(w, "==> %s  UNREADABLE  %v%s\n", sp.Path, err, glob)
		return nil, glob == "" && isEnglishWordToken(sp.Path), false
	}
	return b, false, true
}

// ReadStdin reads standard input whole for `mrw read -` (ADR-161), under the
// bound a file is read under (ADR-104).
func ReadStdin(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, maxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxFileBytes {
		return nil, errOverFileCap{size: int64(len(b))}
	}
	return b, nil
}

// licensedCount is what a quiet read reports: how many lines its observations
// license, each counted once, and how many files hold one.
func licensedCount(observed map[string]seen.Observation, fileLines map[string]int) (n, files int) {
	for k, o := range observed {
		c := fileLines[k]
		if !o.Whole() {
			got := map[int]bool{}
			for _, s := range o.Spans {
				for l := s[0]; l <= s[1]; l++ {
					got[l] = true
				}
			}
			c = len(got)
		}
		if c > 0 {
			n += c
			files++
		}
	}
	return n, files
}

// matchingLines counts the lines a spec's single-pattern ranges match, each
// line once however many of them match it (ADR-153). A --grep spec carries one.
func matchingLines(ranges []Range, lines []string) int {
	n := 0
	for _, l := range lines {
		for _, r := range ranges {
			if r.Re != nil && r.ReEnd == nil && r.Re.MatchString(l) {
				n++
				break
			}
		}
	}
	return n
}

type span struct{ start, end int }

// resolve turns ranges into concrete, merged, ascending spans and reports which
// patterns matched nothing. Merging matters: overlapping context windows would
// otherwise print the same lines twice and pay for them twice.
func resolve(ranges []Range, lines []string, ctx int, b []byte) ([]span, []string) {
	var missed []string
	total := len(lines)
	if total == 0 {
		// An empty file cannot satisfy any range, and saying nothing about it
		// reported success for a request that served nothing: `read
		// empty.txt:1` exited 0 in silence while `read f.txt:6` on a five-line
		// file correctly said so.
		for _, r := range ranges {
			missed = append(missed, fmt.Sprintf("%s (file has 0 lines)", r.Text))
		}
		return nil, missed
	}
	if len(ranges) == 0 {
		return []span{{1, total}}, nil
	}

	var (
		spans []span
	)
	for _, r := range ranges {
		switch {
		case r.JSON != nil:
			// ADR-155: resolved against the file's bytes, not its lines, since a
			// value's span is where its first and last bytes fall.
			start, end, err := jsonSpan(b, r.JSON)
			if err != nil {
				missed = append(missed, fmt.Sprintf("%s (%v)", r.Text, err))
				continue
			}
			spans = append(spans, span{start, end})
		case r.Re != nil && r.ReEnd != nil:
			found := false
			// Tracked separately from `found` so the report can say WHICH half
			// missed, AND so an earlier resolved span cannot suppress it. A
			// caller told only "this range matched nothing" re-reads the file to
			// learn whether it was the start or the end; a caller told nothing
			// at all never learns a start was dropped.
			missingEnd := false
			for i := 0; i < total; i++ {
				if !r.Re.MatchString(lines[i]) {
					continue
				}
				// no-op: the start matched, which only matters if its end does not
				// ADR-036. `j >= i`, not `j > i`: the end is the first match AT
				// OR AFTER the start, which is what internal/apply already did
				// (the THE END IS A DELIMITER block in Apply) and what ed and sed mean by /a/,/b/. An end on
				// the start line closes the span there.
				//
				// 0 means "no end matched", and it is not a valid span start, so
				// it cannot be confused with one. This used to default to
				// `total` — the range silently ran to the end of the file and
				// reported SUCCESS, serving a span the address never described.
				// A read that quietly serves more is exactly as invisible as a
				// write that quietly changes less, and the caller then holds
				// line numbers mrw never agreed to.
				end := 0
				for j := i; j < total; j++ {
					if r.ReEnd.MatchString(lines[j]) {
						end = j + 1
						break
					}
				}
				if end == 0 {
					// ⚠ RECORDED, NOT JUST SKIPPED. `found` is range-wide, so a
					// start that resolved EARLIER used to suppress this one
					// entirely: `START … END … START … EOF` returned the first
					// span with no problem and exit 0, silently dropping a start
					// the address named. That is the same silent-drop this
					// record exists to remove, moved one case along, and it
					// survived the first cut because the only report was gated
					// on `!found`. Found by review of PR #146.
					missingEnd = true
					continue
				}
				spans = append(spans, span{i + 1, end})
				found = true
				i = end - 1
			}
			// Reported INDEPENDENTLY of whether anything resolved, and the valid
			// spans are kept: a caller gets what mrw could resolve plus a named
			// problem for what it could not, rather than one or the other.
			switch {
			case missingEnd:
				missed = append(missed, fmt.Sprintf(
					"%s (the start matched, but the end pattern matched no line at or after it)", r.Text))
			case !found:
				missed = append(missed, r.Text)
			}
		case r.Re != nil:
			found := false
			for i, l := range lines {
				if !r.Re.MatchString(l) {
					continue
				}
				// -C is compared before it is added for the same reason the
				// relative end is: `i + 1 + ctx` OVERFLOWS at a large context
				// and printed `@@ 1--9223372036854775807` at exit 0, serving
				// nothing. The context flag only refuses a NEGATIVE value, so a
				// caller can reach this. Found by the third Codex review of
				// PR #125, beside the relative end it had just protected.
				end := total
				if ctx <= total-(i+1) {
					end = i + 1 + ctx
				}
				if r.RelEnd > 0 {
					// An explicit relative end overrides -C on the trailing
					// An explicit relative end overrides -C on the trailing
					// side: the caller said how many lines they wanted after
					// the match, which is not a guess about context.
					//
					// ⚠ COMPARED BEFORE IT IS ADDED. `i + 1 + r.RelEnd`
					// OVERFLOWS at a large count, and a negative end printed
					// `@@ 2--9223372036854775807` and served nothing at exit 0
					// — a read that returned nothing while reporting success,
					// which is the failure this tool exists to make visible.
					// The numeric branch below was safe by accident, because
					// its `end < start` check caught the wrapped value; this
					// branch has no such check. Found by the second Codex
					// review of PR #125.
					if r.RelEnd > total-(i+1) {
						end = total
					} else {
						end = i + 1 + r.RelEnd
					}
				}
				spans = append(spans, span{max(1, i+1-ctx), min(total, end)})
				found = true
			}
			if !found {
				missed = append(missed, r.Text)
			}
		default:
			start, end := r.Start, r.End
			if start == lastLine {
				start = total
			}
			if end == lastLine {
				end = total
			}
			if start == unbounded {
				start = 1
			}
			if r.RelEnd > 0 && start <= total {
				// Compared before it is added: start+RelEnd overflows at a
				// large count, and a wrapped end is not a clamp.
				if r.RelEnd > total-start {
					end = total
				} else {
					end = start + r.RelEnd
				}
			}
			if end == unbounded || end > total {
				end = total
			}
			if start > total {
				missed = append(missed, fmt.Sprintf("%s (file has %d lines)", r.Text, total))
				continue
			}
			// Reachable only once the length is known, and only for a range
			// written with `$`: `$-5` resolves to 10-5 on a ten-line file.
			// Served, it printed lines 1-5 at exit 0 — content for an address
			// that cannot be satisfied.
			if end < start {
				missed = append(missed, fmt.Sprintf("%s ends before it starts (file has %d lines)", r.Text, total))
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
	// sort.Slice, not a hand-rolled insertion sort. The hand-rolled one was
	// O(n^2) on unsorted input — invisible in practice because spans usually
	// arrive ascending (a pattern scan walks the file forwards), and 1.83s at
	// 30,000 descending ranges when they do not. The stdlib is both faster and
	// less code.
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].start < sorted[j].start })
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

// withNextLine extends each span that ends before the file does by its next
// line, and merges the spans that then touch (ADR-161).
func withNextLine(in []span, total int) []span {
	out := make([]span, len(in))
	for i, s := range in {
		if s.end < total {
			s.end++
		}
		out[i] = s
	}
	return merge(out)
}

// split numbers b's lines as the write engine does (ADR-065, lines.Split) and
// hashes the RAW bytes: the ledger's sha must agree with seen.SHA, and a hash of
// rejoined lines would make every CRLF file read as changed behind mrw's back.
func split(b []byte) ([]string, string) {
	sum := sha256.Sum256(b)
	sha := hex.EncodeToString(sum[:])
	ls, _, _ := lines.Split(string(b))
	return ls, sha
}

// hintUnexpandedGlob explains an unreadable path that holds a glob
// metacharacter, and says nothing otherwise.
//
// "no such file or directory" for `internal/mcp/*.go` is true and answers the
// wrong question. The shape that produces it most often never reaches mrw at
// all: zsh refuses `internal/mcp/*.go:1-3` with "no matches found", because the
// address suffix means the pattern matches no file. Quoting is what a caller
// tries next, and quoting is what produces this message — so this is the second
// step of a wrong path, and the place to say where the right one is.
//
// It fires only on a path that ALREADY failed to open, so it is advice about a
// broken path rather than a claim about what the caller meant. A file whose
// name really does contain a star is served normally and never sees this.
func hintUnexpandedGlob(path string) string {
	if !strings.ContainsAny(path, "*?[") {
		return ""
	}
	return "\n    (that path holds a glob your shell did not expand. An address suffix like" +
		" `:1-3` stops most shells matching, and quoting keeps the star literal." +
		" Use --grep to walk and serve in one call, or --files-from to pipe a list in.)"
}

var englishWordToken = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// isEnglishWordToken reports the shell-split class: a single token with no
// glob metacharacter. mrw sees arguments, not the shell (ADR-015 D4).
func isEnglishWordToken(path string) bool {
	if strings.ContainsAny(path, "*?[") {
		return false
	}
	return englishWordToken.MatchString(path)
}

// hintSplitRegexAddress is one hint per read invocation when two or more
// UNREADABLE paths looked like English-word tokens. A single missing nope.go
// stays silent.
func hintSplitRegexAddress() string {
	return "\n    (those paths look like a shell-split regex address. quoting the spec, or --grep, is the fix.)\n"
}
