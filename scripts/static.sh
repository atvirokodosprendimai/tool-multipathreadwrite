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
#
# DEADLINES (#773: a step hung on a runner for 46 minutes and no log was kept).
# Every step prints its name and the time before it starts, so the last line of
# a hung run's log names the tool even when everything below fails. A step that
# has not finished by its deadline is stopped and named ("static.sh: NAME did
# not finish in N s"). The whole run has a budget too, STATIC_TOTAL_SECS (420 s):
# a step's deadline is the smaller of its own and what is left of it, and when it
# runs out the message names the step that was running. The arithmetic CI relies
# on: the budget, plus the 5 s a stopped tree is given before KILL, plus the
# shell and python start-up, stays under the Static analysis step's
# timeout-minutes in .github/workflows/ci.yml, so the log of a hang is kept.
# Those defaults are for a warm machine (the post-commit hook kills at 280 s
# anyway); CI sets cold-safe values in the step's env, STATIC_TOTAL_SECS=900 and
# STATIC_DEADLINE_SECS=400 with a step limit of 18 minutes, and the two files are
# raised together. STATIC_DEADLINE_SECS=N makes every step's own deadline N
# seconds (a slow network, or a test); the budget still applies.
#
# A step is stopped by walking its process TREE (pgrep -P), TERM first and KILL
# 5 s later, and not by a process group of its own: everything stays in this
# script's group, so a hook or a terminal that kills the group still takes the
# steps with it. Named limits of a tree walk: a descendant that detached and was
# reparented to init before the walk, and a process forked between the walk and
# the signal, are not found. Setup-only contract.sh copies inherit this
# script's process group so that the hook's group kill reaches them too.
set -u
cd "$(dirname "$0")/.." || exit 2
T0=$SECONDS
STATIC_TOTAL_SECS=${STATIC_TOTAL_SECS:-420}
fail=0

# tree PID: the pids of PID's descendants, children before their parent, then PID.
tree() {
  local c
  for c in $(pgrep -P "$1" 2>/dev/null); do tree "$c"; done
  echo "$1"
}
# signal_all SIG PID...
signal_all() {
  local sig=$1 p
  shift
  for p in "$@"; do kill "-$sig" "$p" 2>/dev/null; done
}
# any_alive PID...: some pid in the list still exists.
any_alive() {
  local p
  for p in "$@"; do kill -0 "$p" 2>/dev/null && return 0; done
  return 1
}
# stop_tree PID: TERM the tree, give it 5 s, then KILL what is left of it and of
# what it had spawned meanwhile.
stop_tree() {
  local pids
  pids=$(tree "$1")
  signal_all TERM $pids
  for _ in $(seq 1 50); do any_alive $pids || break; sleep 0.1; done
  # shellcheck disable=SC2046 # a list of pids, split on purpose
  signal_all KILL $pids $(tree "$1")
}

# What the INT and TERM handler needs to know about the step being run, set by
# deadline() while one is: the handler is installed once, here, so it is armed
# for every step and for the toolchain lookups, and it stays armed until the
# step's tree is confirmed gone (a second signal during the clean-up is ignored).
CUR_PID=
CUR_WATCH=
ABORTING=0
abort() {
  local pids
  [ "$ABORTING" -eq 0 ] || return 0
  ABORTING=1
  # The step's tree is stopped first, which also lets the watchdog see it gone; a
  # watchdog that had already fired is waited for, it owes a KILL (it holds the
  # pids it found), and none of that needs a signal to it.
  if [ -n "$CUR_PID" ]; then
    pids=$(tree "$CUR_PID")
    stop_tree "$CUR_PID"
  fi
  if [ -n "$CUR_WATCH" ]; then
    { wait "$CUR_WATCH"; } 2>/dev/null
  fi
  if [ -n "$CUR_PID" ]; then
    # shellcheck disable=SC2086 # a list of pids, split on purpose
    ! any_alive $pids || echo "static.sh: a process of the interrupted step is still running" >&2
  fi
  exit "$1"
}
trap 'abort 130' INT
trap 'abort 143' TERM

