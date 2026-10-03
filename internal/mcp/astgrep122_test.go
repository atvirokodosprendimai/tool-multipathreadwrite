package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ADR-122, the review of #329: an ast_grep answer that serves nothing still
// says what mrw's rules dropped, as an empty grep answer does.
func TestAnEmptyAstGrepAnswerCarriesSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake ast-grep is a POSIX shell script")
	}
	root := licensed(t, map[string]string{".gitignore": "gen/\n", "gen/a.go": "package gen\n"})
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	fake := "#!/bin/sh\nprintf '%s' '[{\"file\":\"gen/a.go\",\"range\":{\"start\":{\"line\":0},\"end\":{\"line\":0}}}]'\n"
	if err := os.WriteFile(filepath.Join(bin, "ast-grep"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	sc := receipt(t, call(t, root, "mrw_read", map[string]any{"ast_grep": "package $A"}))
	sk, _ := sc["skipped"].(map[string]any)
	if sc["matches"] != float64(0) || sk == nil || sk["ignored_dirs"] != float64(1) {
		t.Fatalf("want no match and skipped.ignored_dirs 1: %v", sc)
	}
}
