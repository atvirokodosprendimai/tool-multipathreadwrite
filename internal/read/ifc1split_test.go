package read

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// TestTheUnchangedAnswerCountsTheBytesItDidNotServe holds the line and byte
// counts the --if-changed answer gives for a file that still has the quoted sha,
// for every terminator lines.Split knows, a byte order mark, an empty file, a
// range, context, the line after a range, and --max-lines. Every count is written
// out by hand, so the oracle shares nothing with spanSize.
func TestTheUnchangedAnswerCountsTheBytesItDidNotServe(t *testing.T) {
	one := 1
	for _, tc := range []struct {
		body    string
		ranges  []Range
		context int
		next    bool
		max     *int
		n, size int
	}{
		{body: "a\nbb\n", n: 2, size: 5},
		{body: "a\nbb\n", max: &one, n: 1, size: 2},
		{body: "a\nbb", n: 2, size: 4},
		{body: "a\r\nbb\r\n", n: 2, size: 7},
		{body: "a\r\nbb\r\n", max: &one, n: 1, size: 3},
		{body: "a\r\nbb", n: 2, size: 5},
		{body: "a\rbb\r", n: 2, size: 5},
		{body: "a\rbb\r", max: &one, n: 1, size: 2},
		{body: "a\rbb", n: 2, size: 4},
		{body: "a\nb\r\nc\n", n: 3, size: 7},
		{body: "solo", n: 1, size: 4},
		{body: "solo\n", max: &one, n: 1, size: 5},
		{body: "\xef\xbb\xbfa\nb\n", n: 2, size: 7},
		{body: "\xef\xbb\xbfa\nb\n", max: &one, n: 1, size: 5},
		{body: "", n: 0, size: 0},
		{body: "a\nbb\nccc\n", ranges: []Range{{Start: 2, End: 2, Text: "2"}}, n: 1, size: 3},
		{body: "a\nbb\nccc\n", ranges: []Range{{Re: regexp.MustCompile("bb"), Text: "/bb/"}}, context: 1, n: 3, size: 9},
		{body: "a\nbb\nccc\n", ranges: []Range{{Start: 1, End: 1, Text: "1"}}, next: true, n: 2, size: 5},
		{body: "a\r\nbb\r\nccc", ranges: []Range{{Start: 2, End: 3, Text: "2-3"}}, n: 2, size: 7},
	} {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(tc.body), 0o644); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		opt := Options{IfChanged: map[string]string{"f.txt": seen.SHA([]byte(tc.body))[:8]}, MaxLines: tc.max, Context: tc.context, NextLine: tc.next}
		_, problems := Run(&out, root, []Spec{{Path: "f.txt", Ranges: tc.ranges}}, opt)
		want := fmt.Sprintf("— %d lines, %d bytes not served\n", tc.n, tc.size)
		if problems != 0 || !bytes.HasSuffix(out.Bytes(), []byte(want)) {
			t.Errorf("%q ranges %v -C %d next %v max-lines %v: problems %d, answer %q; want it to end %q", tc.body, tc.ranges, tc.context, tc.next, tc.max != nil, problems, out.String(), want)
		}
	}
}
