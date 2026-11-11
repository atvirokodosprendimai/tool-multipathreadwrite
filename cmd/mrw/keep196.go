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
// (ADR-196 T4). Its re-probe of v1.86.1 found that past a \\.\ or \\?\ prefix a
// path names an object, not only a file: `\\.\GLOBALROOT\Device\ConDrv\Console`
// waited on the console too, so there only a drive, a share, a volume or a pipe
// is accepted (ADR-196 T5).
func windowsKeeps(what, path string) error { return keepsOn(runtime.GOOS, what, path) }

// windowsPipe reports whether path names a named pipe on Windows. A pipe is no
// state file, and the state check opens the path it judges, which spent a
// one-shot pipe's only instance before the read (ADR-196 T5).
func windowsPipe(path string) bool { return pipeOn(runtime.GOOS, path) }

// keepsOn is windowsKeeps for a named GOOS, so the rule is tested on every platform.
func keepsOn(goos, what, path string) error {
	_, err := judgeOn(goos, what, path)
	return err
}

// pipeOn is windowsPipe for a named GOOS.
func pipeOn(goos, path string) bool {
	pipe, err := judgeOn(goos, "", path)
	return pipe && err == nil
}

// judgeOn reports whether path, on goos, names a named pipe, and refuses it when
// Windows would open something other than the file it reads as.
func judgeOn(goos, what, path string) (pipe bool, err error) {
	if goos != "windows" {
		return false, nil
	}
	// A \\?\ path skips Win32's normalization, so a name ending in a dot or a
	// space, or a device name, is kept as written there and opens that name (the
	// Codex review of #628). The filesystem, not Win32, reads a colon as a
	// stream, so that is judged under \\?\ too. Only the volume is skipped: a
	// drive, a UNC server and share (`\\srv\share`, `\\?\UNC\srv\share`), and a
	// device path's prefix (`\\.\`, `//./`, either separator) with the volume
	// after it (the Codex review of #630).
	// `\??\` is the NT form of `\\?\`, which Go's os.Open passes on as it is (the
	// Codex review of #633), so it is judged as a verbatim prefix.
	verbatim := strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\??\`)
	sep := func(i int) bool { return i < len(path) && (path[i] == '/' || path[i] == '\\') }
	comps := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	drive := true
	switch {
	case sep(0) && len(path) > 3 && sep(3) && ((sep(1) && (path[2] == '.' || path[2] == '?')) || path[1:3] == "??"),
		sep(0) && sep(1) && len(path) == 3 && (path[2] == '.' || path[2] == '?'):
		// Past the prefix Windows resolves `.` and `..` before it reads the
		// namespace (not under \\?\), so `\\.\pipe\..\NUL` is `\\.\NUL` and
		// `\\.\.\pipe\x` is a pipe (the Codex reviews of #630). Then the first
		// component says what kind of object the path names; a device, GLOBALROOT
		// or any other object is refused by kind, since no list of names covers
		// them (the re-probe of v1.86.1).
		if comps = comps[1:]; !verbatim {
			comps = resolveDots(comps)
		}
		first := ""
		if len(comps) > 0 {
			first = comps[0]
		}
		// A drive, a volume, a UNC share or the pipe namespace with nothing
		// after it names that object itself: `\\.\C:` opened the raw volume
		// (the field test of v1.86.3), so a file must follow it.
		noFile := func(n int) error {
			return fmt.Errorf("%s %s: it names %s and no file in it, so it would open that volume, share or namespace itself; name a file in it", what, quoted(path), quoted(strings.Join(comps[:min(n, len(comps))], `\`)))
		}
		switch lower := strings.ToLower(first); {
		case len(first) == 2 && first[1] == ':' && 'a' <= lower[0] && lower[0] <= 'z',
			strings.HasPrefix(lower, "volume{") && strings.HasSuffix(lower, "}"):
			if len(comps) < 2 {
				return false, noFile(1)
			}
			comps, drive = comps[1:], false
		case lower == "unc":
			if len(comps) < 4 {
				return false, noFile(3)
			}
			if strings.EqualFold(comps[2], "pipe") {
				return true, nil
			}
			comps, drive = comps[3:], false
		case lower == "pipe":
			if len(comps) < 2 {
				return false, noFile(1)
			}
			return true, nil
		default:
			return false, fmt.Errorf("%s %s: %s names a Windows object that is not a file on a drive or a share (a device, GLOBALROOT, …), so it would open that object; name the file by its drive or share path", what, quoted(path), quoted(first))
		}
	case sep(0) && sep(1):
		// A remote pipe (`\\srv\pipe\…`) is a pipe. Below the share, `.` and
		// `..` are resolved as Windows resolves them, and a `..` cannot climb
		// above the share, so `\\srv\pipe\..\plan` is still a pipe (the Codex
		// review of #633). A server and share with nothing below them name no
		// file (the field test of v1.86.3).
		rest := resolveDots(comps[min(2, len(comps)):])
		if len(comps) < 2 || len(rest) == 0 {
			return false, fmt.Errorf("%s %s: it names %s and no file in it, so it would open that share or namespace itself; name a file in it", what, quoted(path), quoted(strings.Join(comps[:min(2, len(comps))], `\`)))
		}
		if strings.EqualFold(comps[1], "pipe") {
			return true, nil
		}
		comps, drive = rest, false
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
			return false, fmt.Errorf("%s %s: Windows reads the ':' in %s as an NTFS stream, not as part of a name, so it would open a stream that dir and git do not show; name the file as it is", what, quoted(path), quoted(c))
		}
		if verbatim {
			continue
		}
		if strings.TrimRight(c, ". ") != c {
			return false, fmt.Errorf("%s %s: Windows does not keep %s as written — it drops a trailing dot or space — so it would open another file; name the file as it is", what, quoted(path), quoted(c))
		}
		base, _, _ := strings.Cut(c, ".")
		if rooted.DeviceBase(strings.TrimRight(base, " ")) {
			return false, fmt.Errorf("%s %s: %s is a Windows device name, so it would open the device, not a file; name a file (standard input is -)", what, quoted(path), quoted(c))
		}
	}
	return false, nil
}

// resolveDots drops `.` and resolves `..` against the components before it, as
// Windows does past a \\.\ prefix; a `..` with nothing before it is dropped.
func resolveDots(comps []string) []string {
	var out []string
	for _, c := range comps {
		switch c {
		case ".":
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, c)
		}
	}
	return out
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