# deadline NAME SECS CMD...: CMD, a function or a program, in a background
# subshell, with a watchdog that stops its tree when SECS pass. The watchdog
# polls every 0.2 s whether CMD is still there, so it needs no signal and
# nothing to cancel: it exits 3 on seeing CMD gone, and 124 only after it fired
# and did the stopping. The verdict is that exit status, so timed out is never a
# guess from a marker file, from elapsed seconds or from CMD's own status. Then
# `static.sh: NAME did not finish in SECS s` goes to stderr and the status is
# 124; otherwise CMD's own status. The one race: CMD finishing in the instant
# SECS pass counts as finished if the watchdog sees it gone on its last look,
# and as timed out if it does not. The wrapper always waits for the watchdog, so
# its KILL is always sent, and a step costs at most 0.2 s more than CMD. INT and
# TERM to this script are handled by abort() above. bash only: macOS has no
# timeout(1) and a perl alarm cannot run a function.
deadline() {
  local name=$1 secs=$2 pid watcher rc wrc
  shift 2
  ( "$@" ) &
  pid=$!
  CUR_PID=$pid
  ( n=$((secs * 5))
    while [ "$n" -gt 0 ]; do
      kill -0 "$pid" 2>/dev/null || exit 3
      sleep 0.2
      n=$((n - 1))
    done
    kill -0 "$pid" 2>/dev/null || exit 3
    stop_tree "$pid"
    exit 124 ) &
  watcher=$!
  CUR_WATCH=$watcher
  { wait "$pid"; } 2>/dev/null
  rc=$?
  { wait "$watcher"; } 2>/dev/null
  wrc=$?
  CUR_PID=
  CUR_WATCH=
  if [ "$wrc" -eq 124 ]; then
    echo "static.sh: $name did not finish in $secs s" >&2
    return 124
  fi
  return "$rc"
}

# The deadline of a step: the tools that fetch (a Go tool through `go run`, with
# its modules and the vulnerability database, golangci-lint with a cold build
# cache, shellcheck over contract.sh) get the longer one. Measured cold (empty
# GOMODCACHE, GOCACHE and golangci-lint cache, macOS, 2026-11-21): 70 s for the
# whole script, the slowest step 37 s, 51 and 70 s on later runs as contract.sh
# grew; CI's whole Static analysis step has taken at most 115 s.
DEADLINE_TOOL=240
DEADLINE_LOCAL=120
step_deadline() {
  case $1 in
    golangci-lint | deadcode* | staticcheck* | govulncheck* | shellcheck | actionlint | self-test | toolchain*) echo "$DEADLINE_TOOL" ;;
    *) echo "$DEADLINE_LOCAL" ;;
  esac
}
# left: seconds of the budget not yet spent.
left() { echo $((STATIC_TOTAL_SECS - (SECONDS - T0))); }
# limit OWN: OWN, or what is left of the budget when that is less.
limit() {
  local rem
  rem=$(left)
  if [ "$rem" -lt "$1" ]; then echo "$rem"; else echo "$1"; fi
}

# step NAME CMD...: announce, run under the step's deadline, count a failure.
# STEP_FIXED=N in front of it gives the step N seconds whatever
# STATIC_DEADLINE_SECS says (the deadline self-test, which must not be shortened).
step() {
  local name=$1 t0=$SECONDS own secs rc
  shift
  echo "== $name  [$(date +%H:%M:%S)]"
  own=${STEP_FIXED:-${STATIC_DEADLINE_SECS:-$(step_deadline "$name")}}
  secs=$(limit "$own")
  if [ "$secs" -le 0 ]; then
    echo "static.sh: the total budget of $STATIC_TOTAL_SECS s ran out before $name; it was not run"
    echo "FAIL $name"
    fail=1
    return
  fi
  deadline "$name" "$secs" "$@"
  rc=$?
  if [ "$rc" -ne 0 ]; then
    if [ "$rc" -eq 124 ] && [ "$secs" -lt "$own" ]; then
      echo "static.sh: the total budget of $STATIC_TOTAL_SECS s ran out while $name was running"
    fi
    echo "FAIL $name"
    fail=1
  fi
  echo "-- $name: $((SECONDS - t0)) s"
}

