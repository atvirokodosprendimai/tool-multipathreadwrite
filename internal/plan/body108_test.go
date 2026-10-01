package plan

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

// ADR-108 T3. LoadBodyFiles split a body file on \n alone and kept each \r, so
// a CRLF body into a CRLF file came out \r\r\n. A body file is split like every
// other text mrw reads (lines.Split, ADR-065): no body line holds a terminator,
// except a mixed-ending file, where lines.Split keeps the minority one in place.
func TestABodyFileKeepsTheTargetsLineEndings(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"lf":      "a\nb\n",
		"crlf":    "a\r\nb\r\n",
		"cr":      "a\rb\r",
		"mixed":   "a\r\nb\nc\r",
		"nofinal": "a\r\nb",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		hunks := []Hunk{{BodyFile: name, SrcLine: 1}}
		if err := LoadBodyFiles(root, hunks); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want, _, _ := lines.Split(content)
		if !slices.Equal(hunks[0].Body, want) {
			t.Errorf("%s: body %q, want %q", name, hunks[0].Body, want)
		}
		for _, l := range hunks[0].Body {
			if name == "mixed" {
				break
			}
			if strings.ContainsAny(l, "\r\n") {
				t.Errorf("%s: a body line holds a terminator: %q", name, l)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(root, "empty"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	hunks := []Hunk{{BodyFile: "empty", SrcLine: 1}}
	if err := LoadBodyFiles(root, hunks); err != nil || hunks[0].Body != nil {
		t.Errorf("an empty body file: body %q, err %v; want nil", hunks[0].Body, err)
	}
}
