package rooted

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A fake filesystem of names, so the walk that Windows needs can be driven on
// a platform that has no junctions. Each entry is what Lstat reports for that
// path; a link's Readlink answers its target, a placeholder's answers Go's
// "another type of reparse point" (ENOENT), and an unreadable one answers
// permission denied.
type fakeEntry struct {
	kind   string // "dir", "link", "placeholder", "unreadable", "denied", "brokensymlink", "invalid"
	target string
}

type fakeInfo struct {
	name string
	mode os.FileMode
}

func (f fakeInfo) Name() string       { return f.name }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() os.FileMode  { return f.mode }
func (f fakeInfo) ModTime() time.Time { return time.Time{} }
func (f fakeInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeInfo) Sys() any           { return nil }

// fakeLinks builds a linkFS over entries keyed by slash paths, which it
// converts to the running platform's spelling so the same table runs on
// Windows, where this file is compiled too.
func fakeLinks(entries map[string]fakeEntry) linkFS {
	byPath := map[string]fakeEntry{}
	for p, e := range entries {
		byPath[filepath.Clean(filepath.FromSlash(p))] = e
	}
	return linkFS{
		lstat: func(p string) (os.FileInfo, error) {
			e, ok := byPath[filepath.Clean(p)]
			if !ok {
				return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrNotExist}
			}
			switch e.kind {
			case "dir":
				return fakeInfo{filepath.Base(p), os.ModeDir}, nil
			case "invalid":
				return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrInvalid}
			case "denied":
				return nil, &fs.PathError{Op: "lstat", Path: p, Err: fs.ErrPermission}
			case "brokensymlink":
				return fakeInfo{filepath.Base(p), os.ModeSymlink}, nil
			}
			mode := os.ModeIrregular
			return fakeInfo{filepath.Base(p), mode}, nil
		},
		readlink: func(p string) (string, error) {
			switch e := byPath[filepath.Clean(p)]; e.kind {
			case "link":
				return filepath.FromSlash(e.target), nil
			case "placeholder", "brokensymlink":
				return "", &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrNotExist}
			default:
				return "", &fs.PathError{Op: "readlink", Path: p, Err: fs.ErrPermission}
			}
		},
	}
}

func slash(p string) string { return filepath.ToSlash(p) }

// The escape the round found: a junction inside the root pointing outside it.
// The walk must hand back where the path really leads, including through a
// link that sits inside the junction's target, and a relative target resolves
// against the directory that holds the link.
func TestTheLinkWalkReplacesALinkWithItsTarget(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":              {kind: "dir"},
		"/r/j":            {kind: "link", target: "/out"},
		"/out":            {kind: "dir"},
		"/out/k":          {kind: "link", target: "/far"},
		"/far":            {kind: "dir"},
		"/far/secret.txt": {kind: "dir"},
		"/r/rel":          {kind: "link", target: "sub"},
		"/r/sub":          {kind: "dir"},
	})
	for in, want := range map[string]string{
		"/r/j/k/secret.txt": "/far/secret.txt",
		"/r/j":              "/out",
		"/r/rel/x.txt":      "/r/sub/x.txt",
	} {
		got, err := throughLinks(filepath.FromSlash(in), links)
		if err != nil {
			t.Errorf("throughLinks(%s) refused: %v", in, err)
			continue
		}
		if slash(got) != want {
			t.Errorf("throughLinks(%s) = %s, want %s — a link left in place is a boundary check against the wrong path", in, slash(got), want)
		}
	}
}

// A OneDrive placeholder is a reparse point Go marks irregular, and it
// redirects nothing: Readlink says so with ENOENT. Refusing it would refuse
// every file in a synced folder.
func TestTheLinkWalkLeavesAPlaceholderAlone(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":       {kind: "dir"},
		"/r/od":    {kind: "placeholder"},
		"/r/od/f":  {kind: "placeholder"},
		"/r/plain": {kind: "dir"},
	})
	got, err := throughLinks(filepath.FromSlash("/r/od/f"), links)
	if err != nil {
		t.Fatalf("a placeholder was refused as a link: %v", err)
	}
	if slash(got) != "/r/od/f" {
		t.Fatalf("a placeholder was rewritten to %s", slash(got))
	}
}