# Every `go` command below, and every tool's Go subprocess (package loading,
# export data), runs under the toolchain go.mod names, the one every CI job
# builds with (setup-go's go-version-file), whatever go is on PATH; gofmt is
# taken from that toolchain's GOROOT for the same reason. What it cannot pin is
# the Go built INTO an installed binary: golangci-lint's own parser and
# analysers are the ones its release embeds. Printed, so a run says which
# compiler judged it; the first run downloads it, later runs use the cache, so
# both lookups run under a deadline and announce themselves.
echo "== toolchain  [$(date +%H:%M:%S)]"
tcfile=$(mktemp) || exit 2
deadline 'toolchain (go list -m)' "$(limit "${STATIC_DEADLINE_SECS:-$DEADLINE_TOOL}")" go list -m -f '{{.GoVersion}}' > "$tcfile" \
  || { rm -f "$tcfile"; echo "static.sh: no toolchain could be found for go.mod"; exit 2; }
toolchain=$(cat "$tcfile")
GOTOOLCHAIN="go$toolchain"; export GOTOOLCHAIN
deadline 'toolchain (go version)' "$(limit "${STATIC_DEADLINE_SECS:-$DEADLINE_TOOL}")" go version > "$tcfile" \
  || { rm -f "$tcfile"; echo "static.sh: the toolchain go.mod names did not start"; exit 2; }
echo "toolchain: $(cat "$tcfile")"
rm -f "$tcfile"

GOLANGCI_VERSION=2.14.0
DEADCODE=golang.org/x/tools/cmd/deadcode@v0.50.0
STATICCHECK=honnef.co/go/tools/cmd/staticcheck@v0.8.2
GOVULNCHECK=golang.org/x/vuln/cmd/govulncheck@v1.8.0
RUFF_VERSION=0.15.6
SHELLCHECK_VERSION=0.11.0
ACTIONLINT=github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
PLUGIN=cmd/opencode/mrw-plugin

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

