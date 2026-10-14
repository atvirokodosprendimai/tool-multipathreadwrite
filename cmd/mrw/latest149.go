package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/urfave/cli/v3"
)

// releasesAPI is where `mrw version --check-latest` asks for the latest
// release (ADR-149). A package-level string variable, like version, so a test
// and contract §253 point a built binary at a local server with
// -ldflags "-X main.releasesAPI=…"; no environment variable or flag moves it.
var releasesAPI = "https://api.github.com"

const (
	latestReleasePath = "/repos/atvirokodosprendimai/tool-multipathreadwrite/releases/latest"
	latestReleasePage = "https://github.com/atvirokodosprendimai/tool-multipathreadwrite/releases/latest"
	latestTimeout     = 5 * time.Second
)

// strictTag is the shape every release tag has: the build job publishes only
// for a strict vX.Y.Z tag, so a version of any other shape was not released.
var strictTag = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// semver splits a strict vX.Y.Z into its three numbers.
func semver(v string) ([3]int, bool) {
	m := strictTag.FindStringSubmatch(v)
	if m == nil {
		return [3]int{}, false
	}
	var n [3]int
	for i := range n {
		x, err := strconv.Atoi(m[i+1])
		if err != nil {
			return [3]int{}, false
		}
		n[i] = x
	}
	return n, true
}

// compareSemver is -1, 0 or 1 as a is below, equal to or above b.
func compareSemver(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// latestTag asks the releases API for the latest release's tag. It is the one
// network call in mrw (ADR-149 Decision 4), made only for --check-latest.
func latestTag(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, latestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI+latestReleasePath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mrw/"+version)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s answered %s", req.URL.Host, resp.Status)
	}
	var answer struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&answer); err != nil {
		return "", fmt.Errorf("the answer is not a release: %w", err)
	}
	if _, ok := semver(answer.TagName); !ok {
		return "", fmt.Errorf("the answer names no release tag (tag_name %q)", answer.TagName)
	}
	return answer.TagName, nil
}

// checkLatest prints whether a newer release than this mrw exists (ADR-149
// Decisions 1 and 2), after the version line the caller already printed. A
// newer release is exit 1; not getting an answer is exit 2; a build whose
// version is not a released tag gets the latest named and no verdict.
func checkLatest(ctx context.Context, out io.Writer) error {
	latest, err := latestTag(ctx)
	if err != nil {
		return cli.Exit("could not ask for the latest release: "+err.Error(), exitUsage)
	}
	have, ok := semver(version)
	if !ok {
		_, err := fmt.Fprintf(out, "latest release: %s — this build (%s) cannot be compared\n", latest, version)
		return err
	}
	want, _ := semver(latest)
	switch compareSemver(have, want) {
	case 0:
		_, err = fmt.Fprintf(out, "latest release: %s — this mrw is current\n", latest)
	case 1:
		_, err = fmt.Fprintf(out, "latest release: %s — this mrw (%s) is newer\n", latest, version)
	default:
		if _, err := fmt.Fprintf(out, "latest release: %s — this mrw (%s) is older; download %s\n", latest, version, latestReleasePage); err != nil {
			return err
		}
		return cli.Exit("a newer release exists: "+latest, exitNotApplied)
	}
	return err
}
