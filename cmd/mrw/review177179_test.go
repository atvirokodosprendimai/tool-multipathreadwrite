package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

func TestRunLogUsesTheReadersLineBoundaries(t *testing.T) {
	for name, body := range map[string]string{
		"LF":                     "first\nlast\n",
		"CRLF":                   "first\r\nlast\r\n",
		"CR only":                "first\rlast\r",
		"CR no final terminator": "first\rlast",
		"mixed":                  "first\rsecond\nlast\r\n",
		"LF after a buffer":      strings.Repeat("x\r", 40000) + "last\n",
		"empty":                  "",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "run.log")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			want, _, _ := lines.Split(body)
			n, size, last := scanLog(path)
			if n != len(want) || size != int64(len(body)) {
				t.Fatalf("%d lines, %d bytes; want %d, %d", n, size, len(want), len(body))
			}
			if len(want) > 0 && len(want[len(want)-1]) <= lastChars && last != strings.TrimRight(want[len(want)-1], "\r") {
				t.Fatalf("last %q, want %q", last, want[len(want)-1])
			}
		})
	}
}

func TestUnusedGuardNoticeNamesFiveFilesAndCountsTheRest(t *testing.T) {
	files := map[string]string{}
	args := []string{"read", "--grep", "absent"}
	for i := 1; i <= 7; i++ {
		p := fmt.Sprintf("a%d.txt", i)
		files[p] = "present\n"
		args = append(args, "--if-changed", p+"=00000000")
	}
	root := grepTree(t, files)
	args = append(args, ".")
	_, notice, code := project178Run(t, root, "", args...)
	if code != 1 || !strings.Contains(notice, "a1.txt, a2.txt, a3.txt, a4.txt, a5.txt and 2 more") || strings.Contains(notice, "a6.txt") || strings.Count(notice, "-- guard not used:") != 1 {
		t.Fatalf("exit %d: %s", code, notice)
	}
}

func TestAKeptPlanCollisionNeverReplacesTheOriginal(t *testing.T) {
	// Freeze both name ingredients without adding a production test hook.
	synctest.Test(t, func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		old := rand.Reader
		rand.Reader = bytes.NewReader(make([]byte, 8))
		t.Cleanup(func() { rand.Reader = old })
		first := &planTee{}
		first.buf.WriteString("original plan\n")
		path, err := first.write()
		if err != nil {
			t.Fatal(err)
		}
		second := &planTee{}
		second.buf.WriteString("replacement plan\n")
		got, err := second.write()
		if !os.IsExist(err) || got != "" {
			t.Fatalf("collision returned %q, %v; want an existing-path refusal", got, err)
		}
		b, err := os.ReadFile(path)
		if err != nil || string(b) != "original plan\n" {
			t.Fatalf("original changed to %q, %v", b, err)
		}
	})
}
