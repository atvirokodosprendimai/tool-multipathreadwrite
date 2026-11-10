package main

import (
	"fmt"
	"runtime"
	"strings"
)

// windowsKeeps refuses, on Windows, a path the caller handed mrw on the command
// line that Win32 would open as another name: one with a component ending in a
// dot or a space, which Win32 drops (ADR-071). A plan file and a --files-from
// list are opened with os.Open and never pass rooted.Resolve, which refuses that
// spelling for every path inside the root. While urfave/cli v3.11.0 trimmed a
// positional, ADR-069's guard refused `write 'p.mrw '`; with v3.14.0 handing it
// over as typed, `write 'p.mrw '` applied p.mrw on Windows (the Codex review of
// #628, ADR-196). A separate `--files-from 'list '` opened list on every
// release; it is refused too.
func windowsKeeps(what, path string) error { return keepsOn(runtime.GOOS, what, path) }

// keepsOn is windowsKeeps for a named GOOS, so the rule is tested on every platform.
func keepsOn(goos, what, path string) error {
	if goos != "windows" {
		return nil
	}
	// A \\?\ path skips Win32's normalization, so a name ending in a dot or a
	// space is kept as written there and opens that name (the Codex review of
	// #628). A \\.\ device path is normalized, and is judged.
	if strings.HasPrefix(path, `\\?\`) {
		return nil
	}
	if c := unkeptComponent(dropDrive(path)); c != "" {
		return fmt.Errorf("%s %s: Windows does not keep %s as written — it drops a trailing dot or space — so it would open another file; name the file as it is", what, quoted(path), quoted(c))
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
