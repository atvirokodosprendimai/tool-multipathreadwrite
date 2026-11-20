package seen

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

// #702. save leaves out a record the ledger cannot hold (a name with a line
// feed, a record over the bound) and Record returns nil, as it always did: a
// write, an ack and a content read go on. Unrecorded is what a read that exists
// to license lines asks afterwards, and it names each path the ledger lacks.
func TestALicenceTheLedgerCannotHoldIsNamedNotReported(t *testing.T) {
	sha := strings.Repeat("a", 64)
	spans := make([][2]int, 0, 1000)
	for i := 0; i < 1000; i++ {
		spans = append(spans, [2]int{2*i + 1, 2*i + 1})
	}
	cases := []struct {
		name    string
		path    string
		obs     Observation
		bound   int64
		reason  string
		skipWin bool
	}{
		{name: "line feed", path: "a\nb", obs: Observation{SHA: sha}, reason: "line feed", skipWin: true},
		{name: "over the bound", path: "big.txt", obs: Observation{SHA: sha, Spans: spans}, bound: 1000, reason: "over 1000 bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.skipWin && runtime.GOOS == "windows" {
				t.Skip("a line feed is not a legal Windows file name")
			}
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			if c.bound > 0 {
				old := maxRecordBytes
				t.Cleanup(func() { maxRecordBytes = old })
				maxRecordBytes = c.bound
			}
			root := t.TempDir()
			obs := map[string]Observation{c.path: c.obs, "ok.txt": {SHA: sha}, "none.txt": {SHA: sha, Spans: [][2]int{}}}
			if err := Record(root, obs); err != nil {
				t.Fatalf("Record of an unrecordable path returned %v, want nil as on main", err)
			}
			err := Unrecorded(root, obs)
			var om *OmittedError
			if !errors.As(err, &om) || len(om.Paths) != 1 || !strings.Contains(err.Error(), c.reason) {
				t.Fatalf("Unrecorded returned %v, want one omitted path with %q", err, c.reason)
			}
			if err := Unrecorded(root, map[string]Observation{"ok.txt": {SHA: sha}, "none.txt": {SHA: sha, Spans: [][2]int{}}}); err != nil {
				t.Errorf("a recorded sibling and a path with no licensed line were refused: %v", err)
			}
			// A path the ledger could not hold, but that licensed no line, is not asked about.
			if c.skipWin {
				if err := Unrecorded(root, map[string]Observation{"z\nz": {SHA: sha, Spans: [][2]int{}}}); err != nil {
					t.Errorf("an unrecordable path that licensed no line was refused: %v", err)
				}
			}
		})
	}
}
