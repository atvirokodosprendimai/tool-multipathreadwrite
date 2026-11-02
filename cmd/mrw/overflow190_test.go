package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAQuoteRangeAtTheIntegerLimitIsRefusedOnTheCLI(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, `a.go`), []byte(demoFile), 0o644); err != nil {
		t.Fatal(err)
	}
	nl := string(rune(10))
	plan := filepath.Join(root, `p.plan`)
	text := `@@ a.go 1-9223372036854775807 replace anchor=package quote=1.x` + nl + `x` + nl
	if err := os.WriteFile(plan, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(t, func() error {
		cmd := rootCommand()
		var sink bytes.Buffer
		cmd.Writer, cmd.ErrWriter = &sink, &sink
		return cmd.Run(context.Background(), []string{`mrw`, `-C`, root, `write`, `--no-check`, plan})
	})
	if got := exitCode(err); got != exitNotApplied {
		t.Fatal(got, err, out)
	}
}
