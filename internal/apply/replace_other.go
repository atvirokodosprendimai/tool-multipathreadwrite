//go:build !windows

package apply

// platformCause names nothing beyond what causeOf already reads off a portable
// error: on unix a refused open is a permission or a missing file.
func platformCause(error) string { return "" }

// replaceable answers yes on unix: a rename over a file, or its removal, does
// not care who has it open.
func replaceable(string) error { return nil }
