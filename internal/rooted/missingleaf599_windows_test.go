package rooted

import "testing"

func TestAMissingLeafUnderAJunctionedAncestorResolvesInsideTheRoot(t *testing.T) {
	testAMissingLeafUnderALinkedAncestorResolvesInsideTheRoot(t, mklinkJ)
}
