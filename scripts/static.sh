#!/usr/bin/env bash
# scripts/static.sh — every static analyser this repository gates on (ADR-088).
#
# gofmt, go vet, golangci-lint (.golangci.yml), deadcode, staticcheck's U1000
# with tests excluded, govulncheck, and fence-prose (every task fence's grep
# over a tracked file still matches: a prose rewrite turned 8 done fences red
# unnoticed, measured 2026-09-29). Then what is not Go (ADR-147): ruff over the
# Python files, shellcheck over the shell scripts, actionlint over the
# workflows, knip over the opencode plugin, and a self-test that each of those
# four reports a defect planted for it. Each runs whatever the others found,
# and the script exits 1 when any of them found anything. python3 runs
# fence-prose. CI runs the script, and so does the PostToolUse hook after a
# commit (.claude/hooks/static-after-commit.py).
#
# The Go tools run as `go run module@version`: pinned, the same locally and in
# CI, nothing to install, and go.mod untouched; actionlint is one of them.
# golangci-lint, ruff and shellcheck are the binaries to install — golangci-lint
# because its authors advise against building it with go run — and CI installs
# the same versions. knip is the plugin's exact devDependency, so `npm ci` in
# cmd/opencode/mrw-plugin installs it with its whole tree pinned by the lock.
set -u
cd "$(dirname "$0")/.." || exit 2

# Every `go` command below, and every tool's Go subprocess (package loading,
# export data), runs under the toolchain go.mod names, the one every CI job
# builds with (setup-go's go-version-file), whatever go is on PATH; gofmt is
# taken from that toolchain's GOROOT for the same reason. What it cannot pin is
# the Go built INTO an installed binary: golangci-lint's own parser and
# analysers are the ones its release embeds. Printed, so a run says which
# compiler judged it; the first run downloads it, later runs use the cache.
GOTOOLCHAIN="go$(go list -m -f '{{.GoVersion}}')"; export GOTOOLCHAIN
echo "toolchain: $(go version)"

GOLANGCI_VERSION=2.14.0
DEADCODE=golang.org/x/tools/cmd/deadcode@v0.50.0
STATICCHECK=honnef.co/go/tools/cmd/staticcheck@v0.8.1
GOVULNCHECK=golang.org/x/vuln/cmd/govulncheck@v1.8.0
RUFF_VERSION=0.15.6
SHELLCHECK_VERSION=0.11.0
ACTIONLINT=github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
PLUGIN=cmd/opencode/mrw-plugin

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

# gofmt -l over the Go files git sees (tracked, and untracked but not ignored), not
# a walk of `.`: an agent's worktree under .claude/worktrees (ignored) holds files
# mid-edit, and `gofmt -l .` failed every commit made while one was open. A deleted
# tracked file is skipped.
gofmt_seen() {
  local files=() f list
  # Listed into a file first: a process substitution would hide git's exit status, and
  # a listing that failed would pass the step with nothing checked (Codex review of #501).
  list=$(mktemp) || return 1
  git ls-files -z --cached --others --exclude-standard -- '*.go' > "$list" \
    || { echo "gofmt: git ls-files failed, so no Go file was checked"; rm -f "$list"; return 1; }
  while IFS= read -r -d '' f; do [ -e "$f" ] && files+=("$f"); done < "$list"
  rm -f "$list"
  [ "${#files[@]}" -eq 0 ] || "$(go env GOROOT)/bin/gofmt" -l "${files[@]}"
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
# "export data version 5 is greater than maximum supported version 4". The
# GOTOOLCHAIN export at the top is what keeps it on go.mod's Go.
staticcheck_u1000() {
  go run "$STATICCHECK" -tests=false -checks U1000 ./...
}

# have TOOL VERSION HINT: TOOL is on PATH, or say how to install it; a version
# other than the one CI runs is noted, as golangci() does.
have() {
  local tool=$1 want=$2 hint=$3 v
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "$tool is not installed. Install $want: $hint"
    return 1
  fi
  v=$("$tool" --version 2>&1)
  case "$v" in
    *"$want"*) ;;
    *) echo "note: $tool is $(printf '%s' "$v" | tr '\n' ' '); CI runs $want, so findings can differ" ;;
  esac
}

# tracked NAME PATHSPEC...: the tracked files the pathspecs match, into the
# array `files`, counted out loud. None fails the step: an analyser handed no
# file exits 0 having looked at nothing — BSD xargs does not even start it.
tracked() {
  local name=$1 f
  shift
  files=()
  while IFS= read -r -d '' f; do files+=("$f"); done < <(git ls-files -z -- "$@")
  echo "$name: ${#files[@]} file(s)"
  [ "${#files[@]}" -gt 0 ] || { echo "$name: no tracked file matches $*"; return 1; }
}

