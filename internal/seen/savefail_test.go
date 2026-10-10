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
	for _, want := range []string{"could not be updated", "lines served", "changes applied", "not recorded", "read the files again"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the message lacks %q:\n%s", want, msg)
		}
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("the cause is lost to errors.Is: %v", err)
	}
}

// A ledger that cannot be OPENED fails the same way: after a call that acted it
// says what stands (the v1.60.0 retest saw a raw "open … used by another
// process"), and before one that writes it says nothing was written.
func TestAnUnopenableLedgerSaysWhatStandsAfterACallAndThatNothingChangedBefore(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("an unreadable ledger is refused by mode on unix, as a non-root user")
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
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	err = Record(root, map[string]Observation{"b.txt": {SHA: SHA([]byte("two\n"))}})
	if err == nil || !strings.Contains(err.Error(), "could not be read to record") || !strings.Contains(err.Error(), "lines served") || !strings.Contains(err.Error(), "not recorded") || !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Record on an unopenable ledger: %v, want what stands and the cause", err)
	}
	var unreadable *UnreadableError
	if _, err = Snapshot(root); !errors.As(err, &unreadable) || !errors.Is(err, fs.ErrPermission) || strings.Contains(err.Error(), "wrote") {
		t.Errorf("Snapshot on an unopenable ledger: %v, want an UnreadableError that makes no claim about writes, and the cause", err)
	}
}