# staticcheck reads the compiler's export data. v0.8.1 could not decode the
# version Go 1.27 writes ("export data version 5 is greater than maximum
# supported version 4"); v0.8.2 can, and go.mod moved to Go 1.27.4 with it
# (#635). The GOTOOLCHAIN export at the top keeps it on go.mod's Go.
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
  # contract.sh must fail on an error of its own that is not a row (issue #763).
  # Each copy is the real file with one line planted above the marker where its
  # setup ends, run through its real watcher with CONTRACT_STOP_AT_SETUP=1 so no
  # row runs and nothing is built. Four plants must fail and say why, and the
  # copy with nothing planted must pass, so a watcher that fails everything, or
  # one deleted, is caught as well.
  n=$(grep -n '^# --- end of the setup' scripts/contract.sh | cut -d: -f1)
  case $n in '' | *[!0-9]*) echo "self-test: contract.sh has no single end-of-setup marker"; exit 1 ;; esac
  contract_copy() {
    mkdir -p "$d/$1/scripts"
    { head -n $((n - 1)) scripts/contract.sh; [ -z "$2" ] || printf '%s\n' "$2"; tail -n +"$n" scripts/contract.sh; } > "$d/$1/scripts/contract.sh"
  }
  contract_copy stray '12087 insert-before'
  contract_copy bypass '12087 insert-before; exit 0'
  contract_copy nolog 'rm -f "$CONTRACT_STDERR_LOG"'
  contract_copy status 'echo inner-said-seven; exit 7'
  contract_copy teekill 'for _ in $(seq 1 50); do pkill -KILL -f "tee -a $CONTRACT_STDERR_LOG" && break; sleep 0.1; done'
  contract_copy jobreal ': < "12087 Killed"'
  contract_copy jobok $'printf \'%s: line 5:  1234 Killed\\n%s: line 6: 77 Killed: 9   cmd\\n\' "$0" "$0" >&2'
  contract_copy clean ''
  contract_copy group 'perl -e '\''exit(getpgrp(0) == $ENV{STATIC_TEST_PGID} ? 0 : 1)'\'' || exit 1'
  contract_copy nul 'x=$(printf "a\000b")'
  contract_copy nulok 'x=$(printf "a\000b" | LC_ALL=C tr -d "\000")'
  planted 'contract.sh (a stray line)' '12087: command not found' env CONTRACT_STOP_AT_SETUP=1 bash "$d/stray/scripts/contract.sh" || ok=0
  planted 'contract.sh (a stray line, then exit 0)' '12087: command not found' env CONTRACT_STOP_AT_SETUP=1 bash "$d/bypass/scripts/contract.sh" || ok=0
  planted 'contract.sh (its stderr log gone)' 'could not read the log' env CONTRACT_STOP_AT_SETUP=1 bash "$d/nolog/scripts/contract.sh" || ok=0
  planted 'contract.sh (its tee killed)' 'exited 137' env CONTRACT_STOP_AT_SETUP=1 bash "$d/teekill/scripts/contract.sh" || ok=0
  planted 'contract.sh (a real error that begins like a job report)' '12087 Killed: No such file' env CONTRACT_STOP_AT_SETUP=1 bash "$d/jobreal/scripts/contract.sh" || ok=0
  if ! STATIC_TEST_PGID=$(perl -e 'print getpgrp(0)') CONTRACT_STOP_AT_SETUP=1 bash "$d/group/scripts/contract.sh" > "$d/group.out" 2>&1; then
    echo "self-test: the setup-only contract runner left the static check's process group"
    cat "$d/group.out"
    ok=0
  fi
  # The inner run's exit status comes back exactly, not as "non-zero".
  CONTRACT_STOP_AT_SETUP=1 bash "$d/status/scripts/contract.sh" > "$d/status.out" 2>&1
  rc=$?
  if [ "$rc" -ne 7 ] || ! grep -q 'inner-said-seven' "$d/status.out"; then
    echo "self-test: contract.sh did not repeat its inner run's exit 7 (got $rc):"
    tail -5 "$d/status.out"
    ok=0
  fi
  # A job report, in either spelling, is not an error.
  if ! CONTRACT_STOP_AT_SETUP=1 bash "$d/jobok/scripts/contract.sh" > "$d/jobok.out" 2>&1 || ! grep -q 'stops at the end of its setup' "$d/jobok.out"; then
    echo "self-test: contract.sh failed on job reports a row killed a child for:"
    tail -5 "$d/jobok.out"
    ok=0
  fi
  # A NUL a substitution drops is reported by bash 4.4 and later, which fails the
  # run, and by no earlier bash; stripped explicitly it is silent in every one.
  if bash -c '[ "${BASH_VERSINFO[0]}" -gt 4 ] || { [ "${BASH_VERSINFO[0]}" -eq 4 ] && [ "${BASH_VERSINFO[1]}" -ge 4 ]; }'; then
    planted 'contract.sh (a dropped NUL byte)' 'ignored null byte' env CONTRACT_STOP_AT_SETUP=1 bash "$d/nul/scripts/contract.sh" || ok=0
  elif ! CONTRACT_STOP_AT_SETUP=1 bash "$d/nul/scripts/contract.sh" > "$d/nul.out" 2>&1; then
    echo "self-test: contract.sh failed on a dropped NUL byte under a bash that does not report it:"
    tail -5 "$d/nul.out"
    ok=0
  fi
  if ! CONTRACT_STOP_AT_SETUP=1 bash "$d/nulok/scripts/contract.sh" > "$d/nulok.out" 2>&1; then
    echo "self-test: contract.sh failed on a NUL byte stripped explicitly:"
    tail -5 "$d/nulok.out"
    ok=0
  fi
  # The watcher is the first executable statement: nothing above its guard but comments.
  g=$(grep -n '^if \[ -z "\${CONTRACT_INNER:-}" \]; then$' scripts/contract.sh | cut -d: -f1)
  if [ -z "$g" ] || head -n $((g - 1)) scripts/contract.sh | grep -qvE '^(#|$)'; then
    echo "self-test: contract.sh has code above its watcher guard (line ${g:-none})"
    ok=0
  fi
  if ! CONTRACT_STOP_AT_SETUP=1 bash "$d/clean/scripts/contract.sh" > "$d/clean.out" 2>&1 || ! grep -q 'stops at the end of its setup' "$d/clean.out"; then
    echo "self-test: contract.sh with nothing planted did not run to the end of its setup:"
    tail -5 "$d/clean.out"
    ok=0
  fi
  [ "$ok" -eq 1 ] || exit 1
  echo "self-test: ruff, shellcheck, actionlint, knip and contract.sh each reported their planted defect"
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

