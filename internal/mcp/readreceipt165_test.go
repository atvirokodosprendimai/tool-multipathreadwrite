package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/read"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-165 T3. mrw_read's content[1] is built by ReadReceipt, the function
// `mrw read --json` calls too, so one change reaches both surfaces. For a walk
// mrw_read answers whole, the receipt is ReadReceipt's output for that walk and
// carries what each of its parts adds: observed keyed with "/", the problem
// count, and skipped (the binary file the walk passed over).
func TestTheReadReceiptIsComposedOnceForBothSurfaces(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"d/f.txt": "alpha\nbravo\n", "bin.dat": "a\x00b\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, rpcErr := readTool(root, json.RawMessage(`{"grep":"a","specs":["."]}`)) // bin.dat matches too, so the walk skips it as binary
	if rpcErr != nil || res.IsError || len(res.Content) < 2 {
		t.Fatalf("mrw_read: %+v %v", res, rpcErr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(res.Content[1].Text), &got); err != nil {
		t.Fatalf("content[1]: %v\n%s", err, res.Content[1].Text)
	}

	sum := sha256.Sum256([]byte("alpha\nbravo\n"))
	sha := hex.EncodeToString(sum[:])
	observed := map[string]seen.Observation{filepath.FromSlash("d/f.txt"): {SHA: sha, Spans: [][2]int{{1, 2}}}}
	b, err := json.Marshal(ReadReceipt(observed, 0, read.WalkSkipped{Binary: 1}, nil, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("mrw_read's content[1] is not ReadReceipt's output for the same walk:\n got %s\nwant %s", gotJSON, wantJSON)
	}
	if _, ok := got["observed"].(map[string]any)["d/f.txt"]; !ok {
		t.Errorf("observed is not keyed with /: %v", got["observed"])
	}
	if sk, _ := got["skipped"].(map[string]any); sk["binary"] != float64(1) {
		t.Errorf("skipped does not carry the binary file: %v", got["skipped"])
	}
}
