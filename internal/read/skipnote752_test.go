package read

import (
	"strings"
	"testing"
)

// #752. The ignore rules a walk applies come from .gitignore files AND
// .git/info/exclude, so the note does not credit them to .gitignore alone.
func TestTheSkipNoteDoesNotCreditEveryIgnoreRuleToGitignore(t *testing.T) {
	note := SkipNote(WalkSkipped{Ignored: 2, IgnoredDirs: 1}, "--no-ignore")
	for _, want := range []string{"2 file(s) the ignore rules skip", "1 director(ies) the ignore rules skip, not entered"} {
		if !strings.Contains(note, want) {
			t.Errorf("note %q lacks %q", note, want)
		}
	}
	if strings.Contains(note, ".gitignore") {
		t.Errorf("note %q names .gitignore, though an exclude file's rules count too", note)
	}
}

// #752. A reserved device name (aux.txt, nul: ADR-081) is a name Windows keeps and
// opens as a device; it is counted apart from a name Windows will not keep (a
// trailing dot or space), each with its own words, and neither is walked by a flag.
func TestTheSkipNoteCountsDeviceNamesApartFromNamesWindowsWillNotKeep(t *testing.T) {
	// Unkeepable counts BOTH kinds, as it always did (ADR-111): 3 names, 2 of them device names.
	note := SkipNote(WalkSkipped{DeviceNames: 2, Unkeepable: 3}, "--no-ignore")
	for _, want := range []string{"2 file(s) named for a Windows device", "1 file(s) with a name Windows will not keep", "name one to be told why"} {
		if !strings.Contains(note, want) {
			t.Errorf("note %q lacks %q", note, want)
		}
	}
	if strings.Contains(note, "2 file(s) with a name Windows will not keep") || strings.Contains(note, "--no-ignore walks them") {
		t.Errorf("note %q lumps the two, or promises a flag walks them", note)
	}
	if only := SkipNote(WalkSkipped{DeviceNames: 1, Unkeepable: 1}, "--no-ignore"); strings.Contains(only, "will not keep") || !strings.Contains(only, "name one to be told why") {
		t.Errorf("a device name alone gave %q", only)
	}
	// A name of the other kind alone is the ADR-135 sentence, as before.
	if only := SkipNote(WalkSkipped{Unkeepable: 1}, "--no-ignore"); !strings.Contains(only, "1 file(s) with a name Windows will not keep") || strings.Contains(only, "Windows device") {
		t.Errorf("an alias name alone gave %q", only)
	}
}
