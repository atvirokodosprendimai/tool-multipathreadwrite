//go:build !windows

package links

// Follow is false where EvalSymlinks already follows every link, so Real and
// the boundary need no walk of their own (ADR-071).
const Follow = false
