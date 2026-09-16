package adversarial

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// ── ADR-061 arm 3: drive the built binary ──────────────────────────────────
//
// command() choosing ScopedCheck vs Check is only observable from the printed
// command / JSON `command` field. This file builds mrw once and runs
// `mrw check` the way a caller does.
//
// Oracle is ADR-061 Decision + ADR-003 mapped arm, not check.go:
//   {files}-only + named non-Go path → SCOPED <path>, not FULL
//   {packages}-only or mixed on that path → FULL
//   --full (nil paths) → FULL even with {files}-only
//   root-level .go + any non-empty scoped_check → scoped substitution
//   no scoped_check → FULL (Zeus today; not their JSON file)
//
// Pools (v2), measured 2026-09-16:
//   check flags: rg 'Name:.*(json|full)' cmd/mrw/main.go checkCmd → json, full.
//   templates: none / files / packages / mixed / bare (same as the package test).
//   paths: a.rs, Cargo.toml, a.go, a.go+a.rs (the mix that empties packages()).
//   Left out: real cargo/go test (echo only); MCP write (ADR-044: no check);
//   Zeus's checkout JSON (no scoped_check — requiring that file is a tripwire).
//
// A seed is printed on failure and honoured from MRW_SEED.

func TestRandomisedCheckMatrixMatchesTheFilesScopeOracle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the declared checks are sh -c scripts")
	}
	seed := int64(61)
	if s := os.Getenv("MRW_SEED"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		seed = v
	}
	r := rand.New(rand.NewSource(seed))
	const iterations = 120

	type tmpl int
	const (
		tmplNone tmpl = iota
		tmplFiles
		tmplPackages
		tmplMixed
		tmplBare
	)
	scopedOf := func(s tmpl) string {
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
	pathSets := [][]string{
		{"a.rs"},
		{"Cargo.toml"},
		{"a.go"},
		{"a.go", "a.rs"},
	}

	for i := 0; i < iterations; i++ {
		kind := tmpl(r.Intn(5))
		paths := append([]string(nil), pathSets[r.Intn(len(pathSets))]...)
		full := r.Intn(4) == 0
		asJSON := r.Intn(4) == 0

		files := map[string]string{}
		for _, p := range paths {
			if strings.HasSuffix(p, ".go") {
				files[p] = "package probe\n"
			} else {
				files[p] = "fn main() {}\n"
			}
		}
		harness := `{"check":"echo FULL"`
		if s := scopedOf(kind); s != "" {
			harness += `,"scoped_check":"` + s + `"`
		}
		harness += `}`
		files[".quality-harness.json"] = harness
		root := tree(t, files)
		state := t.TempDir()

		args := []string{"check"}
		if asJSON {
			args = append(args, "--json")
		}
		if full {
			args = append(args, "--full")
		} else {
			args = append(args, paths...)
		}
		out, code := run(t, state, root, args...)
		label := fmt.Sprintf("seed=%d iter=%d tmpl=%d full=%v json=%v paths=%v\n%s", seed, i, kind, full, asJSON, paths, out)

		wantCmd, wantScoped := "echo FULL", false
		allGo := true
		for _, p := range paths {
			if !strings.HasSuffix(p, ".go") {
				allGo = false
				break
			}
		}
		switch {
		case full || kind == tmplNone:
			// Check. --full is an empty path list.
		case allGo && kind != tmplNone:
			wantScoped = true
			switch kind {
			case tmplFiles:
				wantCmd = "echo SCOPED " + strings.Join(paths, " ")
			case tmplPackages:
				wantCmd = "echo SCOPED ."
			case tmplMixed:
				wantCmd = "echo SCOPED . " + strings.Join(paths, " ")
			case tmplBare:
				wantCmd = "echo SCOPED"
			}
		case kind == tmplFiles:
			wantScoped, wantCmd = true, "echo SCOPED "+strings.Join(paths, " ")
		}

		if code != 0 {
			t.Fatalf("%s\nexit %d, oracle says 0 (echo)", label, code)
		}
		gotCmd := checkCommand(t, label, out, asJSON)
		if gotCmd != wantCmd {
			t.Fatalf("%s\ncommand %q, oracle says %q scoped=%v", label, gotCmd, wantCmd, wantScoped)
		}
	}
}

func checkCommand(t *testing.T, label, out string, asJSON bool) string {
	t.Helper()
	if asJSON {
		var res struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("%s\njson: %v", label, err)
		}
		return res.Command
	}
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "check (declared): "); ok {
			return rest
		}
	}
	t.Fatalf("%s\nno check (declared): line", label)
	return ""
}

// A {files}-only scoped_check on a default .rs write is the Zeus-shaped win:
// the check that runs is the scoped one, not workspace FULL. The example is
// the template, not Zeus's checkout file (which has no scoped_check).
func TestAFilesOnlyWriteOnRustRunsTheScopedCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the declared check is an sh -c script")
	}
	root := tree(t, map[string]string{
		"a.rs":                  "fn main() {}\n",
		".quality-harness.json": `{"check":"echo FULL","scoped_check":"echo SCOPED {files}"}`,
	})
	state := t.TempDir()
	if _, code := run(t, state, root, "read", "a.rs"); code != 0 {
		t.Fatalf("read: exit %d", code)
	}
	plan := filepath.Join(t.TempDir(), "p.mrw")
	if err := os.WriteFile(plan, []byte("@@ a.rs 1 replace\nfn main() { let _ = 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := run(t, state, root, "write", plan)
	if code != 0 {
		t.Fatalf("write exit %d\n%s", code, out)
	}
	if !strings.Contains(out, "check (declared): echo SCOPED a.rs") {
		t.Fatalf("want scoped check, got:\n%s", out)
	}
	if strings.Contains(out, "echo FULL") {
		t.Fatalf("fell back to FULL:\n%s", out)
	}
}
