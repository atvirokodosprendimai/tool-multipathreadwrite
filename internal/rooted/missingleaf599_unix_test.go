//go:build !windows

package rooted

import (
	"os"
	"testing"
)

func TestAMissingLeafUnderASymlinkedAncestorResolvesInsideTheRoot(t *testing.T) {
	testAMissingLeafUnderALinkedAncestorResolvesInsideTheRoot(t, func(t *testing.T, link, target string) {
		t.Helper()
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
	})
}
