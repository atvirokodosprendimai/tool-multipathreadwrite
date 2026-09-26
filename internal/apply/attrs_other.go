//go:build !windows

package apply

// keepAttributes has nothing to carry outside Windows: a POSIX file's mode is
// kept by stageFile's chmod, and there are no Hidden or System attributes.
func keepAttributes(from, to string) error { return nil }
