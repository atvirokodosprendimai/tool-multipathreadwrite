package main

import (
	"runtime/debug"
	"testing"
)

// `go install github.com/…/cmd/mrw@<sha>` builds in module mode: no linker
// stamp, so version stays "dev", and no vcs.* settings, so versionFrom has no
// revision to add — the binary said "dev" although Go recorded the module
// version it was built from (v1.76.1-0.20261027131114-7c2c46c0504f, a Windows
// session, 2026-10-27). The module version names the commit; the linker stamp,
// when there is one, stays authoritative, and a checkout build keeps "dev".
func TestAGoInstallBuildReportsItsModuleVersion(t *testing.T) {
	pseudo := "v1.76.1-0.20261027131114-7c2c46c0504f"
	rev := []debug.BuildSetting{{Key: "vcs.revision", Value: "9471accd8503d1fd0c29120e954e72483ca682fc"}}
	none := []debug.BuildSetting{{Key: "-buildmode", Value: "exe"}}
	for _, tc := range []struct {
		name, stamp, module string
		settings            []debug.BuildSetting
		want                string
	}{
		{"go install @sha names its pseudo-version", "dev", pseudo, none, pseudo},
		{"go install @tag names the tag", "dev", "v1.77.1", none, "v1.77.1"},
		{"the linker stamp stays authoritative", "v1.77.1", pseudo, none, "v1.77.1"},
		{"a checkout build keeps dev and its revision", "dev", pseudo, rev, "dev (9471acc)"},
		{"a build with no module version stays dev", "dev", "(devel)", none, "dev"},
		{"an empty module version stays dev", "dev", "", none, "dev"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := versionFrom(moduleVersion(tc.stamp, tc.module, tc.settings), tc.settings); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
