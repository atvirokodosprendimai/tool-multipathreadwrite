package read

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// ADR-116: a walk inside a checkout skips what the checkout's .gitignore
// files and .git/info/exclude ignore. The matcher is native, so mrw still runs
// no git and needs none (ADR-004); it follows gitignore(5), and
// TestTheIgnoreMatcherAgreesWithGit holds it to git's own answer where git is
// installed. core.excludesFile is not read: a walk answers the same on every
// machine.

// ignoreRule is one pattern line of an ignore file.
type ignoreRule struct {
	base     string // the ignore file's directory, top-relative with "/"; "" is the top
	re       *regexp.Regexp
	negate   bool
	dirOnly  bool
	anchored bool // matched against the path below base, not the name alone
}

// parseIgnore reads an ignore file whose directory is base. fold matches
// without regard to case, as git does where core.ignorecase is true.
func parseIgnore(base string, data []byte, fold bool) []ignoreRule {
	var rules []ignoreRule
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = trimTrailingSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r := ignoreRule{base: base}
		switch {
		case strings.HasPrefix(line, "!"):
			r.negate = true
			line = line[1:]
		case strings.HasPrefix(line, `\!`), strings.HasPrefix(line, `\#`):
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			r.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		if line == "" {
			continue
		}
		if strings.Contains(line, "/") {
			r.anchored = true
			line = strings.TrimPrefix(line, "/")
		}
		pat, ok := globToRegexp(line)
		if !ok {
			continue // an unclosed class: git abandons the match, so the rule matches nothing
		}
		expr := "^" + pat + "$"
		if fold {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			continue // a pattern git would not match either
		}
		r.re = re
		rules = append(rules, r)
	}
	return rules
}

// trimTrailingSpace drops trailing spaces that are not escaped with a
// backslash, as gitignore(5) says; an escaped one stays, unescaped.
func trimTrailingSpace(s string) string {
	for strings.HasSuffix(s, " ") {
		if strings.HasSuffix(s, `\ `) {
			return s
		}
		s = s[:len(s)-1]
	}
	return s
}

// globToRegexp turns one gitignore glob into a regular expression over a
// "/"-separated path: * and ? stay inside a component, [...] is a class
// ("!" negates it), ** is any number of components where gitignore(5) gives
// it a meaning, and a backslash escapes the next character. It reads the glob
// a byte at a time, as git does, and each byte becomes the rune of the same
// value, so the expression matches a subject spelled by bytewise: "?" is one
// byte of "é", as in git, not the whole character. ok is false for a class
// no "]" closes, which git never matches.
func globToRegexp(g string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(g); i++ {
		c := g[i]
		switch {
		case c == '\\' && i+1 < len(g):
			i++
			b.WriteString(regexp.QuoteMeta(string(rune(g[i]))))
		case c == '*' && i+1 < len(g) && g[i+1] == '*':
			atStart := i == 0
			atEnd := i+2 == len(g)
			slashAfter := i+2 < len(g) && g[i+2] == '/'
			slashBefore := i > 0 && g[i-1] == '/'
			switch {
			case atStart && atEnd: // ** alone: every path (found by the differential against git)
				b.WriteString(".*")
				i++
			case atStart && slashAfter: // **/x
				b.WriteString("(?:.*/)?")
				i += 2
			case slashBefore && atEnd: // x/**
				b.WriteString(".*")
				i++
			case slashBefore && slashAfter: // x/**/y
				b.WriteString("(?:.*/)?")
				i += 2
			default: // ** elsewhere is two stars
				b.WriteString("[^/]*")
				i++
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		case c == '[':
			cls, n, ok := bracket(g[i:])
			if !ok {
				return "", false
			}
			b.WriteString(cls)
			i += n - 1
		default:
			b.WriteString(regexp.QuoteMeta(string(rune(c))))
		}
	}
	return b.String(), true
}

// bracket translates the class at the start of s ("[...]") as git's wildmatch
// reads it, and says how many bytes it took; ok is false when no "]" closes
// it. As git does: "!" or "^" negates; a "]"
// first is a character; a backslash escapes; "a-c" is a range, and a reversed
// one adds nothing beyond the character before it ("[z-a]" is "[z]"); after a
// range a "-" is a character; "[:name:]" is a POSIX class, and an unknown name
// matches nothing; and no class ever matches "/", negated or not. Each piece
// is spelled \x{..} so no character is read as regexp syntax.
func bracket(s string) (string, int, bool) {
	i := 1
	neg := false
	if i < len(s) && (s[i] == '!' || s[i] == '^') {
		neg = true
		i++
	}
	var items []string
	prev := -1 // the last single character, from which a "-" opens a range
	never := false
	for first := true; ; first = false {
		if i >= len(s) {
			return "", 0, false
		}
		c := s[i]
		if c == ']' && !first {
			i++
			break
		}
		switch {
		case c == '[' && i+1 < len(s) && s[i+1] == ':' && posixEnd(s[i+2:]) > 0:
			// "[:name:]" ends at the first "]" after "[:", and only when ":"
			// stands before it; otherwise the "[" is a character, as in git.
			end := posixEnd(s[i+2:])
			set, known := posixClass[s[i+2:i+2+end-1]]
			never = never || !known
			items = append(items, set)
			prev = -1
			i += 2 + end + 1
		case c == '-' && prev >= 0 && i+1 < len(s) && s[i+1] != ']':
			i++
			if s[i] == '\\' && i+1 < len(s) {
				i++
			}
			if hi := int(s[i]); hi >= prev {
				items = append(items, classRange(prev, hi))
			}
			prev = -1
			i++
		default:
			if c == '\\' && i+1 < len(s) {
				i++
				c = s[i]
			}
			items = append(items, classRange(int(c), int(c)))
			prev = int(c)
			i++
		}
	}
	if never {
		return `[^\x00-\x{10FFFF}]`, i, true
	}
	if neg {
		return `[^` + strings.Join(items, "") + `\x{2f}]`, i, true
	}
	if strings.Join(items, "") == "" {
		return `[^\x00-\x{10FFFF}]`, i, true
	}
	return "[" + strings.Join(items, "") + "]", i, true
}

// posixEnd is the offset of the "]" closing a POSIX class name in s, the text
// after "[:": the first "]", when ":" stands before it; else 0.
func posixEnd(s string) int {
	j := strings.IndexByte(s, ']')
	if j < 1 || s[j-1] != ':' {
		return 0
	}
	return j
}

// bytewise spells s one rune per byte, the subject globToRegexp's expressions
// match, so a pattern meets a non-ASCII name byte by byte as git's does.
func bytewise(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			r := make([]rune, len(s))
			for j := 0; j < len(s); j++ {
				r[j] = rune(s[j])
			}
			return string(r)
		}
	}
	return s
}

