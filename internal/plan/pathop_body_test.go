package plan

import (
	"strings"
	"testing"
)

func TestUnlinkWithABodyIsRefused(t *testing.T) {
	_, err := Parse(strings.NewReader("@@ gone.txt - unlink\nstale\n"))
	if err == nil {
		t.Fatal("unlink with a body parsed")
	}
	if !strings.Contains(err.Error(), "unlink takes no body") {
		t.Fatalf("unlink with body: %v", err)
	}
}
