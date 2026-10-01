package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// ADR-111 T1. Receipt keys were only ever added, but nothing said a caller may
// rely on that and nothing failed when one was removed or renamed. Every key of
// the CLI's JSON receipts is listed in docs/receipts.txt, and the receipt types
// and the file agree. stats' counts is a map, but its keys are the vocabulary,
// every one present at zero (ADR-054): names a caller reads, held like fields.
func TestNoShippedReceiptFieldDisappears(t *testing.T) {
	got := map[string]bool{}
	receiptPaths(got, "write", reflect.TypeOf(receipt{}))
	receiptPaths(got, "check", reflect.TypeOf(checkReceipt{}))
	receiptPaths(got, "check_refusal", reflect.TypeOf(checkRefusal{}))
	receiptPaths(got, "stats", reflect.TypeOf(statsReceipt{}))
	for _, name := range authoring.Vocabulary() {
		got["stats counts."+name] = true
	}
	compareWithShipped(t, got, "write", "check", "check_refusal", "stats")
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
