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
