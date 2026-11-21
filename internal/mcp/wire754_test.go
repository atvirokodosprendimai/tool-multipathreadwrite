package mcp

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer"
)

// #754. mrw_write's check.outside_scope and drift name root-relative files, so
// they are spelled with "/" as the other receipt paths are (ADR-091). Driven
// with `\` as the separator so it runs on every platform.
func TestAWriteReceiptSpellsOutsideScopeAndDriftWithSlashes(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	prev := receiptSep
	receiptSep = '\\'
	t.Cleanup(func() { receiptSep = prev })
	v := &writer.Verified{Check: &check.Result{Ran: true, OutsideScope: []string{`d\x.go`}}, Drift: []string{`d\y.go`}}
	out, rpcErr := checkedReceipt(t.TempDir(), apply.Result{Applied: true}, v, nil, false)
	if rpcErr != nil {
		t.Fatal(rpcErr)
	}
	raw, isRaw := out.StructuredContent.(json.RawMessage)
	if !isRaw {
		t.Fatalf("no structured receipt: %v", out)
	}
	var sc struct {
		Check struct {
			OutsideScope []string `json:"outside_scope"`
		} `json:"check"`
		Drift []string `json:"drift"`
	}
	if err := json.Unmarshal(raw, &sc); err != nil {
		t.Fatal(err)
	}
	if want := []string{"d/x.go"}; !slices.Equal(sc.Check.OutsideScope, want) || !slices.Equal(sc.Drift, []string{"d/y.go"}) {
		t.Errorf("outside_scope = %q drift = %q, want d/x.go and d/y.go", sc.Check.OutsideScope, sc.Drift)
	}
	for _, c := range out.Content {
		if strings.Contains(c.Text, `d\x.go`) || strings.Contains(c.Text, `d\\x.go`) {
			t.Errorf("the answer still spells outside_scope with a backslash: %s", c.Text)
		}
	}
}
