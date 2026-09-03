package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

// lookup builds an environment stand-in. ResolveRoot takes one rather than
// calling os.LookupEnv so a test can describe an environment without setting
// one, which is what keeps these tests from leaking into their siblings.
func lookup(vals map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := vals[k]
		return v, ok
	}
}

func TestAnExplicitRootWinsOverTheEnvironment(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	got, src := ResolveRoot(other, lookup(map[string]string{projectDirEnv: dir}))
	if got != other {
		t.Errorf("root = %q, want the explicit %q — a user overriding the host must win", got, other)
	}
	if src != SourceFlag {
		t.Errorf("source = %q, want %q", src, SourceFlag)
	}
}

func TestTheProjectDirEnvironmentIsUsedWhenNoRootIsGiven(t *testing.T) {
	// The bug this task fixes: with no --root the server used to take the
	// working directory, which a host does not guarantee.
	dir := t.TempDir()
	got, src := ResolveRoot("", lookup(map[string]string{projectDirEnv: dir}))
	if got != dir {
		t.Errorf("root = %q, want %q from %s", got, dir, projectDirEnv)
	}
	if src != SourceProjectDir {
		t.Errorf("source = %q, want %q", src, SourceProjectDir)
	}

	// And "" is how the caller says "no --root was given". The caller decides
	// that with IsSet, never by comparing against the flag default — see the
	// next test for why.
	got, src = ResolveRoot("", lookup(map[string]string{projectDirEnv: dir}))
	if got != dir || src != SourceProjectDir {
		t.Errorf(`ResolveRoot("") = %q/%q, want %q/%q`, got, src, dir, SourceProjectDir)
	}
}

func TestANonDirectoryProjectDirFallsBackToTheWorkingDirectory(t *testing.T) {
	// A hostile or stale environment must not make the server unusable. Each
	// of these is a value a host could plausibly set.
	file := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, env := range map[string]map[string]string{
		"unset":         {},
		"empty":         {projectDirEnv: ""},
		"whitespace":    {projectDirEnv: "   "},
		"missing dir":   {projectDirEnv: filepath.Join(t.TempDir(), "nope")},
		"a plain file":  {projectDirEnv: file},
		"relative path": {projectDirEnv: "some/relative/dir"},
	} {
		t.Run(name, func(t *testing.T) {
			got, src := ResolveRoot("", lookup(env))
			if got != "." {
				t.Errorf("root = %q, want %q — an unusable environment falls back, it does not fail", got, ".")
			}
			if src != SourceWorkingDir {
				t.Errorf("source = %q, want %q", src, SourceWorkingDir)
			}
		})
	}
}

func TestResolveRootReportsWhichSourceItUsed(t *testing.T) {
	// The announcement must not claim an environment origin for a fallback, so
	// the source is returned rather than inferred by the caller. A bool would
	// make the third case unrepresentable.
	dir := t.TempDir()
	seen := map[Source]bool{}
	for _, c := range []struct {
		explicit string
		env      map[string]string
		want     Source
	}{
		{dir, nil, SourceFlag},
		{"", map[string]string{projectDirEnv: dir}, SourceProjectDir},
		{"", nil, SourceWorkingDir},
	} {
		_, src := ResolveRoot(c.explicit, lookup(c.env))
		if src != c.want {
			t.Errorf("ResolveRoot(%q, %v) source = %q, want %q", c.explicit, c.env, src, c.want)
		}
		seen[src] = true
	}
	if len(seen) != 3 {
		t.Errorf("only %d distinct sources are reachable; all three must be, or the announcement cannot be truthful", len(seen))
	}
	// And each must render as something a reader can act on.
	for _, s := range []Source{SourceFlag, SourceProjectDir, SourceWorkingDir} {
		if s.String() == "" {
			t.Errorf("source %v has no description", s)
		}
	}
}

func TestAnExplicitDotIsAChoiceNotADefault(t *testing.T) {
	// `--root .` typed on purpose and `--root` omitted both arrive as "." from
	// the flag parser. An earlier version of ResolveRoot compared against the
	// default and treated them as the same, so a deliberate `--root .` beside a
	// CLAUDE_PROJECT_DIR served the environment's tree — silently, and the
	// opposite of what was asked. The caller now distinguishes them with IsSet
	// and passes "" for absent, so "." reaching here IS a choice.
	dir := t.TempDir()
	got, src := ResolveRoot(".", lookup(map[string]string{projectDirEnv: dir}))
	if got != "." || src != SourceFlag {
		t.Errorf(`ResolveRoot(".") = %q/%q, want "."/%q — an explicit dot must win`, got, src, SourceFlag)
	}
}

func TestAProjectDirEndingInSpaceIsNotRewritten(t *testing.T) {
	// Trimming decides whether anything was SAID; it must not change the path.
	// A directory whose name really ends in a space is legal, and trimming it
	// resolves to a different directory than the host named.
	base := t.TempDir()
	odd := filepath.Join(base, "trailing ")
	if err := os.Mkdir(odd, 0o755); err != nil {
		t.Skipf("this filesystem will not hold a trailing space: %v", err)
	}
	got, src := ResolveRoot("", lookup(map[string]string{projectDirEnv: odd}))
	if got != odd || src != SourceProjectDir {
		t.Errorf("root = %q, want the untrimmed %q", got, odd)
	}
}
