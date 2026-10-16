package main

import (
	"bytes"
	"context"
	"regexp"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

func instructionsOut(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer = &buf
	err := cmd.Run(context.Background(), append([]string{"mrw", "instructions"}, args...))
	return buf.String(), err
}

// ADR-141. `instructions --core` prints the core; the plain command is still the
// whole contract; an argument is still exit 2.
func TestInstructionsCoreFlagPrintsTheCore(t *testing.T) {
	got, err := instructionsOut(t, "--core")
	if err != nil || got != guide.Core() {
		t.Errorf("--core: err %v, stdout is not guide.Core():\n%s", err, got)
	}
	if got, err := instructionsOut(t); err != nil || got != guide.CLI() {
		t.Errorf("plain instructions: err %v, stdout is not guide.CLI()", err)
	}
	if _, err := instructionsOut(t, "--core", "extra"); err == nil || exitCode(err) != exitUsage {
		t.Errorf("--core with an argument: %v, want exit %d", err, exitUsage)
	}
}

// The core cannot name a flag the binary lacks: every --flag in it belongs to
// read, write or the root command.
func TestTheCoreNamesOnlyFlagsTheBinaryHas(t *testing.T) {
	have := map[string]bool{"help": true}
	for _, f := range readCmd().Flags {
		have[f.Names()[0]] = true
	}
	for _, f := range writeCmd().Flags {
		have[f.Names()[0]] = true
	}
	for _, f := range rootCommand().Flags {
		have[f.Names()[0]] = true
	}
	for _, m := range regexp.MustCompile(`--([a-z][a-z-]*)`).FindAllStringSubmatch(guide.Core(), -1) {
		if !have[m[1]] {
			t.Errorf("the core names --%s, a flag read, write and the root command all lack", m[1])
		}
	}
}

// ADR-161 Decision 5. The core's first rule names the one-command wide edit, so
// an agent handed only the core meets it, and the core stays under 300 words.
func TestTheCoreNamesTheSubWrite(t *testing.T) {
	core := guide.Core()
	rule1 := regexp.MustCompile(`(?m)^1\. .*$`).FindString(core)
	if !regexp.MustCompile(`mrw write --grep P --sub R`).MatchString(rule1) {
		t.Errorf("core rule 1 does not name mrw write --grep P --sub R: %q", rule1)
	}
	if n := len(regexp.MustCompile(`\S+`).FindAllString(core, -1)); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}
}
