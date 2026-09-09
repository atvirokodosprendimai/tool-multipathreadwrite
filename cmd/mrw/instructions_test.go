package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

func TestInstructionsCommandPrintsCLI(t *testing.T) {
	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer = &buf
	if err := cmd.Run(context.Background(), []string{"mrw", "instructions"}); err != nil {
		t.Fatalf("instructions: %v", err)
	}
	if got, want := buf.String(), guide.CLI(); got != want {
		t.Errorf("stdout is not guide.CLI()\ngot:\n%s\nwant:\n%s", got, want)
	}
}
