package speclist

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// TestAListDropsEveryLeadingByteOrderMarkAndCountsThem pins ADR-117 T3: T2
// dropped one mark, so a list Windows PowerShell 5.1 piped with two served its
// first spec with the second mark in front of the path, UNREADABLE, under a
// note that said the mark was gone. Every mark at the head of the first line
// is dropped; the note counts them when there are several.
func TestAListDropsEveryLeadingByteOrderMarkAndCountsThem(t *testing.T) {
	const mark = "\xef\xbb\xbf"
	cases := []struct {
		name, in string
		want     []string
		note     string
	}{
		{"two marks", mark + mark + "go.mod:1\nREADME.md\n", []string{"go.mod:1", "README.md"},
			"mrw: --files-from l: dropped 2 byte order marks at the start of the list\n"},
		{"three marks before a comment", mark + mark + mark + "# specs\ngo.mod:1\n", []string{"go.mod:1"},
			"mrw: --files-from l: dropped 3 byte order marks at the start of the list\n"},
		{"one mark keeps its note", mark + "go.mod:1\n", []string{"go.mod:1"},
			"mrw: --files-from l: dropped the byte order mark at the start of the list\n"},
		{"marks past the first line are the spec's", "go.mod\n" + mark + mark + "x\n", []string{"go.mod", mark + mark + "x"}, ""},
		{"a mark after ./ is the name's", "./" + mark + "a.txt\n", []string{"./" + mark + "a.txt"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var notes bytes.Buffer
			got, err := Parse(strings.NewReader(c.in), "--files-from l", &notes)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("specs = %q, want %q", got, c.want)
			}
			if notes.String() != c.note {
				t.Errorf("note = %q, want %q", notes.String(), c.note)
			}
		})
	}
}
