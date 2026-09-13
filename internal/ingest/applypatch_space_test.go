package ingest

import (
	"bytes"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

func TestCompileDeleteFileQuotesASpacedPath(t *testing.T) {
	root := writeTree(t, map[string]string{"my file.go": demo})
	doc := "*** Begin Patch\n*** Delete File: my file.go\n*** End Patch\n"
	planText, err := CompileApplyPatch(root, []byte(doc))
	if err != nil {
		t.Fatalf("Delete File with a space compiled: %v", err)
	}
	if !strings.Contains(string(planText), `@@ "my file.go" - unlink`) {
		t.Fatalf("compiled text is not a quoted unlink hunk:\n%s", planText)
	}
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatalf("quoted unlink did not parse: %v\n%s", err, planText)
	}
	if len(hunks) != 1 || hunks[0].Path != "my file.go" || hunks[0].Op != plan.OpUnlink {
		t.Fatalf("hunk = %+v", hunks)
	}
}

func TestCompileMoveToQuotesASpacedPath(t *testing.T) {
	root := writeTree(t, map[string]string{"old file.go": demo})
	doc := "*** Begin Patch\n*** Update File: old file.go\n*** Move to: new file.go\n*** End Patch\n"
	planText, err := CompileApplyPatch(root, []byte(doc))
	if err != nil {
		t.Fatalf("Move to with a space compiled: %v", err)
	}
	if !strings.Contains(string(planText), `@@ "old file.go" - rename`) {
		t.Fatalf("compiled text is not a quoted rename hunk:\n%s", planText)
	}
	hunks, err := plan.Parse(bytes.NewReader(planText))
	if err != nil {
		t.Fatalf("quoted rename did not parse: %v\n%s", err, planText)
	}
	if len(hunks) != 1 || hunks[0].Path != "old file.go" || hunks[0].Op != plan.OpRename {
		t.Fatalf("hunk = %+v", hunks)
	}
	if len(hunks[0].Body) != 1 || hunks[0].Body[0] != "new file.go" {
		t.Fatalf("dest body = %q", hunks[0].Body)
	}
}
