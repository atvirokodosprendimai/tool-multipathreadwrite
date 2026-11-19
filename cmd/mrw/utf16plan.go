package main

import "fmt"

// utf16Mark reports whether bytes open with a UTF-16 byte order mark of either
// order. It is all that is asked of an apply_patch or SEARCH/REPLACE document:
// with no mark their first bytes can be a path or a blank line, and a NUL is
// valid UTF-8 there.
func utf16Mark(head []byte) bool {
	return len(head) >= 2 && (head[0] == 0xff && head[1] == 0xfe || head[0] == 0xfe && head[1] == 0xff)
}

// utf16Plan reports whether a native plan's first bytes are UTF-16: a byte
// order mark, or, with none, the `@@` a plan opens with, each `@` beside a NUL.
// A UTF-8 plan that opens `@`, NUL, `@`, NUL is no header and would be refused
// by the parser anyway; a comment or a blank line first is left to the parser.
func utf16Plan(head []byte) bool {
	if utf16Mark(head) {
		return true
	}
	if len(head) < 4 {
		return false
	}
	return head[0] == '@' && head[1] == 0 && head[2] == '@' && head[3] == 0 ||
		head[0] == 0 && head[1] == '@' && head[2] == 0 && head[3] == '@'
}

// utf16Refusal is the one sentence for a plan in UTF-16. Windows PowerShell
// 5.1 writes UTF-16 when output is redirected with `>`, and the parser's own
// answer was an error per line quoting it as hex. It does not offer a pipe to
// `mrw write -`: that shell's pipe turns non-ASCII text into `?`.
func utf16Refusal(name string) string {
	return fmt.Sprintf("%s: the plan is UTF-16 and mrw reads UTF-8; Windows PowerShell 5.1 writes UTF-16 when output is redirected with >. "+
		"Write it as UTF-8 with `| Out-File -Encoding utf8 FILE` (in Windows PowerShell 5.1 set [Console]::OutputEncoding to UTF-8 first if the plan holds non-ASCII text), or redirect in cmd.exe; "+
		"a --sub plan can be made and written in one call with `mrw write --grep PATTERN --sub REPLACEMENT`", name)
}
