package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func fileSHA(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// lineFeedRoot is a root below a .git whose own name holds a line feed: every
// ledger key there is prefix plus path, so none can be saved (#702).
func lineFeedRoot(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a line feed is not a legal Windows file name")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	top := t.TempDir()
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(top, "sub\ndir")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func toolCall(t *testing.T, root, tool string, args map[string]any) (callToolResult, *rpcError) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"name": tool, "arguments": args})
	if err != nil {
		t.Fatal(err)
	}
	return callTool(root, raw, false, nil)
}

// A quiet read serves no line, so a licence the ledger cannot keep is refused;
// the pair, a root whose keys can be saved, records it.
func TestAQuietReadTheLedgerCannotHoldIsRefused(t *testing.T) {
	root := lineFeedRoot(t)
	_, rpcErr := toolCall(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt"}, "quiet": true})
	if rpcErr == nil || !strings.Contains(rpcErr.Message, "line feed") {
		t.Fatalf("a quiet read of an unrecordable path returned %v, want a refusal naming the line feed", rpcErr)
	}
	// zero licensed lines: nothing to keep, nothing refused.
	if _, rpcErr := toolCall(t, root, "mrw_read", map[string]any{"specs": []any{"f.txt:99"}, "quiet": true}); rpcErr != nil {
		t.Errorf("a quiet read of a range past the end was refused: %v", rpcErr.Message)
	}

	ok, _ := checkout(t, "f.txt", "hello\n")
	if _, rpcErr := toolCall(t, ok, "mrw_read", map[string]any{"specs": []any{"f.txt"}, "quiet": true}); rpcErr != nil {
		t.Fatalf("the pair: a quiet read in an ordinary root was refused: %v", rpcErr.Message)
	}
	if l, err := seen.Load(ok); err != nil || len(l) != 1 {
		t.Errorf("the pair: the ledger holds %d record(s), %v; want 1", len(l), err)
	}
}

// An ack that names a path the ledger cannot hold is consumed as on main, and
// the plan a sub read returns is its text alone, with nothing put in front.
func TestASubPlanWithAnAckForAnUnrecordablePathIsNotPrefixed(t *testing.T) {
	root := lineFeedRoot(t)
	sum := fileSHA(t, filepath.Join(root, "f.txt"))
	if err := hold(root, "f.txt", sum, map[string][2]int{"ck-1": {1, 1}}); err != nil {
		t.Fatal(err)
	}
	res, rpcErr := toolCall(t, root, "mrw_read", map[string]any{"grep": "hello", "sub": "bye", "ack": []any{"ck-1"}})
	if rpcErr != nil {
		t.Fatalf("the call was refused: %v", rpcErr.Message)
	}
	text := res.Content[0].Text
	if !strings.HasPrefix(text, "@@ ") {
		t.Errorf("the plan is not the text's first line: %q", text)
	}
	if store, err := loadPending(root); err != nil || len(store) != 0 {
		t.Errorf("the ack was not consumed: %v, %v", store, err)
	}
}
