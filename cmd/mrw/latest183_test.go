package main

import (
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"testing"
)

// latestServer answers the releases API with the tag in tag and records the
// User-Agent of the last request.
func latestServer(t *testing.T) (srv *httptest.Server, tag, ua *atomic.Value) {
	t.Helper()
	tag, ua = &atomic.Value{}, &atomic.Value{}
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua.Store(r.Header.Get("User-Agent"))
		_, _ = w.Write([]byte(`{"tag_name":"` + tag.Load().(string) + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, tag, ua
}

// verdictCase is one answer: the reported version, the latest tag, the exit
// code, a line stdout must hold, and the error main would print.
type verdictCase struct {
	name, running, latest string
	code                  int
	stdout, err           string
}

// checkVerdict runs `mrw version --check-latest` and holds it to tc, with
// the request's User-Agent naming the reported version.
func checkVerdict(t *testing.T, tc verdictCase, tag, ua *atomic.Value) {
	t.Helper()
	tag.Store(tc.latest)
	ua.Store("")
	stdout, msg, code := runVersion(t, "version", "--check-latest")
	if code != tc.code {
		t.Fatalf("exit %d, want %d; stdout %q; error %q", code, tc.code, stdout, msg)
	}
	if !strings.Contains(stdout, tc.stdout) {
		t.Errorf("stdout %q does not say %q", stdout, tc.stdout)
	}
	if tc.err == "" && msg != "" || tc.err != "" && !strings.Contains(msg, tc.err) {
		t.Errorf("error %q, want one containing %q", msg, tc.err)
	}
	if got := ua.Load().(string); got != "mrw/"+tc.running {
		t.Errorf("User-Agent %q, want %q", got, "mrw/"+tc.running)
	}
}

// ADR-183 Decisions 2, 3 and 5. A pseudo-version of the release form,
// vX.Y.(Z+1)-0.<timestamp>-<rev>, is judged by the release vX.Y.Z it follows:
// a newer latest is older (exit 1), an equal or lower one a commit after the
// base (exit 0). Every other shape has no base, and no verdict.
func TestCheckLatestJudgesAPseudoVersionByItsBaseRelease(t *testing.T) {
	srv, tag, ua := latestServer(t)
	const p = "v1.78.1-0.20261029000000-abcdefabcdef"
	for _, tc := range []verdictCase{
		{"an equal base is a commit after the latest", p, "v1.78.0", 0,
			"latest release: v1.78.0 — this mrw (" + p + ") is a commit after v1.78.0", ""},
		{"the release after the base makes it older", p, "v1.78.1", 1,
			"latest release: v1.78.1 — this mrw (" + p + ") is older; download https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest",
			"a newer release exists: v1.78.1"},
		{"a later minor makes it older", p, "v1.79.0", 1,
			"this mrw (" + p + ") is older; download", "a newer release exists: v1.79.0"},
		{"a base above the latest is a commit after the base", p, "v1.77.0", 0,
			"latest release: v1.77.0 — this mrw (" + p + ") is a commit after v1.78.0", ""},
		{"the base compares as numbers, not text", "v1.10.1-0.20261029000000-abcdefabcdef", "v1.9.9", 0,
			"this mrw (v1.10.1-0.20261029000000-abcdefabcdef) is a commit after v1.10.0", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pointAt(t, tc.running, srv.URL)
			checkVerdict(t, tc, tag, ua)
		})
	}
	for _, v := range []string{
		"v0.0.0-20261029000000-abcdefabcdef",          // no tag before the commit
		"v1.78.0-rc.1.0.20261029000000-abcdefabcdef",  // after a pre-release tag
		"v1.78.0-0.20261029000000-abcdefabcdef",       // patch 0: invalid
		"v1.78.1-0.20261029000000-abcdefabcdef+dirty", // build metadata
		"v1.78.1-0.2026102900000-abcdefabcdef",        // 13 digits
		"v01.78.1-0.20261029000000-abcdefabcdef",      // a leading zero
	} {
		t.Run("no verdict for "+v, func(t *testing.T) {
			pointAt(t, v, srv.URL)
			checkVerdict(t, verdictCase{running: v, latest: "v1.78.0",
				stdout: "latest release: v1.78.0 — this build (" + v + ") cannot be compared"}, tag, ua)
		})
	}
}

// pointBuildInfo replaces what readBuildInfo returns for one test.
func pointBuildInfo(t *testing.T, module string, settings []debug.BuildSetting) {
	t.Helper()
	old := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/atvirokodosprendimai/tool-multipathreadwrite", Version: module}, Settings: settings}, true
	}
	t.Cleanup(func() { readBuildInfo = old })
}

// ADR-183 Decisions 1 and 4. The real command judges the version mrw reports
// — moduleVersion over the build info — not the linker stamp: a go install
// build, stamped nothing, is judged by its pseudo-version or its tag, while a
// checkout build and (devel) keep no verdict. A test binary's own build info
// says (devel), so the build info is the readBuildInfo seam's.
func TestCheckLatestJudgesTheVersionAGoInstallBuildReports(t *testing.T) {
	srv, tag, ua := latestServer(t)
	const p = "v1.77.2-0.20261028222602-46fc0cf9e067"
	none := []debug.BuildSetting{{Key: "-buildmode", Value: "exe"}}
	rev := []debug.BuildSetting{{Key: "vcs.revision", Value: "9471accd8503d1fd0c29120e954e72483ca682fc"}}
	for _, tc := range []struct {
		verdictCase
		module   string
		settings []debug.BuildSetting
		line     string
	}{
		{verdictCase{"go install @sha behind the latest is older", p, "v1.78.0", 1,
			"this mrw (" + p + ") is older; download", "a newer release exists: v1.78.0"}, p, none, p},
		{verdictCase{"go install @tag at the latest is current", "v1.77.1", "v1.77.1", 0,
			"latest release: v1.77.1 — this mrw is current", ""}, "v1.77.1", none, "v1.77.1"},
		{verdictCase{"go install @tag behind the latest is older", "v1.77.1", "v1.78.0", 1,
			"this mrw (v1.77.1) is older; download", "a newer release exists: v1.78.0"}, "v1.77.1", none, "v1.77.1"},
		{verdictCase{"a checkout build gives no verdict", "dev", "v1.78.0", 0,
			"latest release: v1.78.0 — this build (dev) cannot be compared", ""}, p, rev, "dev (9471acc)"},
		{verdictCase{"a (devel) build gives no verdict", "dev", "v1.78.0", 0,
			"latest release: v1.78.0 — this build (dev) cannot be compared", ""}, "(devel)", none, "dev"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pointAt(t, "dev", srv.URL)
			pointBuildInfo(t, tc.module, tc.settings)
			if got := versionString(); got != tc.line {
				t.Fatalf("version line %q, want %q", got, tc.line)
			}
			checkVerdict(t, tc.verdictCase, tag, ua)
		})
	}
}
