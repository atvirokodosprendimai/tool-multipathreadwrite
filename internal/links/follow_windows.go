//go:build windows

package links

// Follow is true on Windows, where Go 1.23+ neither marks a junction as a
// symlink nor follows one in EvalSymlinks, so the walk has to (ADR-071).
const Follow = true
