package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// ADR-092 T3. Every surface a caller learns mrw from names both step flags and
// what --then-sh grants: mrw instructions, AGENTS.md, README.md and write --help.
func TestEverySurfaceTeachesThen(t *testing.T) {
	agents, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	help, _ := runIn(t, t.TempDir(), "write", "--help")
	then := regexp.MustCompile(`--then([^A-Za-z0-9-]|$)`)
	caveat := guide.ThenShCaveat()
	if !strings.Contains(caveat, "--then-sh") || !strings.Contains(caveat, "arbitrary shell") {
		t.Fatalf("the caveat does not say what --then-sh grants: %q", caveat)
	}
	for name, doc := range map[string]string{
		"mrw instructions": guide.CLI(),
		"AGENTS.md":        string(agents),
		"README.md":        string(readme),
		"write --help":     help,
	} {
		if !then.MatchString(doc) || !strings.Contains(doc, "--then-sh") {
			t.Errorf("%s does not name --then and --then-sh", name)
		}
		if !strings.Contains(doc, caveat) {
			t.Errorf("%s does not carry the --then-sh caveat", name)
		}
	}
}
