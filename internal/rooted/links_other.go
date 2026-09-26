//go:build !windows

package rooted

// followLinks is false where EvalSymlinks already follows every link and the
// filesystem keeps a name as written, so Resolve and Abs are unchanged there
// (ADR-071).
const followLinks = false

// opensDevice is asked only where followLinks is true: no name in a POSIX
// directory opens a device by being spelled like one (ADR-076).
func opensDevice(string) bool { return false }
