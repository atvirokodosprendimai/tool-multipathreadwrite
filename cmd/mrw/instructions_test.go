package main

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
	"github.com/urfave/cli/v3"
)

func TestInstructionsCommandPrintsCLI(t *testing.T) {
	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer = &buf
	if err := cmd.Run(context.Background(), []string{"mrw", "instructions"}); err != nil {
		t.Fatalf("instructions: %v", err)
	}
	if got, want := buf.String(), guide.CLI(); got != want {
		t.Errorf("stdout is not guide.CLI()\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestEveryReadFlagIsTaughtByInstructions reads the flags from readCmd()
// itself, so a flag added or renamed there without guide.CLI() following
// turns this red (ADR-063). The other direction — the instructions naming a
// flag read no longer has — is contract §115's half. A flag must appear as
// --name followed by a character that cannot continue a flag name, so --x
// does not pass on --x-y.
func TestEveryReadFlagIsTaughtByInstructions(t *testing.T) {
	doc := guide.CLI()
	for _, f := range readCmd().Flags {
		name := f.Names()[0]
		re := regexp.MustCompile(`--` + regexp.QuoteMeta(name) + `([^A-Za-z0-9-]|$)`)
		if !re.MatchString(doc) {
			t.Errorf("guide.CLI() never names --%s, a flag mrw read accepts", name)
		}
	}
}

// TestHelpSummariesNameTheReadSide holds the one-line summaries a caller sees
// before any other help: mrw finds as well as reads and writes, read serves
// patterns and --grep hits, and --ast-grep's placeholder is its pattern.
func TestHelpSummariesNameTheReadSide(t *testing.T) {
	if u := rootCommand().Usage; !strings.Contains(u, "find") {
		t.Errorf("mrw's NAME line does not say it finds: %q", u)
	}
	u := readCmd().Usage
	for _, must := range []string{"pattern matches", "--grep"} {
		if !strings.Contains(u, must) {
			t.Errorf("read's Usage does not name %q: %q", must, u)
		}
	}
	// urfave/cli takes the FIRST backticked word of a flag's Usage as the
	// placeholder --help prints, so this is what read --help shows; §115
	// checks the rendered help on the built binary.
	found := false
	for _, f := range readCmd().Flags {
		sf, ok := f.(*cli.StringFlag)
		if !ok || sf.Name != "ast-grep" {
			continue
		}
		found = true
		m := regexp.MustCompile("`([^`]*)`").FindStringSubmatch(sf.Usage)
		if m == nil || m[1] != "PATTERN" {
			t.Errorf("--ast-grep's placeholder is not PATTERN: %q", sf.Usage)
		}
	}
	if !found {
		t.Error("read has no --ast-grep string flag; the placeholder check reached nothing")
	}
}
