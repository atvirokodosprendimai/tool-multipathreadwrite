package bench

import (
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
	"path"
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

// CoupledLiteral is one literal ADR-160's check-coupled rule admits, as Zy
// amended it on 2026-10-15 ("Construct from the tree"): the history held no
// commit the first rule admitted.
type CoupledLiteral struct {
	File     string   // the non-test file holding it once, slash-separated from the module root
	Old, New string   // its value, and the value with the first letter's case flipped
	Tests    []string // the _test.go files of its directory that hold the same value
}

// MineCoupledLiterals lists the module's literals at root that ADR-160
// Decision 1 admits as check-coupled tasks, in the order of sha256("adr-160:" +
// path + ":" + value): a string literal in a non-test .go file — not an import
// path, not a struct tag — starting with a letter whose case can flip,
// appearing exactly once in that file and in no other non-test file of the
// module, whose value a _test.go file in the same directory also holds as a
// literal. Generated files, vendor, testdata and dot or underscore directories
// are skipped, as for renames.
func MineCoupledLiterals(root string) ([]CoupledLiteral, error) {
	type lit struct{ file, dir string }
	src := map[string][]lit{}             // value -> the non-test files holding it, once per occurrence
	tests := map[string]map[string]bool{} // dir + "\x00" + value -> the test files holding it
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
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		first, _, _ := strings.Cut(string(body), "\n")
		if generated.MatchString(strings.TrimRight(first, "\r")) {
			return nil
		}
		f, err := parser.ParseFile(fset, p, body, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		skip := map[*ast.BasicLit]bool{}
		for _, imp := range f.Imports {
			skip[imp.Path] = true
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if fl, ok := n.(*ast.Field); ok && fl.Tag != nil {
				skip[fl.Tag] = true
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING || skip[bl] {
				return true
			}
			v, err := strconv.Unquote(bl.Value)
			if err != nil {
				return true
			}
			if strings.HasSuffix(rel, "_test.go") {
				k := path.Dir(rel) + "\x00" + v
				if tests[k] == nil {
					tests[k] = map[string]bool{}
				}
				tests[k][rel] = true
			} else {
				src[v] = append(src[v], lit{rel, path.Dir(rel)})
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	var out []CoupledLiteral
	for v, at := range src {
		nv, ok := flipFirst(v)
		if len(at) != 1 || !ok {
			continue
		}
		held := tests[at[0].dir+"\x00"+v]
		if len(held) == 0 {
			continue
		}
		c := CoupledLiteral{File: at[0].file, Old: v, New: nv}
		for t := range held {
			c.Tests = append(c.Tests, t)
		}
		sort.Strings(c.Tests)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		return saltedHash(mineSalt, out[i].File+":"+out[i].Old) < saltedHash(mineSalt, out[j].File+":"+out[j].Old)
	})
	return out, nil
}

// flipFirst is v with its first letter's case flipped, and false when v does
// not start with a letter whose case changes.
func flipFirst(v string) (string, bool) {
	r, size := utf8.DecodeRuneInString(v)
	if size == 0 || !unicode.IsLetter(r) {
		return "", false
	}
	f := unicode.ToUpper(r)
	if unicode.IsUpper(r) {
		f = unicode.ToLower(r)
	}
	if f == r {
		return "", false
	}
	return string(f) + v[size:], true
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
