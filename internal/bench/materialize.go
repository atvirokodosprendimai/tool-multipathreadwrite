package bench

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repo is where a corpus repository is found on this machine. Only public
// repositories are listed, so another reader can rerun the corpus (ADR-156
// Decision 3).
type Repo struct {
	URL  string `json:"url"`
	Path string `json:"path"` // "~/" is the home directory
	// DropSections are sections of the repository's instruction files the
	// bench removes in both configurations (ADR-158 Decision 2).
	DropSections []Section `json:"drop_sections,omitempty"`
}

// Section names one markdown section: its file, relative to the tree, and its
// heading line exactly as written.
type Section struct {
	File    string `json:"file"`
	Heading string `json:"heading"`
}

// sectionSpan finds heading in lines, outside code fences, and returns the
// span from it to the next heading of the same or a higher level, and how many
// times the heading occurs.
func sectionSpan(lines []string, heading string) (start, end, n int) {
	level := len(heading) - len(strings.TrimLeft(heading, "#"))
	start, end = -1, len(lines)
	// A fence closes only on a run of its own character at least as long as
	// the one that opened it, with nothing after it (CommonMark): a ``` inside
	// a ```` example is the example's text, and so is a heading below it.
	var fenceChar byte
	fenceLen := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceLen > 0 {
			if run := fenceRun(trimmed); run >= fenceLen && run == len(trimmed) && trimmed[0] == fenceChar {
				fenceLen = 0
			}
			continue
		}
		if run := fenceRun(trimmed); run >= 3 {
			fenceChar, fenceLen = trimmed[0], run
			continue
		}
		if strings.TrimRight(line, "\r\n") == heading {
			n++
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 && end == len(lines) && strings.HasPrefix(line, "#") {
			if l := len(line) - len(strings.TrimLeft(line, "#")); l <= level && strings.HasPrefix(line[l:], " ") {
				end = i
			}
		}
	}
	return start, end, n
}

// fenceRun is the length of the run of backticks or tildes s begins with.
func fenceRun(s string) int {
	if s == "" || s[0] != '`' && s[0] != '~' {
		return 0
	}
	return len(s) - len(strings.TrimLeft(s, s[:1]))
}

// dropSections removes each declared section from its file under root. A file
// that is not in the tree is skipped, since a tree may hold only some files; a
// heading found other than exactly once is an error naming both.
func dropSections(root string, ss []Section) error {
	for _, s := range ss {
		p := filepath.Join(root, s.File)
		b, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		lines := strings.SplitAfter(string(b), "\n")
		start, end, n := sectionSpan(lines, strings.TrimRight(s.Heading, "\n"))
		if n != 1 {
			return fmt.Errorf("%s: the declared section %q is there %d times; drop_sections needs it exactly once", s.File, s.Heading, n)
		}
		if err := os.WriteFile(p, []byte(strings.Join(append(lines[:start:start], lines[end:]...), "")), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// checkSections reads each section the task's repository declares at the
// task's commit, so a stage refuses before its first attempt (ADR-158).
func checkSections(t Task, src string) error {
	repos, err := loadRepos(filepath.Dir(t.Dir))
	if err != nil {
		return err
	}
	for _, s := range repos[t.Repo].DropSections {
		out, err := exec.Command("git", "-C", src, "show", t.Commit+":"+s.File).Output()
		if err != nil {
			return fmt.Errorf("task %s: %s has no %s at %s, which drop_sections declares", t.ID, t.Repo, s.File, t.Commit)
		}
		if _, _, n := sectionSpan(strings.SplitAfter(string(out), "\n"), s.Heading); n != 1 {
			return fmt.Errorf("task %s: %s at %s: the declared section %q is there %d times; drop_sections needs it exactly once", t.ID, s.File, t.Commit, s.Heading, n)
		}
	}
	return nil
}

// errUnreachable marks a task whose repository or pinned commit is not on this
// machine; the oracle test skips such a task by name.
var errUnreachable = errors.New("not on this machine")

// loadRepos reads bench/repos.json, one directory above the task directory.
func loadRepos(tasksDir string) (map[string]Repo, error) {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(tasksDir), "repos.json"))
	if err != nil {
		return nil, err
	}
	repos := map[string]Repo{}
	if err := json.Unmarshal(b, &repos); err != nil {
		return nil, fmt.Errorf("repos.json: %w", err)
	}
	return repos, nil
}

// repoPath resolves a task's repository to a local checkout holding its commit.
func repoPath(t Task) (string, error) {
	repos, err := loadRepos(filepath.Dir(t.Dir))
	if err != nil {
		return "", err
	}
	r, ok := repos[t.Repo]
	if !ok {
		return "", fmt.Errorf("task %s: repository %q is not in repos.json", t.ID, t.Repo)
	}
	p := r.Path
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, rest)
	}
	if err := exec.Command("git", "-C", p, "cat-file", "-e", t.Commit+"^{commit}").Run(); err != nil {
		return "", fmt.Errorf("%s (%s) has no commit %s at %s: %w", t.Repo, r.URL, t.Commit, p, errUnreachable)
	}
	return p, nil
}

// Materialize builds a task's base tree in dst, an empty directory: the files
// named (or, with none named, the whole tree) at the pinned commit, or the
// fixture's base/ directory, with its instructions prepared (PrepareInstructions) and the
// result committed, so the oracle can diff against HEAD. It returns HEAD.
func Materialize(t Task, dst string, files []string) (string, error) {
	return materialize(t, dst, files, "")
}

// materialize is Materialize with the stage's declared check: under stage6 it
// is written as .quality-harness.json before the base commit, so it is in
// every arm's tree and never in the diff the harness grades (ADR-163
// Decision 2).
func materialize(t Task, dst string, files []string, check string) (string, error) {
	if t.Repo == "fixture" {
		if err := copyTree(filepath.Join(t.Dir, "base"), dst); err != nil {
			return "", err
		}
	} else {
		src, err := repoPath(t)
		if err != nil {
			return "", err
		}
		args := append([]string{"-C", src, "archive", "--format=tar", t.Commit, "--"}, files...)
		var out, stderr bytes.Buffer
		cmd := exec.Command("git", args...)
		cmd.Stdout, cmd.Stderr = &out, &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git archive: %w: %s", err, stderr.String())
		}
		if err := untar(&out, dst); err != nil {
			return "", err
		}
	}
	if err := PrepareInstructions(dst); err != nil {
		return "", err
	}
	if t.Repo != "fixture" {
		repos, err := loadRepos(filepath.Dir(t.Dir))
		if err != nil {
			return "", err
		}
		if err := dropSections(dst, repos[t.Repo].DropSections); err != nil {
			return "", err
		}
	}
	if check != "" {
		if err := writeCheckFile(dst, check); err != nil {
			return "", err
		}
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "-A"},
		{"-c", "user.name=bench", "-c", "user.email=bench@localhost", "-c", "commit.gpgsign=false", "commit", "-q", "--no-verify", "-m", "base: " + t.ID},
	} {
		if err := git(dst, args...); err != nil {
			return "", err
		}
	}
	var head bytes.Buffer
	cmd := exec.Command("git", "-C", dst, "rev-parse", "HEAD")
	cmd.Stdout = &head
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(head.String()), nil
}

func git(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "core.autocrlf=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return nil
}

func untar(r io.Reader, dst string) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		p := filepath.Join(dst, filepath.FromSlash(h.Name))
		if !strings.HasPrefix(p, filepath.Clean(dst)+string(filepath.Separator)) {
			return fmt.Errorf("archive entry %q leaves the tree", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(p, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(h.Linkname, p); err != nil {
				return err
			}
		}
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
