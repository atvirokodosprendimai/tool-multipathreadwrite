//go:build windows

package rooted

// followLinks is true on Windows, where Go 1.23+ neither marks a junction as a
// symlink nor follows one in EvalSymlinks, where Win32 reads a name that ends
// in a dot or a space as another (ADR-071), and where a reserved device name is
// refused (ADR-081).
const followLinks = true
