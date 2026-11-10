package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// windowsKeeps refuses, on Windows, a path the caller handed mrw on the command
// line that Win32 would open as another thing than the file it reads as: one
// with a component ending in a dot or a space, which Win32 drops (ADR-071); one
// holding a colon, which names an NTFS stream; or one naming a device such as
// CON or NUL (ADR-081). A plan file and a --files-from list are opened with
// os.Open and never pass rooted.Resolve, which refuses those spellings for every
// path inside the root. While urfave/cli v3.11.0 trimmed a positional, ADR-069's
// guard refused `write 'p.mrw '`; with v3.14.0 handing it over as typed, `write
// 'p.mrw '` applied p.mrw on Windows (the Codex review of #628, ADR-196). The
// Windows field test of v1.86.0 found the other two: `write CON` waited on the
// console, and `write p.mrw:alt` applied a plan that `type p.mrw` does not show
// (ADR-196 T4).
func windowsKeeps(what, path string) error { return keepsOn(runtime.GOOS, what, path) }

// keepsOn is windowsKeeps for a named GOOS, so the rule is tested on every platform.
func keepsOn(goos, what, path string) error {
	if goos != "windows" {
		return nil
	}
	// A \\?\ path skips Win32's normalization, so a name ending in a dot or a
	// space, or a device name, is kept as written there and opens that name (the
	// Codex review of #628). The filesystem, not Win32, reads a colon as a
	// stream, so that is judged under \\?\ too. Only the volume is skipped: a
	// drive, a UNC server and share (`\\srv\share`, `\\?\UNC\srv\share`), and a
	// device path's prefix (`\\.\`, `//./`, either separator); a named pipe
	// (`\\.\pipe\…`) is no file at all and is not judged (the Codex review of
	// #630).
	verbatim := strings.HasPrefix(path, `\\?\`)
	sep := func(i int) bool { return i < len(path) && (path[i] == '/' || path[i] == '\\') }
	comps := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	drive := true
	switch {
	case sep(0) && sep(1) && len(path) > 2 && (path[2] == '.' || path[2] == '?') && (len(path) == 3 || sep(3)):
		comps = comps[1:]
		if len(comps) > 0 && strings.EqualFold(comps[0], "pipe") {
			return nil
		}
		if len(comps) > 0 && strings.EqualFold(comps[0], "UNC") {
			comps, drive = comps[min(3, len(comps)):], false
		}
	case sep(0) && sep(1):
		comps, drive = comps[min(2, len(comps)):], false
	}
	if drive && len(comps) > 0 {
		if d := dropDrive(comps[0]); d == "" {
			comps = comps[1:]
		} else {
			comps[0] = d
		}
	}
	for _, c := range comps {
		if c == "." || c == ".." {
			continue
		}
		if strings.Contains(c, ":") {
			return fmt.Errorf("%s %s: Windows reads the ':' in %s as an NTFS stream, not as part of a name, so it would open a stream that dir and git do not show; name the file as it is", what, quoted(path), quoted(c))
		}
		if verbatim {
			continue
		}
		if strings.TrimRight(c, ". ") != c {
			return fmt.Errorf("%s %s: Windows does not keep %s as written — it drops a trailing dot or space — so it would open another file; name the file as it is", what, quoted(path), quoted(c))
		}
		base, _, _ := strings.Cut(c, ".")
		if rooted.DeviceBase(strings.TrimRight(base, " ")) {
			return fmt.Errorf("%s %s: %s is a Windows device name, so it would open the device, not a file; name a file (standard input is -)", what, quoted(path), quoted(c))
		}
	}
	return nil
}

// dropDrive removes a drive prefix (`C:`), so the drive-relative `C:.\p.mrw`
// and `C:..\list` are not judged as a component `C:.` ending in a dot (the
// Codex re-review of #628). It reads the Windows form on every platform, as
// keepsOn does; a UNC path has no drive and keeps its components.
func dropDrive(p string) string {
	if len(p) >= 2 && p[1] == ':' && ('a' <= p[0]|0x20 && p[0]|0x20 <= 'z') {
		return p[2:]
	}
	return p
}
