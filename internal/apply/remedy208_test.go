package apply

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Review of #745. The ledger is per spelling, so a plan that names a file by an
// alias (a case-folded name, a symlink) is told to read it under that spelling,
// and a read of the one line after the range leaves a partial record under the
// alias that the retry refuses again. The remedy names the range from the start
// through the line after it, so following it lands.
func TestTheLineAfterRemedyCoversTheEditedLinesUnderThePlansOwnSpelling(t *testing.T) {
	const body = "func A() {\n\tif x {\n\t}\n}\n"
	sha := seen.SHA([]byte(body))
	span := regexp.MustCompile(`mrw read --max-cols 0 (\S+):(\d+)-(\d+)$`)
	for _, spelling := range []string{"f.go", "F.go", "ln.go"} {
		t.Run(spelling, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "f.go"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			switch spelling {
			case "F.go":
				if _, err := os.Stat(filepath.Join(root, "F.go")); err != nil {
					t.Skip("this filesystem is case-sensitive")
				}
			case "ln.go":
				if err := os.Symlink("f.go", filepath.Join(root, "ln.go")); err != nil {
					t.Skipf("no symlinks: %v", err)
				}
			}
			in := []Input{{Path: spelling, Op: "replace", Start: 1, End: 3, Anchor: "func A", Lines: -1,
				Body: []string{"func A() {", "\ty()", "}"}}}
			res, err := Apply(root, in, Options{Seen: map[string]Seen{spelling: {SHA: sha, Written: true}}})
			if err != nil {
				t.Fatal(err)
			}
			var reason string
			for _, h := range res.Hunks {
				if h.Status == StatusFailed {
					reason = h.Reason
				}
			}
			m := span.FindStringSubmatch(reason)
			if m == nil || m[1] != spelling || m[2] != "1" || m[3] != "4" {
				t.Fatalf("reason %q does not end with a read of %s:1-4", reason, spelling)
			}
			// Following the remedy: the read leaves a record of those lines under
			// the plan's spelling, and the same plan lands.
			from, _ := strconv.Atoi(m[2])
			to, _ := strconv.Atoi(m[3])
			res, err = Apply(root, in, Options{Seen: map[string]Seen{spelling: {SHA: sha, Spans: [][2]int{{from, to}}}}})
			if err != nil {
				t.Fatal(err)
			}
			if !res.Applied || res.Failed != 0 {
				t.Fatalf("the plan after the named read was refused: %+v", res.Hunks)
			}
			got, _ := os.ReadFile(filepath.Join(root, "f.go"))
			if !strings.Contains(string(got), "y()") {
				t.Errorf("f.go = %q", got)
			}
		})
	}
}
