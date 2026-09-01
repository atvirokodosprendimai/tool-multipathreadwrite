package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPrefersTheDeclaredCheck(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"make verify","scoped_check":"make verify PKG={packages}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Declared() || cfg.Check != "make verify" {
		t.Errorf("cfg = %+v", cfg)
	}
}

// An inferred command must be marked as inferred: if it is red on an untouched
// tree, the finding is about the machine, not about the edit.
func TestLoadInfersGoCheckButSaysSo(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Declared() {
		t.Error("an inferred check reported itself as declared")
	}
	if cfg.Check != "go test ./..." {
		t.Errorf("Check = %q", cfg.Check)
	}
}

func TestLoadOnABareDirectoryHasNoCheck(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Check != "" || cfg.ScopedCheck != "" {
		t.Errorf("invented a check: %+v", cfg)
	}
}

func TestScopeDerivation(t *testing.T) {
	// A root with real packages in it, because a path that is not a Go file is
	// a scope only if it resolves to at least one package — which is what tells
	// `mrw check internal/apply` from `mrw check internal/aply`, and both of
	// those from `mrw check docs`.
	root := t.TempDir()
	for path, body := range map[string]string{
		"internal/apply/apply.go":      "package apply\n",
		"internal/apply/a.go":          "package apply\n",
		"internal/apply/b.go":          "package apply\n",
		"internal/apply/sub/sub.go":    "package sub\n",
		"internal/apply/testdata/x.go": "package testdata\n",
		"cmd/mrw/main.go":              "package main\n",
		"main.go":                      "package probe\n",
		"docs/guide.md":                "# prose\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Config{Check: "FULL", ScopedCheck: "go test {packages}"}
	for name, tc := range map[string]struct {
		paths []string
		want  string
	}{
		"one package":       {[]string{"internal/apply/apply.go"}, "go test ./internal/apply"},
		"two packages":      {[]string{"internal/apply/a.go", "cmd/mrw/main.go"}, "go test ./cmd/mrw ./internal/apply"},
		"deduplicated":      {[]string{"internal/apply/a.go", "internal/apply/b.go"}, "go test ./internal/apply"},
		"root package":      {[]string{"main.go"}, "go test ."},
		"non-Go falls back": {[]string{"internal/apply/a.go", "README.md"}, "FULL"},
		"no paths":          {nil, "FULL"},

		// The bug the first round was one row short of catching: the README
		// spells the scoped form with a DIRECTORY, and a directory has no .go
		// extension.
		"a directory":              {[]string{"internal/apply"}, "go test ./internal/apply/..."},
		"the ./dir form":           {[]string{"./internal/apply"}, "go test ./internal/apply/..."},
		"a directory and a file":   {[]string{"internal/apply", "cmd/mrw/main.go"}, "go test ./cmd/mrw ./internal/apply/..."},
		"a directory twice over":   {[]string{"internal/apply", "internal/apply/a.go"}, "go test ./internal/apply/..."},
		"a path that is not there": {[]string{"internal/aply"}, "FULL"},
		"a directory and a non-Go": {[]string{"internal/apply", "README.md"}, "FULL"},

		// And what THAT round was one row short of: a named directory must
		// cover its SUBTREE. Go's `.` is the root package alone, so `mrw check
		// .` reported PASS with a failing package one directory down.
		"the whole tree":              {[]string{"."}, "go test ./..."},
		"the ./... form we print":     {[]string{"./..."}, "go test ./..."},
		"the ./dir/... form we print": {[]string{"./internal/apply/..."}, "go test ./internal/apply/..."},
		"a nested directory as well":  {[]string{"internal/apply", "internal/apply/sub"}, "go test ./internal/apply/..."},

		// A directory holding no package go will build scopes to nothing, and
		// `go test` on it exits 1 for a reason that is not about the code. That
		// is the typo case wearing a second hat, so it gets the same answer.
		"a directory of prose":      {[]string{"docs"}, "FULL"},
		"prose beside a package":    {[]string{"internal/apply", "docs"}, "FULL"},
		"testdata holds no package": {[]string{"internal/apply/testdata"}, "FULL"},

		// read and write both refuse a path that leaves the root. A scope that
		// names one is the same escape wearing a third hat.
		"a directory outside the root": {[]string{".."}, "FULL"},
		"a file outside the root":      {[]string{"../elsewhere/x.go"}, "FULL"},

		// A .go path is refused when it is not there, for the same reason a
		// directory is. The extension is not evidence the file exists, and at a
		// module root a mistyped one places `.` — a check that runs, passes,
		// and never looks at what the caller named.
		"a .go file that is not there":  {[]string{"internal/apply/nope.go"}, "FULL"},
		"a .go file in a missing dir":   {[]string{"nosuchdir/nope.go"}, "FULL"},
		"a mistyped .go file at a root": {[]string{"chek.go"}, "FULL"},
		"a real file beside a phantom":  {[]string{"internal/apply/apply.go", "internal/apply/nope.go"}, "FULL"},
	} {
		if got, _ := command(root, cfg, tc.paths); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestFilesPlaceholder(t *testing.T) {
	// The files have to be real: {files} is still reached through packages,
	// which now refuses a .go path that is not there, so a fixture naming
	// phantoms would exercise the fallback rather than the placeholder.
	root := t.TempDir()
	for _, name := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("package probe\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Config{ScopedCheck: "lint {files}"}
	got, _ := command(root, cfg, []string{"a.go", "b.go"})
	if got != "lint a.go b.go" {
		t.Errorf("got %q", got)
	}
}

func TestRunReportsTheRealExitCode(t *testing.T) {
	root := t.TempDir()
	res, err := Run(context.Background(), root, Config{Check: "echo hi; exit 7"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", res.ExitCode)
	}
	if res.OK() {
		t.Error("OK() true on a failing check")
	}
	if len(res.Tail) == 0 || res.Tail[0] != "hi" {
		t.Errorf("Tail = %q", res.Tail)
	}
	os.Remove(res.OutputFile)
}

func TestRunPasses(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(), Config{Check: "true"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Errorf("res = %+v", res)
	}
	os.Remove(res.OutputFile)
}

// A check that did not run is not a pass. This is the distinction that lets a
// caller tell "verified green" from "no evidence either way".
func TestNoCheckIsNotAPass(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(), Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ran || res.OK() {
		t.Errorf("res = %+v", res)
	}
	if res.Skipped == "" {
		t.Error("skipping was not explained")
	}
}

// Trimming must announce itself; a silent tail reads as the whole output.
func TestTailAnnouncesWhatItLeftOut(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(),
		Config{Check: "seq 1 100", TailLines: 5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tail) != 5 || res.Truncated != 95 {
		t.Errorf("Tail=%d Truncated=%d, want 5 and 95", len(res.Tail), res.Truncated)
	}
	if res.Tail[4] != "100" {
		t.Errorf("tail is not the END of the output: %q", res.Tail)
	}
	// The full output must still be recoverable from the file.
	b, err := os.ReadFile(res.OutputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "1\n") || !strings.HasSuffix(string(b), "100\n") {
		t.Error("the file did not keep the untrimmed output")
	}
	os.Remove(res.OutputFile)
}

func TestTimeoutIsReportedAsAFailureNotAPass(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(),
		Config{Check: "sleep 5", TimeoutSeconds: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Error("a timed-out check reported OK")
	}
	if !strings.Contains(res.Skipped, "timed out") {
		t.Errorf("Skipped = %q", res.Skipped)
	}
	os.Remove(res.OutputFile)
}

// A check command that is only whitespace is one nobody typed on purpose — a
// stray space, a truncated edit, a template that expanded to nothing. Before
// this was trimmed it read as DECLARED, ran as an empty shell command, exited 0
// and reported `check PASS`. That is a check which did not run reporting a
// pass, which ADR-003 rule 2 exists to refuse, and it would have held for as
// long as the typo did — every gate downstream believing the suite was green.
func TestAWhitespaceOnlyCheckIsNotDeclared(t *testing.T) {
	for _, v := range []string{`"   "`, `"\t"`, `"\n"`, `" \t \n "`} {
		t.Run(v, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
				[]byte(`{"check":`+v+`}`), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Declared() {
				t.Errorf("check=%s reads as declared: %+v", v, cfg)
			}
			// The decisive assertion: whatever it falls back to, it must not
			// be able to report success without running anything. With no
			// go.mod there is nothing to infer, so the run is Skipped — and
			// Skipped is not OK().
			res, err := Run(context.Background(), root, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			if res.OK() {
				t.Errorf("check=%s reported a pass: %+v", v, res)
			}
			if res.Ran {
				t.Errorf("check=%s ran something: %+v", v, res)
			}
		})
	}
}

// The trim must not eat a real command, including one whose declared form has
// incidental padding.
func TestAPaddedCheckIsStillTheDeclaredCheck(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"  make verify  "}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Declared() || cfg.Check != "make verify" {
		t.Errorf("cfg = %+v", cfg)
	}
}

// The trim covers ScopedCheck as well, and nothing else asserted it: removing
// only ScopedCheck from the trim assignment left every other test in this file
// and every contract row green (PR #13 review, Codex). A whitespace-only
// scoped_check would otherwise read as declared, suppressing the fallback.
func TestAWhitespaceOnlyScopedCheckIsNotDeclared(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"scoped_check":"   "}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Declared() {
		t.Errorf("a whitespace-only scoped_check reads as declared: %+v", cfg)
	}
	if cfg.ScopedCheck != "" {
		t.Errorf("ScopedCheck = %q, want it trimmed to empty", cfg.ScopedCheck)
	}
	res, err := Run(context.Background(), root, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() || res.Ran {
		t.Errorf("it reported a pass: %+v", res)
	}
}

// A padded scoped_check is a real one and must survive the trim.
func TestAPaddedScopedCheckIsStillDeclared(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"),
		[]byte(`{"check":"make v","scoped_check":"  make v PKG={packages}  "}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Declared() || cfg.ScopedCheck != "make v PKG={packages}" {
		t.Errorf("cfg = %+v", cfg)
	}
}

// ADR-003 rule 2, in the direction nothing asserted: a process that never
// STARTED did not run. Ran was set before exec, so an unresolvable command
// reported exit 3 — "a check ran and did not pass" — about a process that
// never existed. Exit 3 promises a verdict; this has none, and a caller
// branching on 3 would go looking for test output that was never produced.
func TestACheckThatCannotStartDidNotRun(t *testing.T) {
	root := t.TempDir()
	// An already-cancelled context: exec refuses before the process exists.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := Run(ctx, root, Config{Check: "true"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ran {
		t.Errorf("a process that never started is recorded as having run: %+v", res)
	}
	if res.OK() {
		t.Errorf("it reported a pass: %+v", res)
	}
	if res.Skipped == "" {
		t.Errorf("nothing says why it could not run: %+v", res)
	}
}
