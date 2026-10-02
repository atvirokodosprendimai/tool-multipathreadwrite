package mcp

import (
	"maps"
	"testing"
)

// The review of #318's merge: read.Run reads a file once per spec, so a writer
// between two specs of one file leaves slices of two versions. Merging every
// slice held the older version's spans under the newer sha, and an ack of them
// licensed a line nobody was served from the file now on disk. markServed keys
// each slice by its header's sha, and heldSpans keeps only the observed one's.
func TestASpanFromAReplacedVersionIsNotHeld(t *testing.T) {
	report := "==> a.txt  4L  12B  sha aaaaaaaa\n@@ 1-1\n    1| A1\n" +
		"==> a.txt  4L  12B  sha bbbbbbbb\n@@ 3-3\n    3| B3\n" +
		"==> a.txt  4L  12B  sha bbbbbbbb\n@@ 4-4\n    4| B4\n"
	_, byPath := markServed(report)
	bySHA := byPath["a.txt"]
	if len(bySHA["aaaaaaaa"]) != 1 || len(bySHA["bbbbbbbb"]) != 2 {
		t.Fatalf("slices not kept per version: %v", bySHA)
	}
	held := heldSpans(bySHA, "bbbbbbbb"+"0123456789abcdef0123456789abcdef0123456789abcdef01234567")
	if len(held) != 2 {
		t.Fatalf("held %v, want the two spans served from the observed version", held)
	}
	for _, sp := range held {
		if sp[0] == 1 {
			t.Errorf("line 1, served from a replaced version, is held: %v", held)
		}
	}
	if !maps.Equal(held, bySHA["bbbbbbbb"]) {
		t.Errorf("the observed version's spans were not merged whole: %v", held)
	}
}