# #773: a tool past its deadline is stopped, its whole tree with it, and named; a
# command inside its deadline keeps its own status. Four planted tools against a
# 2 s deadline, each with a sleeper whose argument carries this shell's pid so
# that pgrep finds only it: a plain one, one that ignores TERM (stopped only by
# the watchdog's KILL), a grandchild, and one that EXITS 0 on TERM (exec'd, so the
# status the wrapper sees really is 0; timed out must come from the watchdog and
# not from the command's status). A healthy command 0.64 s long under a 1 s
# deadline must pass whichever second boundary it crosses. The deadlines here are
# literal, so STATIC_DEADLINE_SECS cannot shorten the test.
deadline_self_test() {
  local out rc t0 el c n u=$$ ok=1
  planted_plain() { sleep "30.$u"; }
  planted_term() { sh -c 'trap "" TERM; exec sleep 31.'"$u"; :; }
  planted_grand() { sh -c 'sleep 32.'"$u"' & wait'; :; }
  planted_exit0() { exec sh -c 'trap "exit 0" TERM; sleep 33.'"$u"; }
  for c in plain:30 term:31 grand:32 exit0:33; do
    n=${c##*:}
    c=${c%%:*}
    t0=$SECONDS
    out=$(deadline "planted $c" 2 "planted_$c" 2>&1)
    rc=$?
    el=$((SECONDS - t0))
    sleep 0.3
    if [ "$rc" -ne 124 ] || ! grep -q "static.sh: planted $c did not finish in 2 s" <<<"$out"; then
      echo "self-test: planted $c was not stopped and named (exit $rc):"
      printf '%s\n' "$out" | tail -5
      ok=0
    fi
    if pgrep -f "sleep $n\\.$u" > /dev/null; then
      echo "self-test: planted $c outlived its deadline: its sleeper is still running"
      pkill -KILL -f "sleep $n\\.$u"
      ok=0
    fi
    if [ "$el" -ge 12 ]; then
      echo "self-test: planted $c took $el s against a 2 s deadline"
      ok=0
    fi
  done
  deadline 'planted quick' 600 sh -c 'exit 3'
  rc=$?
  if [ "$rc" -ne 3 ]; then
    echo "self-test: a command inside its deadline lost its exit status (got $rc, want 3)"
    ok=0
  fi
  for _ in 1 2 3 4 5; do
    deadline 'planted near' 1 sh -c 'sleep 0.64'
    rc=$?
    if [ "$rc" -ne 0 ]; then
      echo "self-test: a healthy command inside its deadline was reported as timed out (exit $rc)"
      ok=0
    fi
    sleep 0.37
  done
  [ "$ok" -eq 1 ] || return 1
  echo "self-test: a tool past its deadline is stopped with its tree and named; one inside it keeps its status"
}
STEP_FIXED=60 step "deadline self-test" deadline_self_test

if [ "$fail" -eq 0 ]; then
  echo "static analysis clean"
else
  echo "static analysis FAILED"
fi
exit "$fail"