# --no-cache: ruff otherwise writes .ruff_cache into the checkout.
ruff_check() {
  have ruff "$RUFF_VERSION" "brew install ruff, or pipx install ruff==$RUFF_VERSION" || return 1
  tracked ruff '*.py' || return 1
  ruff check --no-cache --select F,E9 "${files[@]}"
}

shellcheck_check() {
  have shellcheck "$SHELLCHECK_VERSION" "brew install shellcheck, or https://github.com/koalaman/shellcheck/releases/tag/v$SHELLCHECK_VERSION" || return 1
  tracked shellcheck '*.sh' || return 1
  shellcheck -S warning "${files[@]}"
}

# actionlint runs each run: script through the shellcheck on PATH, held to the
# bar scripts/ is held to; without one it skips them silently, and the step
# above that runs shellcheck would already have failed.
actionlint_check() {
  tracked actionlint '.github/workflows/*.yml' '.github/workflows/*.yaml' || return 1
  SHELLCHECK_OPTS='-S warning' go run "$ACTIONLINT" "${files[@]}"
}

knip_check() {
  [ -x "$PLUGIN/node_modules/.bin/knip" ] || { echo "knip is not installed: run npm ci in $PLUGIN"; return 1; }
  tracked knip "$PLUGIN/*.ts" "$PLUGIN/*.mjs" || return 1
  (cd "$PLUGIN" && ./node_modules/.bin/knip --no-progress)
}

# planted NAME PATTERN CMD...: CMD exits non-zero AND its output matches
# PATTERN, so a tool that never ran, or one that exits 1 on its own
# configuration, does not count as having found the defect.
planted() {
  local name=$1 pattern=$2 out rc
  shift 2
  out=$("$@" 2>&1)
  rc=$?
  if [ "$rc" -ne 0 ] && grep -qE -- "$pattern" <<<"$out"; then
    return 0
  fi
  echo "self-test: $name did not report its planted defect (exit $rc, wanted /$pattern/):"
  printf '%s\n' "$out" | tail -20
  return 1
}

# Each analyser ADR-147 adds, handed one defect planted for it in a directory
# removed on every exit, must report it. A subshell, so the trap is its own.
self_test() (
  d=$(mktemp -d) || exit 1
  trap 'rm -rf "$d"' EXIT
  printf 'import os\n' > "$d/planted.py"
  printf '#!/bin/sh\ncd /nowhere\necho here\n' > "$d/planted.sh"
  mkdir -p "$d/.github/workflows" "$d/knip/src"
  printf 'on: push\njobs:\n  planted:\n    runs-on: ubuntu-latest\n    steps:\n      - run: |\n          for i in 1 2; do echo here; done\n      - run: echo ${{ github.nosuch }}\n' > "$d/.github/workflows/planted.yml"
  printf '{"name":"planted","type":"module","main":"src/index.ts"}\n' > "$d/knip/package.json"
  printf 'export const used = 1;\n' > "$d/knip/src/index.ts"
  printf 'export const dead = 2;\n' > "$d/knip/src/dead.ts"
  ok=1
  planted ruff 'F401' ruff check --no-cache --select F,E9 "$d/planted.py" || ok=0
  planted shellcheck 'SC2164' shellcheck -S warning "$d/planted.sh" || ok=0
  planted 'actionlint (run: scripts)' 'SC2034' env SHELLCHECK_OPTS='-S warning' go run "$ACTIONLINT" "$d/.github/workflows/planted.yml" || ok=0
  planted 'actionlint (expressions)' 'nosuch' go run "$ACTIONLINT" "$d/.github/workflows/planted.yml" || ok=0
  planted knip 'dead\.ts' sh -c 'cd "$1" && "$2" --no-progress' _ "$d/knip" "$PWD/$PLUGIN/node_modules/.bin/knip" || ok=0
  [ "$ok" -eq 1 ] || exit 1
  echo "self-test: ruff, shellcheck, actionlint and knip each reported its planted defect"
)

step gofmt empty gofmt_seen
step vet go vet ./...
step golangci-lint golangci
step "deadcode (production code nothing reaches, or only tests reach)" empty go run "$DEADCODE" ./...
step "staticcheck U1000 (unused in production, tests excluded)" staticcheck_u1000
step govulncheck go run "$GOVULNCHECK" ./...
step "fence-prose self-test (a red clause is reported)" python3 scripts/fence-prose.py --self-test
step "fence-prose (every task fence's grep over a tracked file still matches)" python3 scripts/fence-prose.py .
step ruff ruff_check
step shellcheck shellcheck_check
step actionlint actionlint_check
step knip knip_check
step self-test self_test

if [ "$fail" -eq 0 ]; then
  echo "static analysis clean"
else
  echo "static analysis FAILED"
fi
exit "$fail"
