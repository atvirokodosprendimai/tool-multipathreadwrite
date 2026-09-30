package mcp

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-107 T2. Promotion hashed each acknowledged file with os.ReadFile, so a
// file grown after an MCP read was held in memory whole. currentSHA streams
// it into the hash: the digest is the ledger's, and a 50 MB file allocates
// under 1 MB.
func TestAcknowledgementHashesInBoundedMemory(t *testing.T) {
	root := t.TempDir()
	b := bytes.Repeat([]byte("0123456789abcdef"), (50<<20)/16)
	if err := os.WriteFile(filepath.Join(root, "big.bin"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	sha, err := currentSHA(root, "big.bin")
	runtime.ReadMemStats(&after)
	if err != nil || sha != seen.SHA(b) {
		t.Fatalf("currentSHA = %q (%v), want the ledger's digest of the file", sha, err)
	}
	if d := after.TotalAlloc - before.TotalAlloc; d > 1<<20 {
		t.Errorf("hashing a 50 MB file allocated %d bytes, want under 1 MB", d)
	}
}
