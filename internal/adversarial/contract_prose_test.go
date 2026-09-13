package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestContractDoesNotGrepReadmeForTutorialPhrases is the pin for ADR-053.
// A tidy of a README heading must not look like a product break. Live
// (non-comment) lines of scripts/contract.sh must not assert those phrases
// against README.md. Comments that name a retired fossil are allowed.
func TestContractDoesNotGrepReadmeForTutorialPhrases(t *testing.T) {
	src := readContract(t)
	fossils := []string{
		"### Use it from an MCP host",
		"^A range is .*`A,\\+N`",
		"and `A,+N` is the line `A` plus the `N` lines AFTER it",
		"A read CLAMPS a relative end at the last",
		"clamps.*exactly as `12-9999`",
	}
	for i, line := range strings.Split(src, "\n") {
		if isCommentOrEmpty(line) {
			continue
		}
		for _, f := range fossils {
			if strings.Contains(line, f) {
				t.Errorf("scripts/contract.sh:%d still greps README for %q (ADR-053: drop prose greps)", i+1, f)
			}
		}
	}
	if liveReadmeMcpServers(src) {
		t.Error("scripts/contract.sh still parses mcpServers out of README.md (ADR-053: drop the block-as-README-fixture)")
	}
	if !strings.Contains(src, "# 75. ADR-037") {
		t.Error("§75 Shared() through the binary was deleted; ADR-053 keeps it")
	}
}

// TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure pins the other half:
// AckRule still requires its sentence, and it does not freeze the MCP host
// heading. A README that drops ### Use it from an MCP host must not fail
// go test except that AckRule check, if the sentence is still in the file.
func TestAReadmeWithoutTheMcpHostHeadingIsNotAGoTestFailure(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "internal/mcp/ack_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	ack := string(b)
	if !strings.Contains(ack, "func TestEverySurfaceCarriesTheOneRule") {
		t.Fatal("AckRule test is gone; ADR-053 keeps TestEverySurfaceCarriesTheOneRule")
	}
	if strings.Contains(ack, "### Use it from an MCP host") {
		t.Error("AckRule froze the MCP host heading; a README tidy of that heading must not fail go test")
	}
}

func readContract(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts/contract.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func isCommentOrEmpty(line string) bool {
	trim := strings.TrimSpace(line)
	return trim == "" || strings.HasPrefix(trim, "#")
}

func liveReadmeMcpServers(src string) bool {
	sawCat := false
	for _, line := range strings.Split(src, "\n") {
		if isCommentOrEmpty(line) {
			continue
		}
		if strings.Contains(line, "cat README.md") {
			sawCat = true
		}
		if sawCat && strings.Contains(line, "mcpServers") {
			return true
		}
	}
	return false
}
