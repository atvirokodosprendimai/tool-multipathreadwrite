package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestExactEditRejectsLossyStrings(t *testing.T) {
	for _, field := range []string{"target", "old", "replacement"} {
		for _, value := range []string{`\ud800`, `\udfff`, `\ud800x`, `\ud800\ud800`} {
			target, old, replacement := "a", "old", "new"
			switch field {
			case "target":
				target = value
			case "old":
				old = value
			case "replacement":
				replacement = value
			}
			doc := fmt.Sprintf(`[{"target":"%s","old":["%s"],"replacement":["%s"]}]`, target, old, replacement)
			if _, err := readEdits(strings.NewReader(doc)); err == nil {
				t.Fatalf("accepted %s containing %s", field, value)
			}
		}
	}
	root := fixture(t, map[string]string{"a": "old\n"})
	doc := `[{"target":"a","old":["old"],"replacement":["�","\uFFFD","\ud83d\ude00","\\uD800"]}]`
	if _, err := execute(root, strings.NewReader(doc)); err != nil {
		t.Fatal(err)
	}
	wantText(t, root, "a", "�\n�\n😀\n\\uD800\n")
}
