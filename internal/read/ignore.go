package read

import (
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
		expr := "^" + globToRegexp(line) + "$"
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
// it a meaning, and a backslash escapes the next character.
func globToRegexp(g string) string {
	var b strings.Builder
	for i := 0; i < len(g); i++ {
		c := g[i]
		switch {
		case c == '\\' && i+1 < len(g):
			i++
			b.WriteString(regexp.QuoteMeta(string(g[i])))
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
			j := i + 1
			if j < len(g) && (g[j] == '!' || g[j] == '^') {
				j++
			}
			if j < len(g) && g[j] == ']' {
				j++
			}
			for j < len(g) && g[j] != ']' {
				j++
			}
			if j >= len(g) {
				b.WriteString(regexp.QuoteMeta("["))
				continue
			}
			class := g[i+1 : j]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			b.WriteString("[" + strings.ReplaceAll(class, `\`, `\\`) + "]")
			i = j
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return b.String()
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
	if b := readIgnoreFile(filepath.Join(ig.top, filepath.FromSlash(dir), ".gitignore")); b != nil {
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
		if r.re.MatchString(subject) {
			ignored = !r.negate
		}
	}
	return ignored
}
