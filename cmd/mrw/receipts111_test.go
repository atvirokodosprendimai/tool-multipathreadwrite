package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ADR-111 T1. Receipt keys were only ever added, but nothing said a caller may
// rely on that and nothing failed when one was removed or renamed. Every key of
// the CLI's JSON receipts is listed in docs/receipts.txt, and the receipt types
// and the file agree.
func TestNoShippedReceiptFieldDisappears(t *testing.T) {
	compareWithShipped(t, map[string]reflect.Type{
		"write":         reflect.TypeOf(receipt{}),
		"check":         reflect.TypeOf(checkReceipt{}),
		"check_refusal": reflect.TypeOf(checkRefusal{}),
		"stats":         reflect.TypeOf(statsReceipt{}),
	})
}

// receiptPaths lists every key path of a receipt type as docs/receipts.txt
// spells it: "<receipt> <path>", nested keys joined by ".", a slice's elements
// as "[]" and a map's values as "{}". An embedded struct's keys are its
// parent's, as encoding/json inlines them; a []byte is a leaf.
func receiptPaths(name string, t reflect.Type) []string {
	var out []string
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
				out = append(out, name+" "+prefix+key)
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
	return out
}

// compareWithShipped requires docs/receipts.txt and the reflected receipts to
// agree, for the receipts named: a shipped path a type lost is a removal, and a
// path the file lacks is an addition nobody wrote down (ADR-111).
func compareWithShipped(t *testing.T, receipts map[string]reflect.Type) {
	t.Helper()
	got := map[string]bool{}
	for name, typ := range receipts {
		for _, p := range receiptPaths(name, typ) {
			got[p] = true
		}
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "receipts.txt"))
	if err != nil {
		t.Fatal(err)
	}
	shipped, per := map[string]bool{}, map[string]int{}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if f := strings.Fields(line); len(f) == 2 && receipts[f[0]] != nil {
			shipped[line] = true
			per[f[0]]++
		}
	}
	for name := range receipts {
		if per[name] == 0 {
			t.Errorf("docs/receipts.txt lists no field of %s; the comparison would hold nothing", name)
		}
	}
	var lost, unlisted []string
	for p := range shipped {
		if !got[p] {
			lost = append(lost, p)
		}
	}
	for p := range got {
		if !shipped[p] {
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
