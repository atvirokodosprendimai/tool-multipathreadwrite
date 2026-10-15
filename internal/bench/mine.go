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

// countDecls counts every identifier f declares, as go/types records them in
// Info.Defs: functions and methods; types and type parameters, a generic
// method's receiver type parameters included; constants and variables; fields,
// an embedded field by its type's name; interface methods; parameters and
// results; range variables, labels and import names; and a short variable
// declaration's names that are new in its scope — a redeclared name is not a
// declaration (the review of #426).
func countDecls(f *ast.File, decls map[string]int) {
	ast.Walk(declVisitor{decls, map[string]bool{}}, f)
}

type declVisitor struct {
	decls map[string]int
	scope map[string]bool // the names declared in the innermost scope
}

func (v declVisitor) add(ids ...*ast.Ident) {
	for _, id := range ids {
		if id != nil && id.Name != "_" {
			v.decls[id.Name]++
			v.scope[id.Name] = true
		}
	}
}

func (v declVisitor) fields(fl *ast.FieldList) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		v.add(f.Names...)
		ast.Walk(v, f.Type)
	}
}

// function declares a function's receiver, type parameters, parameters and
// results in one scope and walks its body's statements in that same scope, as
// Go does: a body's := of a parameter's name declares nothing.
func (v declVisitor) function(recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt) {
	in := declVisitor{v.decls, map[string]bool{}}
	if recv != nil {
		for _, f := range recv.List {
			in.add(f.Names...)
			t := f.Type
			if s, ok := t.(*ast.StarExpr); ok {
				t = s.X
			}
			switch t := t.(type) {
			case *ast.IndexExpr:
				if id, ok := t.Index.(*ast.Ident); ok {
					in.add(id)
				}
			case *ast.IndexListExpr:
				for _, e := range t.Indices {
					if id, ok := e.(*ast.Ident); ok {
						in.add(id)
					}
				}
			}
		}
	}
	in.fields(typ.TypeParams)
	in.fields(typ.Params)
	in.fields(typ.Results)
	if body != nil {
		for _, s := range body.List {
			ast.Walk(in, s)
		}
	}
}

func (v declVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncDecl:
		v.add(n.Name)
		v.function(n.Recv, n.Type, n.Body)
		return nil
	case *ast.FuncLit:
		v.function(nil, n.Type, n.Body)
		return nil
	case *ast.FuncType:
		v.fields(n.TypeParams)
		v.fields(n.Params)
		v.fields(n.Results)
		return nil
	case *ast.BlockStmt, *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.CaseClause, *ast.CommClause:
		return declVisitor{v.decls, map[string]bool{}}
	case *ast.RangeStmt:
		in := declVisitor{v.decls, map[string]bool{}}
		if n.Tok == token.DEFINE {
			for _, e := range []ast.Expr{n.Key, n.Value} {
				if id, ok := e.(*ast.Ident); ok {
					in.add(id)
				}
			}
		}
		return in
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, e := range n.Lhs {
				if id, ok := e.(*ast.Ident); ok && !v.scope[id.Name] {
					v.add(id)
				}
			}
		}
	case *ast.TypeSpec:
		v.add(n.Name)
		v.fields(n.TypeParams)
		ast.Walk(v, n.Type)
		return nil
	case *ast.ValueSpec:
		v.add(n.Names...)
	case *ast.ImportSpec:
		v.add(n.Name)
	case *ast.LabeledStmt:
		v.add(n.Label)
	case *ast.StructType:
		for _, f := range n.Fields.List {
			if len(f.Names) == 0 {
				v.add(embeddedName(f.Type))
			}
			v.add(f.Names...)
			ast.Walk(v, f.Type)
		}
		return nil
	case *ast.InterfaceType:
		for _, f := range n.Methods.List {
			v.add(f.Names...)
			ast.Walk(v, f.Type)
		}
		return nil
	}
	return v
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
		if c.coupled(body, maxFiles) {
			c.Commit, c.Parent = ids[0], ids[1]
			cs = append(cs, c)
		}
	}
	sort.Slice(cs, func(i, j int) bool { return saltedHash(mineSalt, cs[i].Commit) < saltedHash(mineSalt, cs[j].Commit) })
	return cs, nil
}

// diffPath reads the path of a `diff --git a/P b/P` header whose two sides
// are one path, spaces included (git quotes a path only for characters such
// as a tab or a quote, and such a header is not read: the commit is skipped).
func diffPath(line string) (string, bool) {
	rest, ok := strings.CutPrefix(line, "diff --git a/")
	if !ok || (len(rest)-3)%2 != 0 || len(rest) < 5 {
		return "", false
	}
	p := rest[:(len(rest)-3)/2]
	return p, rest == p+" b/"+p
}

// coupled judges one commit's -U0 diff: every file it touches is changed only
// by replacing string literals' text, one pair throughout, with at least one
// replacement in each file; a mode change, a new, deleted, renamed, copied or
// binary file, or a change to a final newline disqualifies it (the review of
// #426).
func (c *CoupledCandidate) coupled(diff string, maxFiles int) bool {
	var minus, plus []string
	subs := 0
	ok := true
	flush := func() {
		if len(minus) != len(plus) {
			ok = false
		}
		for i := range minus {
			if i >= len(plus) || !c.replaces(minus[i], plus[i]) {
				ok = false
			}
		}
		subs += len(minus)
		minus, plus = nil, nil
	}
	endFile := func() {
		flush()
		if len(c.Files) > 0 && subs == 0 {
			ok = false
		}
		subs = 0
	}
	sc := bufio.NewScanner(strings.NewReader(diff))
	sc.Buffer(make([]byte, 0, 1<<16), 1<<24)
	for sc.Scan() && ok {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			endFile()
			p, same := diffPath(line)
			if !same {
				return false
			}
			c.Files = append(c.Files, p)
		case strings.HasPrefix(line, "@@"):
			flush()
		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "index "):
		case strings.HasPrefix(line, "-"):
			minus = append(minus, line[1:])
		case strings.HasPrefix(line, "+"):
			plus = append(plus, line[1:])
		case line == "":
		default: // a mode change, a new, deleted, renamed or copied file, binary, "\ No newline at end of file"
			return false
		}
	}
	endFile()
	if !ok || !c.paired || len(c.Files) < 2 || len(c.Files) > maxFiles {
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

// lineTokens scans one line as Go, comments included; a line that does not
// scan — inside a raw string that spans lines, say — is not a substitution.
func lineTokens(s string) ([]lineToken, bool) {
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
			continue // inserted at the line's end
		}
		out = append(out, lineToken{int(pos) - 1, tok, lit})
	}
}

// replaces says whether after is before with string literal tokens replaced,
// and nothing else: every other token, comments included, and the text between
// tokens stay as they were. Every replacement in the commit must be the same
// pair of literal values; the first one found fixes it.
func (c *CoupledCandidate) replaces(before, after string) bool {
	bt, ok1 := lineTokens(before)
	at, ok2 := lineTokens(after)
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
