package seen

import (
	"errors"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"testing"
)

// A ledger that cannot be replaced (on Windows, one another process holds open)
// failed a read as "rename <tmp> <ledger>: Access is denied" with exit 2, after
// the lines had been served: four of five Windows sessions on v1.59.0 read it
// as a failed read. It now says what stands and what does not, and keeps the
// cause for errors.Is.
func TestALedgerThatCannotBeUpdatedSaysWhatStandsAndWhatIsNotRecorded(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only ledger is refused by mode on unix, as a non-root user")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.txt": {SHA: SHA([]byte("one\n"))}}); err != nil {
		t.Fatal(err)
	}
	path, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	err = Record(root, map[string]Observation{"b.txt": {SHA: SHA([]byte("two\n"))}})
	if err == nil {
		t.Fatal("a ledger that cannot be replaced was recorded into")
	}
	msg := err.Error()
	for _, want := range []string{"could not be updated", "already served", "already applied", "not recorded", "read the files again"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the message lacks %q:\n%s", want, msg)
		}
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("the cause is lost to errors.Is: %v", err)
	}
}
