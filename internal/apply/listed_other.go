//go:build !darwin

package apply

// spelledByDescriptor is darwin's cheap way to the stored name of a file; elsewhere
// the directory is listed.
func spelledByDescriptor(*tree, string) (string, bool, bool) { return "", false, false }
