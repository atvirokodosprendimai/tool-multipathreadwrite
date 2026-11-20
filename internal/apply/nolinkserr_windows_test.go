//go:build windows

package apply

// The Win32 errors noHardLinks accepts: Go's synthetic EXDEV and EPERM are not
// among them on Windows.
var (
	errNoLinksA error = errorNotSameDevice
	errNoLinksB error = errorInvalidFunction
)
