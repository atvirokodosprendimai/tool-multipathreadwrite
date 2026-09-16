package check

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ── ADR-061 under stress ───────────────────────────────────────────────────
//
// Oracle is the Decision, not command():
//   {files}-only + non-empty paths + packages() empty → scoped, {files} quoted
//   {packages}-only or mixed + empty map → Check
//   empty path list → Check (mrw check --full)
//   packages() maps → scoped, both placeholders substituted (ADR-003)
//
// Layer 1 quotes with a byte walk + an inert charset string; shellArg uses
// ContainsFunc + a switch. Layer 2 draws template × path-class; mapped vs
// unmapped is a fixture class the generator guarantees, not a call to
// packages() inside the oracle.
//
// Pools (v2), measured 2026-09-16:
//   templates: rg -n '\{packages\}|\{files\}' internal/check/check.go
//     → both placeholders, plus absent / neither (the Decision's other arms).
//   path class: empty | allGo (root-level .go) | unmapped (.rs / .toml / mix).
//   Left out: `{files_extra}` substring; nested packages() trees (would force
//   the oracle to reimplement packages()); Zeus's real JSON (no scoped_check —
//   a fixture that required that file is a tripwire against adding one).

const inertCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-/=:,+@"

func allInert(v string) bool {
	for i := 0; i < len(v); i++ {
		if !strings.ContainsRune(inertCharset, rune(v[i])) {
			return false
		}
	}
	return true
}

// refShellArg is the quoting oracle: single quotes, with a close-escape-reopen
// for a literal quote, walking bytes. Empty is quoted (it is not inert).
func refShellArg(v string) string {
	if v != "" && allInert(v) {
		return v
	}
	var b bytes.Buffer
	b.WriteByte('\'')
	for i := 0; i < len(v); i++ {
		if v[i] == '\'' {
			b.WriteString(`'\''`)
		} else {
			b.WriteByte(v[i])
		}
	}
	b.WriteByte('\'')
	return b.String()
}

func refShellArgs(vals []string) string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = refShellArg(v)
	}
	return strings.Join(out, " ")
}

func TestRefShellArgAgreesOnTheSpecExamples(t *testing.T) {
	for _, v := range []string{
		"a.go",
		"two words.rs",
		"pkg; true #",
		"d$(touch owned)",
		"quo'te",
		"",
		"./internal/check",
		".",
	} {
		if got, want := shellArg(v), refShellArg(v); got != want {
			t.Errorf("shellArg(%q)=%q want %q", v, got, want)
		}
	}
}

func FuzzShellArg(f *testing.F) {
	for _, v := range []string{"a.go", "two words.rs", "pkg; true #", "d$(x)", "quo'te", "", ".", "./x"} {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, v string) {
		if got, want := shellArg(v), refShellArg(v); got != want {
			t.Fatalf("shellArg(%q)=%q want %q", v, got, want)
		}
	})
}

type scopeTmpl int

const (
	tmplNone scopeTmpl = iota
	tmplFiles
	tmplPackages
	tmplMixed
	tmplBare
)

func (s scopeTmpl) String() string {
	return [...]string{"none", "files", "packages", "mixed", "bare"}[s]
}

func (s scopeTmpl) text() string {
	switch s {
	case tmplFiles:
		return "echo SCOPED {files}"
	case tmplPackages:
		return "echo SCOPED {packages}"
	case tmplMixed:
		return "echo SCOPED {packages} {files}"
	case tmplBare:
		return "echo SCOPED"
	default:
		return ""
	}
}

type pathClass int

const (
	classEmpty pathClass = iota
	classAllGo
	classUnmapped
)

func (c pathClass) String() string {
	return [...]string{"empty", "allGo", "unmapped"}[c]
}

const checkFull = "echo FULL"

// filesScopeOracle is the Decision as a function of the generator's classes.
// mapped means "every path is a root-level .go file we created"; those place
// `.`, so the oracle never calls packages().
func filesScopeOracle(tmpl scopeTmpl, class pathClass, paths []string) (string, bool) {
	scoped := tmpl.text()
	if scoped == "" {
		return checkFull, false
	}
	if class == classAllGo {
		r := strings.NewReplacer("{packages}", refShellArg("."), "{files}", refShellArgs(paths))
		return r.Replace(scoped), true
	}
	if class != classEmpty && tmpl == tmplFiles {
		r := strings.NewReplacer("{files}", refShellArgs(paths))
		return r.Replace(scoped), true
	}
	return checkFull, false
}

func TestRandomisedCommandMatchesTheFilesScopeOracle(t *testing.T) {
	seed := int64(61)
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	r := rand.New(rand.NewSource(seed))
	const iterations = 400

	unmappedPool := [][]string{
		{"a.rs"},
		{"Cargo.toml"},
		{"a.rs", "Cargo.toml"},
		{"a.go", "a.rs"},
		{"two words.rs"},
	}

	for i := 0; i < iterations; i++ {
		tmpl := scopeTmpl(r.Intn(5))
		class := pathClass(r.Intn(3))
		root := t.TempDir()
		var paths []string
		switch class {
		case classEmpty:
			paths = nil
		case classAllGo:
			paths = []string{"a.go"}
			if r.Intn(2) == 0 {
				paths = []string{"a.go", "b.go"}
			}
			for _, p := range paths {
				if err := os.WriteFile(filepath.Join(root, p), []byte("package probe\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		case classUnmapped:
			paths = append([]string(nil), unmappedPool[r.Intn(len(unmappedPool))]...)
			for _, p := range paths {
				full := filepath.Join(root, p)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				body := "fn main() {}\n"
				if strings.HasSuffix(p, ".go") {
					body = "package probe\n"
				}
				if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}

		if class == classAllGo && len(packages(root, paths)) == 0 {
			t.Fatalf("seed=%d iter=%d: allGo fixture did not map: %v", seed, i, paths)
		}
		if class == classUnmapped && len(packages(root, paths)) > 0 {
			t.Fatalf("seed=%d iter=%d: unmapped fixture mapped: %v → %v", seed, i, paths, packages(root, paths))
		}
		if class == classEmpty && len(packages(root, paths)) > 0 {
			t.Fatalf("seed=%d iter=%d: empty paths mapped: %v", seed, i, packages(root, paths))
		}

		cfg := Config{Check: checkFull, ScopedCheck: tmpl.text()}
		got, gotScoped := command(root, cfg, paths)
		want, wantScoped := filesScopeOracle(tmpl, class, paths)
		if got != want || gotScoped != wantScoped {
			t.Fatalf("seed=%d iter=%d tmpl=%s class=%s paths=%q\ngot  %q scoped=%v\nwant %q scoped=%v",
				seed, i, tmpl, class, paths, got, gotScoped, want, wantScoped)
		}
	}
}
