package adversarial

import (
	"os"
	"testing"
)

// TestMain clears MRW_SHOWN for the package: the mrw these tests build and run
// inherits this process's environment, and a developer's own MRW_SHOWN=off,
// which a host sets for a session (ADR-218), would make its reads licence-only
// and refuse the numbered plans the tests submit.
func TestMain(m *testing.M) {
	os.Unsetenv("MRW_SHOWN")
	os.Exit(m.Run())
}
