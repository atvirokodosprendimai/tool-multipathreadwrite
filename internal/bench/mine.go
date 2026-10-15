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
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	pkgs := map[string][]*ast.File{} // a directory's files by package clause
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
		pkg := filepath.Dir(p) + "\x00" + f.Name.Name
		pkgs[pkg] = append(pkgs[pkg], f)
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
	for _, fs := range pkgs {
		countDecls(fset, fs, decls)
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

// countDecls counts the identifiers one package's files declare. It type
// checks them with go/types and reads Info.Defs, so every scope rule — a :=
// that redeclares, a type switch's implicit variable, a field that shares a
// local's name — is the compiler's, not an approximation of it (the reviews of
// #426). Imports are not resolved: the importer refuses each one and checking
// goes on. A package with errors can leave declarations out of Info.Defs — a
// second declaration of one name in two build-tagged files, a field inside a
// type argument of an unresolved generic — so every declaration that is not a
// := or a range variable is also counted from the syntax, and a name counts
// the larger of the two (the third review of #426). A package clause's name
// declares nothing.
func countDecls(fset *token.FileSet, files []*ast.File, decls map[string]int) {
	clause := map[*ast.Ident]bool{}
	for _, f := range files {
		clause[f.Name] = true
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: refuseImports{}, Error: func(error) {}}
	_, _ = conf.Check(files[0].Name.Name, fset, files, info) // errors are expected: no import resolves
	defs := map[string]int{}
	for id := range info.Defs {
		if !clause[id] && id.Name != "_" {
			defs[id.Name]++
		}
	}
	syn := map[string]int{}
	add := func(ids ...*ast.Ident) {
		for _, id := range ids {
			if id != nil && id.Name != "_" {
				syn[id.Name]++
			}
		}
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				add(n.Name)
				if n.Recv != nil {
					for _, r := range n.Recv.List {
						t := r.Type
						if s, ok := t.(*ast.StarExpr); ok {
							t = s.X
						}
						switch t := t.(type) {
						case *ast.IndexExpr:
							add(identOf(t.Index))
						case *ast.IndexListExpr:
							for _, e := range t.Indices {
								add(identOf(e))
							}
						}
					}
				}
			case *ast.TypeSpec:
				add(n.Name)
			case *ast.ValueSpec:
				add(n.Names...)
			case *ast.Field:
				add(n.Names...)
			case *ast.StructType:
				for _, fl := range n.Fields.List {
					if len(fl.Names) == 0 {
						add(embeddedName(fl.Type))
					}
				}
			case *ast.ImportSpec:
				add(n.Name)
			case *ast.LabeledStmt:
				add(n.Label)
			}
			return true
		})
	}
	for name, n := range defs {
		decls[name] += max(n, syn[name])
	}
	for name, n := range syn {
		if _, ok := defs[name]; !ok {
			decls[name] += n
		}
	}
}

func identOf(e ast.Expr) *ast.Ident {
	id, _ := e.(*ast.Ident)
	return id
}

// embeddedName is the field name an embedded type declares: T for T, *T,
// pkg.T and T[X].
func embeddedName(t ast.Expr) *ast.Ident {
	for {
		switch e := t.(type) {
		case *ast.StarExpr:
			t = e.X
		case *ast.IndexExpr:
			t = e.X
		case *ast.IndexListExpr:
			t = e.X
		case *ast.SelectorExpr:
			return e.Sel
		case *ast.Ident:
			return e
		default:
			return nil
		}
	}
}

// refuseImports resolves no import, so mining needs no build of the module.
type refuseImports struct{}

func (refuseImports) Import(path string) (*types.Package, error) {
	return nil, fmt.Errorf("mining resolves no import (%s)", path)
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
	Old, New       string // the literal's value before and after, unquoted
	paired         bool   // Old and New are fixed; Old may be empty
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
		var c CoupledCandidate
		if !c.coupled(body, maxFiles) {
			continue
		}
		// The diff's lines are fragments: a line inside a block comment or a
		// multi-line raw string scans as something else. Each file is judged
		// whole, before and after, as the compiler would read it (the review of
		// #426).
		whole := true
		for _, f := range c.Files {
			before, err1 := gitOut(repo, "show", ids[1]+":"+f)
			after, err2 := gitOut(repo, "show", ids[0]+":"+f)
			if err1 != nil || err2 != nil || !c.replaces(before, after) {
				whole = false
				break
			}
		}
		if whole {
			c.Commit, c.Parent = ids[0], ids[1]
			cs = append(cs, c)
		}
	}
	sort.Slice(cs, func(i, j int) bool { return saltedHash(mineSalt, cs[i].Commit) < saltedHash(mineSalt, cs[j].Commit) })
	return cs, nil
}

