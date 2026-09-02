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

// A timeout_seconds that does not survive being used is a value nobody typed
// on purpose — the same rule as the whitespace-only check command, one field
// over.
//
// time.Duration is an int64 NANOsecond count, so a large seconds value
// overflows when multiplied. 9999999999 produced a deadline ~2.3 million hours
// in the PAST: exec refused before the process existed and the run was
// reported as one that could not start. It was not even monotonic —
// 99999999999 wrapped back to positive and the check ran normally, so a bigger
// number meant a working timeout again. Both are pinned here, because a fix
// that only handled the first value would leave the wrap.
func TestAnOverlargeTimeoutIsClampedNotOverflowed(t *testing.T) {
	root := t.TempDir()
	for _, secs := range []int{9999999999, 99999999999, maxTimeoutSeconds + 1} {
		cfg := Config{Check: "exit 0", TimeoutSeconds: secs, declared: true}
		res, err := Run(context.Background(), root, cfg, nil)
		if err != nil {
			t.Fatalf("timeout_seconds=%d: %v", secs, err)
		}
		// The check is trivial and instant, so the ONLY way it fails here is a
		// deadline that was already in the past when the run started.
		if !res.OK() {
			t.Errorf("timeout_seconds=%d: %+v", secs, res)
		}
		if res.Skipped != "" {
			t.Errorf("timeout_seconds=%d reported %q", secs, res.Skipped)
		}
	}
}

