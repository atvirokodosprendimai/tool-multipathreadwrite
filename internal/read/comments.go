package read

import (
	"go/scanner"
	"go/token"
	"strings"
)

// commentLines marks, 1-based, the lines of a Go source that hold a comment and
// nothing else (ADR-179 Decision h): scanned whole with go/scanner, so a line
// inside a /* */ block is known whatever range was asked for and a // inside a
// string is no comment. A line with code before or after its comment, a
// //go:, //line or //export directive, and the comment directly above an import
// spec "C" (the cgo preamble), grouped or not, are not marked. nil when the
// source does not scan.
func commentLines(src []byte) []bool {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var sc scanner.Scanner
	bad := false
	sc.Init(file, src, func(token.Position, string) { bad = true }, scanner.ScanComments)
	n := strings.Count(string(src), "\n") + 2
	com, code, keep := make([]bool, n+1), make([]bool, n+1), make([]bool, n+1)
	prev := token.ILLEGAL
	var importC []int // the lines of the "C" import specs
	inImport, group := false, false
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		line := file.PositionFor(pos, false).Line // not Line: it follows //line directives
		end := line + strings.Count(lit, "\n")
		if end >= len(com) {
			return nil
		}
		switch {
		case tok == token.COMMENT:
			directive := strings.HasPrefix(lit, "//go:") || strings.HasPrefix(lit, "//line ") || strings.HasPrefix(lit, "/*line ") || strings.HasPrefix(lit, "//export ")
			for l := line; l <= end; l++ {
				com[l] = true
				keep[l] = keep[l] || directive
			}
		case tok == token.SEMICOLON && lit == "\n":
		default:
			for l := line; l <= end; l++ {
				code[l] = true
			}
		}
		if tok != token.COMMENT {
			switch {
			case tok == token.IMPORT:
				inImport, group = true, false
			case tok == token.LPAREN && inImport && prev == token.IMPORT:
				group = true
			case tok == token.RPAREN && inImport && group, tok == token.SEMICOLON && inImport && !group:
				inImport = false
			case tok == token.STRING && inImport && lit == `"C"`:
				importC = append(importC, line)
			}
			prev = tok
		}
	}
	if bad {
		return nil
	}
	for _, il := range importC {
		for l := il - 1; l > 0 && com[l] && !code[l]; l-- {
			keep[l] = true
		}
	}
	out := make([]bool, len(com))
	for l := range out {
		out[l] = com[l] && !code[l] && !keep[l]
	}
	return out
}
