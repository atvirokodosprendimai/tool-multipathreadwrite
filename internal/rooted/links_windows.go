//go:build windows

package rooted

// followLinks is true on Windows, where Go 1.23+ neither marks a junction as a
// symlink nor follows one in EvalSymlinks, and where Win32 reads a name that
// ends in a dot or a space as another (ADR-071).
const followLinks = true
