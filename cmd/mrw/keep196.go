package main

import (
	"fmt"
	"runtime"
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
	if c := unkeptComponent(path); c != "" {
		return fmt.Errorf("%s %s: Windows does not keep %s as written — it drops a trailing dot or space — so it would open another file; name the file as it is", what, quoted(path), quoted(c))
	}
	return nil
}