// classRange spells lo-hi for a regexp class, leaving out "/".
func classRange(lo, hi int) string {
	if lo <= '/' && '/' <= hi {
		s := ""
		if lo < '/' {
			s += classRange(lo, '/'-1)
		}
		if hi > '/' {
			s += classRange('/'+1, hi)
		}
		return s
	}
	if lo == hi {
		return fmt.Sprintf(`\x{%x}`, lo)
	}
	return fmt.Sprintf(`\x{%x}-\x{%x}`, lo, hi)
}

// posixClass is each "[:name:]" git knows, in the ASCII it means, without "/".
var posixClass = map[string]string{
	"alnum":  `0-9A-Za-z`,
	"alpha":  `A-Za-z`,
	"blank":  `\t `,
	"cntrl":  `\x00-\x1f\x7f`,
	"digit":  `0-9`,
	"graph":  `\x{21}-\x{2e}\x{30}-\x{7e}`,
	"lower":  `a-z`,
	"print":  `\x{20}-\x{2e}\x{30}-\x{7e}`,
	"punct":  `\x{21}-\x{2e}\x{3a}-\x{40}\x{5b}-\x{60}\x{7b}-\x{7e}`,
	"space":  `\t\n\v\f\r `,
	"upper":  `A-Z`,
	"xdigit": `0-9A-Fa-f`,
}

// ignorer answers whether a path below the walk root is ignored. It loads
// each directory's .gitignore the first time a path below it is asked about.
type ignorer struct {
	top    string // the checkout's top, absolute
	prefix string // the walk root relative to top, "/"-joined; "" at the top
	rules  []ignoreRule
	loaded map[string]bool // top-relative directories whose .gitignore was read
	static bool            // rules given whole: nothing is read from disk
	fold   bool            // match without regard to case (foldsCase)
}

