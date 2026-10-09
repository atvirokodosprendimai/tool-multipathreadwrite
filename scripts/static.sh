#!/usr/bin/env bash
# scripts/static.sh — every static analyser this repository gates on (ADR-088).
#
# gofmt, go vet, golangci-lint (.golangci.yml), deadcode, staticcheck's U1000
# with tests excluded, govulncheck, and fence-prose (every task fence's grep
# over a tracked file still matches: a prose rewrite turned 8 done fences red
# unnoticed, measured 2026-09-29). Each runs whatever the others found, and the
# script exits 1 when any of them found anything. python3 runs fence-prose.
# and the script exits 1 when any of them found anything. CI runs it, and so
# does the PostToolUse hook after a commit (.claude/hooks/static-after-commit.py).
#
# The Go tools run as `go run module@version`: pinned, the same locally and in
# CI, nothing to install, and go.mod untouched. golangci-lint is the one binary
# to install, because its authors advise against building it with go run; CI
# installs the same version.
set -u
cd "$(dirname "$0")/.." || exit 2

GOLANGCI_VERSION=2.14.0
DEADCODE=golang.org/x/tools/cmd/deadcode@v0.50.0
STATICCHECK=honnef.co/go/tools/cmd/staticcheck@v0.8.1
GOVULNCHECK=golang.org/x/vuln/cmd/govulncheck@v1.8.0

fail=0
step() {
  local name=$1
  shift
  echo "== $name"
  if ! "$@"; then
    echo "FAIL $name"
    fail=1
  fi
}

# gofmt -l and deadcode exit 0 while printing findings, so their output is the
# verdict: anything printed fails the step.
empty() {
  local out
  out=$("$@") || { printf '%s\n' "$out"; return 1; }
  [ -z "$out" ] || { printf '%s\n' "$out"; return 1; }
}

golangci() {
  if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint is not installed. Install v$GOLANGCI_VERSION: brew install golangci-lint, or https://golangci-lint.run/docs/welcome/install/"
    return 1
  fi
  local v
  v=$(golangci-lint version --short 2>/dev/null || golangci-lint --version)
  case "$v" in
    *"$GOLANGCI_VERSION"*) ;;
    *) echo "note: golangci-lint is $v; CI runs $GOLANGCI_VERSION, so findings can differ" ;;
  esac
  golangci-lint run ./...
}

# staticcheck reads the compiler's export data, and v0.8.1 (still the newest
# release, and master too, 2026-10-09) cannot decode the version Go 1.27 writes:
# "export data version 5 is greater than maximum supported version 4". It runs
# under the toolchain go.mod names, the one CI builds with, whatever go is on PATH.
staticcheck_u1000() {
  GOTOOLCHAIN="go$(go list -m -f '{{.GoVersion}}')" go run "$STATICCHECK" -tests=false -checks U1000 ./...
}

step gofmt empty gofmt -l .
step vet go vet ./...
step golangci-lint golangci
step "deadcode (production code nothing reaches, or only tests reach)" empty go run "$DEADCODE" ./...
step "staticcheck U1000 (unused in production, tests excluded)" staticcheck_u1000
step govulncheck go run "$GOVULNCHECK" ./...
step "fence-prose self-test (a red clause is reported)" python3 scripts/fence-prose.py --self-test
step "fence-prose (every task fence's grep over a tracked file still matches)" python3 scripts/fence-prose.py .

if [ "$fail" -eq 0 ]; then
  echo "static analysis clean"
else
  echo "static analysis FAILED"
fi
exit "$fail"