// Any other failure to read a link is not knowledge of where it leads, so the
// path is refused rather than judged by its spelling — which is the escape.
func TestTheLinkWalkRefusesALinkItCannotRead(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":   {kind: "dir"},
		"/r/u": {kind: "unreadable"},
	})
	_, err := throughLinks(filepath.FromSlash("/r/u/secret.txt"), links)
	if err == nil {
		t.Fatal("a link that could not be read was followed lexically")
	}
	if !strings.Contains(slash(err.Error()), "/r/u") || !strings.Contains(err.Error(), "cannot follow") {
		t.Fatalf("the refusal must name the link and say why: %v", err)
	}
}

// A create names a file that is not there yet. The links above it are
// followed, and the missing tail is kept as written.
func TestTheLinkWalkStopsAtAMissingComponent(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":   {kind: "dir"},
		"/r/j": {kind: "link", target: "/out"},
		"/out": {kind: "dir"},
	})
	got, err := throughLinks(filepath.FromSlash("/r/j/new/deep.txt"), links)
	if err != nil {
		t.Fatal(err)
	}
	if slash(got) != "/out/new/deep.txt" {
		t.Fatalf("got %s, want /out/new/deep.txt — the junction above a missing tail must still be followed", slash(got))
	}
}

// Two links pointing at each other must refuse, not spin.
func TestTheLinkWalkBoundsALoop(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":   {kind: "dir"},
		"/r/a": {kind: "link", target: "/r/b"},
		"/r/b": {kind: "link", target: "/r/a"},
	})
	if _, err := throughLinks(filepath.FromSlash("/r/a/x"), links); err == nil {
		t.Fatal("a link loop was accepted")
	}
}

// Win32 drops a trailing dot or space from a name and reads a colon as an NTFS
// stream, so each of these names does not reach the file written. The check
// names the component and what is left once those characters are dropped.
func TestWin32AliasNamesTheComponentWindowsWouldRemap(t *testing.T) {
	for in, want := range map[string][2]string{
		"b.txt.":        {"b.txt.", "b.txt"},
		"sp.txt ":       {"sp.txt ", "sp.txt"},
		"b.txt::$DATA":  {"b.txt::$DATA", "b.txt"},
		"dir /a.txt":    {"dir ", "dir"},
		`sub\dir.\a.go`: {"dir.", "dir"},
		"...":           {"...", ""},
	} {
		comp, reads := win32Alias(in)
		if comp != want[0] || reads != want[1] {
			t.Errorf("win32Alias(%q) = (%q, %q), want (%q, %q)", in, comp, reads, want[0], want[1])
		}
	}
}

// "." and ".." end in a dot and are not aliases; a cleaned "./a.txt" must not
// be refused on Windows.
func TestWin32AliasLeavesDotAndDotDotAlone(t *testing.T) {
	for _, in := range []string{".", "..", "a/./b", "../a", "a.b/c.d", `x\..\y`, ""} {
		if comp, _ := win32Alias(in); comp != "" {
			t.Errorf("win32Alias(%q) named %q, which Windows reads as written", in, comp)
		}
	}
}

// A symlink whose Readlink answers ENOENT is a link mrw cannot follow, not a
// placeholder: only an irregular entry may say "not a link" and be kept
// (review of #228, A2).
func TestTheLinkWalkRefusesASymlinkItCannotRead(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":   {kind: "dir"},
		"/r/s": {kind: "brokensymlink"},
	})
	if _, err := throughLinks(filepath.FromSlash("/r/s/x"), links); err == nil {
		t.Fatal("a symlink that could not be read was kept as a plain component")
	}
}

// Only a component that is not there ends the walk. One that cannot be
// examined is refused, not judged by its spelling (review of #228, A1).
func TestTheLinkWalkRefusesAComponentItCannotExamine(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":   {kind: "dir"},
		"/r/d": {kind: "denied"},
	})
	if _, err := throughLinks(filepath.FromSlash("/r/d/x"), links); err == nil {
		t.Fatal("a component that could not be examined ended the walk as if it were missing")
	}
}

// A name that cannot exist (Windows answers "invalid name" for a "*" a shell
// left unexpanded) ends the walk like a missing one: it cannot be a link, and
// refusing it replaced read's own report of the missing file and its glob hint
// (found by the Windows shard on 1be35ef).
func TestTheLinkWalkEndsAtANameThatCannotExist(t *testing.T) {
	links := fakeLinks(map[string]fakeEntry{
		"/r":      {kind: "dir"},
		"/r/*.go": {kind: "invalid"},
	})
	got, err := throughLinks(filepath.FromSlash("/r/*.go"), links)
	if err != nil {
		t.Fatalf("a name that cannot exist was refused: %v", err)
	}
	if slash(got) != "/r/*.go" {
		t.Fatalf("got %s, want the name kept as written", slash(got))
	}
}
