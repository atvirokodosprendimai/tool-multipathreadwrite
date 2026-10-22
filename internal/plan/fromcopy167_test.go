package plan

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-167 T1: from=PATH:ADDR parses into a source and a numeric address, and
// every shape the record refuses is refused at parse.
func TestAFromCopyParsesIntoASourceAndRefusesWhatItCannotCopy(t *testing.T) {
	type accepted struct {
		addr string
		want Addr
	}
	for _, c := range []accepted{
		{"3-5", Addr{Start: 3, End: 5}},
		{"7", Addr{Start: 7, End: 7}},
		{"2,+3", Addr{Start: 2, End: 2, RelEnd: 3}},
		{"$", Addr{Start: EOF, End: EOF}},
		{"4-$", Addr{Start: 4, End: EOF}},
	} {
		for _, hdr := range []string{
			"@@ dest.go 9 replace anchor=\"x\" from=a.go:" + c.addr,
			"@@ dest.go 9 insert-after from=a.go:" + c.addr,
			"@@ dest.go 9 insert-before from=a.go:" + c.addr,
			"@@ dest.go 0 create from=a.go:" + c.addr,
		} {
			hunks, err := Parse(strings.NewReader(hdr + "\n"))
			if err != nil {
				t.Errorf("%s: refused: %v", hdr, err)
				continue
			}
			h := hunks[0]
			if h.BodyFrom != "a.go" || h.BodyFromAddr != c.want || h.CountedBody || len(h.Body) != 0 {
				t.Errorf("%s: BodyFrom=%q addr=%+v counted=%v body=%q, want a.go %+v, uncounted, empty",
					hdr, h.BodyFrom, h.BodyFromAddr, h.CountedBody, h.Body, c.want)
			}
		}
	}

	t.Run("a one-letter source is a file, not a drive", func(t *testing.T) {
		// The whole value `a:3` reads as a drive on Windows; only the path part
		// `a` is asked whether it is rooted.
		hunks, err := Parse(strings.NewReader("@@ d.txt 0 create from=a:3\n"))
		if err != nil || hunks[0].BodyFrom != "a" || hunks[0].BodyFromAddr.Start != 3 {
			t.Errorf("from=a:3: %v %+v", err, hunks)
		}
	})

	t.Run("a colon in the source's own name", func(t *testing.T) {
		hunks, err := Parse(strings.NewReader("@@ d.txt 0 create from=a:b.txt:3\n"))
		if runtime.GOOS == "windows" {
			// `a:b.txt` is the drive-relative path `a:` + `b.txt` there (rooted.IsRooted).
			if err == nil {
				t.Errorf("from=a:b.txt:3 parsed on Windows, where it names a drive")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if hunks[0].BodyFrom != "a:b.txt" || hunks[0].BodyFromAddr.Start != 3 {
			t.Errorf("BodyFrom=%q addr=%+v, want a:b.txt line 3", hunks[0].BodyFrom, hunks[0].BodyFromAddr)
		}
	})

	refused := []string{
		"a.go:0", "a.go:-", "a.go:4-", "a.go:$-2", "a.go:$,+2", "a.go:1-2,5-6",
		"a.go:/x/", "a.go:/a/,/b/", "a.go:/a:b/", "a.go:$.k", "a.go", `C:\x`,
		"/x:3", "@1:3-5", "a.go:3-5,+2", "a.go:",
	}
	if runtime.GOOS == "windows" {
		// A drive path is rooted only where Windows reads it so (rooted.IsRooted);
		// elsewhere `C:x` is an ordinary file name with a colon.
		refused = append(refused, `C:\x:3`, "C:x:3")
	}
	for _, v := range refused {
		hdr := "@@ dest.go 9 insert-after from=" + v
		_, err := Parse(strings.NewReader(hdr + "\n"))
		if err == nil {
			t.Errorf("%s: parsed, want a refusal", hdr)
			continue
		}
		if !strings.Contains(err.Error(), "from=") {
			t.Errorf("%s: refusal does not name from=: %v", hdr, err)
		}
	}

	for name, plan := range map[string]string{
		"body=2":                "@@ d.go 9 insert-after from=a.go:3 body=2\nx\ny\n",
		"body=@":                "@@ d.go 9 insert-after from=a.go:3 body=@b.txt\n",
		"a body line":           "@@ d.go 9 insert-after from=a.go:3\nextra\n",
		"raw=true":              "@@ d.go 9 insert-after from=a.go:3 raw=true\n",
		"delete":                "@@ d.go 3-4 delete from=a.go:3\n",
		"unlink":                "@@ d.go - unlink from=a.go:3\n",
		"rename":                "@@ d.go - rename from=a.go:3\nn.go\n",
		"from= given twice":     "@@ d.go 9 insert-after from=a.go:3 from=a.go:4\n",
		"replace needs a range": "@@ d.go 0 replace from=a.go:3\n",
	} {
		if _, err := Parse(strings.NewReader(plan)); err == nil {
			t.Errorf("%s: parsed, want a refusal", name)
		}
	}

	t.Run("the loader opens nothing for a from= hunk", func(t *testing.T) {
		hunks, err := Parse(strings.NewReader("@@ d.go 0 create from=nowhere.go:3\n"))
		if err != nil {
			t.Fatalf("a create with from= is refused at parse: %v", err)
		}
		if err := LoadBodyFiles(t.TempDir(), hunks); err != nil {
			t.Fatalf("LoadBodyFiles opened the source: %v", err)
		}
		if len(hunks[0].Body) != 0 {
			t.Errorf("Body = %q, want empty", hunks[0].Body)
		}
	})
}

// body=@ keeps ADR-060's meaning exactly: a colon in its value is part of the path.
func TestABodyFileWhoseNameHoldsAColonStillLoads(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a:3"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Skipf("this filesystem keeps no colon in a name: %v", err)
	}
	hunks, err := Parse(strings.NewReader("@@ d.txt 0 create body=@a:3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if hunks[0].BodyFrom != "" {
		t.Fatalf("body=@a:3 was read as a from= source: %q", hunks[0].BodyFrom)
	}
	if err := LoadBodyFiles(root, hunks); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(hunks[0].Body, ","); got != "one,two" {
		t.Errorf("Body = %q, want one,two", got)
	}
}
