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
	maxAnswer         = 1 << 20
)

// strictTag is the shape every release tag has: the build job publishes only
// for a strict vX.Y.Z tag, so a version of any other shape was not released.
var strictTag = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

// semver splits a strict vX.Y.Z into its three numbers.
func semver(v string) ([3]int, bool) {
	return numbers(strictTag.FindStringSubmatch(v))
}

// pseudoRelease is the one pseudo-version form with a release before it: Go
// names a commit after tag vX.Y.Z vX.Y.(Z+1)-0.<14-digit timestamp>-<revision>
// (ADR-183 Decision 2). Every other form — vX.0.0-<timestamp>-<revision> with
// no tag before it, a commit after a pre-release tag, build metadata — has no
// base mrw judges.
var pseudoRelease = regexp.MustCompile(`^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)-0\.\d{14}-[0-9A-Za-z]+$`)

// pseudoVersionBase is the release a release-form pseudo-version follows.
func pseudoVersionBase(v string) ([3]int, bool) {
	n, ok := numbers(pseudoRelease.FindStringSubmatch(v))
	if !ok || n[2] == 0 {
		// vX.Y.0-0.… has no patch to step back from; Go never writes it.
		return [3]int{}, false
	}
	n[2]--
	return n, true
}

// numbers reads the three version numbers a match of strictTag or
// pseudoRelease captured.
func numbers(m []string) ([3]int, bool) {
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
// network call in mrw (ADR-149 Decision 4), made only for --check-latest; its
// User-Agent names the version mrw reports (ADR-183 Decision 1).
func latestTag(ctx context.Context, running string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, latestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAPI+latestReleasePath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mrw/"+running)
	req.Header.Set("Accept", "application/vnd.github+json")
	// Redirects are followed, as Go's client follows them (at most ten): GitHub
	// answers 301 for a renamed repository. The answer they end at must be 200.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s answered %s", req.URL.Host, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswer+1))
	if err != nil {
		return "", fmt.Errorf("reading the answer: %w", err)
	}
	if len(body) > maxAnswer {
		return "", fmt.Errorf("the answer is larger than %d bytes", maxAnswer)
	}
	var answer struct {
		TagName string `json:"tag_name"`
	}
	// Unmarshal, not one Decode: the body is one JSON value, and a body with
	// anything after it is not a release (the Codex review of #402).
	if err := json.Unmarshal(body, &answer); err != nil {
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
// version is neither a released tag nor a pseudo-version with a release
// before it gets the latest named and no verdict. running is the version mrw
// reports, so a go install build is judged too (ADR-183).
func checkLatest(ctx context.Context, out io.Writer, running string) error {
	latest, err := latestTag(ctx, running)
	if err != nil {
		return cli.Exit("could not ask for the latest release: "+err.Error(), exitUsage)
	}
	want, _ := semver(latest)
	// A commit after release vB is older only when the latest is above vB; an
	// equal latest is the release it follows, and a lower one a tag GitHub does
	// not call latest (ADR-183 Decision 3).
	if base, ok := pseudoVersionBase(running); ok {
		if compareSemver(base, want) < 0 {
			return older(out, latest, running)
		}
		_, err = fmt.Fprintf(out, "latest release: %s — this mrw (%s) is a commit after v%d.%d.%d\n", latest, running, base[0], base[1], base[2])
		return err
	}
	have, ok := semver(running)
	if !ok {
		_, err := fmt.Fprintf(out, "latest release: %s — this build (%s) cannot be compared\n", latest, running)
		return err
	}
	switch compareSemver(have, want) {
	case 0:
		_, err = fmt.Fprintf(out, "latest release: %s — this mrw is current\n", latest)
	case 1:
		_, err = fmt.Fprintf(out, "latest release: %s — this mrw (%s) is newer\n", latest, running)
	default:
		return older(out, latest, running)
	}
	return err
}

// older prints that a newer release exists and where to get it, exit 1.
func older(out io.Writer, latest, running string) error {
	if _, err := fmt.Fprintf(out, "latest release: %s — this mrw (%s) is older; download %s\n", latest, running, latestReleasePage); err != nil {
		return err
	}
	return cli.Exit("a newer release exists: "+latest, exitNotApplied)
}
