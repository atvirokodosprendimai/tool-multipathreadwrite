package read

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnIncompleteDeclarationHasNoInventedEnd(t *testing.T) {
	for name, tail := range map[string]string{
		"function":                "func Bad() {\n",
		"function with statement": "func Bad() {\n println(1)\n",
		"struct":                  "type Bad struct {\n X int\n",
		"interface":               "type Bad interface {\n M()\n",
		"group":                   "var (\n Bad = 1\n",
		"literal":                 "var Bad = []int{1,\n",
		"call":                    "var Bad = f(1,\n",
	} {
		t.Run(name, func(t *testing.T) {
			out, err := goOutline("a.go", []byte("package p\nfunc Good() {}\n"+tail))
			if err == nil || len(out) != 2 || out[0] != "2-2 func Good" || !strings.HasPrefix(out[1], "3-? ") {
				t.Fatalf("outline %q, error %v: want the complete Good and an unknown end for Bad", out, err)
			}
		})
	}
	out, err := goOutline("a.go", []byte("package p\nfunc Good() { _ = missing + }\n"))
	if err == nil || len(out) != 1 || out[0] != "2-2 func Good" {
		t.Fatalf("a syntax error inside a closed declaration lost its known end: %q, %v", out, err)
	}
}

func TestAByteNoticeNamesFiveFilesAndCountsTheRest(t *testing.T) {
	files := map[string]string{}
	var paths []string
	for i := 1; i <= 7; i++ {
		name := fmt.Sprintf("u%d.txt", i)
		files[name] = "\xff\xfeh\x00i\x00\n\x00"
		paths = append(paths, name)
	}
	root := t.TempDir()
	var specs []Spec
	for _, p := range paths {
		if err := os.WriteFile(filepath.Join(root, p), []byte(files[p]), 0o600); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, Spec{Path: p, Raw: p})
	}
	var out bytes.Buffer
	Run(&out, root, specs, Options{Numbers: true})
	text := out.String()
	at := strings.LastIndex(text, "-- note:")
	if at < 0 {
		t.Fatalf("missing byte notice: %s", text)
	}
	note := text[at:]
	if !strings.Contains(note, "u5.txt is UTF-16 (BOM FF FE) and 2 more") || strings.Contains(note, "u6.txt") || strings.Count(note, "-- note:") != 1 {
		t.Fatalf("unbounded byte notice: %s", note)
	}
}
