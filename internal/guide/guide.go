// Package guide holds the sentences every mrw surface must teach.
//
// ADR-037: Shared is the contract a caller with only the binary is entitled
// to; a surface may print more after it, never a different version of it.
package guide

// Shared is the five sentences both the CLI pamphlet and the MCP handshake
// must contain verbatim. It is not interpolated: examples stay in the
// surface that executes them.
func Shared() string {
	return shared
}

const shared = `Reach for mrw when the task touches 3 or more edits, 2 or more files, or several ranges you need to read.
A plan applies whole or not at all: if any hunk fails, nothing is written.
Read before you write, per line, not per file.
mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.
A refusal names the file, the plan line, and the reason.`

// CLI is Shared plus the operator traps that only the shell surface can hit.
// Stdout of `mrw instructions` is exactly this string.
func CLI() string {
	return Shared() + `

Never read an exit code through a pipe: mrw write plan | head returns head's status.
Exit 3 means the write applied and the check failed, so the tree is changed and unverified.
MSYS rewrites a regex address before mrw starts. Export MSYS2_ARG_CONV_EXCL='*', or use PowerShell or WSL.
A shell glob and an address suffix do not mix.
`
}
