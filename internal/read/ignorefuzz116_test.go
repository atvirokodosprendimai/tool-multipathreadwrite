package read

import (
	"bytes"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ADR-116 differential: random gitignore files over random trees, the
// matcher against `git check-ignore`, path by path. git is the oracle; the
// test skips without it. MRW_IGNORE_FUZZ sets the number of cases (default 40).
func TestTheIgnoreMatcherAgreesWithGitOnRandomRules(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH: the differential needs it as an oracle")
	}
	n := 40
	if v, err := strconv.Atoi(os.Getenv("MRW_IGNORE_FUZZ")); err == nil && v > 0 {
		n = v
	}
	dirs := []string{"a", "b", "ab", "sub"}
	leaves := []string{"a.x", "b.x", "ab.x", "c.y", "a", "é.x", "aé", "A.x", "Ab.x", "É.x"}
	tokens := []string{"a", "b", "ab", "*", "?", "*.x", "a*", "[ab]", "[!a]*", "**", "c.y", "sub", "[a-c].x", "a?", "*b*", "?.x",
		"[[:alpha:]]*", "[[:punct:]]x", "[c-a]*", "[!c-a].x", "[a-b-c]*", "[]a]*", "[!]]*", "[\\]a]*", "a[a\\-z]", "[[:bogus:]]*", "[!/]*",
		"[[:x]a:]*", "[[:]:]", "[ab", "a[b-\\]", "?.x", "??.x", "[!a]?", "é*", "*é", "[é]*",
		"A*", "[A]*", "[!A]*", "[A-C].x", "[[:upper:]]*", "\\A*", "*B*", "É*"}
	for seed := int64(1); seed <= int64(n); seed++ {
		r := rand.New(rand.NewSource(seed))
		root := t.TempDir()
		if out, err := exec.Command(git, "-C", root, "init", "-q").CombinedOutput(); err != nil {
			t.Skipf("git init failed: %v %s", err, out)
		}
		var paths []string
		seen := map[string]bool{}
		for i := 0; i < 25; i++ {
			depth := r.Intn(3)
			var parts []string
			for d := 0; d < depth; d++ {
				parts = append(parts, dirs[r.Intn(len(dirs))])
			}
			parts = append(parts, leaves[r.Intn(len(leaves))])
			p := strings.Join(parts, "/")
			// A leaf named like a directory on the same path would collide.
			if seen[p] || clashes(p, seen) {
				continue
			}
			seen[p] = true
			paths = append(paths, p)
		}
		gi := func() string {
			var lines []string
			for i := 0; i < 1+r.Intn(4); i++ {
				var segs []string
				for s := 0; s < 1+r.Intn(3); s++ {
					segs = append(segs, tokens[r.Intn(len(tokens))])
				}
				l := strings.Join(segs, "/")
				if r.Intn(4) == 0 {
					l = "/" + l
				}
				if r.Intn(5) == 0 {
					l += "/"
				}
				if r.Intn(4) == 0 {
					l = "!" + l
				}
				switch r.Intn(8) {
				case 0:
					l += "   " // trailing spaces are trimmed
				case 1:
					lines = append(lines, "# "+l) // a comment changes nothing
				}
				lines = append(lines, l)
			}
			return strings.Join(lines, "\n") + "\n"
		}
		files := map[string]string{".gitignore": gi()}
		if r.Intn(2) == 0 {
			files["sub/.gitignore"] = gi()
		}
		for _, p := range paths {
			files[p] = "x\n"
		}
		ok := true
		for name, body := range files {
			full := filepath.Join(root, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				ok = false
				break
			}
			if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		// -z: git quotes a non-ASCII path in line output ("\303\251"), so the
		// oracle is read NUL-separated, as written.
		cmd := exec.Command(git, "-C", root, "check-ignore", "--no-index", "--stdin", "-z")
		cmd.Stdin = strings.NewReader(strings.Join(paths, "\x00") + "\x00")
		var out bytes.Buffer
		cmd.Stdout = &out
		_ = cmd.Run()
		gitSays := map[string]bool{}
		for _, l := range strings.Split(out.String(), "\x00") {
			if l != "" {
				gitSays[l] = true
			}
		}
		ig := newIgnorer(root)
		for _, p := range paths {
			if got := ig.Ignored(p, false, 0); got != gitSays[p] {
				t.Errorf("case %d: %q: matcher %v, git %v\n.gitignore:\n%s\nsub/.gitignore:\n%s", seed, p, got, gitSays[p], files[".gitignore"], files["sub/.gitignore"])
			}
		}
		if t.Failed() {
			t.Fatalf("case %d disagreed with git (MRW_IGNORE_FUZZ=%d reruns it)", seed, seed)
		}
	}
}

// clashes says whether p would need a file where another path needs a
// directory, or the reverse.
func clashes(p string, seen map[string]bool) bool {
	for q := range seen {
		if strings.HasPrefix(q, p+"/") || strings.HasPrefix(p, q+"/") {
			return true
		}
	}
	return false
}
