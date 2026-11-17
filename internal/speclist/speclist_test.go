package speclist

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo pins ADR-117 T2: a
// list saved by an editor or a PowerShell pipe that writes UTF-8 with a byte
// order mark served its first spec as "\ufeffgo.mod", UNREADABLE, and read a
// leading "# comment" as a path. The mark is dropped from the first line only,
// and the note names the list.
func TestAListThatStartsWithAByteOrderMarkDropsItAndSaysSo(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
		noted    bool
	}{
		{"first spec", "\ufeffgo.mod:1\nREADME.md\n", []string{"go.mod:1", "README.md"}, true},
		{"first comment", "\ufeff# specs\ngo.mod:1\n", []string{"go.mod:1"}, true},
		{"no mark", "go.mod:1\n", []string{"go.mod:1"}, false},
		{"a mark past the first line is the spec's", "go.mod\n\ufeffx\n", []string{"go.mod", "\ufeffx"}, false},
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
			if said := strings.Contains(notes.String(), "--files-from l: dropped the byte order mark"); said != c.noted {
				t.Errorf("note = %q, want a note: %v", notes.String(), c.noted)
			}
		})
	}
	// A list that is only the mark holds no spec, and says so as any empty list does.
	if _, err := Parse(strings.NewReader("\ufeff\n"), "--files-from l", nil); err == nil || !strings.Contains(err.Error(), "no specs") {
		t.Errorf("a list of only the mark: err = %v, want no specs", err)
	}
}