// TestSubstitutedPathsAreOneShellArgumentEach pins the quoting of the values
// mrw puts into the scoped command. The command is a shell line, so an
// unquoted path is syntax: a space splits it, and `;` ends the command.
func TestSubstitutedPathsAreOneShellArgumentEach(t *testing.T) {
	root := t.TempDir()
	// Built at runtime, never committed: a checked-in path holding `;` or `#`
	// is miserable for every tool that walks the tree.
	for _, dir := range []string{"pkg; true #", "two words", "d$(touch owned)", "quo'te"} {
		full := filepath.Join(root, dir)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Skipf("filesystem will not hold %q: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(full, "x.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "plain.go"), []byte("package probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Check: "FULL", ScopedCheck: "go test {packages}"}
	for name, tc := range map[string]struct {
		paths []string
		want  string
	}{
		"a semicolon ends the command": {[]string{"pkg; true #/x.go"}, `go test './pkg; true #'`},
		"a space splits the argument":  {[]string{"two words/x.go"}, `go test './two words'`},
		"a subshell runs":              {[]string{"d$(touch owned)/x.go"}, `go test './d$(touch owned)'`},
		"a quote closes the quoting":   {[]string{"quo'te/x.go"}, `go test './quo'\''te'`},

		// The negative control. An ordinary scope must stay byte-identical, or
		// this test would pass while quoting everything indiscriminately.
		"an ordinary path is untouched": {[]string{"plain.go"}, "go test ."},
	} {
		if got, _ := command(root, cfg, tc.paths); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}

	// {files} is substituted from the caller's paths rather than derived ones,
	// and reaches the same shell.
	filesCfg := Config{ScopedCheck: "lint {files}"}
	got, _ := command(root, filesCfg, []string{"two words/x.go", "plain.go"})
	if want := `lint 'two words/x.go' plain.go`; got != want {
		t.Errorf("{files}: got %q, want %q", got, want)
	}
}

// TestAShellInjectedScopeStillFails runs the real command. The unit rows above
// pin the string; this pins the verdict, because the defect was not a wrong
// string — it was PASS at exit 0 with the package never tested.
func TestAShellInjectedScopeStillFails(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "pkg; true #")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skipf("filesystem will not hold the directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module probe\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A package that does not compile: `go test` on it exits non-zero, so a
	// scope that really covers it cannot report success.
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n\nfunc F() { undefinedSymbol() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Check: "false", ScopedCheck: "go test {packages}", declared: true}
	res, err := Run(t.Context(), root, cfg, []string{"pkg; true #/x.go"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Errorf("a scope naming a broken package reported OK: command %q, exit %d", res.Command, res.ExitCode)
	}
}

// A scope outside the root is refused, not fallen back on. The fallback is
// what makes the other unplaceable paths safe — a typo, a prose directory —
// because the whole-project run still covers them. Outside the root it covers
// nothing the caller named, so the verdict it produces is about a different
// tree entirely.
//
// The root's own check is `exit 0` here, so a fallback would report a PASS.
// Asserting on the error alone would not catch that: the row has to show that
// NOTHING ran, which is what an empty Command and Ran=false say.
func TestAScopeOutsideTheRootIsRefusedNotFallenBackTo(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(outer, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "o.go"), []byte("package outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	cfg := Config{Check: "exit 0", ScopedCheck: "exit 0", declared: true}
	for _, scope := range []string{"../outside", "../outside/...", outside, "link"} {
		res, err := Run(context.Background(), root, cfg, []string{scope})
		if err == nil {
			t.Errorf("%s: accepted, result %+v", scope, res)
			continue
		}
		if res.Ran || res.Command != "" || res.OK() {
			t.Errorf("%s: refused but a result was filled in: %+v", scope, res)
		}
		if !strings.Contains(err.Error(), "--root") {
			t.Errorf("%s: error does not say where to point --root: %v", scope, err)
		}
	}
}

// The discriminating half: the old fallback answered about the ROOT, so its
// verdict tracked the root's own tests and never the argument. A failing root
// made `check ../outside` report a failure — right code, wrong subject — and a
// row asserting only "not zero" would have passed throughout.
func TestARefusedScopeDoesNotInheritTheRootsVerdict(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Check: "exit 1", declared: true}
	res, err := Run(context.Background(), root, cfg, []string{"../elsewhere"})
	if err == nil {
		t.Fatalf("accepted: %+v", res)
	}
	if res.ExitCode != 0 || res.Ran {
		t.Errorf("the root's failing check reached a refused scope: %+v", res)
	}
}

// An in-root path that cannot be placed must still fall back: that is the
// behaviour the refusal above is carved out of, and breaking it would trade
// one silent omission for a wall of refusals on ordinary typos.
func TestAnUnplaceableInRootScopeStillFallsBack(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Check: "exit 0", ScopedCheck: "exit 7", declared: true}
	res, err := Run(context.Background(), root, cfg, []string{"nosuchdir"})
	if err != nil {
		t.Fatalf("an in-root typo was refused: %v", err)
	}
	if !res.OK() || res.Command != "exit 0" {
		t.Errorf("did not fall back to the whole-project command: %+v", res)
	}
}

// A directory that exists and cannot be READ is refused, not treated as one
// that holds no package.
//
// holdsPackage walks the directory and discards the walk's error, so a
// permission failure came back as `found = false` — indistinguishable from a
// directory of prose. The scope then fell back to the whole-project command,
// which is sound for a typo (the full run covers the root) and vacuous here:
// the caller named a directory, mrw could not open it, and answered PASS.
//
// Found by sweeping wing_craft's "a zero that meant I could not look" against
// this repository — a read error reported as an absence.
func TestADirectoryThatCannotBeReadIsRefusedNotTreatedAsEmpty(t *testing.T) {
	root := t.TempDir()
	blind := filepath.Join(root, "blind")
	if err := os.MkdirAll(blind, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blind, "b.go"), []byte("package blind\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blind, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blind, 0o755) })
	// chmod is a no-op for uid 0, so PROVE the bits bite before asserting on
	// them. Under root this test would otherwise pass having exercised the
	// happy path — the shape scripts/contract.sh guards the same way.
	if _, err := os.ReadDir(blind); err == nil {
		t.Skip("permission bits not enforced here (running as root?)")
	}

	// The check itself exits 0, so a row asserting only "not OK" would not
	// separate a refusal from a passing run. The refusal is an ERROR from Run,
	// before anything executes.
	cfg := Config{Check: "exit 0", ScopedCheck: "exit 0", declared: true}
	res, err := Run(context.Background(), root, cfg, []string{"blind"})
	if err == nil {
		t.Fatalf("an unreadable directory was accepted: %+v", res)
	}
	if res.Ran || res.Command != "" {
		t.Errorf("something ran for a scope mrw could not read: %+v", res)
	}
	if !strings.Contains(err.Error(), "cannot be read") {
		t.Errorf("the reason was not reported: %v", err)
	}
}

// The two controls. A readable directory still scopes, and a path that is NOT
// THERE still falls back — that is the decided behaviour for a typo, and
// trading it for a refusal would break every ordinary mistyped scope.
func TestReadableAndMissingScopesAreUnchanged(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "good"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "good", "g.go"), []byte("package good\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Check: "exit 0", ScopedCheck: "exit 0", declared: true}

	if res, err := Run(context.Background(), root, cfg, []string{"good"}); err != nil || !res.OK() {
		t.Errorf("a readable directory was refused: %v %+v", err, res)
	}
	if res, err := Run(context.Background(), root, cfg, []string{"nosuchdir"}); err != nil || !res.OK() {
		t.Errorf("a missing path no longer falls back: %v %+v", err, res)
	}
}
