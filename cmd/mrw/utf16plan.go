package main

import "fmt"

// utf16Plan reports whether a plan's first bytes are UTF-16: a byte order mark
// of either order, or two characters of ASCII each beside a NUL, which is what
// a plan's `@@` is with no mark. A UTF-8 plan has no NUL in its first four
// bytes, and a NUL deeper in a body line is not looked at.
func utf16Plan(head []byte) bool {
	if len(head) >= 2 && (head[0] == 0xff && head[1] == 0xfe || head[0] == 0xfe && head[1] == 0xff) {
		return true
	}
	if len(head) < 4 {
		return false
	}
	return head[0] != 0 && head[1] == 0 && head[2] != 0 && head[3] == 0 ||
		head[0] == 0 && head[1] != 0 && head[2] == 0 && head[3] != 0
}

// utf16Refusal is the one sentence for a plan in UTF-16. Windows PowerShell
// 5.1 writes UTF-16 when output is redirected with `>`, and the parser's own
// answer was an error per line quoting it as hex.
func utf16Refusal(name string) string {
	return fmt.Sprintf("%s: the plan is UTF-16 and mrw reads UTF-8; Windows PowerShell 5.1 writes UTF-16 when output is redirected with >. "+
		"Write it with `| Out-File -Encoding utf8 FILE`, redirect in cmd.exe, or pipe the plan to `mrw write -`; "+
		"a --sub plan can be made and written in one call with `mrw write --grep PATTERN --sub REPLACEMENT`", name)
}
