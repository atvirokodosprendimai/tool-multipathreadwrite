package refusal

import (
	"errors"
	"fmt"
	"testing"
)

// A kind survives wrapping and its text is the message unchanged; an error
// with no kind reports "", which is not any named kind.
func TestKindOfFindsAWrappedRefusal(t *testing.T) {
	err := New(NotRead, "%s has not been read", "a.go")
	if err.Error() != "a.go has not been read" {
		t.Errorf("text = %q, want the message unchanged", err.Error())
	}
	if got := KindOf(fmt.Errorf("line 3: %w", err)); got != NotRead {
		t.Errorf("KindOf(wrapped) = %q, want %q", got, NotRead)
	}
	if got := KindOf(errors.New("plain")); got != "" {
		t.Errorf("KindOf(plain) = %q, want empty", got)
	}
}
