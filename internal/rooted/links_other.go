//go:build !windows

package rooted

// followLinks is false where EvalSymlinks already follows every link and the
// filesystem keeps a name as written, so Resolve and Abs are unchanged there
// (ADR-071).
const followLinks = false
