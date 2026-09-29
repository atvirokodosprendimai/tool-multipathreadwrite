package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stepsBlock is a steps block holding cmd under the name fmt, beside a step
// holding no token, as .quality-harness.json would carry it.
func stepsBlock(t *testing.T, cmd string) Config {
	t.Helper()
	b, err := json.Marshal(map[string]string{"fmt": cmd, "vet": "go vet ./..."})
	if err != nil {
		t.Fatal(err)
	}
	return Config{Check: "true", Steps: b}
}

// ADR-094 T1. A declared step whose command holds {files} or {packages} is
// refused when a step is asked for, naming the step and the token: mrw expands
// them only in scoped_check, and a step given the literal text checked a file
// named {files} and passed. A near-miss is not mrw's grammar and runs as
// written.
func TestADeclaredStepHoldingAPlaceholderIsRefused(t *testing.T) {
	for _, tok := range []string{"{files}", "{packages}"} {
		_, err := stepsBlock(t, "gofmt -l "+tok).StepCommands()
		if err == nil {
			t.Errorf("a step holding %s was accepted", tok)
			continue
		}
		if msg := err.Error(); !strings.Contains(msg, "step \"fmt\"") || !strings.Contains(msg, tok) || !strings.Contains(msg, "scoped_check") {
			t.Errorf("the refusal of %s does not name the step, the token and why: %q", tok, msg)
		}
	}
	for _, near := range []string{"{file}", "{ files }", "{FILES}", "{package}"} {
		cmds, err := stepsBlock(t, "gofmt -l "+near).StepCommands()
		if err != nil || cmds["fmt"] != "gofmt -l "+near {
			t.Errorf("the near-miss %s was refused or changed: %v %q", near, err, cmds["fmt"])
		}
	}
}

// ADR-094 T1. The tokens Placeholder refuses are exactly the tokens command()
// substitutes: every token it reports is expanded in a scoped_check on a mapped
// Go path, both of command()'s tokens are reported, and a near-miss is not.
func TestPlaceholderNamesEveryTokenTheScopedCheckExpands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reported := map[string]bool{}
	for _, c := range []string{"{packages}", "{files}", "{file}", "{ files }", "{FILES}", "{package}", "{}"} {
		tok := Placeholder("run " + c + " now")
		if tok == "" {
			continue
		}
		if tok != c {
			t.Errorf("Placeholder(%q) = %q, want the token the command holds", c, tok)
		}
		reported[tok] = true
		got, scoped := command(root, Config{Check: "FULL", ScopedCheck: "run " + tok}, []string{"a.go"})
		if !scoped || strings.Contains(got, tok) {
			t.Errorf("Placeholder reports %s, which command() does not substitute: %q scoped=%v", tok, got, scoped)
		}
	}
	if !reported["{packages}"] || !reported["{files}"] || len(reported) != 2 {
		t.Errorf("Placeholder reported %v, want exactly {packages} and {files}", reported)
	}
	if got := Placeholder("go vet ./..."); got != "" {
		t.Errorf("Placeholder of a command with no token = %q", got)
	}
}
