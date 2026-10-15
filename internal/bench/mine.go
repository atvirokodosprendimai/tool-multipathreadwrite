package bench

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// mineSalt orders Stage 4's candidates (BACKLOG, "Pre-registered for
// ADR-160"): by sha256 of the salt and the candidate's key.
const mineSalt = "adr-160:"

func saltedHash(salt, key string) string {
	s := sha256.Sum256([]byte(salt + key))
	return hex.EncodeToString(s[:])
}

// ExportTree writes repo's tree at commit into dst, as git archive gives it:
// mining reads the pinned commit, never whatever the clone has checked out.
func ExportTree(repo, commit, dst string) error {
	var out, stderr bytes.Buffer
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", commit)
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git archive %s: %w: %s", commit, err, stderr.String())
	}
	return untar(&out, dst)
}

// RenameCandidate is one identifier ADR-160's wide-rename predicate admits.
type RenameCandidate struct {
	Name        string
	Occurrences int // identifier tokens and whole-word mentions in comments
	Files       int
}

// generated is the first line Go's convention marks a generated file with.
var generated = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// MineWideRenames lists the identifiers of the Go module at root that ADR-160
// Decision 1 admits as wide-rename tasks, in registered order: declared exactly
// once in the module; named in no string literal and in no generated file; not
// a package name; with at least minOcc occurrences — identifier tokens and
// whole-word mentions in comments — across at least minFiles files. It sees no
// types, so a name also used as a selector on a standard-library value is
// listed: the reference's build gate (T2) is what filters it.
func MineWideRenames(root string, minOcc, minFiles int) ([]RenameCandidate, error) {
	decls := map[string]int{}
	occ := map[string]int{}
	files := map[string]map[string]bool{}
	tainted := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (name == "vendor" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		first, _, _ := strings.Cut(string(src), "\n")
		if generated.MatchString(strings.TrimRight(first, "\r")) {
			for _, w := range words(string(src)) {
				tainted[w] = true
			}
			return nil
		}
		f, err := parser.ParseFile(fset, p, src, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		tainted[f.Name.Name] = true
		for _, imp := range f.Imports {
			base := strings.Trim(imp.Path.Value, "\"`")
			tainted[base[strings.LastIndex(base, "/")+1:]] = true
			if imp.Name != nil {
				tainted[imp.Name.Name] = true
			}
		}
		countDecls(f, decls)
		var s scanner.Scanner
		s.Init(fset.AddFile(p, -1, len(src)), src, nil, scanner.ScanComments)
		for {
			_, tok, lit := s.Scan()
			if tok == token.EOF {
				break
			}
			var found []string
			switch tok {
			case token.IDENT:
				found = []string{lit}
			case token.COMMENT:
				found = words(lit)
			case token.STRING, token.CHAR:
				for _, w := range words(lit) {
					tainted[w] = true
				}
			}
			for _, w := range found {
				occ[w]++
				if files[w] == nil {
					files[w] = map[string]bool{}
				}
				files[w][p] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []RenameCandidate
	for name, n := range decls {
		if n != 1 || name == "_" || tainted[name] || occ[name] < minOcc || len(files[name]) < minFiles {
			continue
		}
		out = append(out, RenameCandidate{Name: name, Occurrences: occ[name], Files: len(files[name])})
	}
	sort.Slice(out, func(i, j int) bool { return saltedHash(mineSalt, out[i].Name) < saltedHash(mineSalt, out[j].Name) })
	return out, nil
}

// countDecls counts every identifier f declares: functions and methods, types
// and type parameters, constants and variables, fields, parameters and results,
// short variable declarations, range variables, labels and import names.
func countDecls(f *ast.File, decls map[string]int) {
	add := func(ids ...*ast.Ident) {
		for _, id := range ids {
			if id != nil {
				decls[id.Name]++
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			add(n.Name)
		case *ast.TypeSpec:
			add(n.Name)
		case *ast.ValueSpec:
			add(n.Names...)
		case *ast.Field:
			add(n.Names...)
		case *ast.ImportSpec:
			add(n.Name)
		case *ast.LabeledStmt:
			add(n.Label)
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				for _, e := range n.Lhs {
					if id, ok := e.(*ast.Ident); ok {
						add(id)
					}
				}
			}
		case *ast.RangeStmt:
			if n.Tok == token.DEFINE {
				for _, e := range []ast.Expr{n.Key, n.Value} {
					if id, ok := e.(*ast.Ident); ok {
						add(id)
					}
				}
			}
		}
		return true
	})
}

// words returns the whole-word Go identifiers in s: maximal runs of letters,
// digits and underscores that do not start with a digit.
func words(s string) []string {
	var out []string
	start := -1
	for i := 0; i <= len(s); {
		r, size := rune(0), 1
		if i < len(s) {
			r, size = utf8.DecodeRuneInString(s[i:])
		}
		in := i < len(s) && (r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r))
		if in && start < 0 {
			start = i
		}
		if !in && start >= 0 {
			w := s[start:i]
			if f, _ := utf8.DecodeRuneInString(w); !unicode.IsDigit(f) {
				out = append(out, w)
			}
			start = -1
		}
		i += size
	}
	return out
}

// CoupledCandidate is one commit ADR-160's check-coupled predicate admits.
type CoupledCandidate struct {
	Commit, Parent string
	Files          []string
	Old, New       string // the literal's text before and after
}

// MineCheckCoupled lists the commits reachable from commit in repo, merges
// excluded, that ADR-160 Decision 1 admits as check-coupled tasks, in
// registered order: at most maxFiles files, every one of them changed only by
// replacing one string literal's text with another — the same pair in every
// file — and among them a non-test .go file and a _test.go file in the same
// directory. Whether the package's tests pass at the parent is T2's check.
func MineCheckCoupled(repo, commit string, maxFiles int) ([]CoupledCandidate, error) {
	out, err := gitOut(repo, "log", "-p", "--no-merges", "--no-renames", "--no-color", "--no-ext-diff", "-U0", "--format=%x00%H %P", commit)
	if err != nil {
		return nil, err
	}
	var cs []CoupledCandidate
	for _, rec := range strings.Split(out, "\x00")[1:] {
		head, body, _ := strings.Cut(rec, "\n")
		ids := strings.Fields(head)
		if len(ids) != 2 { // a root commit has no parent
			continue
		}
		if c, ok := coupled(body, maxFiles); ok {
			c.Commit, c.Parent = ids[0], ids[1]
			cs = append(cs, c)
		}
	}
	sort.Slice(cs, func(i, j int) bool { return saltedHash(mineSalt, cs[i].Commit) < saltedHash(mineSalt, cs[j].Commit) })
	return cs, nil
}

var diffFile = regexp.MustCompile(`^diff --git a/(\S+) b/(\S+)$`)

// coupled judges one commit's -U0 diff.
func coupled(diff string, maxFiles int) (CoupledCandidate, bool) {
	var c CoupledCandidate
	var minus, plus []string
	pairOK := true
	flush := func() {
		if len(minus) != len(plus) {
			pairOK = false
		}
		for i := range minus {
			if i >= len(plus) || !c.replaces(minus[i], plus[i]) {
				pairOK = false
			}
		}
		minus, plus = nil, nil
	}
	sc := bufio.NewScanner(strings.NewReader(diff))
	sc.Buffer(make([]byte, 0, 1<<16), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			m := diffFile.FindStringSubmatch(line)
			if m == nil || m[1] != m[2] {
				return c, false
			}
			c.Files = append(c.Files, m[1])
		case strings.HasPrefix(line, "@@"):
			flush()
		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "):
		case strings.HasPrefix(line, "-"):
			minus = append(minus, line[1:])
		case strings.HasPrefix(line, "+"):
			plus = append(plus, line[1:])
		case strings.HasPrefix(line, "new file"), strings.HasPrefix(line, "deleted file"), strings.HasPrefix(line, "Binary"):
			return c, false
		}
	}
	flush()
	if !pairOK || c.Old == "" || len(c.Files) < 2 || len(c.Files) > maxFiles {
		return c, false
	}
	dirs := map[string][2]bool{}
	for _, f := range c.Files {
		if !strings.HasSuffix(f, ".go") {
			continue
		}
		d := dirs[filepath.Dir(f)]
		if strings.HasSuffix(f, "_test.go") {
			d[1] = true
		} else {
			d[0] = true
		}
		dirs[filepath.Dir(f)] = d
	}
	for _, d := range dirs {
		if d[0] && d[1] {
			sort.Strings(c.Files)
			return c, true
		}
	}
	return c, false
}

// replaces says whether after is before with one string literal's text
// replaced, and that the replacement is the commit's one pair: the first
// replacement found fixes Old and New, and every later one must repeat it.
func (c *CoupledCandidate) replaces(before, after string) bool {
	p := 0
	for p < len(before) && p < len(after) && before[p] == after[p] {
		p++
	}
	s := 0
	for s < len(before)-p && s < len(after)-p && before[len(before)-1-s] == after[len(after)-1-s] {
		s++
	}
	// Widen the difference to the whole literal it sits in.
	open := strings.LastIndexAny(before[:p], "\"`")
	end := len(before) - s
	closeB := strings.IndexAny(before[end:], "\"`")
	if open < 0 || closeB < 0 || before[open] != before[end+closeB] {
		return false
	}
	old := before[open+1 : end+closeB]
	endA := len(after) - s
	closeA := strings.IndexAny(after[endA:], "\"`")
	if closeA < 0 || strings.ContainsAny(old, "\"`") {
		return false
	}
	nw := after[open+1 : endA+closeA]
	if strings.ContainsAny(nw, "\"`") || old == nw || strings.TrimSpace(old) == "" {
		return false
	}
	if before[:open+1]+nw+before[end+closeB:] != after {
		return false
	}
	if c.Old == "" {
		c.Old, c.New = old, nw
		return true
	}
	return c.Old == old && c.New == nw
}
