package main

import (
	"bytes"
	"context"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// runVersion runs the real root command with argv after "mrw" and returns
// stdout, the error main would print, and the exit code main would use.
func runVersion(t *testing.T, argv ...string) (string, string, int) {
	t.Helper()
	cmd := rootCommand()
	var out bytes.Buffer
	cmd.Writer, cmd.ErrWriter = &out, &out
	err := cmd.Run(context.Background(), append([]string{"mrw"}, argv...))
	if err == nil {
		return out.String(), "", 0
	}
	return out.String(), err.Error(), exitCode(err)
}

// pointAt sets version and releasesAPI for one test and restores both.
func pointAt(t *testing.T, v, api string) {
	t.Helper()
	oldV, oldAPI := version, releasesAPI
	version, releasesAPI = v, api
	t.Cleanup(func() { version, releasesAPI = oldV, oldAPI })
}

// ADR-149 Decisions 1, 2 and 5. Each answer a local server gives is one line
// and one exit code: current and newer 0, older 1, a dev build 0 with no
// verdict, and every way of not getting an answer 2.
func TestCheckLatestSaysWhetherANewerReleaseExists(t *testing.T) {
	const latestPath = "/repos/atvirokodosprendimai/tool-multipathreadwrite/releases/latest"
	var status atomic.Int32
	var body, gotPath, gotUA atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath.Store(r.URL.Path)
		gotUA.Store(r.Header.Get("User-Agent"))
		w.WriteHeader(int(status.Load()))
		_, _ = w.Write([]byte(body.Load().(string)))
	}))
	defer srv.Close()

	for _, tc := range []struct {
		name, running, status, body string
		code                        int
		stdout, err                 string
	}{
		{"the same release is current", "v1.65.0", "200", `{"tag_name":"v1.65.0"}`, 0,
			"latest release: v1.65.0 — this mrw is current", ""},
		{"an older running version is told to update", "v1.64.0", "200", `{"tag_name":"v1.65.0"}`, 1,
			"latest release: v1.65.0 — this mrw (v1.64.0) is older; download https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest",
			"a newer release exists: v1.65.0"},
		{"a minor below a higher patch is older", "v1.9.9", "200", `{"tag_name":"v1.10.0"}`, 1,
			"this mrw (v1.9.9) is older", "a newer release exists: v1.10.0"},
		{"a newer running version is not older", "v1.66.0", "200", `{"tag_name":"v1.65.0"}`, 0,
			"latest release: v1.65.0 — this mrw (v1.66.0) is newer", ""},
		{"a dev build gives no verdict", "dev", "200", `{"tag_name":"v1.65.0"}`, 0,
			"latest release: v1.65.0 — this build (dev) cannot be compared", ""},
		{"a status other than 200 could not ask", "v1.65.0", "404", `{"message":"Not Found"}`, 2,
			"", "could not ask for the latest release: "},
		{"a body that is not JSON could not ask", "v1.65.0", "200", `<html>`, 2,
			"", "could not ask for the latest release: "},
		{"a tag that is not strict could not ask", "v1.65.0", "200", `{"tag_name":"v1.66.0-rc1"}`, 2,
			"", "could not ask for the latest release: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, _ := strconv.Atoi(tc.status)
			status.Store(int32(n))
			body.Store(tc.body)
			gotPath.Store("")
			gotUA.Store("")
			pointAt(t, tc.running, srv.URL)
			stdout, msg, code := runVersion(t, "version", "--check-latest")
			if code != tc.code {
				t.Fatalf("exit %d, want %d; stdout %q; error %q", code, tc.code, stdout, msg)
			}
			if !strings.HasPrefix(stdout, versionString()+"\n") {
				t.Errorf("stdout does not start with the version line %q: %q", versionString(), stdout)
			}
			if tc.stdout != "" && !strings.Contains(stdout, tc.stdout) {
				t.Errorf("stdout %q does not say %q", stdout, tc.stdout)
			}
			if tc.err == "" && msg != "" || tc.err != "" && !strings.Contains(msg, tc.err) {
				t.Errorf("error %q, want one containing %q", msg, tc.err)
			}
			if p := gotPath.Load().(string); p != latestPath {
				t.Errorf("asked %q, want %q", p, latestPath)
			}
			if ua := gotUA.Load().(string); ua != "mrw/"+tc.running {
				t.Errorf("User-Agent %q, want %q", ua, "mrw/"+tc.running)
			}
		})
	}

	t.Run("a server that is gone could not ask", func(t *testing.T) {
		gone := httptest.NewServer(http.NotFoundHandler())
		gone.Close()
		pointAt(t, "v1.65.0", gone.URL)
		stdout, msg, code := runVersion(t, "version", "--check-latest")
		if code != 2 || !strings.Contains(msg, "could not ask for the latest release: ") {
			t.Fatalf("exit %d, error %q; want 2 and could not ask", code, msg)
		}
		if strings.Contains(stdout, "latest release:") {
			t.Errorf("stdout claims an answer it never got: %q", stdout)
		}
	})
}

// ADR-149 Decision 4. Plain `mrw version` asks nothing: the flag is what asks.
func TestVersionWithoutTheFlagAsksNothing(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	defer srv.Close()
	pointAt(t, "v1.65.0", srv.URL)
	for _, argv := range [][]string{{"version"}, {"--version"}, {"-v"}} {
		stdout, msg, code := runVersion(t, argv...)
		if code != 0 || msg != "" || strings.Contains(stdout, "latest release") {
			t.Errorf("mrw %v: exit %d, error %q, stdout %q", argv, code, msg, stdout)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("a version query without --check-latest made %d request(s)", n)
	}
}

// ADR-149 Decision 4. No shipped file but latest149.go imports the network:
// a call nobody asked for would need an import this test refuses.
func TestOnlyTheCheckLatestFileImportsTheNetwork(t *testing.T) {
	const allowed = "cmd/mrw/latest149.go"
	found := false
	for _, dir := range []string{"../../cmd", "../../internal"} {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" || d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			rel := filepath.ToSlash(strings.TrimPrefix(p, "../../"))
			for _, imp := range f.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if path != "net" && !strings.HasPrefix(path, "net/") {
					continue
				}
				if rel == allowed {
					found = true
					continue
				}
				t.Errorf("%s imports %q; only %s may reach the network (ADR-149)", rel, path, allowed)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Errorf("%s imports no net package: the walk is not reaching the file it allows", allowed)
	}
}
