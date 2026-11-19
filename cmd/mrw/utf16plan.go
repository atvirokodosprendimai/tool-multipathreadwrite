package main

import "fmt"

// utf16Plan reports whether a plan's first bytes are UTF-16: a byte order mark
// of either order, or, with no mark, the two characters every plan document
// opens with (`@@`, apply_patch's `**`, a SEARCH block's `<<`) each beside a
// NUL. A NUL is valid UTF-8 and the parsers allow one in a comment or a
// preamble, so any other pair beside NULs is left to them.
func utf16Plan(head []byte) bool {
	if len(head) >= 2 && (head[0] == 0xff && head[1] == 0xfe || head[0] == 0xfe && head[1] == 0xff) {
		return true
	}
	if len(head) < 4 {
		return false
	}
	opens := func(c byte) bool { return c == '@' || c == '*' || c == '<' }
	return opens(head[0]) && head[1] == 0 && head[2] == head[0] && head[3] == 0 ||
		head[0] == 0 && opens(head[1]) && head[2] == 0 && head[3] == head[1]
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
