package read

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// mdHeading is an ATX heading: up to three spaces, one to six #, then either
// nothing (an empty heading) or whitespace and the text; mdClosing is the
// optional closing sequence, which needs whitespace before it (`# C#` ends in a
// C#); mdFence is a code fence, ``` or ~~~ at least three long.
var (
	mdHeading = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$`)
	mdClosing = regexp.MustCompile(`^(.*?)[ \t]+#+$`)
	mdFence   = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})(.*)$")
)

// outlineRows is a file's structure for --outline (ADR-179 Decision b): one row
// `A-B text` per top-level item, from go/parser for .go files and a scan of ATX
// headings outside fenced blocks for .md and .markdown. supported is false for
// any other extension; err is the parse error of a .go file, the rows being
// what parsed before it.
func outlineRows(path string, src []byte, lines []string) (rows []string, supported bool, err error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		rows, err = goOutline(path, src)
		return rows, true, err
	case ".md", ".markdown":
		return markdownOutline(lines), true, nil
	}
	return nil, false, nil
}

func goOutline(path string, src []byte) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution|parser.ParseComments)
	if f == nil {
		return nil, err
	}
	var rows []string
	for _, d := range f.Decls {
		var doc *ast.CommentGroup
		var text string
		switch d := d.(type) {
		case *ast.FuncDecl:
			doc, text = d.Doc, "func "+recvText(d)+d.Name.Name
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			doc, text = d.Doc, d.Tok.String()+" "+specNames(d)
		default:
			continue
		}
		start := d.Pos()
		if doc != nil {
			start = doc.Pos()
		}
		first := fset.PositionFor(start, false).Line
		last := "?" // a declaration cut off by a parse error has no end
		if e := fset.PositionFor(d.End(), false).Line; e >= first {
			last = strconv.Itoa(e)
		}
		rows = append(rows, fmt.Sprintf("%d-%s %s", first, last, strings.TrimSpace(text)))
	}
	return rows, err
}

// recvText is a method's receiver as written, `(s *Store) ` or `(*Store) `.
func recvText(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	fld := d.Recv.List[0]
	if len(fld.Names) > 0 {
		return "(" + fld.Names[0].Name + " " + types.ExprString(fld.Type) + ") "
	}
	return "(" + types.ExprString(fld.Type) + ") "
}

// specNames are the names a type, var or const declaration declares: all of
// them up to four, then the first four and an ellipsis.
func specNames(d *ast.GenDecl) string {
	var names []string
	for _, s := range d.Specs {
		switch s := s.(type) {
		case *ast.TypeSpec:
			names = append(names, s.Name.Name)
		case *ast.ValueSpec:
			for _, n := range s.Names {
				names = append(names, n.Name)
			}
		}
	}
	if len(names) > 4 {
		return strings.Join(names[:4], ", ") + ", …"
	}
	return strings.Join(names, ", ")
}

// markdownOutline lists the ATX headings outside fenced blocks; a heading's
// range ends before the next heading of its level or a higher one.
func markdownOutline(lines []string) []string {
	type heading struct {
		at, level int
		text      string
	}
	var hs []heading
	fenceCh, fenceLen := byte(0), 0 // the open fence's character and length; 0 length is none
	for i, l := range lines {
		f := mdFence.FindStringSubmatch(l)
		if fenceLen > 0 { // closes on the same character, at least as long, nothing after it
			if f != nil && f[2][0] == fenceCh && len(f[2]) >= fenceLen && strings.TrimSpace(f[3]) == "" {
				fenceLen = 0
			}
			continue
		}
		if f != nil && !(f[2][0] == '`' && strings.Contains(f[3], "`")) {
			fenceCh, fenceLen = f[2][0], len(f[2])
			continue
		}
		if m := mdHeading.FindStringSubmatch(l); m != nil {
			text := strings.TrimSpace(m[2])
			if strings.Trim(text, "#") == "" {
				text = ""
			} else if c := mdClosing.FindStringSubmatch(text); c != nil {
				text = strings.TrimSpace(c[1])
			}
			label := m[1]
			if text != "" {
				label += " " + text
			}
			hs = append(hs, heading{at: i + 1, level: len(m[1]), text: label})
		}
	}
	rows := make([]string, 0, len(hs))
	for i, h := range hs {
		end := len(lines)
		for _, n := range hs[i+1:] {
			if n.level <= h.level {
				end = n.at - 1
				break
			}
		}
		rows = append(rows, fmt.Sprintf("%d-%d %s", h.at, end, h.text))
	}
	return rows
}
