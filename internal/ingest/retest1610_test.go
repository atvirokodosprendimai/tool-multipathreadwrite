package ingest

import (
	"strings"
	"testing"
)

// The v1.61.0 Windows retest (four sessions). A mix of CRLF and LF lines holds no
// bare CR, so the refusal says the endings are mixed; a lone CR still says so.
func TestMixedCRLFAndLFEndingsAreNotCalledABareCR(t *testing.T) {
	_, err := CompileCreate("m.txt", []byte("a\r\nb\nc\r\n"))
	if err == nil || !strings.Contains(err.Error(), "mixed") || strings.Contains(err.Error(), "bare CR") {
		t.Errorf("CRLF and LF lines: %v, want a refusal naming mixed endings and no bare CR", err)
	}
	if _, err := CompileCreate("m.txt", []byte("a\nb\r")); err == nil || !strings.Contains(err.Error(), "bare CR") {
		t.Errorf("a lone CR: %v, want it named", err)
	}
}

// The same retest: cmd and Git Bash pipe the same bytes, so the note says the
// dropped CRLF is most likely PowerShell's, not that PowerShell sent it.
func TestTheDroppedCRLFNoteDoesNotAssertWhoSentIt(t *testing.T) {
	_, notes := CreateContent([]byte("a\nb\n\r\n"))
	if len(notes) != 1 || !strings.Contains(notes[0], "most likely") || strings.Contains(notes[0], "so that one was dropped") {
		t.Errorf("notes = %q, want one note saying the CRLF is most likely PowerShell's", notes)
	}
}
