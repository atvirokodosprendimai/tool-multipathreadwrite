package mcp

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ADR-111 T2. MCP's receipts are held to docs/receipts.txt as the CLI's are.
// mrw_write's is a type. mrw_read's has variants — served, paged, no match,
// index — built as maps; readSchema declares the union of their keys, and
// TestTheReadReceiptMatchesItsSchema holds that declaration to a real answer of
// each variant both ways, so the keys are taken from it.
func TestNoShippedMCPReceiptFieldDisappears(t *testing.T) {
	got := map[string]bool{}
	receiptPaths(got, "mcp_write", reflect.TypeOf(writeReceipt{}))
	props, _ := readSchema()["properties"].(map[string]any)
	for k := range props {
		got["mcp_read "+k] = true
		// A nested object's keys are listed under it, as skipped.binary is.
		inner, _ := props[k].(map[string]any)["properties"].(map[string]any)
		for sub := range inner {
			got["mcp_read "+k+"."+sub] = true
		}
	}
	observed, _ := props["observed"].(map[string]any)
	entry, _ := observed["additionalProperties"].(map[string]any)
	entryProps, _ := entry["properties"].(map[string]any)
	for k := range entryProps {
		got["mcp_read observed{}."+k] = true
	}
	compareWithShipped(t, got, "mcp_write", "mcp_read")
}

// receiptPaths adds every key path of a receipt type to got, as
// docs/receipts.txt spells it: "<receipt> <path>", nested keys joined by ".", a
// slice's elements as "[]" and a map's values as "{}". An embedded struct's keys
// are its parent's, as encoding/json inlines them; a []byte is a leaf.
func receiptPaths(got map[string]bool, name string, t reflect.Type) {
	var walk func(t reflect.Type, prefix string, depth int)
	walk = func(t reflect.Type, prefix string, depth int) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if depth > 8 {
			return
		}
		switch t.Kind() {
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if !f.IsExported() {
					continue
				}
				key, _, _ := strings.Cut(f.Tag.Get("json"), ",")
				if key == "-" {
					continue
				}
				if f.Anonymous && key == "" {
					walk(f.Type, prefix, depth+1)
					continue
				}
				if key == "" {
					key = f.Name
				}
				got[name+" "+prefix+key] = true
				walk(f.Type, prefix+key+".", depth+1)
			}
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Uint8 {
				return
			}
			walk(t.Elem(), strings.TrimSuffix(prefix, ".")+"[].", depth+1)
		case reflect.Map:
			walk(t.Elem(), strings.TrimSuffix(prefix, ".")+"{}.", depth+1)
		}
	}
	walk(t, "", 0)
}

// compareWithShipped requires docs/receipts.txt and got to agree for the
// receipts named: a shipped path got lacks is a removal, and a path the file
// lacks is an addition nobody wrote down (ADR-111).
func compareWithShipped(t *testing.T, got map[string]bool, names ...string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "receipts.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	shipped, per := map[string]bool{}, map[string]int{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if f := strings.Fields(line); len(f) == 2 && want[f[0]] {
			shipped[line] = true
			per[f[0]]++
		}
	}
	for _, n := range names {
		if per[n] == 0 {
			t.Errorf("docs/receipts.txt lists no field of %s; the comparison would hold nothing", n)
		}
	}
	var lost, unlisted []string
	for p := range shipped {
		if !got[p] {
			lost = append(lost, p)
		}
	}
	for p := range got {
		name, _, _ := strings.Cut(p, " ")
		if want[name] && !shipped[p] {
			unlisted = append(unlisted, p)
		}
	}
	sort.Strings(lost)
	sort.Strings(unlisted)
	for _, p := range lost {
		t.Errorf("a shipped receipt field disappeared: %s (ADR-111: keys are only added)", p)
	}
	for _, p := range unlisted {
		t.Errorf("a receipt field is not in docs/receipts.txt: %s", p)
	}
}