// newIgnorer finds the checkout at or above absRoot — a .git directory or
// file — and returns an ignorer for it, or nil outside a checkout, where git
// applies no .gitignore and neither does mrw.
func newIgnorer(absRoot string) *ignorer {
	top := absRoot
	for {
		if _, err := os.Lstat(filepath.Join(top, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(top)
		if parent == top {
			return nil
		}
		top = parent
	}
	prefix, err := filepath.Rel(top, absRoot)
	if err != nil {
		return nil
	}
	prefix = filepath.ToSlash(prefix)
	if prefix == "." {
		prefix = ""
	}
	ig := &ignorer{top: top, prefix: prefix, loaded: map[string]bool{}, fold: foldsCase(top)}
	if b := readIgnoreFile(filepath.Join(gitDir(top), "info", "exclude")); b != nil {
		ig.rules = append(ig.rules, parseIgnore("", b, ig.fold)...)
	}
	return ig
}

// foldsCase reports whether the checkout at top sits on a filesystem that
// folds case, by asking whether .GIT names the same file as .git. It is the
// probe git init makes to set core.ignorecase, and git matches ignore rules
// without regard to case where that is true — on macOS and Windows by default,
// where `*.log` ignores A.LOG. An explicit core.ignorecase is not read: mrw
// parses no git config.
func foldsCase(top string) bool {
	a, err := os.Lstat(filepath.Join(top, ".git"))
	if err != nil {
		return false
	}
	b, err := os.Lstat(filepath.Join(top, ".GIT"))
	return err == nil && os.SameFile(a, b)
}

// gitDir is the directory holding the checkout's info/exclude: .git itself,
// or, where .git is a file — a worktree or a submodule — the gitdir it names,
// and that gitdir's commondir when it has one, where a worktree's info/exclude
// lives. A gitdir is a path git wrote, not one a caller gave, so IsAbs is the
// question; a path it cannot follow gives no rules, and the walk serves more.
func gitDir(top string) string {
	dir := filepath.Join(top, ".git")
	if b := readIgnoreFile(dir); b != nil {
		line, _, _ := strings.Cut(string(b), "\n")
		p, ok := strings.CutPrefix(strings.TrimSpace(line), "gitdir:")
		if !ok {
			return dir
		}
		dir = strings.TrimSpace(p)
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(top, dir)
		}
	}
	if b := readIgnoreFile(filepath.Join(dir, "commondir")); b != nil {
		c := strings.TrimSpace(string(b))
		if !filepath.IsAbs(c) {
			c = filepath.Join(dir, c)
		}
		return c
	}
	return dir
}

// load reads dir's .gitignore once (dir top-relative).
func (ig *ignorer) load(dir string) {
	if ig.static || ig.loaded[dir] {
		return
	}
	ig.loaded[dir] = true
	p := filepath.Join(ig.top, filepath.FromSlash(dir), ".gitignore")
	// A .gitignore that is a link is not followed, as git does not follow
	// one: followed, a link out of the root let a file mrw refuses to serve
	// decide what it serves, and the skip count told what it held.
	if fi, err := os.Lstat(p); err != nil || fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return
	}
	if b := readIgnoreFile(p); b != nil {
		ig.rules = append(ig.rules, parseIgnore(dir, b, ig.fold)...)
	}
}

// readIgnoreFile reads an ignore file the way a read serves a file: without
// blocking on a FIFO or a device (ADR-109) and no more than maxFileBytes
// (ADR-104). A .gitignore that is a FIFO hung every walk that met it, and one
// linked to /dev/zero read without end. Anything it will not read — missing, a
// directory, not regular, too large — gives no rules, so the walk serves more
// rather than waiting.
func readIgnoreFile(p string) []byte {
	f, fi, err := regular.Open(p)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	if fi.IsDir() || fi.Size() > maxFileBytes {
		return nil
	}
	b, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil || int64(len(b)) > maxFileBytes {
		return nil
	}
	return b
}

// Ignored says whether rel, a root-relative path spelled with "/", is
// ignored: a directory above it is ignored (gitignore(5): nothing below an
// ignored directory can be re-included), or the last rule that matches it is
// not a negation. from is how many components of rel the caller named — the
// directory a walk started at — and those directories do not prune what is
// below them, as --exclude prunes only below where the walk starts: naming an
// ignored directory walks it. Rules still apply to what is inside.
func (ig *ignorer) Ignored(rel string, isDir bool, from int) bool {
	full := rel
	pd := 0
	if ig.prefix != "" {
		full = ig.prefix + "/" + rel
		pd = strings.Count(ig.prefix, "/") + 1
	}
	parts := strings.Split(full, "/")
	for i := 0; i < len(parts); i++ {
		ig.load(strings.Join(parts[:i], "/"))
	}
	for i := 1; i < len(parts); i++ {
		if i > pd && i <= pd+from {
			continue
		}
		if ig.match(strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return ig.match(full, isDir)
}

// match applies the rules to one top-relative path; the last match wins.
func (ig *ignorer) match(p string, isDir bool) bool {
	ignored := false
	for _, r := range ig.rules {
		if r.dirOnly && !isDir {
			continue
		}
		below := p
		if r.base != "" {
			if !strings.HasPrefix(p, r.base+"/") {
				continue
			}
			below = strings.TrimPrefix(p, r.base+"/")
		}
		subject := path.Base(below)
		if r.anchored {
			subject = below
		}
		if r.re.MatchString(bytewise(subject)) {
			ignored = !r.negate
		}
	}
	return ignored
}