// diffPath reads the path of a `diff --git a/P b/P` header whose two sides
// are one path: spaces included, and a path git quotes (for a quote, a
// backslash or a control character) decoded (the reviews of #426).
func diffPath(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "diff --git ")
	if !ok {
		return "", false
	}
	if strings.HasPrefix(rest, `"`) {
		qa, err := strconv.QuotedPrefix(rest)
		if err != nil {
			return "", false
		}
		qb := strings.TrimPrefix(rest[len(qa):], " ")
		a, err1 := strconv.Unquote(qa)
		b, err2 := strconv.Unquote(qb)
		if err1 != nil || err2 != nil || !strings.HasPrefix(a, "a/") || !strings.HasPrefix(b, "b/") {
			return "", false
		}
		return a[2:], a[2:] == b[2:]
	}
	rest, ok = strings.CutPrefix(rest, "a/")
	if !ok || (len(rest)-3)%2 != 0 || len(rest) < 5 {
		return "", false
	}
	p := rest[:(len(rest)-3)/2]
	return p, rest == p+" b/"+p
}

// coupled screens one commit's -U0 diff before its files are read whole: at
// most maxFiles files, each changed in place — a mode change, or a new,
// deleted, renamed, copied or binary file disqualifies it — and among them a
// non-test .go file and a _test.go file in one directory. A file that is not
// Go is kept for the whole-file judgment beside them, and neither the number
// of lines a hunk changes nor a missing final newline is judged here: the
// whole-file judgment decides whether anything but literals changed (the third
// review of #426).
func (c *CoupledCandidate) coupled(diff string, maxFiles int) bool {
	sc := bufio.NewScanner(strings.NewReader(diff))
	sc.Buffer(make([]byte, 0, 1<<16), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			p, same := diffPath(line)
			if !same {
				return false
			}
			c.Files = append(c.Files, p)
		case strings.HasPrefix(line, "@@"), strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "-"), strings.HasPrefix(line, "+"), strings.HasPrefix(line, `\ `), line == "":
		default: // a mode change, a new, deleted, renamed or copied file, binary
			return false
		}
	}
	if len(c.Files) < 2 || len(c.Files) > maxFiles {
		return false
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
			return true
		}
	}
	return false
}

type lineToken struct {
	off int
	tok token.Token
	lit string
}

// goTokens scans a Go file, comments included; source that does not scan is
// not a substitution.
func goTokens(s string) ([]lineToken, bool) {
	var sc scanner.Scanner
	bad := false
	src := []byte(s)
	sc.Init(token.NewFileSet().AddFile("", -1, len(src)), src, func(token.Position, string) { bad = true }, scanner.ScanComments)
	var out []lineToken
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			return out, !bad
		}
		if tok == token.SEMICOLON && lit == "\n" {
			continue // inserted at a line's end
		}
		out = append(out, lineToken{int(pos) - 1, tok, lit})
	}
}

// replaces says whether the file after is the file before with string literal
// tokens replaced, and nothing else: every other token, comments included,
// and the text between tokens stay as they were. Every replacement in the
// commit must be the same pair of literal values; the first one found fixes
// it.
func (c *CoupledCandidate) replaces(before, after string) bool {
	bt, ok1 := goTokens(before)
	at, ok2 := goTokens(after)
	if !ok1 || !ok2 || len(bt) != len(at) {
		return false
	}
	rebuilt, last, n := "", 0, 0
	for i := range bt {
		if bt[i].tok == at[i].tok && bt[i].lit == at[i].lit {
			continue
		}
		if bt[i].tok != token.STRING || at[i].tok != token.STRING {
			return false
		}
		old, err1 := strconv.Unquote(bt[i].lit)
		nw, err2 := strconv.Unquote(at[i].lit)
		if err1 != nil || err2 != nil || old == nw {
			return false
		}
		if !c.paired {
			c.Old, c.New, c.paired = old, nw, true
		} else if c.Old != old || c.New != nw {
			return false
		}
		rebuilt += before[last:bt[i].off] + at[i].lit
		last = bt[i].off + len(bt[i].lit)
		n++
	}
	return n > 0 && rebuilt+before[last:] == after
}
