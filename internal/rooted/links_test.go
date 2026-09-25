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
	kind   string // "dir", "link", "placeholder", "unreadable"
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
			mode := os.ModeDir
			if e.kind != "dir" {
				mode = os.ModeIrregular
			}
			return fakeInfo{filepath.Base(p), mode}, nil
		},
		readlink: func(p string) (string, error) {
			switch e := byPath[filepath.Clean(p)]; e.kind {
			case "link":
				return filepath.FromSlash(e.target), nil
			case "placeholder":
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

// Win32 strips a trailing dot or space from every component and reads a colon
// as an NTFS stream, so each of these names reaches a different file than the
// one written. The check names the component and what Windows reads.
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
