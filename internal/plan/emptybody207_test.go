package plan

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ADR-207 Decision 1. A delete's body is the lines it expects to remove, checked
// against the file before anything is written; `body=@FILE` loads it after the
// parse, and a file of zero bytes loaded as no body at all, so the check was
// skipped exactly when the producer of the file had failed. On a delete it is
// refused now, naming the plan line and the file. Every other op keeps what it did.
func TestAnEmptyBodyFileOnADeleteIsRefused(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{"empty.txt": "", "crlf.txt": "\r\n", "b.txt": "b"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	load := func(t *testing.T, header string) ([]Hunk, error) {
		t.Helper()
		hunks, err := Parse(strings.NewReader("\n\n" + header + "\n"))
		if err != nil {
			t.Fatalf("%s: parse: %v", header, err)
		}
		return hunks, LoadBodyFiles(root, hunks)
	}

	t.Run("delete_zero_bytes", func(t *testing.T) {
		_, err := load(t, "@@ f.txt 2 delete body=@empty.txt")
		if err == nil {
			t.Fatal("a delete with a zero-byte body file was loaded")
		}
		for _, want := range []string{"line 3", "body=@empty.txt", "0 bytes", "a delete's body=@ is the lines it expects to remove", "nothing was deleted"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("the refusal %q does not say %q", err, want)
			}
		}
	})
	t.Run("delete_one_blank_line", func(t *testing.T) {
		hunks, err := load(t, "@@ f.txt 2 delete body=@crlf.txt")
		if err != nil || !slices.Equal(hunks[0].Body, []string{""}) {
			t.Errorf("one blank line: body %q, err %v; want [\"\"] and no error", hunks[0].Body, err)
		}
	})
	t.Run("delete_non_empty", func(t *testing.T) {
		hunks, err := load(t, "@@ f.txt 2 delete body=@b.txt")
		if err != nil || !slices.Equal(hunks[0].Body, []string{"b"}) {
			t.Errorf("a non-empty body file: body %q, err %v; want [\"b\"] and no error", hunks[0].Body, err)
		}
	})
	t.Run("create_zero_bytes", func(t *testing.T) {
		hunks, err := load(t, "@@ new.txt 0 create body=@empty.txt")
		if err != nil || hunks[0].Body != nil || !hunks[0].CountedBody {
			t.Errorf("an empty create from a file: body %q counted %v, err %v; want nil, counted, no error", hunks[0].Body, hunks[0].CountedBody, err)
		}
	})
	t.Run("replace_zero_bytes", func(t *testing.T) {
		hunks, err := load(t, "@@ f.txt 1 replace anchor=\"a\" body=@empty.txt")
		if err != nil || hunks[0].Body != nil {
			t.Errorf("a replace with an empty body file: body %q, err %v; want nil and no error (apply refuses it)", hunks[0].Body, err)
		}
	})
	t.Run("insert_before_zero_bytes", func(t *testing.T) {
		hunks, err := load(t, "@@ f.txt 1 insert-before body=@empty.txt")
		if err != nil || hunks[0].Body != nil {
			t.Errorf("an insert with an empty body file: body %q, err %v; want nil and no error (apply refuses it)", hunks[0].Body, err)
		}
	})
	t.Run("delete_body_zero", func(t *testing.T) {
		if _, err := load(t, "@@ f.txt 2 delete body=0"); err != nil {
			t.Errorf("body=0 on a delete keeps its meaning, no expectation: %v", err)
		}
	})
}
