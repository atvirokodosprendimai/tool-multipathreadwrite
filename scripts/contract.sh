#!/usr/bin/env bash
# Run mrw against its own promises, in a throwaway repo, and assert each one.
#
# The README quotes this script's results. It exists because a contract stated
# in prose is a hope: every row here is something mrw claims, and each is
# checked by making it go wrong on purpose.
#
#   ./scripts/contract.sh
#
# Exit 0 only if every row holds. Deliberately NOT a go test: it drives the real
# binary through a real shell, which is how a caller actually meets the tool —
# and exit statuses are half of what is being asserted.
set -uo pipefail

# The runner is a fresh process group of its own, always, so that
# `pgrep -g $$` at the end names every process this run started and nothing
# else: a descendant re-parented away from its spawner keeps its group, so
# #101's two orphans would have been named the day they were made rather
# than fifteen hours later. Being a leader already is not enough — as the
# first command of an interactive pipeline (`./scripts/contract.sh | tee`)
# the script leads a group that `tee` shares, and a group-wide kill takes
# `tee` with it (both reviews of the previous form found this). So the
# original process is a thin wrapper: it forks, the child takes a new group
# and becomes the runner, and the wrapper — still in the caller's group, so
# a terminal Ctrl-C or hangup reaches it — forwards INT, TERM and HUP to the
# runner's group as TERM, waits, and repeats the runner's exit status
# (signal n → 128+n). perl, because macOS has no setsid(1) and the script
# depends on perl already. CONTRACT_GROUP marks the runner so the wrapper is
# entered exactly once; nothing before this line spawns a child. Measured on
# darwin: exit 7 repeated as 7; INT to the wrapper emptied a three-member
# runner group inside a second, and the caller's pipeline peer finished on
# its own with 143 from the runner. The group form was proposed by a peer
# session on 2026-09-05. One consequence: with `stty tostop` set, a write to
# the terminal from this background group stops it — off by default, and
# nothing here reads the terminal.
if [ "${CONTRACT_GROUP:-}" != "$$" ]; then
  exec perl -e '
    my ($pid, $pending);
    # Handlers before the fork, and the group set from BOTH sides: a signal
    # in the window between them is recorded and forwarded once the group
    # exists, instead of killing the wrapper and leaving the runner behind.
    $SIG{INT} = $SIG{TERM} = $SIG{HUP} = sub { $pending = 1; kill "TERM", -$pid if $pid };
    $pid = fork; defined $pid or die "fork: $!";
    if ($pid) {
      setpgrp($pid, $pid);
      kill "TERM", -$pid if $pending;
      my $r; do { $r = waitpid $pid, 0 } while ($r == -1 && $!{EINTR});
      my $s = $?; exit(($s & 127) ? 128 + ($s & 127) : $s >> 8);
    }
    setpgrp(0, 0);
    $SIG{INT} = $SIG{TERM} = $SIG{HUP} = "DEFAULT";
    # A forward that landed before that reset ran the inherited handler with
    # $pid still 0 and was swallowed; replay it here (review, round five).
    kill "TERM", $$ if $pending;
    $ENV{CONTRACT_GROUP} = $$;
    exec @ARGV or die "exec: $!";
  ' "$BASH" "$0" "$@"
fi
# The marker is consumed here so no descendant inherits it: a stale exported
# value equal to a reused pid would otherwise skip the wrapper (review).
# Nothing accidental can match it after this — live pids are unique and this
# is unset before anything is spawned; a caller that forges it to its own
# pid on purpose is defeating the wrapper deliberately, and may.
unset CONTRACT_GROUP

# The repository and this script, captured once and absolutely. Every later
# path used to be rebuilt from "$0" after this cd, and from inside scripts/ that
# named the repository's parent: §30 and §43 failed, and the conflict-marker
# check passed without looking at anything (BACKLOG :76).
SELF=$(cd "$(dirname "$0")" && pwd)/$(basename "$0")
SRC=$(cd "$(dirname "$0")/.." && pwd)
# A relative MRW names a binary relative to where the caller stood, so it is
# made absolute before the cd below moves that (review of #236).
if [ -n "${MRW:-}" ]; then MRW=$(cd "$(dirname "$MRW")" && pwd)/$(basename "$MRW"); fi
cd "$SRC"
WORK=$(mktemp -d)
# Reap the whole group on exit: every child, and every orphan that kept the
# group after its parent died. TERM is ignored FIRST, before the rm, so a
# second forwarded signal cannot cut the trap short — measured unnecessary
# on darwin, where bash carried exit status through it; kept as a guard for
# the Linux runner.
trap 'trap "" TERM; rm -rf "$WORK"; kill -- -$$ 2>/dev/null' EXIT

# ⚠ AND PIN mrw's OWN STATE INTO $WORK. Every `fixture` gets a fresh root, which
# is a fresh state KEY, so an unpinned run converted one temporary directory
# into one PERMANENT one per fixture: measured 2026-09-07, +111 entries per run
# and 22,836 directories (242 MB of disk by du) accumulated on one machine, 22,591 of them
# naming a checkout that no longer exists (ADR-034). Fresh-root isolation is
# unchanged and still the mechanism; this puts the state beside the fixtures
# that caused it, so a run leaves the machine as it found it. Section 71 pins a
# sub-base of its own under $WORK because it COUNTS entries and needs a base
# holding only what it planted.
export XDG_STATE_HOME="$WORK/state"
mkdir -p "$XDG_STATE_HOME"

# And pin TMPDIR into $WORK. mrw's check writes a `mrw-check-*.log` into the
# system temp directory and keeps it whenever the check FAILS or is truncated
# (a passing one removes its own), and many rows here fail a check on purpose —
# so a contract run left 13 behind (reported by the WSL peer on 2026-09-24).
# Everything this run writes to a temp directory now goes with $WORK.
export TMPDIR="$WORK/tmp"
mkdir -p "$TMPDIR"

# jq is required, not optional: 16 rows parse a --json receipt with it, most
# without a guard, so a machine without jq failed those rows as if mrw were
# wrong. Say so once, before anything runs (reported by the WSL peer).
command -v jq >/dev/null 2>&1 || { echo "contract.sh needs jq on PATH (the --json rows parse receipts with it)" >&2; exit 2; }

# Build our OWN binary inside WORK rather than sharing bin/mrw. Two fences ran
# concurrently under `adr-verify --sweep` on 2026-08-31 — one starting with
# `go build -o bin/mrw`, two others executing it — and the binary was rewritten
# under a running process. Both contract.sh-invoking fences failed, and passed
# on a re-run: a flake with a cause. A shared mutable artifact is the cause, so
# it is removed rather than retried. The build is cached, so this is cheap.
MRW=${MRW:-$WORK/mrw}
go build -o "$MRW" ./cmd/mrw
fails=0

ok()   { printf '  PASS  %s\n' "$1"; }
# A skip is honest; a silent pass is not. Used where the trigger is a
# permission bit, which uid 0 ignores — under root the row would go green
# without exercising anything, which is the defect class this file exists for.
skip() { printf '  SKIP  %s\n' "$1"; }
bad()  { printf '  FAIL  %s\n' "$1"; fails=$((fails + 1)); }
# want <expected-exit> <actual-exit> <description>
want() { [ "$1" = "$2" ] && ok "$3" || bad "$3 (exit $2, want $1)"; }
# bounded SECS OUT CMD... runs CMD with its stdout and stderr in OUT and waits
# at most SECS seconds for it. It returns CMD's exit status, or 124 after
# killing a CMD still running, with a line saying so appended to OUT. A row
# that must bound mrw uses this, never `perl -e 'alarm N; exec @ARGV'`: Go
# ignores SIGALRM unless it asks for the signal, so the alarm fired and mrw
# ran on (BACKLOG.md, the ADR-072 hang-guard entry). The alarm still bounds a child that is not Go, such
# as §55's Python hook.
#
# The kill is SIGKILL to CMD alone. A child CMD put in a process group of its
# own — mrw does that to ast-grep and to the check (internal/subproc) — is out
# of this run's group, so neither this kill, the EXIT trap nor the survivors
# check at the end reaches it: a row whose fake child can hang records the
# child's pid and kills it itself (§111, Codex review of #236).
#
# CMD's stdin is /dev/null: bash gives a background job no input when job
# control is off, so a row that pipes a plan into CMD writes it to a file and
# names the file instead (review of #236).
bounded() {
  local secs=$1 out=$2 pid i rc
  shift 2
  "$@" > "$out" 2>&1 & pid=$!
  for ((i = 0; i < secs * 10; i++)); do
    kill -0 "$pid" 2>/dev/null || { wait "$pid"; return; }
    sleep 0.1
  done
  # A CMD that finished during the last interval keeps its own status: the
  # kill is then a no-op on a process already gone, and wait reports how it
  # ended. Only a CMD the kill ended is a timeout (Codex review of #236).
  kill -9 "$pid" 2>/dev/null; wait "$pid" 2>/dev/null; rc=$?
  [ "$rc" -eq 137 ] || return "$rc"
  echo "bounded: still running after ${secs}s, killed" >> "$out"
  return 124
}

# Each fixture is its OWN checkout at its own path, not a rebuild at a shared
# one. mrw keys per-checkout state on the absolute root, so reusing one path
# carried a ledger from the previous row into the next — which broke row 6 the
# moment state moved out of the tree (ADR-004). A shared path between
# independent cases was always wrong; it only became visible here.
fixture() {
  R=$(mktemp -d "$WORK/r-XXXXXX")
  printf 'module demo\n\ngo 1.26\n'                                  > "$R/go.mod"
  printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
  printf 'package demo\n\nfunc D() int { return 4 }\n'               > "$R/b.go"
  printf 'package demo\n\nimport "testing"\n\nfunc TestAll(t *testing.T) {\n\tif A()+B()+C()+D() != 10 {\n\t\tt.Fatal("bad")\n\t}\n}\n' > "$R/a_test.go"
  # A REAL read, not --stat: since the ledger records what was actually
  # SERVED, a stat prints no content and licenses no edit.
  "$MRW" -C "$R" read a.go b.go a_test.go >"$WORK/served.out"
}
m() { "$MRW" -C "$R" "$@"; }

echo "mrw contract — $(git rev-parse --short HEAD)$(git diff --quiet || echo ' (dirty tree)')"

# 1. One bad hunk of three aborts everything, and says which.
fixture
out=$(printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 10 }\n@@ a.go 4 replace anchor="NOPE"\nx\n@@ b.go 3 replace anchor="func D"\nfunc D() int { return 40 }\n' | m write - 2>&1)
rc=$?
want 1 "$rc" "3 hunks, 1 bad anchor -> exit 1"
grep -q 'FAIL a.go 4' <<<"$out" && ok "the offender is named" || bad "the offender is not named"
grep -q '^skip'       <<<"$out" && ok "siblings report skip, never ok" || bad "siblings did not report skip"
grep -q 'return 1 }' "$R/a.go" && ok "nothing was written" || bad "a file was written"

# 2. A valid multi-file plan applies, and --check runs the project's own tests.
fixture
printf 'package demo\n\nimport "testing"\n\nfunc TestAll(t *testing.T) {\n\tif A()+B()+C()+D() != 82 {\n\t\tt.Fatal("bad")\n\t}\n}\n' > "$R/a_test.go"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 10 }\n@@ a.go 5 replace anchor="func C"\nfunc C() int { return 30 }\n@@ b.go 3 replace anchor="func D"\nfunc D() int { return 40 }\n' > "$R/ok.mrw"
out=$(m write --check "$R/ok.mrw" 2>&1); rc=$?
want 0 "$rc" "3 hunks / 2 files valid + --check -> exit 0"
grep -q 'check PASS' <<<"$out" && ok "the scoped check ran and passed" || bad "no passing check in the output"

# 3. A good write followed by a red suite: the write STAYS, and says so.
fixture
printf 'package demo\n\nimport "testing"\n\nfunc TestAll(t *testing.T) {\n\tt.Fatal("deliberately red")\n}\n' > "$R/a_test.go"
out=$(printf '@@ b.go 3 replace anchor="func D"\nfunc D() int { return 41 }\n' | m write --check - 2>&1); rc=$?
want 3 "$rc" "good write + red test -> exit 3"
grep -q 'return 41' "$R/b.go" && ok "the write was kept, not reverted" || bad "the write was reverted"
grep -q 'deliberately red' <<<"$out" && ok "the failing output is shown" || bad "the failure was not shown"

# 4. Pointers resolve, and an out-of-range one ERRORS rather than resolving to
#    nothing — an empty result is how a batch silently does less than it was
#    asked to.
fixture
m iter add a.go b.go a_test.go >/dev/null
m read @1:1-2 >"$WORK/served.out"; want 0 "$?" "@1:1-2 resolves"
m read @3     >"$WORK/served.out"; want 0 "$?" "@3 resolves"
out=$(m read @9 2>&1); rc=$?
want 2 "$rc" "@9 (out of range) errors"
grep -q '3 entr' <<<"$out" && ok "the error says how many entries exist" || bad "the error is unhelpful"

# 5. The adversarial one: a check whose OUTPUT says PASS while the process
#    exits 1. Believing the text is what a tail in the pipeline would do.
fixture
printf '{"check":"echo PASS; echo \\"ok demo 0.1s\\"; exit 1"}' > "$R/.quality-harness.json"
out=$(m check --full 2>&1); rc=$?
want 3 "$rc" "output says PASS, process exits 1 -> reported FAIL"
grep -q 'check FAIL' <<<"$out" && ok "the process is believed, not the text" || bad "the printed word was believed"

# 6. Read before modify: an unseen file, and one changed behind mrw's back.
# A checkout mrw has genuinely never looked at — NOT fixture(), which reads its
# files as its last step. Deleting a ledger file used to be enough when state
# lived in the tree; since ADR-004 the honest way to have an unseen file is to
# not read it.
R=$(mktemp -d "$WORK/unseen-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\n' > "$R/a.go"
out=$(printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "editing a file mrw has never read -> refused"
grep -q 'has not been read' <<<"$out" && ok "the reason names the cause" || bad "unclear reason"
m read a.go >"$WORK/served.out"
printf 'package demo\n\nfunc A() int { return 99 }\n' > "$R/a.go"   # changed elsewhere
out=$(printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "editing a file changed behind mrw's back -> refused"
grep -q 'changed since' <<<"$out" && ok "the reason names the staleness" || bad "unclear reason"

# 7. The tree stays clean. The reported bug was that `.mrw/` appeared silently
#    in whatever repository you ran mrw in, ignored by nothing, and got
#    committed. Asserted as a property — the root holds only what was put there
#    — so it keeps holding when mrw learns to store something new.
fixture
before=$(ls -A "$R" | sort | tr '\n' ' ')
m read --stat a.go >"$WORK/served.out"
m iter add a.go >/dev/null
after=$(ls -A "$R" | sort | tr '\n' ' ')
[ "$before" = "$after" ] && ok "read and iter leave the working tree untouched" \
  || bad "the working tree changed: '$before' -> '$after'"
[ -d "$R/.mrw" ] && bad "mrw created .mrw/ in the working tree" \
  || ok "no .mrw/ in the working tree"

# 8. Guards are checked on every op, and a body= count is honoured. Each row
#    here is a README claim that used to be false in silence: anchor= and
#    lines= were consulted only on replace/delete, and body= was never counted.
fixture
out=$(printf '@@ a.go 3 insert-after anchor="NOPE"\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a false anchor on an insertion -> refused"
out=$(printf '@@ a.go 3 insert-after lines=9\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "lines=9 on an insertion -> refused"
out=$(printf '@@ a.go 3 replace body=5\nx\n' | m write - 2>&1); rc=$?
want 2 "$rc" "body= asking for more lines than the plan holds -> parse error"
printf 'package demo\n\nvar S = "muted"\n' > "$R/q.go"
m read q.go >"$WORK/served.out"
out=$(printf '@@ q.go 3 replace anchor="= \\"muted\\""\nvar S = "MUTED"\n' | m write - 2>&1); rc=$?
want 0 "$rc" "an anchor may contain an escaped quote"

# 9. The ledger records what the caller was SHOWN, and the root is a boundary.
#    Both were silent before: a --stat read licensed an edit to a file whose
#    content had never been printed, and a ../ path walked straight out of the
#    directory mrw was pointed at.
R=$(mktemp -d "$WORK/shown-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\n' > "$R/a.go"
m read --stat a.go >"$WORK/served.out"
out=$(printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "--stat prints no content, so it licenses no edit"
grep -q 'has not been read' <<<"$out" && ok "the reason names what was not shown" || bad "unclear reason"
m read a.go:1-1 >"$WORK/served.out"
out=$(printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "reading line 1 does not license an edit to line 3"
# THE POSITIVE HALF, and it is the remedy ADR-002's risk table now names.
# Before ADR-005 the cheap way back from a refusal was `--stat`; a stat licenses
# nothing now, so the record would leave a caller with no remedy at all unless
# the RANGED re-read is asserted to work. One call, only the lines the edit
# addresses. Without this row the two above pin only that things are REFUSED,
# which a build that refused everything would also satisfy.
#
# Its OWN checkout, because unlike its neighbours this row's write SUCCEEDS —
# run in the shared one it left a.go modified and broke a later row that reads
# the whole file. The file says every fixture is its own checkout for exactly
# this reason and the first draft of this row ignored it.
# R is SAVED and RESTORED, not just reassigned: the rows after this one belong
# to section 9's own checkout, and leaving them in this one carried a written
# a.go into them — the second way the same shared-state trap bit this row.
PREV_R=$R
R=$(mktemp -d "$WORK/remedy-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\n' > "$R/a.go"
m read a.go:3 >"$WORK/served.out"
printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "but re-reading line 3 licenses line 3 — one ranged call is the remedy"
R=$PREV_R
m read a.go >"$WORK/served.out"
out=$(printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 2 }\n' | m write - 2>&1); rc=$?
want 0 "$rc" "reading the whole file licenses the whole file"
out=$(printf '@@ ../escaped.txt - create\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a hunk that leaves the root -> refused"
[ -e "$(dirname "$R")/escaped.txt" ] && bad "mrw wrote outside the root" \
  || ok "nothing was written outside the root"

# 10. --max-lines withholds, and what is withheld is not observed. This is the
#     same gap as --stat wearing a different hat, and it is the one a caller
#     reaches for on a big file — exactly when they cannot count what they were
#     not shown.
R=$(mktemp -d "$WORK/withheld-XXXXXX")
: > "$R/big.txt"
for i in $(seq 1 40); do echo "line $i" >> "$R/big.txt"; done
m read --max-lines 5 big.txt >"$WORK/served.out" 2>&1
out=$(printf '@@ big.txt 40 replace\nREWRITTEN\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a truncated read does not license an edit to a withheld line"
grep -q 'has not been read' <<<"$out" && ok "the reason names the unseen lines" || bad "unclear reason"
m read big.txt >"$WORK/served.out"
out=$(printf '@@ big.txt 40 replace\nREWRITTEN\n' | m write - 2>&1); rc=$?
want 0 "$rc" "reading it whole afterwards licenses the edit"

# 11. body= may contain a real header when the plan says raw=true, and may not
#     when it does not.
fixture
out=$(printf '@@ a.go 3 replace body=1\n@@ b.go 3 replace\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a valid header inside a counted body -> parse error"
out=$(printf '@@ a.go 3 replace body=1 raw=true\n@@ b.go 3 replace\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "raw=true writes that header as content"
grep -q '@@ b.go 3 replace' "$R/a.go" && ok "the header landed as text" || bad "the body was not written"

# 12. A plan file is a SHELL argument: -C moves the paths INSIDE it, not the
#     file itself. The split is defensible and invisible, so the refusal has to
#     say where it looked — and the framework must not swallow the error, which
#     it did until the exit handler was unwired (main's "mrw:" prefix was dead
#     code and nothing printed it).
fixture
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 7 }\n' > "$R/plan.mrw"
out=$(cd "$WORK" && "$MRW" -C "$R" write --dry-run plan.mrw 2>&1); rc=$?
want 2 "$rc" "a plan file is not looked for under --root"
grep -q 'working directory' <<<"$out" && ok "the refusal says where it looked" || bad "bare error: $out"
grep -q '^mrw: ' <<<"$out" && ok "the error carries mrw's own prefix" || bad "no mrw: prefix: $out"
out=$(cd "$R" && "$MRW" -C "$R" write --dry-run plan.mrw 2>&1); rc=$?
want 0 "$rc" "the same plan, run from beside it, applies"

# 13. The root confines READS as well as writes, and a replace with no body is
#     refused rather than deleting the line it addresses. Both were reported
#     from outside: `read ../outside.txt` served the file at exit 0 and a
#     symlink out of the tree was followed, while the identical path in a plan
#     was refused; and an empty-bodied replace deleted a line while the receipt
#     said ok.
fixture
printf 'SECRET\n' > "$WORK/outside.txt"
ln -sf /etc/hosts "$R/hosts.link"
out=$(m read ../outside.txt 2>&1); rc=$?
want 1 "$rc" "reading outside the root -> refused"
grep -q 'SECRET' <<<"$out" && bad "the file outside the root was printed" \
  || ok "nothing outside the root was printed"
out=$(m read hosts.link 2>&1); rc=$?
want 1 "$rc" "a symlink out of the root -> refused"
out=$(m read a.go 2>&1); rc=$?
want 0 "$rc" "an ordinary read still works"
out=$(printf '@@ a.go 3 replace\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a replace with no body -> parse error, not a deletion"
grep -q 'say delete' <<<"$out" && ok "the error names the op that means it" || bad "unclear reason: $out"
grep -q 'func A' "$R/a.go" && ok "the line is still there" || bad "the line was deleted"

# 14. The README lists four reasons `read` exits 1 and names the word each one
#     prints. That table is only useful if the words are the binary's, so each
#     row is driven here — this is the same drift that let the README describe
#     anchor= escaping backwards for a whole PR.
fixture
printf 'SECRET\n' > "$(dirname "$R")/outside.txt"
: > "$R/big.txt"; for i in $(seq 1 40); do echo "line $i" >> "$R/big.txt"; done
out=$(m read nosuch.txt 2>&1); rc=$?
want 1 "$rc" "an unreadable file -> exit 1"
grep -q 'UNREADABLE' <<<"$out" && ok "and prints UNREADABLE" || bad "wrong word: $out"
out=$(m read ../outside.txt 2>&1); rc=$?
want 1 "$rc" "a path outside the root -> exit 1"
grep -q 'REFUSED' <<<"$out" && ok "and prints REFUSED" || bad "wrong word: $out"
out=$(m read 'big.txt:/nomatch/' 2>&1); rc=$?
want 1 "$rc" "a pattern that matches nothing -> exit 1"
grep -q 'no match for' <<<"$out" && ok "and prints no match for" || bad "wrong word: $out"
out=$(m read --max-lines 5 big.txt 2>&1); rc=$?
want 1 "$rc" "a --max-lines cap -> exit 1"
grep -q 'withheld' <<<"$out" && ok "and says what it withheld" || bad "wrong word: $out"
# The row names two spellings and the case above reaches only the lowercase one:
# a whole-file cap cuts a span in half, while WITHHELD needs a span dropped
# entirely, which needs several ranges in one spec.
out=$(m read --max-lines 2 'big.txt:1-2,10-12,30-32' 2>&1); rc=$?
want 1 "$rc" "a span withheld whole -> exit 1"
grep -q 'WITHHELD' <<<"$out" && ok "and prints WITHHELD" || bad "wrong word: $out"

# 15. `mrw check <dir>` scopes to that package AND EVERYTHING UNDER IT. The
#     README spells the scoped form with a directory — `mrw check
#     internal/apply` — and it ran the WHOLE-project command instead, because a
#     directory has no .go extension. Fixing that with `./dir` alone bought a
#     second silent failure: go's `./dir` is the package at the top, so
#     `mrw check .` reported PASS with a failing package one level down. A path
#     mrw cannot place as a package — a directory of prose, one named
#     testdata — must still fall back rather than scope to nothing. A path
#     that is NOT THERE is refused (ADR-042 / §86): the fallback used to PASS
#     the whole project, so a typo read as green. A path OUTSIDE the root is
#     refused too: the fallback covers a present unplaceable path, not a name
#     that pointed elsewhere. These rows used to assert that fallback, under
#     the name "as read and write refuse it".
fixture
mkdir -p "$R/internal/apply/testdata" "$R/docs"
printf 'package apply\n\nfunc A() int { return 1 }\n' > "$R/internal/apply/a.go"
printf 'package testdata\n'                          > "$R/internal/apply/testdata/t.go"
printf '# prose\n'                                   > "$R/docs/guide.md"
# The directory outside the root holds a REAL package: pointed at /etc these two
# rows passed with the boundary check removed, because /etc has no Go files in
# it. A row that cannot fail asserts nothing.
mkdir -p "$WORK/outside"
printf 'package outside\n' > "$WORK/outside/o.go"
ln -s "$WORK/outside" "$R/link"
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check internal/apply 2>&1); rc=$?
want 0 "$rc" "check on a directory runs"
grep -qF 'SCOPED ./internal/apply/...' <<<"$out" && ok "and scopes to that package and its subtree" || bad "not scoped recursively: $out"
out=$(m check ./internal/apply 2>&1)
grep -qF 'SCOPED ./internal/apply/...' <<<"$out" && ok "the ./dir form scopes too" || bad "not scoped: $out"
out=$(m check ./internal/apply/... 2>&1)
grep -qF 'SCOPED ./internal/apply/...' <<<"$out" && ok "the ./dir/... form we print round-trips" || bad "no round trip: $out"
out=$(m check internal/apply/a.go 2>&1)
grep -qF 'SCOPED ./internal/apply' <<<"$out" && ok "a .go file still scopes to its own package" || bad "not scoped: $out"
out=$(m check . 2>&1)
grep -qF 'SCOPED ./...' <<<"$out" && ok "the root scopes to every package, not just the top one" || bad "not recursive: $out"
out=$(m check internal/aply 2>&1); rc=$?
want 2 "$rc" "a mistyped path is refused, not a silent whole-project PASS"
grep -q 'FULL' <<<"$out" && bad "fell back and answered about the root: $out" \
  || ok "and nothing ran under the mistyped path"
out=$(m check docs 2>&1)
grep -q 'echo FULL' <<<"$out" && ok "a directory with no package falls back" || bad "scoped to a non-package: $out"
out=$(m check internal/apply/testdata 2>&1)
grep -q 'echo FULL' <<<"$out" && ok "testdata holds nothing the ... form matches, so it falls back" || bad "scoped to testdata: $out"
# The root's own check is `echo FULL`, which exits 0. So a row asserting only a
# non-zero exit would not distinguish a refusal from anything; what says the
# check never ran is that FULL was not printed. Both halves are asserted.
for spelling in ../outside "$WORK/outside" link; do
  out=$(m check "$spelling" 2>&1); rc=$?
  want 2 "$rc" "a scope outside the root ($spelling) is refused, as read and write refuse one"
  grep -q 'FULL' <<<"$out" && bad "fell back and answered about the root: $out" \
    || ok "and nothing ran under it ($spelling)"
done
# The discriminator. The fallback answered about the ROOT, so its verdict moved
# with the root's own tests and never with the argument: point the root's check
# at a failure and the old code reported 3 — the right shape of wrongness for a
# row that only checks "not zero" to sail through.
printf '{"check":"echo FULL; exit 1"}\n' > "$R/.quality-harness.json"
m check ../outside >/dev/null 2>&1; rc=$?
want 2 "$rc" "a refused scope does not inherit the root's failing verdict"
m check . >/dev/null 2>&1; rc=$?
want 3 "$rc" "while the root itself still reports its own failure"
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
# An ABSOLUTE path outside the root is the same escape in the spelling that
# hides it: it is JOINED onto the root rather than honoured (deliberate, and
# tested in internal/rooted), so it lands inside, places no package and fell
# back — read and apply survive that because the joined path then fails to
# exist and they say so, and a check has no such tell.
m check /etc >/dev/null 2>&1; rc=$?
want 2 "$rc" "an absolute path outside the root is refused, not silently re-rooted"
# The machine-readable surface is the one a refusal must not leak a shape into:
# a consumer reading exit_code out of a document has no way to tell a verdict
# from a refusal, and unlike a human it never sees the message on stderr.
out=$(m check --json ../outside 2>/dev/null); rc=$?
want 2 "$rc" "--json refuses the same scope"
grep -q 'exit_code' <<<"$out" && bad "a refusal emitted a result document: $out" \
  || ok "and emits no result document to read a verdict out of"
out=$(m check --full 2>&1)
grep -q 'echo FULL' <<<"$out" && ok "--full still ignores every scope" || bad "not full: $out"

# 15c. A .go path that is not there is refused (ADR-042), like the directory
#      typo it is. Placing `filepath.Dir` without asking whether the FILE
#      exists let a mistyped name at a module root scope to `.`, run the root
#      package and report PASS — then the fallback hid the same miss behind
#      a whole-project PASS. Both are closed.
fixture
mkdir -p "$R/pkg"
printf 'package pkg\n' > "$R/pkg/p.go"
# The fixture root already holds a package, which is what makes these rows able
# to fail: without the existence check a phantom .go name places `.` and scopes
# to the root package, so the run is green and covers nothing the caller named.
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check chek.go 2>&1); rc=$?
want 2 "$rc" "a mistyped .go file at the root is refused"
grep -q 'FULL' <<<"$out" && bad "fell back on a phantom .go: $out" \
  || ok "and nothing ran under it"
out=$(m check nosuchdir/nope.go 2>&1); rc=$?
want 2 "$rc" "a .go file in a missing directory is refused"
grep -q 'FULL' <<<"$out" && bad "fell back on a phantom dir: $out" \
  || ok "and nothing ran under the missing dir"
out=$(m check a.go 2>&1)
grep -qF 'SCOPED .' <<<"$out" && ok "a .go file that IS there still scopes" || bad "did not scope a real file: $out"
out=$(m check pkg/p.go 2>&1)
grep -qF 'SCOPED ./pkg' <<<"$out" && ok "and so does one in a subpackage" || bad "did not scope a real subpackage file: $out"

# 15b. The row the ./dir form could not fail: a REAL check, on a tree whose
#      failing package is one level below the one the scope names. With
#      `go test .` this exits 0 and reports PASS.
fixture
mkdir -p "$R/sub"
printf 'package sub\n'                                                                      > "$R/sub/s.go"
printf 'package sub\n\nimport "testing"\n\nfunc TestBroken(t *testing.T) {\n\tt.Fatal("BROKEN")\n}\n' > "$R/sub/s_test.go"
printf '{"check":"echo NOT-SCOPED","scoped_check":"go test {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check . 2>&1); rc=$?
want 3 "$rc" "check . fails on a broken package one level down"
grep -q 'TestBroken' <<<"$out" && ok "and the failure it reports is that package's" || bad "did not reach it: $out"

# 15d. A derived path reaches `sh -c`, so an unquoted one is shell SYNTAX. A
#      directory named `pkg; true #` turned `go test {packages}` into
#      `go test ./pkg; true #`: the package was never tested, sh exited 0, and
#      mrw reported PASS. Same silent pass as section 15, one layer down.
fixture
mkdir -p "$R/pkg; true #" "$R/two words" "$R/plain"
printf 'package x\n' > "$R/pkg; true #/x.go"
printf 'package y\n' > "$R/two words/y.go"
printf 'package plain\n' > "$R/plain/p.go"
# printf one line PER ARGUMENT, so the output says how many arguments the shell
# saw and what each held. An `echo` row could not fail: mrw prints the command
# it built above the output, and unquoted that line holds the very characters
# the row was grepping for.
printf '{"check":"echo FULL","scoped_check":"printf A:%%s:Z {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check 'pkg; true #/x.go' 2>&1)
grep -qF 'A:./pkg; true #:Z' <<<"$out" && ok "a semicolon in a path is one argument, not a second command" || bad "the shell ate the path: $out"
out=$(m check 'two words/y.go' 2>&1)
grep -qF 'A:./two words:Z' <<<"$out" && ok "a space in a path is one argument, not two scopes" || bad "split on the space: $out"
# The negative control. An ordinary scope must reach the shell exactly as it
# always has, or these rows would pass while quoting was applied to everything.
out=$(m check plain/p.go 2>&1)
grep -qF 'A:./plain:Z' <<<"$out" && ! grep -qF "'" <<<"$out" && ok "an ordinary scope is still unquoted" || bad "quoted a safe path: $out"

# 15e. The verdict, not the string: a REAL check whose only package is behind
#      the injected name. Unquoted this exits 0 with nothing tested.
fixture
mkdir -p "$R/pkg; true #"
printf 'package x\n\nfunc F() { undefinedSymbol() }\n' > "$R/pkg; true #/x.go"
printf '{"check":"echo NOT-SCOPED","scoped_check":"go test {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check 'pkg; true #/x.go' 2>&1); rc=$?
want 3 "$rc" "a scope naming a broken package behind a semicolon still fails"
grep -qF 'FAIL	demo/pkg; true #' <<<"$out" && ok "and the failure it reports is that package's" || bad "did not reach it: $out"

# 16. ADR-008: a bodyless delete consumes a range while asserting nothing about
#     it, so the receipt is where a wrong range first becomes visible. This is
#     the incident that produced the record, reproduced: a range one line too
#     long takes the closing brace of the function above, and the old receipt
#     said `-4 +0  ok`.
fixture
printf 'package demo\n\nfunc E() int {\n\treturn 5\n}\n\nvar _ = 1\nvar _ = 2\n' > "$R/c.go"
m read c.go >"$WORK/served.out"
out=$(printf '@@ c.go 5-8 delete\n' | m write --dry-run - 2>&1)
rc=$?
want 0 "$rc" "the ADR-008 delete applies"
grep -qF 'from "}" to "var _ = 2"' <<<"$out" \
  && ok "a delete receipt names the first and last line it removed" \
  || bad "the delete receipt carries no bounds: $out"
out=$(printf '@@ c.go 3 replace\nfunc E() int { return 5 }\n' | m write --dry-run --json - 2>&1)
grep -qF 'removed_first' <<<"$out" \
  && bad "a replace reported removed_first: $out" \
  || ok "and no other op reports them"
out=$(printf '@@ c.go 5-8 delete\n' | m write --json - 2>&1)
grep -qF '"removed_first": "}"' <<<"$out" && grep -qF '"removed_last": "var _ = 2"' <<<"$out" \
  && ok "--json carries removed_first and removed_last" \
  || bad "--json is missing the bounds: $out"

# 17. ADR-008: a delete may carry the lines the caller EXPECTS to remove. A
#     match applies; a mismatch refuses the whole plan and names the line.
fixture
printf 'package demo\n\nfunc E() int {\n\treturn 5\n}\n\nvar _ = 1\nvar _ = 2\n' > "$R/c.go"
m read c.go >"$WORK/served.out"
out=$(printf '@@ c.go 7-8 delete\nvar _ = 1\nvar _ = 2\n' | m write --dry-run - 2>&1)
rc=$?
want 0 "$rc" "a delete whose expected removal matches applies"
out=$(printf '@@ c.go 7-8 delete\nvar _ = 1\nvar _ = 3\n' | m write - 2>&1)
rc=$?
want 1 "$rc" "a delete whose expected removal differs is refused"
grep -q 'expected removal differs at line 8' <<<"$out" \
  && ok "and the refusal names the line that differed" \
  || bad "the refusal does not name the line: $out"
grep -qF 'var _ = 2' <<<"$out" && grep -qF 'var _ = 3' <<<"$out" \
  && ok "printing what the plan said beside what the file holds" \
  || bad "the refusal shows only one side: $out"
grep -qF 'var _ = 1' "$R/c.go" \
  && ok "and nothing was written" \
  || bad "the file was modified despite the refusal"

# 18. ADR-008, found by probing the built binary rather than by a test: a
#     whitespace-only mismatch must not be reported as two identical strings,
#     and a delete of BLANK lines still reports its bounds in --json.
fixture
printf 'alpha\n\tindented\nomega\n\n\ndone\n' > "$R/w.txt"
m read w.txt >"$WORK/served.out"
out=$(printf '@@ w.txt 2 delete\n    indented\n' | m write - 2>&1)
rc=$?
want 1 "$rc" "a tab-against-spaces expected removal is refused"
grep -qF '\tindented' <<<"$out" \
  && ok "and the refusal shows the tab instead of trimming it away" \
  || bad "the whitespace difference was trimmed out of the message: $out"
out=$(printf '@@ w.txt 4-5 delete\n' | m write --dry-run --json - 2>&1)
grep -qF '"removed_first": ""' <<<"$out" && grep -qF '"removed_last": ""' <<<"$out" \
  && ok "a delete of blank lines still carries its bounds in --json" \
  || bad "--json dropped the bounds for a blank-line delete: $out"

# 19. Found by probing the built binary with wrong input rather than by a test.
#     A typo'd subcommand used to exit 3 — the status this tool documents as
#     "a check ran and did not pass" — because the framework fell through to
#     its help machinery. A hook branching on 3 would read a typo as a landed
#     write with a red suite.
fixture
out=$(m frobnicate 2>&1); rc=$?
want 2 "$rc" "an unknown subcommand is a usage error, not a failed check"
grep -q 'unknown command "frobnicate"' <<<"$out" && ok "and it names what was typed" || bad "does not name the command: $out"
grep -q 'write' <<<"$out" && ok "and lists the real ones" || bad "does not list the alternatives: $out"

# 19b. --check under --dry-run cannot verify anything, and silently dropping it
#      returned exit 0 to a caller who believed their preview had been checked.
#      ADR-003 rule 2 already decides what that is worth: a check that did not
#      run is not a pass, and its exit table files a missing check under 2.
printf '{"check":"echo THE-CHECK-RAN"}\n' > "$R/.quality-harness.json"
out=$(printf '@@ a.go 3 delete\n' | m write --dry-run --check - 2>&1)
rc=$?
want 2 "$rc" "--check under --dry-run is refused, not silently dropped"
grep -q 'cannot run under --dry-run' <<<"$out" \
  && ok "and the refusal says why" \
  || bad "unclear refusal: $out"
grep -q 'THE-CHECK-RAN' <<<"$out" && bad "the check ran against an unwritten tree" || ok "and no check was run"
#      The PRECEDENCE, not just the refusal: a usage error must preempt every
#      kind of plan error, or it preempts inconsistently. With this test below
#      the parse, an unparseable plan beat the flag pair while a plan whose
#      HUNK failed lost to it — so the caller was told about their flags and
#      never learned their address was out of range, and exit 1 (which promises
#      an untouched tree) became exit 2.
printf '@@ a.go 99 delete\n'        > "$R/fail.mrw"
printf '@@ a.go notanaddr delete\n' > "$R/parse.mrw"
out=$(m write --dry-run "$R/fail.mrw" 2>&1); rc=$?
want 1 "$rc" "a failing hunk alone is exit 1, and says which address"
grep -q 'out of range' <<<"$out" && ok "the caller learns the real problem" || bad "no diagnosis: $out"
m write --dry-run --check "$R/fail.mrw"  >/dev/null 2>&1; want 2 "$?" "the flag pair preempts a failing hunk"
m write --dry-run --check "$R/parse.mrw" >/dev/null 2>&1; want 2 "$?" "and preempts a parse error the same way"
m write --dry-run --check "$R/nope.mrw"  >/dev/null 2>&1; want 2 "$?" "and preempts a missing plan file"

# 20. Probing round two. A check command that is only WHITESPACE read as
#     declared, ran as an empty shell command, exited 0 and reported PASS —
#     a check that did not run reporting a pass, which is what ADR-003 rule 2
#     refuses, and it would hold for as long as the typo did.
#     The decisive fixture has NO go.mod, so there is nothing to infer either:
#     the only way to report PASS would be to run the empty command.
R=$(mktemp -d "$WORK/nogomod-XXXXXX")
printf 'hello\n' > "$R/a.txt"
printf '{"check":"   "}\n' > "$R/.quality-harness.json"
out=$(m check --full 2>&1); rc=$?
want 2 "$rc" "a whitespace-only check with nothing to infer exits 2, not 0"
grep -q 'check PASS' <<<"$out" && bad "a whitespace-only check reported PASS: $out" \
  || ok "a whitespace-only check cannot report PASS"
grep -q 'no check declared' <<<"$out" && ok "it is reported as no check at all" || bad "not skipped: $out"
#     Same for scoped_check, which nothing else covers: untrimmed it reads as
#     declared and suppresses the fallback.
printf '{"scoped_check":"   "}\n' > "$R/.quality-harness.json"
out=$(m check --full 2>&1); rc=$?
want 2 "$rc" "a whitespace-only scoped_check exits 2 too"
grep -q 'check PASS' <<<"$out" && bad "a whitespace-only scoped_check reported PASS: $out" \
  || ok "and cannot report PASS"
#     And in a Go tree it falls back the way an EMPTY value already did.
fixture
printf '{"check":"   "}\n' > "$R/.quality-harness.json"
out=$(m check --full 2>&1); rc=$?
want 0 "$rc" "in a Go tree the fallback runs and passes"
grep -q 'inferred' <<<"$out" && ok "in a Go tree it falls back to the inferred check" || bad "did not fall back: $out"
grep -q 'declared' <<<"$out" && bad "it still reads as declared: $out" \
  || ok "and never reads as declared"
printf '{"check":"  echo REAL; exit 1  "}\n' > "$R/.quality-harness.json"
out=$(m check --full 2>&1); rc=$?
want 3 "$rc" "a padded REAL check still runs and still fails"
grep -q 'REAL' <<<"$out" && ok "and the padding did not eat the command" || bad "command lost: $out"

# 20b. A negative -C printed `@@ 5-3` — an address mrw's own parser refuses —
#      with no content, at exit 0. The README promises the header is exactly
#      the address a write plan takes, and a silently empty result is the
#      failure this tool exists to refuse. A negative --max-lines was ignored.
fixture
out=$(m read -C -1 "a.go:/func A/" 2>&1); rc=$?
want 2 "$rc" "a negative -C is refused"
grep -qE '@@ [0-9]+-[0-9]+' <<<"$out" && bad "it still emitted a range header: $out" \
  || ok "and emits no address at all"
out=$(m read --max-lines -5 a.go 2>&1); rc=$?
want 2 "$rc" "a negative --max-lines is refused"
out=$(m read -C 1 "a.go:/func B/" 2>&1)
grep -qE '@@ 3-5' <<<"$out" && ok "a positive -C still widens the range" || bad "-C broke: $out"
#      PRECEDENCE, not just the value: a usage error must preempt every other
#      kind of bad input, or which error the caller sees depends on which OTHER
#      mistake they made. Placed after the parse, these reported the range and
#      the pointer instead.
out=$(m read -C -1 "a.go:" 2>&1); rc=$?
want 2 "$rc" "a negative -C preempts a bad range spec"
grep -q 'context cannot be negative' <<<"$out" && ok "and the flag is what it names" || bad "named the range: $out"
out=$(m read --max-lines -5 @999 2>&1); rc=$?
want 2 "$rc" "a negative --max-lines preempts a bad pointer"
grep -q 'cap cannot be negative' <<<"$out" && ok "and the flag is what it names" || bad "named the pointer: $out"
out=$(m read "a.go:" 2>&1); rc=$?
want 2 "$rc" "and a bad range alone is still reported as itself"
grep -q 'empty range' <<<"$out" && ok "with its own diagnosis intact" || bad "lost the range diagnosis: $out"

echo

# 21. A plan aborts with the tree UNTOUCHED when a file cannot be written.
#     The write phase used to write each file and move on, so a file that could
#     not be written left the EARLIER ones already renamed into place: a
#     partially applied plan, no receipt at all, and — since the ledger is
#     recorded from that receipt — mrw then refused the caller's next edit to a
#     file it had modified itself, reporting it as "changed since mrw last saw
#     it". Staging every file before renaming any is what makes ADR-001 rule 2
#     hold against a filesystem failure and not only a validation one.
fixture
mkdir -p "$R/locked"
printf 'package locked\n\nfunc F() int { return 1 }\n' > "$R/locked/f.go"
m read a.go locked/f.go >"$WORK/served.out"
chmod 555 "$R/locked"
# uid 0 ignores the permission bits, so PROVE they bite before asserting on
# them. A row that cannot fail asserts nothing, and under root this one cannot.
if touch "$R/locked/.probe" 2>/dev/null; then
  rm -f "$R/locked/.probe"; chmod 755 "$R/locked"
  skip "an unwritable directory aborts the plan (permission bits not enforced here — running as root?)"
else
  before=$(cat "$R/a.go")
  printf '@@ a.go 3 replace\nfunc A() int { return 99 }\n@@ locked/f.go 3 replace\nfunc F() int { return 99 }\n' \
    | m write - >/dev/null 2>&1; rc=$?
  chmod 755 "$R/locked"
  want 1 "$rc" "a plan naming an unwritable file is refused (exit 1, ADR-132: a permission is the target's)"
  [ "$(cat "$R/a.go")" = "$before" ] \
    && ok "and the file that COULD be written was left untouched" \
    || bad "partially applied: a.go was written while the plan failed"
  [ -z "$(find "$R" -name '.mrw-*')" ] \
    && ok "and no staged temp file was left in the tree (ADR-004)" \
    || bad "staging littered: $(find "$R" -name '.mrw-*')"
  # ADR-001 RULE 3 ON THE FILESYSTEM PATH — OPEN until 2026-09-02. The abort
  # above was always correct; what was missing is the receipt. The run returned
  # the error and rendered nothing, so a caller learned which error occurred and
  # nothing about which hunks it affected or which files the plan addressed.
  # Rule 3 says every hunk carries its own verdict and every addressed file
  # appears, written or not.
  chmod 555 "$R/locked"
  out=$(printf '@@ a.go 3 replace\nfunc A() int { return 99 }\n@@ locked/f.go 3 replace\nfunc F() int { return 99 }\n' \
    | m write - 2>&1)
  jout=$(printf '@@ a.go 3 replace\nfunc A() int { return 99 }\n@@ locked/f.go 3 replace\nfunc F() int { return 99 }\n' \
    | m write --json - 2>/dev/null)
  chmod 755 "$R/locked"
  grep -qE '^FAIL locked/f\.go ' <<<"$out" \
    && ok "a filesystem failure gives the unstageable hunk a FAIL verdict" \
    || bad "no FAIL verdict for the unstageable hunk: $(head -1 <<<"$out")"
  grep -qE '^skip a\.go ' <<<"$out" \
    && ok "and its sibling reports skip, never ok" \
    || bad "the sibling hunk got no skip verdict: $(head -2 <<<"$out" | tr '\n' ' ')"
  # "2 file(s)" IS the rule-3 claim. Asserting only on the FAIL line would pass
  # on a receipt that forgot the sibling file entirely.
  grep -qE '2 hunk\(s\), 2 file\(s\), 1 failed, 0 advisories — NOTHING WRITTEN' <<<"$out" \
    && ok "and the summary names every file the plan addressed" \
    || bad "summary does not name both addressed files: $(grep 'hunk(s)' <<<"$out")"
  # --json used to emit a bare error and no JSON at all. jq -e is the assertion
  # because "looks like JSON" and "parses" are different claims.
  if jq -e . <<<"$jout" >/dev/null 2>&1; then
    ok "and --json emits a receipt that PARSES on a filesystem failure"
    n=$(jq -r '[.hunks[] | select(.status != null)] | length' <<<"$jout")
    [ "$n" = 2 ] && ok "and every hunk in it carries a status" || bad "$n hunk(s) carry a status, want 2"
    n=$(jq -r '[.files[] | select(.written == false)] | length' <<<"$jout")
    [ "$n" = 2 ] && ok "and both addressed files appear, neither written" || bad "$n unwritten file(s) in the JSON receipt, want 2"
  else
    bad "--json emitted nothing parseable on a filesystem failure"
  fi
  # The ledger consequence: a.go is unchanged, so the recorded hash still
  # matches and an ordinary edit to it still applies. When a.go had been
  # written behind the receipt, this refused.
  printf '@@ a.go 3 replace\nfunc A() int { return 5 }\n' | m write --no-check - >/dev/null 2>&1; rc=$?
  want 0 "$rc" "and the ledger still matches the tree, so the next edit applies"

  # Staging a create calls MkdirAll, so an abort that unlinked only the temp
  # files left the DIRECTORY standing — a change to the tree made by a run that
  # wrote nothing, which is what ADR-004 forbids. Both halves are asserted,
  # because a cleanup that ate a pre-existing directory would be no less wrong.
  mkdir -p "$R/already"
  chmod 555 "$R/locked"
  printf '@@ fresh/deep/n.go - create\npackage n\n@@ already/e.go - create\npackage e\n@@ locked/f.go 3 replace\nfunc F() int { return 99 }\n' \
    | m write - >/dev/null 2>&1; rc=$?
  chmod 755 "$R/locked"
  want 1 "$rc" "a create into a new directory is refused with the rest of the plan (exit 1, ADR-132)"
  [ ! -e "$R/fresh" ] \
    && ok "and the directories staging created are taken back" \
    || bad "left behind: $(find "$R/fresh" 2>/dev/null | tr '\n' ' ')"
  [ -d "$R/already" ] \
    && ok "while a directory that predated the run is left alone" \
    || bad "the cleanup removed a pre-existing directory"
fi

# 22. A ranged read that serves NOTHING must license NOTHING. seen.Observation
#     draws the distinction already — a nil span list is "the whole file", an
#     empty one is "hashed, and none of it shown" — but read left it nil when a
#     range printed nothing, so `mrw read f.txt:/nomatch/` served no lines,
#     exited 1, and then licensed a write to a line the caller had never seen.
#     A FAILED read granted strictly more than a successful partial one, which
#     is ADR-005 inverted. Shipped in v0.0.11.
#
#     The WRITE is the assertion here. Both the broken and the fixed binary
#     print the same thing for this read — nothing, plus a `!!` line — so no
#     row grepping the read's output could tell them apart, and `mrw seen` is
#     one more rendering of the same record. Only trying the edit settles it.
#
#     Each case gets its OWN checkout: state is keyed on the absolute root, so
#     a shared path carries the previous case's ledger into the next one. That
#     is not hypothetical — it is how this defect first looked four times worse
#     than it is, with the correct rows appearing broken too.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/led.txt"
m read 'led.txt:/nomatch/' >"$WORK/served.out" 2>&1; rc=$?
want 1 "$rc" "a range that matches nothing is reported, not served"
printf '@@ led.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 1 "$rc" "and licenses NO edit — the line was never shown"
grep -q '^c$' "$R/led.txt" && ok "and the file is untouched" || bad "the write landed: $(sed -n 3p "$R/led.txt")"

fixture
printf 'a\nb\nc\nd\ne\n' > "$R/oob.txt"
m read 'oob.txt:99' >"$WORK/served.out" 2>&1; rc=$?
want 1 "$rc" "a line past the end is reported the same way"
printf '@@ oob.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 1 "$rc" "and licenses no edit either"

# CONTROL A: a whole-file read still licenses the whole file. Breaking this
# trades a permissive bug for a restrictive one, and a guard that refuses
# ordinary work is a guard people turn off with --force.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/whole.txt"
m read whole.txt >"$WORK/served.out"
printf '@@ whole.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "while a whole-file read still licenses the whole file"

# CONTROL B: a partial read still licenses exactly its own lines — the line it
# showed, and not the one it did not.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/part.txt"
m read 'part.txt:2' >"$WORK/served.out"
printf '@@ part.txt 2 replace\nBBB\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "and a partial read still licenses the line it showed"
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/part2.txt"
m read 'part2.txt:2' >"$WORK/served.out"
printf '@@ part2.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 1 "$rc" "and still refuses the line it did not"

# An empty file cannot satisfy a range either, and used to say so by staying
# silent at exit 0.
fixture
: > "$R/none.txt"
m read 'none.txt:1' >"$WORK/served.out" 2>&1; rc=$?
want 1 "$rc" "a range against an empty file is reported, not silently fine"

# 22b. UPGRADING DOES NOT HEAL A LEDGER v0.0.11 ALREADY POISONED. Section 22
#      stops new bad entries; it does nothing about the ones already on disk,
#      and the fixed binary would honour them — because a poisoned entry and a
#      legitimate whole-file read are the SAME BYTES, `<sha>  -  <path>`. No
#      parse-time rule can separate them, so the ledger carries a version
#      header and a file without it is discarded rather than parsed.
#
#      The pre-v2 file is written directly here rather than produced by an old
#      binary: what matters is the on-disk shape a v0.0.11 mrw left behind, and
#      writing it is the honest way to have it without shipping an old build.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/stale.txt"
SHA=$(m read stale.txt 2>/dev/null | sed -n 's/.*sha \([0-9a-f]*\).*/\1/p' | head -1)
SD=$(m seen | head -1 | sed 's/.*: //')
# A whole-file licence for a file this checkout has NOT read in this state —
# exactly what a v0.0.11 failed read left behind. No header line.
m forget stale.txt >/dev/null 2>&1 || true
FULLSHA=$(shasum -a 256 "$R/stale.txt" | cut -d' ' -f1)
# TWO entries, and the count is the assertion's whole strength. Load consumes
# line 1 as the header ALWAYS, before deciding whether it is one — so a
# ONE-entry headerless ledger has its only record eaten either way, the ledger
# is empty whether or not the discard runs, and the write is refused for the
# wrong reason. With one entry this row passed with the version check disabled.
# The second entry is what the discard has to remove.
printf '%s  -  first.txt\n%s  -  stale.txt\n' "$FULLSHA" "$FULLSHA" > "$SD/seen"
grep -qv '^#mrw-seen' "$SD/seen" && ok "a pre-v2 ledger has no header, as v0.0.11 wrote it" || bad "fixture is wrong"
out=$(m write - <<<"$(printf '@@ stale.txt 3 replace\nCCC\n')" 2>&1); rc=$?
want 1 "$rc" "a pre-v2 whole-file licence is NOT honoured after upgrading"
grep -q '^c$' "$R/stale.txt" && ok "and the file is untouched" || bad "the poisoned licence applied: $(sed -n 3p "$R/stale.txt")"
grep -q 'written by an older mrw' <<<"$out" && ok "and the caller is told why, not just refused" || bad "silent refusal: $out"

# It must HEAL, or every later run repeats the refusal: Record loads before it
# saves, so a stale ledger that returned an error would stop the header ever
# being written.
m read stale.txt:2 >"$WORK/served.out" 2>&1
head -1 "$SD/seen" | grep -q '^#mrw-seen' && ok "and the next read rewrites the ledger with a header" || bad "the ledger never heals: $(head -1 "$SD/seen")"
out=$(m read stale.txt:2 2>&1 >/dev/null)
grep -q 'written by an older mrw' <<<"$out" && bad "the notice repeats after healing" || ok "and the notice does not repeat once healed"
printf '@@ stale.txt 2 replace\nBBB\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "and the healed ledger licenses exactly the line that was re-read"
# 23. `$` is the LAST LINE on the read path too. read kept one sentinel for
#     `$` and for an omitted end, and downstream that sentinel means unbounded
#     in whichever direction it appears — so `f.txt:$` resolved to 1-total and
#     served the WHOLE file for a one-line request, while `@@ f.txt $ replace`
#     on the write path correctly touched one line and the README said "the
#     last line". Nothing asserted the read side, which is why it drifted.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/five.txt"
seq 1 10 > "$R/ten.txt"
m read five.txt ten.txt >"$WORK/served.out"
# The HEADER is derived from the resolved range so it is fair evidence, but the
# LINE COUNT is what a header alone could not fake: 1-5 and 5-5 differ by four
# printed lines.
out=$(m read 'five.txt:$' 2>&1)
grep -qF '@@ 5-5' <<<"$out" && ok '`$` resolves to the last line' || bad "not the last line: $out"
n=$(grep -cE '^ +[0-9]+\|' <<<"$out")
[ "$n" = 1 ] && ok "and serves exactly one line" || bad "served $n lines for a one-line address"
out=$(m read 'five.txt:$-$' 2>&1)
grep -qF '@@ 5-5' <<<"$out" && ok '`$-$` is that same one line' || bad "wrong: $out"
# The control: `$` as an END was already correct, by accident. If this moves,
# the fix has traded one wrong reading of `$` for another.
out=$(m read 'five.txt:3-$' 2>&1)
grep -qF '@@ 3-5' <<<"$out" && ok '`3-$` still runs to EOF' || bad "the end form moved: $out"
# A range written with `$` can only be judged reversed once the length is
# known: `$-5` is fine on a five-line file and reversed on a ten-line one.
# Before this it was neither — it SERVED lines 1-5 at exit 0.
out=$(m read 'ten.txt:$-5' 2>&1); rc=$?
want 1 "$rc" 'a `$` range that ends before it starts is refused'
grep -q 'ends before it starts' <<<"$out" && ok "and says so" || bad "no reason given: $out"
grep -qE '^ +[0-9]+\|' <<<"$out" && bad "served content for an unsatisfiable range: $out" \
  || ok "and serves no content for it"
out=$(m read 'five.txt:$-5' 2>&1)
grep -qF '@@ 5-5' <<<"$out" && ok 'while the same address on a shorter file is satisfiable' || bad "wrong: $out"

# THE ASSERTION NO HEADER TEXT CAN SATISFY: the ledger records what was SERVED,
# so it is independent of anything printed above the content. Before the fix a
# one-line request recorded the whole file as seen. It needs its OWN checkout,
# because the reads above already recorded these files whole — and it goes LAST
# for that reason: an earlier fixture call here left the files the rows below
# it were using in a previous root, and `ten.txt:$-5` then "passed" on an
# UNREADABLE rather than on the reversed range. A row that passes for the wrong
# reason is the thing this file exists to catch.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/led.txt"
m read 'led.txt:$' >"$WORK/served.out"
out=$(m seen 2>&1)
grep -E 'led\.txt' <<<"$out" | grep -qE 'lines 5( |$)' \
  && ok "and the ledger records only the line that was served" \
  || bad "ledger over-records: $(grep led.txt <<<"$out")"

# 24. CONCURRENT invocations serialize on the ledger (ADR-038), and writability
#     still follows the ledger exactly. ADR-038 closed the accepted loss: 40
#     racing reads must keep 40. The skip-when-no-race branch is gone — a host
#     that serialises is now the pass, not a hole. What must STILL hold is the
#     failure DIRECTION ADR-004 leans on: a missing entry costs a re-read and
#     never licenses a wrong write. So after the reads, the files still IN the
#     ledger are exactly the ones that can be written — not merely that some
#     writes were refused, which would pass on a build that refused everything,
#     and not that changed == applied, which would pass on a build that failed
#     OPEN. §76 is the keep-40 row; this one is the safety half.
fixture
N=40
for i in $(seq 1 $N); do printf 'a\nb\nc\n' > "$R/r$i.txt"; done
for i in $(seq 1 $N); do m read "r$i.txt" >"$WORK/served.out" 2>&1 & done
wait
kept=$(m seen 2>/dev/null | grep -cE '(^| )r[0-9]+\.txt$')
if [ "$kept" -lt "$N" ]; then
  bad "concurrent reads lost ledger entries ($kept/$N kept) — ADR-038 requires all 40"
else
  ok "concurrent reads keep every entry ($kept/$N kept)"
fi
applied=0
for i in $(seq 1 $N); do
  printf '@@ r%s.txt 2 replace\nB\n' "$i" | m write - >/dev/null 2>&1 && applied=$((applied + 1))
done
# THE ASSERTION: writability follows the ledger exactly. Fail-open makes this
# 40 with a short ledger; refuse-everything makes it 0; only honouring the
# surviving entries makes it equal to what survived.
want "$kept" "$applied" "and exactly the files still in the ledger are writable"
changed=$(grep -lx 'B' "$R"/r*.txt 2>/dev/null | wc -l | tr -d ' ')
want "$applied" "$changed" "and no file changed that was not applied"

# 25. A directory that EXISTS and cannot be READ is refused, not read as one
#     holding no package. holdsPackage walks the directory and discarded the
#     walk's error, so a permission failure came back indistinguishable from a
#     directory of prose — and the scope then fell back to the whole-project
#     command. That used to be sound for a typo (ADR-042 now refuses a miss)
#     and is vacuous here: the caller named a directory, mrw could not open
#     it, and the answer was PASS at exit 0.
fixture
mkdir -p "$R/blind" "$R/seen-dir"
printf 'package blind\n' > "$R/blind/b.go"
printf 'package seendir\n' > "$R/seen-dir/s.go"
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
chmod 000 "$R/blind"
if [ -r "$R/blind" ] || ls "$R/blind" >/dev/null 2>&1; then
  chmod 755 "$R/blind"
  skip "an unreadable scope is refused (permission bits not enforced here — running as root?)"
else
  out=$(m check blind 2>&1); rc=$?
  chmod 755 "$R/blind"
  want 2 "$rc" "a directory that cannot be read is refused, not scoped to nothing"
  grep -q 'cannot be read' <<<"$out" && ok "and the reason names the path" || bad "no reason given: $out"
  # The declared check is `echo FULL` and exits 0, so a row asserting only a
  # non-zero exit would not separate a refusal from the fallback it replaces.
  # What says nothing ran is that FULL was never printed.
  grep -q 'FULL' <<<"$out" && bad "fell back and answered about the whole project: $out" \
    || ok "and nothing ran under it"
fi
# CONTROLS: a readable directory still scopes. A path that is not there is
# refused (ADR-042) — the leftover this row used to protect.
out=$(m check seen-dir 2>&1)
grep -qF 'SCOPED ./seen-dir/...' <<<"$out" && ok "while a readable directory still scopes" || bad "readable dir broke: $out"
out=$(m check nosuchdir 2>&1); rc=$?
want 2 "$rc" "and a mistyped path is refused, not a silent PASS"
grep -q 'FULL' <<<"$out" && bad "fell back on the typo: $out" \
  || ok "and nothing ran under the typo"

# 26. `iter add` is the FOURTH way into the tree — after read, write and check —
#     and it was the one that did not enforce the root boundary. It validated
#     with filepath.Join, which CLEANS `../outside/x` into a path that exists,
#     so the entry was accepted. Nothing leaked, because read refuses it when
#     serving — but the working set is what `mrw check` scopes to by default, so
#     one accepted entry made every later check refuse until it was removed.
#     ADR-006 rule 2: the boundary lives in one place, and this was the caller
#     that did not use it.
fixture
mkdir -p "$WORK/beyond"
printf 'secret\n' > "$WORK/beyond/s.txt"
ln -s "$WORK/beyond" "$R/wayout"
m iter add ../beyond/s.txt >/dev/null 2>&1; rc=$?
want 2 "$rc" "iter add refuses a path outside the root"
out=$(m iter add ../beyond/s.txt 2>&1)
grep -q 'outside the root' <<<"$out" && ok "and says so, rather than 'no such file'" || bad "wrong reason: $out"
m iter add wayout/s.txt >/dev/null 2>&1; rc=$?
want 2 "$rc" "and refuses it through a symlink too"
# THE CONSEQUENCE, which is what makes this more than a tidiness fix: the set is
# what `mrw check` scopes to, so an accepted entry wedged every later check.
out=$(m iter 2>&1)
grep -q 'beyond' <<<"$out" && bad "the out-of-root entry landed in the working set: $out" \
  || ok "so nothing out-of-root reaches the working set"
m iter add a.go >/dev/null 2>&1
m check >/dev/null 2>&1; rc=$?
want 0 "$rc" "and a set-scoped check still runs"
# CONTROLS: ordinary specs must still be accepted, and a MISSING in-root path
# keeps its own message — a different mistake with a different remedy.
m iter add b.go >/dev/null 2>&1; rc=$?
want 0 "$rc" "an in-root file is still added"
m iter add 'a.go:1-2' >/dev/null 2>&1; rc=$?
want 0 "$rc" "and so is a ranged spec"
out=$(m iter add nosuch.go 2>&1); rc=$?
want 2 "$rc" "a missing in-root path is still refused"
grep -q 'no such file' <<<"$out" && ok "and still says 'no such file', not 'outside the root'" || bad "wrong reason: $out"

# 27. THE SCALE PEOPLE ACTUALLY USE. Every row above this one edits 1-3 hunks,
#     and the README's examples total six `@@` headers. Observed in real use on
#     2026-09-02: THIRTEEN hunks in one `mrw write`, and a read naming TWELVE
#     comma-separated ranges of one file — several times the size of anything the
#     corpus exercised. awk's own bug history says defects cluster at input
#     SHAPES rather than at missing assertions, and scale is a shape.
#
#     The load-bearing assertion is not that the 13 land: it is that the 987
#     lines nobody addressed come back BYTE-IDENTICAL. A batch edit that
#     disturbs a neighbour is the failure this tool exists to refuse, and it is
#     invisible if you only check the lines you meant to change.
#
#     THE MUTATION THAT PRODUCES A SILENT DISTURBANCE, named because it is the
#     one a reader will try to reproduce and the guard is what makes it silent:
#
#       res = append(res, orig[cursor-1:max(cursor-1, h.Start-2)]...)
#
#     WITHOUT the max() the slice goes invalid on the first hunk and the write
#     fails outright, exit 2, no verdicts — a different bug, loudly. WITH it the
#     write SUCCEEDS, reports 13 ok verdicts, and eats the line before each
#     hunk. Four rows go red: these two, and two earlier ones whose chained
#     check fails. Those two say only "the check did not pass"; the rows below
#     are what NAME the disturbance.
fixture
awk 'BEGIN{for(i=1;i<=1000;i++) print "line " i}' > "$R/big.css"
cp "$R/big.css" "$WORK/big.css.orig"
SITES="40,80,102-103,126,244,446,500,600,700,800,900,978"
out=$(m read "big.css:$SITES" 2>&1); rc=$?
want 0 "$rc" "a read naming 12 ranges of one file in ONE call"
# TWELVE ranges covering THIRTEEN lines — 102-103 is one range, two lines. The
# first draft of this row asserted 13 spans and went red: mrw was right and the
# arithmetic was mine. Worth keeping as written, because "count the sites" and
# "count the ranges" are exactly the confusion a comma-separated spec invites.
n=$(grep -cE '^@@ ' <<<"$out")
[ "$n" = 12 ] && ok "and serves 12 separate spans, not one merged blur" || bad "served $n span(s), want 12"
grep -qE '^@@ 102-103$' <<<"$out" && ok "keeping 102-103 as one span covering two lines" || bad "the two-line range was split or lost"

# One command, thirteen hunks. Built with printf, which is how the plan was
# generated in the wild — nothing in the docs showed that, so every caller
# reinvents it.
{ for i in 40 80 102 126 244 446 500 600 700 800 900 978; do
    printf '@@ big.css %d replace\nCHANGED-%d\n' "$i" "$i"
  done
  printf '@@ big.css 103 replace\nCHANGED-103\n'
} > "$WORK/13.plan"
out=$(m write "$WORK/13.plan" 2>&1); rc=$?
want 0 "$rc" "and 13 hunks in ONE write command"
n=$(grep -cE '^ok ' <<<"$out")
[ "$n" = 13 ] && ok "with a verdict for every one of the 13" || bad "$n verdict(s), want 13"
n=$(grep -c '^CHANGED-' "$R/big.css")
[ "$n" = 13 ] && ok "and all 13 sites changed" || bad "$n site(s) changed, want 13"

# THE ROW THAT MATTERS. Strip the 13 addressed lines from both files and compare
# the rest byte for byte: a batch that quietly moved or ate a neighbour fails
# here and nowhere else.
ADDR='^(40|80|102|103|126|244|446|500|600|700|800|900|978)$'
awk -v a="$ADDR" 'NR !~ a' "$WORK/big.css.orig" > "$WORK/before.rest" 2>/dev/null || true
awk 'BEGIN{split("40 80 102 103 126 244 446 500 600 700 800 900 978",k," ");for(i in k)skip[k[i]]=1} !(FNR in skip)' \
  "$WORK/big.css.orig" > "$WORK/before.rest"
awk 'BEGIN{split("40 80 102 103 126 244 446 500 600 700 800 900 978",k," ");for(i in k)skip[k[i]]=1} !(FNR in skip)' \
  "$R/big.css" > "$WORK/after.rest"
if cmp -s "$WORK/before.rest" "$WORK/after.rest"; then
  ok "and the 987 lines nobody addressed are byte-identical"
else
  bad "a 13-hunk write disturbed lines it did not address: $(diff "$WORK/before.rest" "$WORK/after.rest" | head -3 | tr '\n' ' ')"
fi
n=$(wc -l < "$R/big.css" | tr -d ' ')
[ "$n" = 1000 ] && ok "and the file is still 1000 lines" || bad "line count moved to $n"

# --- 28. hunk ORDER does not change the result (ADR-001 rule 1) ------------
# The rule says a plan's hunks are applied to the ORIGINAL file, so their order
# in the plan cannot matter. Nothing pinned that until now: every plan the suite
# writes happens to be in ascending line order, which is exactly the arrangement
# a naive sequential implementation also gets right. Two identical files, the
# same five edits, opposite orders, compared byte for byte.
seq 1 20 > "$R/ord-a.txt"; cp "$R/ord-a.txt" "$R/ord-b.txt"
m read ord-a.txt ord-b.txt >"$WORK/served.out" 2>&1
# The ops must SHIFT lines, or the row cannot fail: a plan of nothing but
# `replace` leaves every later line number valid, so a naive implementation that
# walks the plan top-to-bottom against a mutating buffer passes it too. Mixing
# insert-after and delete is what makes order observable at all.
ord_plan() { # $1 = file
  printf '@@ %s 3 insert-after\nINS-3\n'  "$1"
  printf '@@ %s 7 delete\n'                "$1"
  printf '@@ %s 11 replace\nX-11\n'       "$1"
  printf '@@ %s 15 insert-after\nINS-15\n' "$1"
  printf '@@ %s 19 delete\n'               "$1"
}
ord_plan ord-a.txt > "$WORK/asc.plan"
# Reversed by HUNK, not by line: a hunk is a header plus its body, so reversing
# the plan's lines would put each body above its own header.
{ printf '@@ ord-b.txt 19 delete\n'
  printf '@@ ord-b.txt 15 insert-after\nINS-15\n'
  printf '@@ ord-b.txt 11 replace\nX-11\n'
  printf '@@ ord-b.txt 7 delete\n'
  printf '@@ ord-b.txt 3 insert-after\nINS-3\n'
} > "$WORK/desc.plan"
m write "$WORK/asc.plan"  >/dev/null 2>&1; want 0 "$?" "five hunks in ascending line order apply"
m write "$WORK/desc.plan" >/dev/null 2>&1; want 0 "$?" "and the same five in descending order apply too"
if cmp -s "$R/ord-a.txt" "$R/ord-b.txt"; then
  ok "and both orders produce a byte-identical file"
else
  bad "hunk order changed the result: $(diff "$R/ord-a.txt" "$R/ord-b.txt" | head -3 | tr '\n' ' ')"
fi
n=$(wc -l < "$R/ord-b.txt" | tr -d ' ')
[ "$n" = 20 ] && ok "and the descending write did not shift the file (still 20 lines)" || bad "line count moved to $n"

# --- 29. a plan file with CRLF line endings --------------------------------
# A plan is often written by an editor, and an editor on Windows writes CRLF.
# Two ways to get this wrong: reject the plan because the op parses as
# "replace\r", or accept it and carry the CR into the body, silently giving an
# LF file one CRLF line. The second is the dangerous one, so it gets its own row.
printf 'a\nb\nc\n' > "$R/crlf-plan.txt"
m read crlf-plan.txt >"$WORK/served.out" 2>&1
printf '@@ crlf-plan.txt 2 replace\r\nBBB\r\n' > "$WORK/crlf.plan"
m write "$WORK/crlf.plan" >/dev/null 2>&1; want 0 "$?" "a plan file written with CRLF endings still parses"
[ "$(sed -n 2p "$R/crlf-plan.txt")" = "BBB" ] && ok "and line 2 is the intended content" || bad "line 2 is $(sed -n 2p "$R/crlf-plan.txt" | od -c | head -1)"
if LC_ALL=C grep -q $'\r' "$R/crlf-plan.txt"; then
  bad "the plan's CR leaked into an LF file"
else
  ok "and the CR did not leak into the LF file"
fi


# --- 30. concurrent WRITES lose edits, and the loss must stay SAFE ----------
# Section 24 covers concurrent READS. Nothing covered concurrent writes, and
# docs/adr/BACKLOG.md now records what they actually do: 20 writers racing on
# one file kept 1, 17 and 20 of 20 edits across three trials, and a writer that
# LOST still printed "applied" and exited 0. That silent loss is accepted (ADR-002
# puts locking permanently out of scope) and is deliberately NOT asserted here:
# it is nondeterministic, so a row for it would skip on a serialising machine
# and assert nothing.
#
# What IS asserted are the three invariants that held in every trial, because
# they are what makes the accepted loss survivable: a racing write may be
# discarded whole, but it must never leave the file TORN, the wrong LENGTH, or
# a temp file behind. Those are unconditional — no skip guard, no race needed.
#
# HONESTY NOTE, and it is a limitation of these three rows: they are NOT
# mutation-proven. Replacing the atomic rename with a non-atomic in-place write
# (truncate, write half, sleep 3ms, write the rest) did NOT turn any of them
# red. The reason is nameable rather than mysterious: under that mutant the
# file changes visibly mid-flight, so the sha guard REFUSES more writers, not
# fewer — exit-0 writers dropped from 17/20 to 6/20 — and the writers that
# would have had to overlap were serialised by the very corruption meant to be
# detected. Process startup also dwarfs the tear window.
#
# So these rows currently assert a property nothing has been shown to break.
# They are kept because they are unconditional and cost nothing, and because a
# future refactor of the write path is exactly what they exist to catch — but
# do not read a green here as evidence the invariant is protected.
fixture
seq 1 100 > "$R/race.txt"
m read race.txt >"$WORK/served.out" 2>&1
for i in $(seq 1 20); do
  printf '@@ race.txt %d replace\nW-%d\n' $((i * 3)) "$i" | m write - >/dev/null 2>&1 &
done
wait
n=$(wc -l < "$R/race.txt" | tr -d ' ')
[ "$n" = 100 ] && ok "20 racing writers leave the file its original 100 lines" || bad "line count moved to $n under concurrent writes"
# Every line must be either an untouched original number or a complete marker.
# Anything else is interleaved output — two writers' bytes in one file.
torn=$(grep -cvE '^([0-9]+|W-[0-9]+)$' "$R/race.txt" || true)
[ "$torn" = 0 ] && ok "and no line is torn between two writers" || bad "$torn torn line(s): $(grep -vE '^([0-9]+|W-[0-9]+)$' "$R/race.txt" | head -2 | tr '\n' ' ')"
strays=$(find "$R" -name '.mrw-*' 2>/dev/null | wc -l | tr -d ' ')
[ "$strays" = 0 ] && ok "and no staging temp survives the race (ADR-004)" || bad "$strays stray temp(s) left by concurrent writers"


# --- 31. an ABSOLUTE path given as a command-line argument -----------------
# Reported by a user running mrw on another project: `mrw -C repo read
# /elsewhere/x.md` joined the absolute path onto the root, looked for
# `repo/elsewhere/x.md`, and reported "no such file" about a path nobody wrote
# — while the `==>` header above it echoed the path they DID write. They read
# that as a defect in the tool that called mrw.
#
# Joining stays correct for a PLAN, which is a document whose paths are
# relative by design, and section 26 pins apply refusing an absolute one by
# name. A command-line argument is the other convention: tab completion emits
# absolute paths. Containment is still decided in one place — the argument is
# made root-relative and goes through the same Resolve.
fixture
out=$(m read "$R/a.go" 2>&1); rc=$?
want 0 "$rc" "an absolute path INSIDE the root is served, not joined"
grep -qE '^==> a\.go ' <<<"$out" \
  && ok "and the receipt names it by its root-relative path" \
  || bad "header is not the root-relative path: $(head -1 <<<"$out")"

# The shape the user actually hit. $WORK is a second mktemp dir, so this is a
# real path that exists and is genuinely outside the root — not a missing file
# dressed up as a boundary case.
printf 'outside\n' > "$WORK/outside.txt"
out=$(m read "$WORK/outside.txt" 2>&1); rc=$?
want 1 "$rc" "an absolute path OUTSIDE the root is refused"
grep -q 'is outside the root' <<<"$out" \
  && ok "and says so, rather than 'no such file'" \
  || bad "wrong diagnosis: $(head -1 <<<"$out")"
# THE ROW THAT MATTERS: no line of the output may name root+absolute glued
# together. That concatenation is what sent the user looking for a file they
# could see with their own eyes.
if grep -qF "$R$WORK" <<<"$out"; then
  bad "the receipt still names a concatenated path: $(grep -oF "$R$WORK" <<<"$out" | head -1)"
else
  ok "and no line names the root and the absolute path glued together"
fi


# --- 32. the plan surface names the path that is actually missing ----------
# Section 31 fixed the read surface; review of it found the same misdirection
# one surface over. apply refuses an absolute path in a plan — correctly, a
# plan is a document whose paths are relative by design — and then said "it
# named <the caller's own path>, which does not exist". That file DOES exist;
# it is the path the join produced that does not, and it was never shown. The
# clause written to stop a caller hunting for a file they can see was telling
# them exactly that.
fixture
mkdir -p "$R/sub"; printf 'a\nb\nc\n' > "$R/sub/real.go"
m read sub/real.go >"$WORK/served.out" 2>&1
out=$(printf '@@ %s 2 replace\nB\n' "$R/sub/real.go" | m write - 2>&1); rc=$?
want 1 "$rc" "an absolute path in a PLAN is still refused"
grep -q 'is absolute, and every path in a plan is relative to the root' <<<"$out" \
  && ok "and still says why, by name" \
  || bad "lost the absolute-path diagnosis: $(head -1 <<<"$out")"
# THE ROW: the path reported missing must be the JOINED one, which contains the
# root twice. A message naming only the caller's path is the bug.
if grep -qF "$R$R/sub/real.go" <<<"$out"; then
  ok "and names the joined path it actually looked for"
else
  bad "does not name the joined path: $(head -1 <<<"$out")"
fi
[ -f "$R/sub/real.go" ] \
  && ok "and the file it called missing is still right there, untouched" \
  || bad "the fixture file vanished"

# 28. ADR-007: mrw FINDS the files it serves. Every flag and every documented
# usage error, driven through the real binary — because the precedence table is
# where a caller's mental model breaks, and a table is only a promise until
# something exits non-zero over it.
#
# Numbered 28 and not 15. T3's acceptance fence says `grep -q '^# 15\.'`, and
# section 15 has existed since the check work — so that clause was satisfied by
# an UNTOUCHED contract.sh from the moment it was written. The fence built to
# prove new rows exist could not fail. Found 2026-09-03 while adding the rows;
# the fence now names this section.
fixture
mkdir -p "$R/pkg" "$R/vendor"
printf 'package demo\n\n// NEEDLE lives here\nfunc E() int { return 5 }\n' > "$R/pkg/e.go"
printf 'package demo\n\n// NEEDLE again\n'                                 > "$R/pkg/f.txt"
printf 'NEEDLE in a vendored file\n'                                       > "$R/vendor/v.md"

out=$(m read --grep NEEDLE 2>&1); rc=$?
want 0 "$rc" "--grep with no paths walks the root"
grep -q 'pkg/e.go' <<<"$out" && grep -q 'vendor/v.md' <<<"$out" \
  && ok "and serves every matching file it found" \
  || bad "the walk missed a match: $(head -3 <<<"$out")"

out=$(m read --grep NEEDLE --exclude vendor --exclude '*.txt' 2>&1); rc=$?
want 0 "$rc" "--exclude prunes a directory and drops a basename match"
grep -q 'pkg/e.go' <<<"$out" \
  && ! grep -q 'vendor/v.md' <<<"$out" \
  && ! grep -q 'f.txt' <<<"$out" \
  && ok "and keeps exactly what was not excluded" \
  || bad "exclusion served the wrong set: $(head -3 <<<"$out")"

# The basename half is the difference between the flag working and doing
# nothing: path.Match's * does not cross a separator, so '*.txt' against the
# root-relative path alone would match no file at any depth.
out=$(m read --grep NEEDLE --exclude '*.txt' 2>&1)
grep -q 'f.txt' <<<"$out" \
  && bad "--exclude '*.txt' did not drop a nested .txt — the basename match is gone" \
  || ok "--exclude matches the basename, not only the root-relative path"

out=$(m read --grep 'no-such-needle-anywhere' 2>&1); rc=$?
want 1 "$rc" "a pattern that matched nothing exits 1"
grep -q 'no-such-needle-anywhere' <<<"$out" \
  && ok "and names the pattern rather than printing nothing" \
  || bad "silence, or a report that does not name the pattern: $(head -1 <<<"$out")"

out=$(m read --grep NEEDLE pkg/e.go absent.go 2>&1); rc=$?
want 1 "$rc" "a refused path during a walk exits 1"
grep -q 'pkg/e.go' <<<"$out" && grep -q 'absent.go' <<<"$out" \
  && ok "and the good path is still served beside the refusal" \
  || bad "rule 5 broken — one bad path cost the good one: $(head -3 <<<"$out")"

out=$(printf '# a comment\n\npkg/e.go:/NEEDLE/\n' | m read --files-from - 2>&1); rc=$?
want 0 "$rc" "--files-from - reads specs from stdin, skipping blanks and comments"
grep -q 'NEEDLE lives here' <<<"$out" \
  && ok "and serves them" \
  || bad "--files-from served nothing: $(head -3 <<<"$out")"

out=$(printf '\n#only comments\n' | m read --files-from - 2>&1); rc=$?
want 2 "$rc" "--files-from with no specs is a usage error, not silence"

# Every row of the precedence table that says "usage error". Each is exit 2.
m read --exclude '*.go'                    >"$WORK/served.out" 2>&1; want 2 $? "--exclude without --grep is a usage error"
m read --grep X --files-from -             >"$WORK/served.out" 2>&1; want 2 $? "--grep with --files-from is a usage error"
m read --files-from - a.go                 >"$WORK/served.out" 2>&1; want 2 $? "--files-from with positional paths is a usage error"
m read --grep X a.go:1-2                   >"$WORK/served.out" 2>&1; want 2 $? "--grep with a positional range is a usage error"
m read --grep X --exclude '['              >"$WORK/served.out" 2>&1; want 2 $? "a glob path.Match rejects is a usage error"
m read --grep '('                          >"$WORK/served.out" 2>&1; want 2 $? "a pattern regexp rejects is a usage error"

# The no-argument behaviour is UNCHANGED: without --grep it is still the
# working set, and an empty one is still the same usage error it always was.
m read >"$WORK/served.out" 2>&1; want 2 $? "no arguments and no --grep still means the working set"

# 29. A PASSING check leaves no log behind, and a FAILING one keeps its
# evidence. Measured 2026-09-03 before the fix: 11,129 mrw-check-*.log files
# totalling 43 MB in one machine's temp directory, one per --check run ever
# made, none ever removed.
#
# Two conditions, not one. The tail is a summary and the file is the evidence,
# so a failure keeps it; and a truncated report NAMES the file, so a pass that
# withheld lines keeps it too — deleting a file the report points at would be
# worse than leaving it.
fixture
before=$(ls ${TMPDIR:-/tmp}/mrw-check-*.log 2>/dev/null | wc -l | tr -d ' ')

m read a.go >"$WORK/served.out" 2>&1
out=$(printf '@@ a.go 3 replace\nfunc A() int { return 1 }\n' | m write --check - 2>&1); rc=$?
want 0 "$rc" "a passing --check exits 0"
grep -q 'full output:' <<<"$out" \
  && bad "a passing check still names a log file: $(grep 'full output:' <<<"$out")" \
  || ok "and names no log file, because there is none to name"

after=$(ls ${TMPDIR:-/tmp}/mrw-check-*.log 2>/dev/null | wc -l | tr -d ' ')
[ "$after" -le "$before" ] \
  && ok "and left no mrw-check log behind ($before -> $after)" \
  || bad "a passing check leaked $((after - before)) log file(s)"

# The other half: a FAILING check must keep the file it points at, or the
# caller is told to read evidence that has been deleted.
fixture
printf 'package demo\n\nimport "testing"\n\nfunc TestNo(t *testing.T) { t.Fatal("red") }\n' > "$R/a_test.go"
m read a.go >"$WORK/served.out" 2>&1
out=$(printf '@@ a.go 3 replace\nfunc A() int { return 1 }\n' | m write --check - 2>&1); rc=$?
want 3 "$rc" "a failing --check still exits 3, tree changed and unverified"
logf=$(grep -o '/[^ ]*mrw-check-[^ ]*\.log' <<<"$out" | head -1)
if [ -n "$logf" ] && [ -f "$logf" ]; then
  ok "and the log it names is still there to read"
  rm -f "$logf"
else
  bad "a failing check named no readable log: $(tail -1 <<<"$out")"
fi

# 30. THE DOCUMENTED EXAMPLE IS EXECUTED, not admired. AGENTS.md carries the
# plan-generation loop — the one that turns N calls into 2 — and it is the
# highest-leverage thing the docs teach. A worked example that has drifted from
# the tool teaches the drift, confidently.
#
# The block is EXTRACTED from AGENTS.md and run, so this row fails when the doc
# changes and the code does not, or the reverse. Verified by hand on 2026-09-03
# and now on every CI run.
fixture
awk '/^```bash$/{f=1;next} /^```$/{f=0} f' "$SRC/AGENTS.md" > "$WORK/example.sh"
[ -s "$WORK/example.sh" ] \
  && ok "AGENTS.md still carries a runnable bash example" \
  || bad "no fenced bash block found in AGENTS.md — the example was renamed or removed"

# The example walks git-tracked *.go files, so give it a git repo with some.
( cd "$R" && git init -q . && git add -A >/dev/null 2>&1 && git -c user.email=t@t -c user.name=t commit -qm x >/dev/null 2>&1 )
out=$(cd "$R" && PATH="$(dirname "$MRW"):$PATH" bash "$WORK/example.sh" 2>&1); rc=$?
if [ "$rc" -eq 0 ] || grep -q 'hunk(s)' <<<"$out"; then
  ok "and it runs against the real binary"
else
  bad "the documented example no longer works: $(head -2 <<<"$out")"
fi
grep -q 'package fresh' "$R/a.go" \
  && ok "and it actually rewrote the files it claims to" \
  || bad "the example ran but changed nothing — a write that changes nothing is the bug mrw exists to prevent"

# 31. A GUARD THE CALLER WROTE MUST BE ABLE TO FIRE. Two ways it could not,
# both found by probing on 2026-09-01 and both still true on 2026-09-03:
#
#   * a REPEATED key was last-wins and SILENT. `anchor="NOPE" anchor="a"`
#     applied at exit 0 with the false guard gone. That is internal/apply's own
#     stated principle inverted — "a guard that is parsed and then discarded
#     would be worse than no guard at all, the caller believes the edit is
#     pinned".
#   * `raw=true` without `body=` was accepted and did nothing, because raw=
#     only switches off the header check INSIDE a counted body.
#
# Both are refused at parse time now. Refused rather than resolved: two guards
# on one hunk are two different claims about one edit, and picking either
# silently is how the caller keeps believing the other.
fixture
m read a.go >"$WORK/served.out" 2>&1

out=$(printf '@@ a.go 3 replace anchor="NOPE" anchor="func A"\nX\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a repeated guard key is a usage error, not last-wins"
grep -q 'given twice' <<<"$out" \
  && ok "and says which key was given twice" \
  || bad "refused without naming the repetition: $(head -2 <<<"$out"|tail -1)"
grep -q 'func A() int { return 1 }' "$R/a.go" \
  && ok "and nothing was written" \
  || bad "the file changed despite the refusal"

# Every key, not just anchor — sha= and lines= are the ones that matter most.
for pair in 'sha=aaaaaaaa sha=bbbbbbbb' 'lines=1 lines=2' 'body=1 body=2'; do
  printf '@@ a.go 3 replace %s\nX\n' "$pair" | m write - >/dev/null 2>&1
  want 2 $? "a repeated key is refused for: $pair"
done

out=$(printf '@@ a.go 3 replace raw=true\nX\n' | m write - 2>&1); rc=$?
want 2 "$rc" "raw=true without body= is a usage error"
grep -q 'without body=' <<<"$out" \
  && ok "and says why it guards nothing" \
  || bad "refused without explaining: $(head -2 <<<"$out"|tail -1)"

# The LEGITIMATE pairing must still work, or this row has broken the escape
# hatch that lets a plan carry a line beginning with @@.
out=$(printf '@@ a.go 3 replace body=1 raw=true\n@@ still just a body line\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "body= with raw=true still applies — the escape hatch is intact"

# And a single guard of each kind is untouched. FRESH fixture: the hunk above
# rewrote line 3, so reusing the file here would fail on the anchor for a
# reason that has nothing to do with guards.
fixture
m read a.go >"$WORK/served.out" 2>&1
out=$(printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 9 }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "one anchor= still guards an edit"

# 32. A UTF-8 BOM does not disqualify the first header (issue #46). Windows
# PowerShell 5.1 — the powershell.exe that ships with Windows — writes one for
# `-Encoding utf8` and has no BOM-less option, so the most obvious way to
# author a plan in the native Windows shell produced a file mrw refused. The
# refusal was misleading twice: it said "text before the first @@ header" about
# a line that IS a header, then reported the body as a SECOND error, so the
# reader concluded the plan format was wrong.
#
# Driven through a shell because that is where the bytes come from. Stripped
# only at offset 0 — a BOM anywhere else is content, and editing a caller's
# body would be the silent-corruption class this project refuses.
fixture
m read a.go >"$WORK/served.out" 2>&1
printf '\357\273\277@@ a.go 3 replace\nfunc A() int { return 7 }\n' > "$WORK/bom.mrw"
out=$(m write --no-check "$WORK/bom.mrw" 2>&1); rc=$?
want 0 "$rc" "a plan with a UTF-8 BOM applies"
grep -q 'return 7' "$R/a.go" \
  && ok "and the edit actually landed" \
  || bad "exit 0 but the file is unchanged: $(sed -n 3p "$R/a.go")"

# The same bytes without the BOM must still apply — the control that proves the
# BOM was the whole difference.
fixture
m read a.go >"$WORK/served.out" 2>&1
printf '@@ a.go 3 replace\nfunc A() int { return 7 }\n' > "$WORK/nobom.mrw"
m write --no-check "$WORK/nobom.mrw" >/dev/null 2>&1
want 0 $? "and the same plan without a BOM still applies"

# A BOM in the BODY is content, not syntax.
fixture
m read a.go >"$WORK/served.out" 2>&1
printf '@@ a.go 3 replace\n\357\273\277KEEP\n' > "$WORK/inner.mrw"
m write --no-check "$WORK/inner.mrw" >/dev/null 2>&1
want 0 $? "a BOM inside a body is accepted"
# tr -s: od on macOS separates bytes with TWO spaces and on GNU with one, so a
# literal 'ef bb bf' matches on Linux and silently never matches here.
if sed -n 3p "$R/a.go" | od -An -tx1 | tr -s ' ' | head -1 | grep -q 'ef bb bf'; then
  ok "and preserved as content rather than stripped"
else
  bad "a BOM in the body was eaten — that is a silent edit to the caller's text"
fi

# 33. A DISCOVERED symlink out of the root is neither read nor revealed.
#
# `consider` resolves a NAMED path, so an explicit ../outside was always
# refused. An entry found by WALKING went to os.Stat and os.ReadFile with the
# path the walk built, never through rooted.Resolve — so mrw READ a file
# outside the root to match against it. read.Run refused to SERVE the result,
# but the two outcomes differed: a match printed a REFUSED line naming the
# resolved outside path, a non-match printed "no file matched". That
# difference is a pattern oracle over files outside the root.
#
# THE ROW THAT MATTERS is the last one: the two answers must be the same
# except for the caller's own pattern text. Asserting only "the match was
# refused" would have passed on the broken build.
#
# Found by an independent review 2026-09-03, not by the suite: the existing
# cases covered an explicit ../ and an IN-root symlink, and this fell between.
fixture
mkdir -p "$R/sub"
OUTSIDE=$(mktemp -d "$WORK/outside-XXXXXX")
printf 'SECRET-TOKEN-abc123\n' > "$OUTSIDE/secret.txt"
printf 'package demo\n' > "$R/sub/ordinary.go"
if ln -s "$OUTSIDE/secret.txt" "$R/sub/link.txt" 2>/dev/null; then
  hit=$(m read --grep 'SECRET-TOKEN' . 2>&1)
  miss=$(m read --grep 'ABSENT-EVERYWHERE' . 2>&1)

  grep -qE 'REFUSED|secret\.txt' <<<"$hit" \
    && bad "a match against an out-of-root file was announced: $(head -1 <<<"$hit")" \
    || ok "a discovered out-of-root symlink produces no REFUSED line and names no target"

  # Normalise the caller's own pattern out of both, then compare. What is left
  # must be identical, or the output distinguishes 'matched outside the root'
  # from 'did not match' — which is the oracle.
  h=$(sed 's|/SECRET-TOKEN/|/P/|' <<<"$hit")
  ms=$(sed 's|/ABSENT-EVERYWHERE/|/P/|' <<<"$miss")
  [ "$h" = "$ms" ] \
    && ok "and matching is indistinguishable from not matching" \
    || bad "the answers differ, which is the oracle: [$h] vs [$ms]"
else
  skip "symlinks unavailable — out-of-root walk boundary not exercised"
  skip "symlinks unavailable — the oracle comparison not exercised"
fi

# The legitimate case must not regress: a symlink to a file INSIDE the root is
# a candidate of its own, because mrw addresses files by path.
fixture
mkdir -p "$R/sub"
printf 'package t\nINROOT-MATCH\n' > "$R/sub/real.go"
if ln -s "$R/sub/real.go" "$R/sub/alias.go" 2>/dev/null; then
  n=$(m read --grep 'INROOT-MATCH' . 2>&1 | grep -c '^==> ')
  [ "$n" = "2" ] \
    && ok "an IN-root symlink is still served beside its target (two names, two addresses)" \
    || bad "expected 2 served files, got $n — the boundary fix broke rule 4"
else
  skip "symlinks unavailable — in-root symlink serving not exercised"
fi
# 34. CONCATENATED BOM-CARRYING FRAGMENTS STAY SEPARATE HUNKS.
#
# ⚠ This row exists because §32's fix caused a SILENT WRONG-WRITE. Stripping
# the BOM only from line 1 meant a PowerShell user — who builds a plan by
# concatenating fragments, each BOMed by the shell that wrote it — got the
# SECOND header treated as BODY TEXT of the first hunk. Two hunks applied as
# ONE, at exit 0, writing the swallowed header into their source file. The
# same input was REFUSED before §32: a loud failure became a quiet corruption,
# which is the precise defect this whole project is built to refuse.
#
# The assertion is on the FILE, not the exit code. Exit 0 was the bug.
fixture
m read a.go >"$WORK/served.out" 2>&1
printf '\357\273\277@@ a.go 3 replace\nfunc A() int { return 7 }\n' >  "$WORK/f1.mrw"
printf '\357\273\277@@ a.go 4 replace\nfunc B() int { return 8 }\n' >  "$WORK/f2.mrw"
cat "$WORK/f1.mrw" "$WORK/f2.mrw" > "$WORK/both.mrw"
out=$(m write --no-check "$WORK/both.mrw" 2>&1); rc=$?
want 0 "$rc" "two BOM-carrying fragments apply"
[ "$(grep -c 'ok  ' <<<"$out")" = "2" ] \
  && ok "as TWO hunks, not one" \
  || bad "got $(grep -c 'ok  ' <<<"$out") hunk(s): a header was swallowed into a body"
grep -q 'return 7' "$R/a.go" && grep -q 'return 8' "$R/a.go" \
  && ok "and both edits landed" \
  || bad "an edit is missing: $(sed -n '3,4p' "$R/a.go" | tr '\n' '|')"
# THE ROW THAT WOULD HAVE CAUGHT IT: no plan header may survive INTO the file.
grep -q '@@ a.go' "$R/a.go" \
  && bad "a plan header was written into the caller's source file — the silent wrong-write" \
  || ok "and no plan header leaked into the source"

# 35. THE MSYS DIAGNOSTIC, driven through the real binary (issue #45).
#
# Added because a reviewer pointed out that AGENTS.md says "a new promise needs
# a row in scripts/contract.sh that makes it go wrong on purpose" and carves
# out nothing for diagnostics — and I had argued the Go tests were enough. They
# do bite, but a MESSAGE is exactly the kind of promise that rots quietly: a
# reworded hint breaks nothing, compiles, and passes any test that greps for a
# substring the same edit happened to change. The suite already asserts message
# text elsewhere, so diagnostics were never conventionally exempt here.
#
# BOTH HALVES. The quiet one is the one that matters: a hint appended to every
# parse failure would be read once and ignored forever.
fixture

# The mangled spec exactly as MSYS2 hands it over: the ':' became ';' and the
# /pattern/ was expanded against the Git install prefix.
out=$(m read 'cmd\mrw\main.go;C:\Program Files\Git\^func main\' 2>&1); rc=$?
want 2 "$rc" "a spec MSYS mangled is a usage error"
grep -q 'MSYS2 argument conversion' <<<"$out" \
  && ok "and the diagnostic names MSYS2 rather than blaming a line number" \
  || bad "no MSYS diagnosis: $(head -1 <<<"$out")"
grep -q 'MSYS2_ARG_CONV_EXCL' <<<"$out" \
  && ok "and names the environment variable that fixes it" \
  || bad "diagnosed without a remedy: $(head -1 <<<"$out")"

# THE QUIET HALF. An ordinary bad range must NOT collect the hint.
out=$(m read 'a.go:notanumber' 2>&1); rc=$?
want 2 "$rc" "an ordinary bad range is still a usage error"
grep -q 'MSYS' <<<"$out" \
  && bad "the MSYS hint fired on an ordinary mistake — noise on every bad spec" \
  || ok "and carries no MSYS hint"

# The signature is the PAIR. A backslash alone is an ordinary filename
# character on POSIX and must not trigger it either.
out=$(m read 'weird\name.go:notanumber' 2>&1)
grep -q 'MSYS' <<<"$out" \
  && bad "a backslash alone triggered the hint; the pair is ';' AND a Windows path" \
  || ok "and a backslash without a ';' does not trigger it"

# 36. ADR-009: mrw COUNTS what happens to the plans it is given, and the tally
# holds nothing of the caller's work.
#
# The last row is the one that matters. A tally is a standing temptation to
# record "just the path", and a test that only checks the counts would pass on
# the day somebody adds one — so this greps the written file for the things a
# plan is made of and fails if any reached disk.
fixture
m read a.go >"$WORK/served.out" 2>&1

printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' | m write --no-check - >/dev/null 2>&1
want 0 $? "a plan that applies exits 0"
printf '@@ a.go 3 frobnicate\nx\n' | m write - >/dev/null 2>&1
want 2 $? "a plan that does not parse is a usage error"
printf '@@ a.go 3 replace anchor="NOPE"\nx\n' | m write - >/dev/null 2>&1
want 1 $? "a plan that parses but does not apply exits 1"

# mrw's own answer for where this checkout's state lives, rather than guessing
# at XDG layout. The base is pinned into $WORK at the top of this file, but the
# KEY is still a hash, so the directory is asked for rather than constructed.
tally="$(m seen | head -1)/authoring"
if [ -f "$tally" ]; then
  ok "a tally is written"
  grep -q '^applied 1$'       "$tally" && ok "and counts the applied plan"     || bad "applied not counted: $(tr '\n' '|' < "$tally")"
  grep -q '^refused_parse 1$' "$tally" && ok "and counts the unparseable one"  || bad "refused_parse not counted: $(tr '\n' '|' < "$tally")"
  grep -q '^refused_apply 1$' "$tally" && ok "and counts the one that did not apply" || bad "refused_apply not counted: $(tr '\n' '|' < "$tally")"

  # THE BOUNDARY ROW. Nothing of the caller's work may be in this file.
  if grep -qE '/|\\|@@|\.go|anchor|sha=|replace|insert|delete|create' "$tally"; then
    bad "the tally holds a plan fragment, path or address: $(tr '\n' '|' < "$tally")"
  else
    ok "and holds nothing of the caller's plan, paths or anchors"
  fi

  # Every line is 'name count'. A field nobody anticipated fails here.
  if [ -z "$(grep -vE '^[a-z_]+ [0-9]+$' "$tally")" ]; then
    ok "and every line is a counter, not a record"
  else
    bad "a non-counter line reached the tally: $(grep -vE '^[a-z_]+ [0-9]+$' "$tally" | head -1)"
  fi
else
  bad "no tally was written — the call site in mrw write is not reached"
fi

# "Record never fails a write" is NOT asserted here. An unusable state home
# fails the LEDGER first (seen.Record returns an error and the write exits 2),
# so a row here would grade the ledger and report it as the tally. It is proved
# where it can be isolated: TestRecordNeverFailsAWrite.

# 37. ADR-009 T2: `mrw stats` reads the tally, and never prints a rate without
# its denominator.
#
# The denominator row is the one worth having. A bare percentage is the form
# that gets quoted out of the population it was measured on, and ADR-009's
# criterion is explicitly valid FOR a population — so the shape of the output
# is part of the decision, not presentation.
fixture

out=$(m stats 2>&1); rc=$?
want 0 "$rc" "stats on a checkout with no tally exits 0"
grep -qi 'no plans recorded' <<<"$out" \
  && ok "and says nothing is recorded rather than printing zeros" \
  || bad "an empty tally did not announce itself: $(head -1 <<<"$out")"

m read a.go >"$WORK/served.out" 2>&1
printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' | m write --no-check - >/dev/null 2>&1
printf '@@ a.go 3 frobnicate\nx\n' | m write - >/dev/null 2>&1

out=$(m stats 2>&1); rc=$?
want 0 "$rc" "stats after two plans exits 0"
grep -q 'applied' <<<"$out"       && ok "and counts the applied plan"        || bad "applied missing: $out"
grep -q 'refused_parse' <<<"$out" && ok "and counts the unparseable one"     || bad "refused_parse missing: $out"
# THE ROW: every rate carries its sample size.
grep -qE 'of [0-9]+ plan' <<<"$out" \
  && ok "and every rate carries its denominator" \
  || bad "a rate was printed without its sample size: $out"

out=$(m stats --json 2>&1); rc=$?
want 0 "$rc" "stats --json exits 0"
grep -q '"plans"' <<<"$out" && grep -q '"counts"' <<<"$out" \
  && ok "and emits the plans/counts shape" \
  || bad "--json shape wrong: $(head -3 <<<"$out"|tr '\n' ' ')"

out=$(m stats --reset 2>&1); rc=$?
want 0 "$rc" "stats --reset exits 0"
grep -qE '[0-9]+ plan' <<<"$out" \
  && ok "and says how many records it discarded" \
  || bad "a silent reset is indistinguishable from a no-op: $out"
grep -qi 'no plans recorded' <<<"$(m stats 2>&1)" \
  && ok "and the tally is empty afterwards" \
  || bad "the tally survived --reset"

# 38. ADR-010 T2: the MCP server is the SAME engine, driven through a real pipe.
#
# A handler-level test would pass on a frame no host sends — which is exactly
# what happened while this task's own spec said `Content-Length`, the Language
# Server Protocol's framing rather than MCP's. So this row starts the real
# binary as a subprocess and speaks newline-delimited JSON-RPC down its stdin.
#
# The claim being checked is ADR-010's whole thesis: the verdict a caller gets
# over MCP is the value the CLI would have produced for the same plan. Two
# identical checkouts, one plan, both transports, compared field by field.
fixture
R_CLI=$R
PLAN='@@ a.go 3 replace
func A() int { return 11 }
'
cli=$(printf '%s' "$PLAN" | "$MRW" -C "$R_CLI" write --no-check --json - 2>/dev/null); rc=$?
want 0 "$rc" "the CLI applies the plan and emits a receipt"

fixture
R_MCP=$R
# ADR-113: mrw_write runs the check by default; check: false matches the CLI's --no-check above.
req=$(printf '%s' "$PLAN" | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read(),"check":False}}}))')
mcpout=$(printf '%s\n' "$req" | "$MRW" -C "$R_MCP" mcp 2>"$WORK/mcp.err"); rc=$?
want 0 "$rc" "mrw mcp applies the same plan over a real pipe"

# The spec: a server MUST NOT write anything to stdout that is not a valid MCP
# message. This binary prints to stdout everywhere else, so the rule is live.
python3 - "$mcpout" <<'PY'
import json,sys
lines=[l for l in sys.argv[1].splitlines() if l.strip()]
if not lines:
    print("the server wrote nothing at all"); sys.exit(1)
for l in lines:
    m=json.loads(l)
    assert m.get("jsonrpc")=="2.0", l
PY
[ $? -eq 0 ] && ok "every stdout line is a valid MCP message" \
             || bad "the server wrote something to stdout that is not an MCP message"
# stderr is where the spec says diagnostics go — "MAY write UTF-8 strings to its
# standard error for logging purposes" — so this asserts stderr carries ONLY the
# startup announcement ADR-011-T1 added, not that it is empty. It was `-s` until
# 2026-09-03, and the announcement turned that row red the day it landed: an
# assertion of "nothing" is the wrong shape for a channel the spec permits.
# NOT `grep -cv ... || echo 0`: grep prints 0 AND exits 1 when nothing is
# selected, so the `||` fires and the substitution captures "0\n0". That form
# was written here on 2026-09-03 and failed for exactly that reason.
[ -z "$(grep -v '^mrw mcp: serving ' "$WORK/mcp.err" | tr -d '[:space:]')" ] \
  && ok "and stderr carried only the startup announcement" \
  || bad "the server wrote something unexpected to stderr: $(grep -v '^mrw mcp: serving ' "$WORK/mcp.err" | head -1)"

# THE ROW: one engine, one answer. Root differs by construction (two temp
# dirs); every other field of the receipt must be identical.
python3 - "$cli" "$mcpout" <<'PY'
import json,sys
cli=json.loads(sys.argv[1])
mcp=json.loads(sys.argv[2].splitlines()[0])["result"]["structuredContent"]
cli.pop("root",None); cli.pop("check",None); mcp.pop("root",None)
if cli!=mcp:
    print("MCP:",json.dumps(mcp,sort_keys=True)); print("CLI:",json.dumps(cli,sort_keys=True)); sys.exit(1)
PY
[ $? -eq 0 ] && ok "the MCP receipt equals the CLI receipt for the same plan" \
             || bad "the two transports disagree about what happened"
grep -q 'return 11 }' "$R_MCP/a.go" && ok "and the file really changed" || bad "the receipt claimed a write that did not happen"

# Recovery — ADR-001's original objection, tested rather than argued. Killing
# the server mid-session must lose nothing: the ledger is on disk, so a NEW
# server and the CLI are both still licensed to write what the dead one read.
fixture
R_KILL=$R
rm -f "$WORK/in" "$WORK/out"; mkfifo "$WORK/in" "$WORK/out"
"$MRW" -C "$R_KILL" mcp < "$WORK/in" > "$WORK/out" 2>/dev/null &
srv=$!
exec 9>"$WORK/in"; exec 8<"$WORK/out"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go"]}}}' >&9
IFS= read -r -t 10 line <&8
[ -n "${line:-}" ] && ok "a live server answers a read over a real pipe" || bad "the server did not answer within 10s"
kill -9 "$srv" 2>/dev/null; wait "$srv" 2>/dev/null
exec 9>&-; exec 8<&-
kill -0 "$srv" 2>/dev/null && bad "the server survived SIGKILL" || ok "the server is killed mid-session"

# A NEW server completes a write the dead one licensed.
req=$(printf '@@ a.go 3 replace\nfunc A() int { return 12 }\n' | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read()}}}))')
out=$(printf '%s\n' "$req" | "$MRW" -C "$R_KILL" mcp 2>/dev/null)
grep -q '"applied":true' <<<"$out" \
  && ok "a NEW server completes a write the killed one licensed" \
  || bad "the ledger did not survive the kill: $(head -c 200 <<<"$out")"

# And so is the CLI: one ledger, not one per transport.
out=$(printf '@@ a.go 4 replace\nfunc B() int { return 22 }\n' | "$MRW" -C "$R_KILL" write --no-check - 2>&1); rc=$?
want 0 "$rc" "and a CLI write after the killed server is licensed too"
# 39. ADR-010 T3: mcp is a subcommand this binary has, and it starts a server.
#
# README heading / mcpServers JSON-fixture greps retired — ADR-053. A tidy of
# a tutorial heading is not a product break. This drives --help and the built
# binary.
#
# Redirect before grepping. Under this file's `set -o pipefail`, `grep -q` exits
# on its first match, `--help` takes SIGPIPE writing the rest, and the pipeline
# reports 141 — a row that fails for a reason having nothing to do with what it
# asks. The same trap ate ADR-010-T1's fence earlier the same day.
"$MRW" --help > "$WORK/help.out" 2>&1
grep -qE '^[[:space:]]+mcp[[:space:]]' "$WORK/help.out" \
  && ok "mrw --help lists mcp" \
  || bad "mrw --help does not list mcp"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | "$MRW" mcp 2>&1); rc=$?
want 0 "$rc" "mrw mcp starts a server"
grep -q 'mrw_write' <<<"$out" \
  && ok "which advertises mrw_write" \
  || bad "the server started but did not advertise mrw_write: $(head -c 120 <<<"$out")"


# 40. ADR-011 T1: the server binds to the checkout the HOST meant, not to the
# working directory it happened to inherit.
#
# Measured 2026-09-03 before this row existed: launched with cwd /tmp and
# CLAUDE_PROJECT_DIR naming a repository, `mrw mcp` served /private/tmp. Every
# ADR-006 refusal it gave was correct and about a tree nobody asked about, and
# the ADR-002 ledger is keyed per checkout, so a wrong root silently starts a
# second one. This drives the REAL binary from a foreign cwd, which is the only
# place the defect is visible.
fixture
R_HOST=$R
fixture
R_CWD=$R
printf 'package demo\n\nfunc Only() int { return 7 }\n' > "$R_HOST/only-in-host.go"

# No --root: the environment must decide, not the working directory.
out=$(cd "$R_CWD" && CLAUDE_PROJECT_DIR="$R_HOST" printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["only-in-host.go"]}}}' \
  | CLAUDE_PROJECT_DIR="$R_HOST" "$MRW" mcp 2>"$WORK/root.err"); rc=$?
want 0 "$rc" "mrw mcp starts with no --root"
grep -q 'func Only' <<<"$out" \
  && ok "and serves the tree CLAUDE_PROJECT_DIR names, not its own cwd" \
  || bad "the server bound to its working directory: $(head -c 160 <<<"$out")"

# THE ANNOUNCEMENT: stderr names the tree and the reason, stdout stays clean.
grep -q 'CLAUDE_PROJECT_DIR' "$WORK/root.err" \
  && ok "and says on stderr which tree it chose and why" \
  || bad "the server did not announce its root: $(head -c 120 "$WORK/root.err")"
python3 -c "
import json,sys
for l in open(sys.argv[1]) if False else sys.argv[1].splitlines():
    if l.strip(): json.loads(l)
" "$out" && ok "and stdout carried only MCP messages" || bad "stdout was polluted by the announcement"

# An explicit --root beats the host: a user overriding on purpose must win.
out=$(printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["only-in-host.go"]}}}' \
  | CLAUDE_PROJECT_DIR="$R_HOST" "$MRW" --root "$R_CWD" mcp 2>/dev/null)
grep -q 'UNREADABLE\|REFUSED' <<<"$out" \
  && ok "and an explicit --root overrides the host's environment" \
  || bad "--root did not win over CLAUDE_PROJECT_DIR: $(head -c 160 <<<"$out")"

# 41. ADR-011 T2: the tool surface declares what a host needs to protect a user,
# and the declaration is true.
#
# A host shows annotations to a person before asking them to approve a call, so
# a readOnlyHint on a tool that writes is a lie that person acts on. This row
# therefore checks read-only-ness BY OBSERVATION — it runs the tool and looks at
# the tree — rather than by reading the field back out of the descriptor.
fixture

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
want 0 $? "tools/list answers"
python3 - "$out" <<'PY'
import json,sys
tools={t["name"]: t for t in json.loads(sys.argv[1])["result"]["tools"]}
for name in ("mrw_read","mrw_write"):
    t=tools[name]
    for field in ("title","annotations","_meta"):
        assert field in t, "%s has no %s" % (name, field)
    assert t["_meta"]["anthropic/maxResultSizeChars"] > 0, "%s declares no size limit" % name
# Only mrw_write declares an outputSchema: ADR-023 (§61) removed mrw_read's
# with its structuredContent, and a declared schema promises one.
props=tools["mrw_write"]["outputSchema"].get("properties") or {}
assert props, "mrw_write declares a property-less schema, which validates anything"
assert "outputSchema" not in tools["mrw_read"], "mrw_read declares an outputSchema and returns no structuredContent (ADR-023)"
assert tools["mrw_read"]["annotations"]["readOnlyHint"] is True
assert tools["mrw_write"]["annotations"]["readOnlyHint"] is False
assert tools["mrw_write"]["annotations"]["destructiveHint"] is True
PY
[ $? -eq 0 ] && ok "both tools declare title, annotations and _meta, and only mrw_write an outputSchema" \
             || bad "the declared tool surface is incomplete or dishonest"

# THE ROW: mrw_read says readOnlyHint, so running it must leave the tree alone.
before=$(cd "$R" && find . -type f -newer go.mod -o -type f | sort | xargs shasum -a 256 | shasum -a 256)
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go"]}}}\n' \
  | m mcp >/dev/null 2>&1
after=$(cd "$R" && find . -type f -newer go.mod -o -type f | sort | xargs shasum -a 256 | shasum -a 256)
[ "$before" = "$after" ] \
  && ok "and mrw_read really is read-only, as its annotation claims" \
  || bad "mrw_read is annotated readOnlyHint and changed the tree"

# Both content blocks, with the first one machine-readable.
m read a.go >"$WORK/served.out" 2>&1
# The plan's newlines must reach mrw as the two-character escape \n INSIDE the
# JSON string, not as real newlines — a real one would split the message across
# two lines and the server would see two malformed frames. Hence \\n here.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ a.go 3 replace\\nfunc A() int { return 41 }\\n"}}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
c=r["content"]
assert len(c)>=2, "want two content blocks, got %d" % len(c)
# The spec asks for the serialized JSON in "a TextContent block", not the first.
# The report stays at content[0] because for mrw_read that block is where the
# FILE CONTENT lives, and a caller already reading content[0] must not silently
# start receiving a receipt instead.
assert c[0]["text"].strip(), "the human-readable report is missing from content[0]"
second=json.loads(c[1]["text"])         # must parse: the spec's SHOULD
assert second==r["structuredContent"], "content[1] and structuredContent disagree"
PY
[ $? -eq 0 ] && ok "and content[1] is the serialized structuredContent, with the report first" \
             || bad "the result does not carry the serialized JSON the spec asks for"

# 42. ADR-011 T3: a read too large for the wire is REFUSED, cheaply, and only
# over MCP.
#
# ADR-007's cap reports itself when it fires, which is right for a person at a
# terminal. Over MCP the consumer is a model, and a truncated file that arrives
# looking like the whole file is the silent wrong answer this project exists to
# refuse. The host truncates at 25,000 tokens regardless, so refusing legibly is
# the only option that does not spend memory to be overruled.
fixture
python3 -c "
with open('$R/wide.go','w') as f:
    f.write('package demo\n')
    for i in range(60000): f.write('// padding padding padding padding padding %d\n' % i)
"

# ⚠ RETARGETED BY ADR-014 at the MULTI-SPEC branch. A single oversized spec now
# returns a first page and a continuation and records the span it served, so
# "carries no file content" and "licenses nothing" are deliberately false there;
# §47 covers that case. mrw still cannot know WHICH of several specs to narrow,
# so the flat refusal survives for them, and this row pins it where its
# assertions remain true.
printf '// small\n' > "$R/small.go"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["wide.go","small.go"]}}}\n' | m mcp 2>/dev/null)
want 0 $? "the server answers a read that is too large"
python3 - "$out" <<'PY'
import json,re,sys
r=json.loads(sys.argv[1])["result"]
assert r.get("isError") is True, "an oversized read was not refused"
t=r["content"][0]["text"]
assert "200000" in t, "the refusal does not name the limit: %s" % t[:120]
m=re.search(r"around (\d+) lines per file", t)
assert m, "the refusal gives no line budget to retry with: %s" % t[:200]
assert int(m.group(1)) >= 100, "the budget %s is too small to be a useful retry" % m.group(1)
assert "padding padding" not in t, "the refusal carries file content; it truncated rather than refused"
# On the RESULT, not on structuredContent: a multi-spec refusal carries no
# structuredContent at all, so asserting the key's absence inside it passed over
# an empty object and proved nothing. Non-blocking item 6 of PR #76's review.
assert "next_read" not in json.dumps(r), "a multi-spec refusal must not pretend to page"
PY
[ $? -eq 0 ] && ok "and refuses it, naming the limit and a range to retry with" \
             || bad "the oversized read was not refused legibly"

# A refused read must license NOTHING: no ledger entry claiming the caller saw it.
out=$(printf '@@ wide.go 2 replace\n// changed\n' | m write - 2>&1); rc=$?
want 1 "$rc" "and a multi-spec refusal still licenses nothing"

# THE GO/NO-GO: one transport is bounded, the engine is not.
out=$(m read wide.go 2>&1); rc=$?
want 0 "$rc" "and the same file still reads whole on the CLI"
[ "$(wc -c <<<"$out")" -gt 200000 ] \
  && ok "and the CLI answer is larger than the MCP limit, so only the transport is capped" \
  || bad "the CLI read was also bounded; the engine was changed"

# 43. ADR-012 T1: the wire teaches the format, because nothing else can.
#
# A host driving `mrw mcp` is in a checkout it did not clone from here: AGENTS.md
# does not exist for it, and no model has this plan format in training data. So
# `initialize` and `tools/list` are the whole of its education. This row reads
# what the SHIPPED BINARY says, not what the source says, and it holds the wire
# against AGENTS.md — the duplication ADR-012 accepts deliberately, asserted here
# rather than trusted.
fixture

rule='Use mrw always: plan the activity as one read of every site, then one plan, then one write.'
# AGENTS.md may wrap the sentence across two lines, so compare on a whitespace-
# folded copy: the rule is the words, not where the paragraph happened to break.
tr -s '[:space:]' ' ' < "$SRC/AGENTS.md" | grep -qF "$rule" \
  && ok "AGENTS.md still states the always + plan sentence this row holds the wire against" \
  || bad "the always + plan sentence moved in AGENTS.md; the wire and the repository now teach different rules"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
want 0 $? "initialize answers"
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
i=r.get("instructions")
assert isinstance(i,str) and i.strip(), "initialize carries no instructions"
assert "@@" in i, "the instructions never show a plan header"
for f in ("AGENTS.md","README.md","CONTRIBUTING.md"):
    assert f not in i, "the instructions point at %s, which an MCP-only caller cannot open" % f
assert len(i.encode()) <= 4096, "the instructions are %d bytes; they are paid once per session" % len(i.encode())
# ADR-037: Shared is the contract both surfaces must carry. The Go test
# proves instructionsText contains it; this row proves the SHIPPED binary
# does. Five literals, not a package import — a drifted splice that still
# compiles must fail here.
shared = (
    "Use mrw always: plan the activity as one read of every site, then one plan, then one write.",
    "A plan applies whole or not at all: if any hunk fails validation, nothing is written; a failed commit reports what reached disk (CLI: PARTIALLY APPLIED).",
    "Read before you write, per line, not per file.",
    "mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.",
    "A refusal names the file, the plan line, and the reason.",
)
for s in shared:
    assert s in i, "handshake omitted Shared sentence: %r" % s
PY
[ $? -eq 0 ] && ok "and the handshake teaches the format without pointing at a file the caller cannot open" \
             || bad "the initialize instructions are missing, unbounded, or a pointer to nothing"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
want 0 $? "tools/list answers"
python3 - "$out" "$rule" <<'PY'
import json,sys
tools={t["name"]: t for t in json.loads(sys.argv[1])["result"]["tools"]}
rule=sys.argv[2]
for name in ("mrw_read","mrw_write"):
    assert rule in tools[name]["description"], "%s never says when to reach for it" % name
props=tools["mrw_write"]["inputSchema"]["properties"]
ex=props["plan"].get("examples")
assert ex and ex[0].startswith("@@ "), "mrw_write publishes no worked plan"
ex=tools["mrw_read"]["inputSchema"]["properties"]["specs"].get("examples")
assert ex and isinstance(ex[0],list) and len(ex[0])>1, "mrw_read publishes no worked spec list"
# ADR-027 changed a SERVED path — a bare `create` fails and names the fix — and
# ADR-012 is the promise that this surface teaches the format it demands. The
# handshake has no room for it (its 4096-byte bound is asserted three tests
# over, and the claims it would displace are guarded), so the tool description
# is where a plan author meets it, and this is what drives that through the
# built server rather than trusting the string in the source.
d=props["plan"]["description"]
assert "body=0" in d, "the plan description does not teach that an empty file is body=0 (ADR-027)"
assert "refused" in d.lower(), "the plan description lists create without saying a bare one is refused"
# ADR-035. This check has been unsound twice, and each cut was defeated by a
# string a careless edit could plausibly produce:
#   `"REQUIRED" in d`                     passed `anchor= is not REQUIRED ...`
#   one word order + two capitalisations  passed `anchor= is optional. It is
#                                         REQUIRED only on delete`
#   the clause without its SCOPE          passed `... REQUIRED on a replace
#                                         unless it addresses more than one line`
# So the scope is pinned too, and the checks are run over a table of strings
# that MUST fail as well as the live description — a check nobody has watched
# reject anything is a claim, not a gate.
import re as _re

def _adr035_desc_problems(text):
    out = []
    # THE WHOLE SENTENCE, TO ITS FULL STOP. An open-ended substring let a
    # qualifier be appended after the scope and still pass — `... more than one
    # line only when --force is absent.` was accepted, while the guard at
    # apply.go is unconditional.
    if not _re.search(r"anchor=\s+is\s+REQUIRED\s+on\s+a\s+replace\s+addressing\s+more\s+than\s+one\s+line,\s+and\s+may\s+be\s+omitted\s+elsewhere\.", text):
        out.append("does not state the requirement as the whole canonical clause")
    if _re.search(r"(?i)required[^.]*\b(unless|except)\b", text):
        out.append("qualifies REQUIRED away with unless/except")
    if _re.search(r"(?i)required[^.]*exactly\s+one\s+line", text):
        out.append("inverts the scope to exactly one line")
    if _re.search(r"(?i)anchor=[^.]*\bnot\s+required", text):
        out.append("says anchor= is not required")
    if _re.search(r"(?i)optional[^.]*anchor=", text):
        out.append("calls anchor= optional before requiring it")
    if _re.search(r"(?i)anchor=[^.]*\boptional\b", text):
        out.append("calls anchor= optional after requiring it")
    return out

_bad_wordings = [
    'anchor= is not REQUIRED on a replace addressing more than one line',
    'anchor= is optional. It is REQUIRED only on delete',
    'anchor= is REQUIRED on a replace unless it addresses more than one line',
    'anchor= is REQUIRED on a replace addressing exactly one line',
    'Optional guards: sha=<hex>, lines=<n>, anchor="<text>"',
    'anchor= is REQUIRED on a replace',
    'anchor= is REQUIRED on a replace addressing more than one line only when --force is absent.',
    'anchor= is REQUIRED on a replace addressing more than one line',
]
for _w in _bad_wordings:
    assert _adr035_desc_problems(_w), \
        "the ADR-035 wording check accepts a misleading description: %r" % _w
assert not _adr035_desc_problems(d), \
    "the plan description misstates the anchor= requirement (ADR-035): %s" % _adr035_desc_problems(d)
PY
[ $? -eq 0 ] && ok "and both tools say when to reach for them, publish a worked example, and teach the body-less create refusal" \
             || bad "the tool descriptions still say only what the tools do"

# THE ROW: the published examples run on the tree they are written against. An
# example asserted to be present stays green long after it stops being valid,
# and it is the one thing a caller copies verbatim. ADR-090: the tree is the
# hand-written one in the checkout, not one built from the plan — a tree built
# from a plan cannot fail that plan's own anchor, and a spec list checked only
# for its shape shipped a regexp that matched no Go ever written.
python3 - "$out" "$WORK/published.plan" "$WORK/published.specs" <<'PY'
import json,sys
tools={t["name"]: t for t in json.loads(sys.argv[1])["result"]["tools"]}
with open(sys.argv[2],"w") as f:
    f.write(tools["mrw_write"]["inputSchema"]["properties"]["plan"]["examples"][0])
with open(sys.argv[3],"w") as f:
    f.write("\n".join(tools["mrw_read"]["inputSchema"]["properties"]["specs"]["examples"][0])+"\n")
PY
cp -R "$SRC/internal/mcp/testdata/example/." "$R/"
m read --files-from "$WORK/published.specs" >"$WORK/served.out" 2>&1
want 0 $? "the read example mrw publishes serves every spec on the tree it is written against"
paths=$(python3 -c "
import re,sys
print(' '.join(sorted({m.group(1) for m in re.finditer(r'^@@ (\S+) ', open('$WORK/published.plan').read(), re.M)})))
")
# shellcheck disable=SC2086
m read $paths >"$WORK/served.out" 2>&1
want 0 $? "the files the published plan names can be read"
out=$(m write --dry-run "$WORK/published.plan" 2>&1); rc=$?
want 0 "$rc" "and the plan mrw publishes to a host is one mrw itself accepts"
# "0 failed" contains the word, so match a VERDICT line: `fail` or `skip` in the
# first column. A substring check here would have been green on every run.
# The rendered report prints a failure as `FAIL` and a skipped sibling as
# `skip`, so this must be case-insensitive: matching only lowercase bound on
# this two-hunk example purely because a sibling gets skipped, and a ONE-hunk
# failing plan would have walked straight past it.
grep -qiE '^(fail|skip)' <<<"$out" \
  && bad "a hunk of the published example failed: $out" \
  || ok "and every hunk of the published example passes"
# The pair: the same plan with its pattern changed to one the tree does not
# hold is refused, so the pass above is the pattern resolving, not a pattern
# nothing checks.
python3 - "$WORK/published.plan" "$WORK/unmatched.plan" <<'PY'
import re,sys
plan=open(sys.argv[1]).read()
bad,n=re.subn(r'^(@@ \S+ )/[^/]+/( )', r'\1/^no such line in the example tree$/\2', plan, flags=re.M)
assert n==1, "the published plan carries %d pattern-addressed hunk(s), want exactly one" % n
open(sys.argv[2],"w").write(bad)
PY
want 0 $? "the published plan addresses one hunk by pattern"
out=$(m write --dry-run "$WORK/unmatched.plan" 2>&1)
want 1 $? "and the same plan with a pattern that matches nothing is refused"
# Exit 1 is shared by every refused hunk, so the reason is asserted too (Codex on #261).
grep -q 'matched no line' <<<"$out" \
  && ok "and the refusal says the pattern matched no line" \
  || bad "the unmatched-pattern plan was refused for another reason: $out"

# 44. ADR-012 T2: the machine-readable half of the contract says what it means.
#
# ADR-011 made the output shapes GENERATED from the Go types, which is right and
# left them mute: a caller was told `failed` is an integer and never what it
# counts. This row reads the descriptions off the SHIPPED BINARY, at every
# depth — the fields worth describing are one level down, in a hunk's verdict
# and a file's `written`.
fixture

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
want 0 $? "tools/list answers"
python3 - "$out" <<'PY'
import json,sys
tools=json.loads(sys.argv[1])["result"]["tools"]
missing=[]; seen=[]
def walk(schema, where, prefix=""):
    for name,p in (schema.get("properties") or {}).items():
        path=prefix+name
        seen.append("%s:%s" % (where,path))
        if not (p.get("description") or "").strip(): missing.append("%s:%s" % (where,path))
        walk(p, where, path+".")
        for sub in ("items","additionalProperties"):
            if isinstance(p.get(sub), dict): walk(p[sub], where, path+".")
# Only the tools that DECLARE a schema: mrw_read's went with its
# structuredContent (ADR-023, §61); its receipt's descriptions are held by
# TestEveryOutputSchemaPropertyIsDescribed, which walks readSchema() by name.
declared=[t for t in tools if "outputSchema" in t]
assert [t["name"] for t in declared]==["mrw_write"], "declared schemas: %r, want mrw_write only" % [t["name"] for t in declared]
for t in declared:
    walk(t["outputSchema"], t["name"])
assert not missing, "undescribed propert(ies): %s" % ", ".join(missing)
# A walk that never descended would find only the top level and report success.
assert len(seen) >= 10, "the walk found only %d properties, so it is not descending: %s" % (len(seen), seen)
assert any(s.startswith("mrw_write:hunks.status") for s in seen), "the walk never reached a hunk verdict"
# ADR-090: a floor survives the loss of a whole container, so a file's landing is named too.
assert "mrw_write:files.written" in seen, "the walk never reached a file's written flag"
PY
[ $? -eq 0 ] && ok "every property of every declared outputSchema says what it means, at every depth" \
             || bad "the declared output schemas still describe only their types"
# 45. ADR-013 T2: a pattern address resolves exactly once, or it is refused —
# and it is never a way to edit a file you have not read.
#
# The fixture here is written by hand and the plans bend to it, deliberately.
# ADR-012 shipped a mutant that survived because its fixture was generated FROM
# the plan, so the plan's own guard could not fail. `func (s *Store) Get`
# appearing twice below is the ordinary shape of a Go file, not a contrivance.
fixture
mkdir -p "$R/store"
cat > "$R/store/store.go" <<'GO'
package store

// Get returns a row. See func (s *Store) Get below.
func (s *Store) Get(id string) (string, bool) {
	r, ok := s.rows[id]
	return r, ok
}

func (s *Store) Put(id, v string) {
	s.rows[id] = v
}
GO

m read store/store.go >"$WORK/served.out" 2>&1
want 0 $? "the fixture reads"

out=$(printf '@@ store/store.go /^func \\(s \\*Store\\) Put/ replace\nfunc (s *Store) Put(id, v string) { s.rows[id] = v }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a pattern that matches exactly one line applies"
grep -q '^ok .*/\^func' <<<"$out" \
  && ok "and the verdict echoes the PATTERN the caller wrote, not the line it resolved to" \
  || bad "the verdict does not name the pattern: $out"

# THE ROW: two matches is a refusal that names both lines, never a choice.
m read store/store.go >"$WORK/served.out" 2>&1
out=$(printf '@@ store/store.go /func \\(s \\*Store\\) Get/ replace\n// no\n' | m write - 2>&1); rc=$?
want 1 "$rc" "an ambiguous pattern is refused"
grep -qi 'matched 2 lines' <<<"$out" \
  && ok "and the refusal says how many it matched" \
  || bad "the refusal does not say how many: $out"
grep -qE 'lines 3, 4' <<<"$out" \
  && ok "and names them, so the caller can narrow it or address by number" \
  || bad "the refusal does not name the matching lines: $out"
grep -q 'NOTHING WRITTEN' <<<"$out" \
  && ok "and nothing was written" \
  || bad "an ambiguous pattern wrote something"

# A pattern must NOT be a ledger bypass: the resolved line still has to be read.
fixture
mkdir -p "$R/store"
printf 'package store\n\nfunc (s *Store) Put(id, v string) {\n\ts.rows[id] = v\n}\n' > "$R/store/store.go"
out=$(printf '@@ store/store.go /^func \\(s \\*Store\\) Put/ replace\n// no\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a pattern against an unread file is refused"
grep -q 'has not been read' <<<"$out" \
  && ok "and refused as UNREAD, so a pattern is not a way past the ledger" \
  || bad "the refusal was not the unread one: $out"

# The RANGE form, which the first cut of this record shipped with no resolution
# test at all — and whose own headline example failed, because `^}` closes every
# function and exactly-once was being applied to the end as well as the start.
# The end is a delimiter: first match AT OR AFTER the start.
fixture
mkdir -p "$R/store"
cat > "$R/store/store.go" <<'GO'
package store

func (s *Store) Get(id string) (string, bool) {
	r, ok := s.rows[id]
	return r, ok
}

func (s *Store) Put(id, v string) {
	s.rows[id] = v
}
GO
m read store/store.go >"$WORK/served.out" 2>&1
out=$(printf '@@ store/store.go /^func \\(s \\*Store\\) Get/,/^\\}/ replace anchor="Store) Get(id"\nfunc (s *Store) Get(id string) (string, bool) { return s.rows[id], true }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "the range form applies on a file where the end pattern matches twice"
grep -q 'func (s \*Store) Put' "$R/store/store.go" \
  && ok "and it stopped at the FIRST closing brace, leaving Put intact" \
  || bad "the range ran past the first closing brace: $(cat "$R/store/store.go")"

# An end that only matches ABOVE the start delimits nothing and is refused.
m read store/store.go >"$WORK/served.out" 2>&1
out=$(printf '@@ store/store.go /^func \\(s \\*Store\\) Put/,/^package/ replace\n// x\n' | m write - 2>&1); rc=$?
want 1 "$rc" "an end pattern above the start is refused"
grep -q 'above the start' <<<"$out" \
  && ok "and the refusal says why, rather than silently inverting the range" \
  || bad "the refusal does not explain the inversion: $out"

# Two address forms in ONE grammar must be refused on the same inputs.
out=$(printf '@@ store/nope.go /x/ create\n// x\n' | m write - 2>&1); rc=$?
# Exit 2, not 1: both are PARSE errors — a malformed document, refused before
# anything touches the tree — where 1 is a hunk that parsed and failed to apply.
want 2 "$rc" "create refuses a pattern address exactly as it refuses a number"
out=$(printf '@@ store/store.go /^package/,/^func/ insert-after\n// x\n' | m write - 2>&1); rc=$?
want 2 "$rc" "and an insertion refuses a RANGE, whether it is written 3-6 or /a/,/b/"

# 46. ADR-013 T3: the rule the wire teaches is the rule the binary enforces.
#
# ADR-012 taught an enum the engine never sent and two independent reviewers
# caught it. This row is that lesson applied to a RULE rather than a value: the
# exactly-once refusal is asserted on the wire and in the binary, in one place.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
i=json.loads(sys.argv[1])["result"]["instructions"]
assert "/regexp/" in i, "the instructions never teach the pattern form"
assert "EXACTLY ONE" in i, "the instructions teach the form without the exactly-once rule"
assert "have not read" in i, "the instructions do not say a pattern is still subject to the ledger"
PY
[ $? -eq 0 ] && ok "the wire teaches the pattern form, its exactly-once rule, and the ledger caveat" \
             || bad "the taught rule is incomplete"
# 50. ADR-016 T1: the surface says what it is NOT.
#
# A registered MCP tool outcompetes a CLI an agent must remember exists — it
# arrives with a schema, in the tool list, while the CLI is a string in a file
# the agent may never read. Observed directly: agents settling for the smaller
# surface because it is the one they can see. So the wire routes them, and this
# row checks the routing is on the wire AND that the flags it names are real.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
i=json.loads(sys.argv[1])["result"]["instructions"]
# ADR-075: "one writer per checkout" replaced "serialized" — the CLI's writes
# take turns too, so serialized writes are no longer this surface's advantage.
for w in ("--context","--check","--root","shell","one writer per checkout","ONE fixed checkout","ack","LICENSES NOTHING"):
    assert w in i, "the instructions never mention %r" % w
# The routing must come BEFORE the format details. It is no longer literally
# first: it is merged into the WHEN TO REACH paragraph, because a separate
# opening block competed with the old "Use mrw_read and mrw_write" guidance
# further down and told a shell-capable caller two different things. What has
# to hold is the ORDER — a caller meets the choice before it meets the grammar.
assert i.index("WHICH SURFACE") < i.index("READING."), "the routing comes after the format details, so the choice is made before it is offered"
assert len(i.encode()) <= 4096, "instructions are %d bytes, over the bound every session pays" % len(i.encode())
# ⚠ THE RULE ITSELF, read out of internal/mcp/ack.go's AckRule constant and
# required VERBATIM on the wire. Listing tokens is what let a mutant negate the
# clause while "ack", "LICENSES NOTHING" and the rest stayed present — twice.
import re as _re
_src=open("internal/mcp/ack.go").read()
_m=_re.search(r'const AckRule = "([^"]*)"', _src)
assert _m, "internal/mcp/ack.go no longer declares AckRule, so nothing pins what the surfaces must say"
# ⚠ AN INDEPENDENT ORACLE. Taking the expected text from the implementation and
# then finding it in the implementation's output is satisfied by an EMPTY rule.
# The sentence is written here so that weakening the constant fails.
_want = ("Send an id in ack only if you hold BOTH its open and close markers AND counted the N "
         "numbered lines the open marker says follow: one marker is not enough, because a cut "
         "starting inside a span leaves the other end.")
assert _m.group(1) == _want, "AckRule no longer states the rule ADR-031 decided:\n%s" % _m.group(1)
assert _want in i, "the served instructions do not carry the acknowledgement rule verbatim"
PY
[ $? -eq 0 ] && ok "the handshake routes a shell-capable caller to the CLI, first" \
             || bad "the surface does not say it is the smaller one"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
for t in json.loads(sys.argv[1])["result"]["tools"]:
    assert "CLI" in t["description"], "%s does not route to the CLI" % t["name"]
PY
[ $? -eq 0 ] && ok "and both descriptions do too, for a host that ignores instructions" \
             || bad "a host reading only tools/list is not routed"

# THE ROW: the advice must be TRUE. Every flag it names comes from the CLI's own
# help, so a rename turns this red instead of leaving the wire recommending a
# flag that is gone.
# PER SUBCOMMAND, and --root against the ROOT help. A concatenated blob would
# pass a wire text that recommended --check to mrw_read's caller, and it would
# pass `-C` for choosing a checkout — which is the context flag after `read`,
# so `mrw read -C DIR` errors. That was the first cut of this record's advice.
missing=""
grep -q -- '--files-from' <<<"$(m read --help 2>&1)"  || missing="$missing read:--files-from"
grep -q -- '--check'      <<<"$(m write --help 2>&1)" || missing="$missing write:--check"
grep -q -- '--root'       <<<"$(m --help 2>&1)"       || missing="$missing root:--root"
[ -z "$missing" ] \
  && ok "and every flag the wire recommends exists on the SUBCOMMAND it names" \
  || bad "the wire recommends flags that subcommand does not have:$missing"

# THE ROW THE FIRST CUT WOULD HAVE FAILED: the recommended checkout form must
# actually work against a second tree.
OTHER=$(mktemp -d); printf 'alpha\n' > "$OTHER/o.txt"
out=$("$MRW" --root "$OTHER" read o.txt 2>&1); rc=$?
want 0 "$rc" "the recommended --root form really points the CLI at another checkout"
grep -q 'alpha' <<<"$out" \
  && ok "and serves it" \
  || bad "the recommended form did not serve the other checkout: $out"
rm -rf "$OTHER"


# 51. ADR-017 T1: the MCP surface can FIND, and an oversized find is an index.
#
# The population this is for has no shell: an analyst on Claude Desktop cannot
# run `rg -l | mrw read --files-from -`, so over MCP "which files" was simply
# unanswerable. This row drives the real binary over a real tree.
#
# THE ROW THAT MATTERS is the second one. Serving matches is the easy half; the
# half that decides whether this is usable is what happens when the matches do
# not fit — and a refusal there is ADR-014's dead end reappearing on the
# ORDINARY case for this caller rather than an exotic one.
fixture
python3 -c "
import os
for i in range(40):
    with open('$R/doc%03d.csv' % i,'w') as f:
        f.write('a line that matches nothing\n')
        for j in range(400): f.write('the NEEDLE is here\n')
"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"grep":"NEEDLE"}}}\n' | m --root "$R" mcp 2>/dev/null)
MRW_BIN="$MRW" python3 - "$R" "$out" <<'PY'
import json, subprocess, sys, os
root, raw = sys.argv[1], sys.argv[2]
res = json.loads(raw)["result"]
sc = json.loads(res["content"][1]["text"])   # the receipt lives in content[1] (ADR-023)
assert "isError" not in res, "ADR-024: an oversized grep index SERVED an index, so the key must be ABSENT — a host truncates a flagged result and a shortened index names no gap"
assert sc["matches"] == 40, "the index reports %r matching files, want 40" % sc["matches"]
idx = sc["index"]
assert idx, "an oversized grep returned no index at all"
# THE ENTRIES MUST BE PATHS THAT ROUND-TRIP. Send the first one back to the real
# binary WITH the same grep, which is what the result tells the caller to do.
first = idx[0]
assert ":" not in first, "index entry %r carries an address; a pattern containing a slash makes that ambiguous" % first
req = {"jsonrpc":"2.0","id":1,"method":"tools/call",
       "params":{"name":"mrw_read","arguments":{"specs":[first],"grep":"NEEDLE"}}}
p = subprocess.run([os.environ["MRW_BIN"], "--root", root, "mcp"],
                   input=json.dumps(req)+"\n", capture_output=True, text=True)
back = json.loads(p.stdout.splitlines()[0])["result"]
assert back.get("isError") is not True, "index entry %r is not a spec the tool accepts" % first
body = "".join(c.get("text","") for c in back["content"])
assert "NEEDLE" in body, "reading index entry %r served no match" % first
# AND THE INDEX ITSELF MUST FIT THE CAP IT EXISTS TO RESPECT. An index built to
# avoid an oversized result that is ITSELF oversized is the spill this answer
# was added to prevent. Measured on the encoded line, not on the entry list:
# earlier cuts counted each entry once (650,000 chars) and then twice (210,289),
# because the JSON block is escaped again inside the envelope.
assert len(raw) <= 200000, "the index result is %d characters, over the 200000 cap it exists to respect" % len(raw)
PY
[ $? -eq 0 ] && ok "an oversized grep returns an index whose entries really read" \
             || bad "the index is missing, wrong, or not made of specs"

# And a grep that FITS serves content and records it, so the ledger licenses a
# write to what the caller was actually shown.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["doc000.csv"],"grep":"NEEDLE"}}}\n' | m --root "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
res = json.loads(sys.argv[1])["result"]
assert res.get("isError") is not True, "a grep that fits should not be an error"
assert json.loads(res["content"][1]["text"])["observed"], "a served grep recorded nothing, so it licenses no write"
PY
[ $? -eq 0 ] && ok "and a grep that fits serves content and records what it served" \
             || bad "a fitting grep did not serve or did not record"

# The grammar matches the CLI's: a range and a grep are two answers to one
# question, and BOTH surfaces say so.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["doc000.csv:1-2"],"grep":"NEEDLE"}}}\n' | m --root "$R" mcp 2>/dev/null)
grep -q 'two answers to one question' <<<"$out" \
  && ok "and a ranged spec with grep is refused, in the CLI's own words" \
  || bad "the two surfaces disagree on the grammar: $out"
# THE INDEX MUST PAGE TO EXHAUSTION, AND FIT THE CAP WHILE DOING IT.
#
# This fixture exists because the 40-file one above cannot reach either failure:
# its index is tiny, so a cap assertion passes trivially and a paging assertion
# never fires. Many small files is also the shape the Desktop population
# actually has.
fixture
# 12000, not 8000: ADR-023 removed the structuredContent copy of the index, so
# an 8000-entry index now FITS the cap, and the fixture grew past it.
python3 -c "
for i in range(12000):
    f=open('$R/document%05d.csv' % i,'w'); f.write('x\n'); f.write('the NEEDLE is here\n')
"
MRW_BIN="$MRW" python3 - "$R" <<'PY'
import json, subprocess, sys, os
root = sys.argv[1]
mrw = os.environ["MRW_BIN"]

def call(args):
    req = {"jsonrpc":"2.0","id":1,"method":"tools/call",
           "params":{"name":"mrw_read","arguments":args}}
    p = subprocess.run([mrw, "--root", root, "mcp"],
                       input=json.dumps(req)+"\n", capture_output=True, text=True)
    line = p.stdout.splitlines()[0]
    # THE ENCODED LINE is what the host receives, so it is what the cap governs.
    assert len(line) <= 200000, "an index result was %d characters, over the 200000 cap it exists to respect" % len(line)
    return json.loads(line)["result"]

res = call({"grep": "NEEDLE"})
sc = json.loads(res["content"][1]["text"])   # the receipt lives in content[1] (ADR-023)
assert sc["matches"] == 12000, "matches = %r, want 12000" % sc["matches"]
assert len(sc["index"]) < 12000, "the index served all 12000 entries; this fixture exists to overflow it"
assert sc["next_index"], "the index was cut short and named no resume point"

# FOLLOW IT TO EXHAUSTION and compare the union, rather than checking that a
# continuation is present. next_index named the FIRST WITHHELD file in an
# earlier cut, and `after` skips everything at or before its value — so the
# cursor skipped its own file and lost exactly one per page boundary. A
# presence check passed that; only the union caught it.
seen_paths, nxt, pages = set(sc["index"]), sc["next_index"], 1
while nxt:
    pages += 1
    assert pages <= 30, "following next_index did not terminate"
    r = call({"grep": "NEEDLE", "after": nxt})
    s = json.loads(r["content"][1]["text"])
    if "index" in s:
        page = s["index"]
        assert page, "page %d came back empty; after=%r is not making progress" % (pages, nxt)
        assert not (seen_paths & set(page)), "page %d repeats entries already shown" % pages
        seen_paths |= set(page)
        nxt = s.get("next_index") or ""
    else:
        seen_paths |= set(s["observed"].keys())
        nxt = ""
assert len(seen_paths) == 12000, "paging to exhaustion yielded %d distinct files, want 12000" % len(seen_paths)
PY
[ $? -eq 0 ] && ok "an index pages to exhaustion, loses nothing, and stays under the cap" \
             || bad "the index loses entries, does not terminate, or exceeds the cap"


# 52. ADR-017 T2: the wire teaches finding, and no longer calls --grep a thing
# only the CLI has.
#
# ADR-016 shipped "only it has --grep". That sentence became FALSE the moment
# §51 passed, and NOTHING ELSE COULD HAVE CAUGHT IT: §50 asserts the flags the
# routing names EXIST in the CLI's help, and --grep still exists. A gate that
# checks a thing exists cannot catch a thing that is present and false.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
i = json.loads(sys.argv[1])["result"]["instructions"]
assert "grep" in i, "the wire never mentions grep, so an MCP-only caller cannot learn it can search"
assert "INDEX" in i and "no content" in i, "the wire does not say an oversized grep returns an index carrying NO content"
assert "only it has --grep" not in i, "the wire still calls --grep a CLI exclusive, which it is not"
# BYTES, NOT RUNES. Python's len() on a str counts code points and Go's
# len() counts bytes, so these two gates were measuring different things
# against the same constant: 4,073 here against 4,091 there, 18 bytes of
# headroom the contract believed it had and did not. The text is full of em
# dashes, so the gap grows with every one added. Caught while writing this
# off as non-blocking — which is the habit this corpus keeps recording.
assert len(i.encode()) <= 4096, "instructions are %d bytes, over the bound every session pays" % len(i.encode())
PY
[ $? -eq 0 ] && ok "the handshake teaches finding and stops claiming --grep for the CLI" \
             || bad "the wire is wrong about finding"

# THE ROW THAT BINDS: every long flag the wire still names as the CLI's must NOT
# be an argument mrw_read declares. This is the direction §50 cannot check.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
ins=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}\n' | m mcp 2>/dev/null)
python3 - "$out" "$ins" <<'PY'
import json,re,sys
tools = json.loads(sys.argv[1])["result"]["tools"]
i = json.loads(sys.argv[2])["result"]["instructions"]
props = {}
for t in tools:
    if t["name"] == "mrw_read":
        props = t["inputSchema"].get("properties", {})
for m in re.findall(r"--[a-z][a-z-]+", i):
    arg = m[2:].replace("-", "_")
    assert arg not in props, "the wire names %s as the CLI's, but mrw_read declares %r" % (m, arg)
PY
[ $? -eq 0 ] && ok "and no flag it calls the CLI's is an argument this tool has" \
             || bad "the routing tells callers to leave a surface that has the flag"

# 53. ADR-018: a root nobody named, that cannot be a project, is refused.
#
# Issue #81. The reported symptom is a server bound to `/`, but the guard is on
# the SOURCE: with no --root and no CLAUDE_PROJECT_DIR the server serves
# whichever directory the host happened to launch it in, and everything
# downstream is then correct about a tree nobody asked about — on a surface
# that also writes.
#
# EXPLICITNESS IS THE LICENCE, and the third row is the one that matters: an
# explicit --root / must still be SERVED. A guard that refused it would be a
# list of paths somebody found distasteful rather than a rule, and it would
# break the population whose documents really do live under a wide root.
( cd / && env -u CLAUDE_PROJECT_DIR "$MRW" mcp < /dev/null > /dev/null 2>&1 )
want 2 "$?" "a fallback onto the filesystem root is refused"

out=$( cd / && env -u CLAUDE_PROJECT_DIR "$MRW" mcp < /dev/null 2>&1 )
grep -q -- '--root' <<<"$out" \
  && ok "and the refusal names the flag that fixes it" \
  || bad "the refusal is a dead end, it does not say how to name a tree: $out"

( cd / && env -u CLAUDE_PROJECT_DIR "$MRW" --root / mcp < /dev/null > /dev/null 2>&1 )
want 0 "$?" "but an EXPLICIT --root / is still served — explicitness is the licence"

R53=$(mktemp -d)
( cd "$R53" && env -u CLAUDE_PROJECT_DIR "$MRW" mcp < /dev/null > /dev/null 2>&1 )
want 0 "$?" "and an ordinary directory reached by fallback still serves"
rm -rf "$R53"

# 54. ADR-020: the served-size curve is measured, not asserted — and the
# scorer can see a wrong line.
#
# A harness that has only ever been shown correct plans proves nothing: a
# scorer that always says "hit" passes every such row. So this row generates
# a real trial with the BUILT curve binary, authors one plan at the planted
# line and one at a distractor's byte-identical line, and requires the two
# verdicts to differ in the right direction. It also pastes a result from
# "another trial" and requires a refusal at exit 2, because a manifest
# answered from a different cell would otherwise score cleanly and mean
# nothing. The engine is untouched by this record; it is only called.
CURVE="$WORK/curve"
go build -o "$CURVE" ./cmd/curve
R54=$(mktemp -d)
"$CURVE" generate -out "$R54/cell" -bytes 6000 -position middle -distractors 4 -seed 54 > "$R54/generate.out" 2>&1
want 0 "$?" "curve generate writes a trial from the built binary"
target=$(grep -o '"line": *[0-9]*' "$R54/cell/answer.json" | grep -o '[0-9]*$')
trial=$(grep -o '"trial_id": *"[0-9a-f]*"' "$R54/cell/manifest.json" | grep -o '[0-9a-f]*"$' | tr -d '"')
served=$(grep -o '"served_bytes": *[0-9]*' "$R54/cell/manifest.json" | grep -o '[0-9]*$')
wrong=$(grep -n '^timeout = 30$' "$R54/cell/tree/services.conf" | cut -d: -f1 | grep -vx "$target" | head -1)
[ -n "$target" ] && [ -n "$wrong" ] && [ "${served:-0}" -ge 6000 ] \
  && ok "the trial planted line $target, has a byte-identical distractor at line $wrong, and served $served bytes" \
  || bad "the trial did not come out as generated: target=$target wrong=$wrong served=$served"
result54() {
  python3 -c 'import json,sys; print(json.dumps({"trial_id":sys.argv[1],"served_bytes":int(sys.argv[2]),"plan":sys.argv[3]}))' "$1" "$2" "$3"
}
result54 "$trial" "$served" "@@ services.conf $target replace
timeout = 45
" > "$R54/right.json"
result54 "$trial" "$served" "@@ services.conf $wrong replace
timeout = 45
" > "$R54/wrong.json"
result54 "someone-else" "$served" "@@ services.conf $target replace
timeout = 45
" > "$R54/other.json"

# Assertions decode the JSON rather than grepping it: `"n": 2` is a SUBSTRING of
# `"n": 20`, so a grep-shaped check passes on a tally twice the size it claims.
# python3 is already required by earlier rows.
j54() { python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(json.dumps(d))' "$1"; }
assert54() {
  python3 - "$1" "$2" <<'PY'
import json, sys
doc = json.load(open(sys.argv[1]))
for clause in sys.argv[2].split(";"):
    path, want = clause.split("=", 1)
    node = doc
    for part in path.split("."):
        node = node[int(part)] if part.isdigit() else node[part]
    if json.dumps(node, sort_keys=True) != want and str(node) != want:
        raise SystemExit("%s is %r, want %s" % (path, node, want))
PY
}

"$CURVE" score -cell "$R54/cell" -result "$R54/right.json" > "$R54/right.out"
want 0 "$?" "a plan at the planted line is scored"
assert54 "$R54/right.out" "outcome=hit;changed=[$target];target=$target" \
  && ok "and it is a hit that changed exactly the planted line" \
  || bad "the planted line did not score a clean hit: $(j54 "$R54/right.out")"

"$CURVE" score -cell "$R54/cell" -result "$R54/wrong.json" > "$R54/wrong.out"
want 0 "$?" "a plan at a distractor's identical line is scored, not refused — it parsed and applied"
assert54 "$R54/wrong.out" "outcome=miss;changed=[$wrong]" \
  && ok "and it is a MISS: the scorer can see a wrong line that applied cleanly" \
  || bad "a wrong line did not score a miss: $(j54 "$R54/wrong.out")"

# The plan that fixes the right line and ALSO writes elsewhere. Diffing only the
# target file scores this a clean hit, which is not "exactly the planted line".
result54 "$trial" "$served" "@@ services.conf $target replace
timeout = 45
$(printf '@@ extra.txt - create\nuninvited\n')" > "$R54/both.json"
"$CURVE" score -cell "$R54/cell" -result "$R54/both.json" > "$R54/both.out"
want 0 "$?" "a plan that fixes the target and also creates another file is scored"
assert54 "$R54/both.out" 'outcome=miss;touched=["extra.txt"]' \
  && ok "and it is a MISS naming the file it also wrote" \
  || bad "an extra written file was scored as a hit: $(j54 "$R54/both.out")"

"$CURVE" score -cell "$R54/cell" -result "$R54/other.json" > /dev/null 2> "$R54/other.err"
want 2 "$?" "a result echoing another trial's id is refused, not scored"
grep -q 'trial' "$R54/other.err" \
  && ok "and the refusal names the trial it was expecting" \
  || bad "the refusal does not say what mismatched: $(cat "$R54/other.err")"

# The manifest must not carry the stratum or the distractor count: together they
# name the target's block, so a client could count instead of read.
grep -qE '"(position|distractors)"' "$R54/cell/manifest.json" \
  && bad "manifest.json leaks the stratum or the distractor count: $(cat "$R54/cell/manifest.json")" \
  || ok "and the client's manifest carries no ground truth"

# Repeats of ONE cell must tally together. Padding overshoots by a seed-dependent
# amount, so keying on measured bytes would make N repetitions N cells of one.
for s in 1 2 3; do
  "$CURVE" generate -out "$R54/rep$s" -bytes 6000 -position middle -distractors 4 -seed "$s" > /dev/null
  rt=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["trial_id"])' "$R54/rep$s/manifest.json")
  rb=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["served_bytes"])' "$R54/rep$s/manifest.json")
  rl=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["line"])' "$R54/rep$s/answer.json")
  result54 "$rt" "$rb" "$(printf '@@ services.conf %s replace\ntimeout = 45\n' "$rl")" > "$R54/rep$s.json"
  "$CURVE" score -cell "$R54/rep$s" -result "$R54/rep$s.json" > "$R54/rep$s.out"
done
"$CURVE" tally "$R54/rep1.out" "$R54/rep2.out" "$R54/rep3.out" > "$R54/reps.out"
want 0 "$?" "tally accepts three repeats of one size cell"
assert54 "$R54/reps.out" "0.n=3;0.hits=3;0.cell=6000" \
  && [ "$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))))' "$R54/reps.out")" = 1 ] \
  && ok "and groups them into ONE cell of 3, not three cells of one" \
  || bad "repeats did not group: $(j54 "$R54/reps.out")"

"$CURVE" tally "$R54/right.out" "$R54/wrong.out" > "$R54/tally.out"
want 0 "$?" "tally accepts the two scores"
assert54 "$R54/tally.out" "0.n=2;0.hits=1;0.refused=0;0.misses=1" \
  && [ "$(python3 -c 'import json,sys; print(len(json.load(open(sys.argv[1]))))' "$R54/tally.out")" = 1 ] \
  && ok "and reports 1 of 2 in exactly one cell, with nothing in the refused column" \
  || bad "tally did not report the cell: $(j54 "$R54/tally.out")"

# A malformed outcome must be REFUSED, not bucketed: silently counting it as a
# refusal is how bad score data enters the measurement.
python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); d["outcome"]="garbage"; json.dump(d, open(sys.argv[2],"w"))' "$R54/right.out" "$R54/garbage.out"
"$CURVE" tally "$R54/garbage.out" > /dev/null 2>&1
want 2 "$?" "and a score carrying an unrecognised outcome is refused, not tallied"
rm -rf "$R54"
# 55. ADR-022: a path-scoped rule arrives on an mrw read too, through the hook.
#
# Claude Code injects a `.claude/rules/*.md` file with `paths:` frontmatter
# only when its own Read tool reads a matching file (issue #86). This row
# drives .claude/hooks/rules-on-read.py the way the harness does — JSON on
# stdin, the documented envelope on stdout, $CLAUDE_PROJECT_DIR in the
# environment — and DECODES what comes back rather than grepping it, because
# `"n": 2` is a substring of `"n": 20` and body text is a substring of a
# broken envelope. Every case that must deliver is paired with one that must
# not. State goes under the fixture's HOME, so nothing is left in /tmp.
# The hook driven below is the one settings.json WIRES, resolved the way the
# harness resolves it — ${CLAUDE_PROJECT_DIR} is this checkout — so an entry
# that names a file which is not there fails this row, not a session.
HOOK=$(CLAUDE_PROJECT_DIR="$PWD" python3 -c '
import json, os, re
entries = json.load(open(".claude/settings.json"))["hooks"]["PostToolUse"]
ours = [e for e in entries if "rules-on-read" in json.dumps(e)]
assert len(ours) == 1, entries
e = ours[0]
assert set(e["matcher"].split("|")) == {"Bash", "Write", "mcp__mrw__mrw_read", "mcp__mrw__mrw_write"}, e["matcher"]
assert len(e["hooks"]) == 1 and e["hooks"][0]["type"] == "command", e
m = re.fullmatch(r"python3 \"\$\{CLAUDE_PROJECT_DIR\}/([^\"]+)\"", e["hooks"][0]["command"])
assert m, e["hooks"][0]["command"]
print(os.path.join(os.environ["CLAUDE_PROJECT_DIR"], m.group(1)))' 2>/dev/null)
[ -n "$HOOK" ] && [ -f "$HOOK" ] \
  && ok "settings.json wires exactly the four tools to a command whose file, resolved through CLAUDE_PROJECT_DIR, exists" \
  || bad "the settings entry does not wire the hook this row drives: ${HOOK:-no command resolved}"
R55=$(mktemp -d)
mkdir -p "$R55/proj/.claude/rules" "$R55/proj/docs/adr" "$R55/proj/pkg" "$R55/home"
printf -- '---\npaths:\n  - "docs/adr/**"   # the records\n  - "**/*_test.go"\n---\n\nSCOPED RULE BODY 55\n' > "$R55/proj/.claude/rules/scoped.md"
printf -- '---\npaths: ["src/**/*.{ts,tsx}", "*.md"]\n---\n\nINLINE RULE BODY 55\n' > "$R55/proj/.claude/rules/inline.md"
printf -- '# plain rule\n\nPLAIN RULE BODY 55\n' > "$R55/proj/.claude/rules/plain.md"
printf 'a record\n' > "$R55/proj/docs/adr/x.md"
printf 'spaced\n' > "$R55/proj/docs/adr/my file.md"
printf 'package pkg\n' > "$R55/proj/pkg/a_test.go"
printf 'readme\n' > "$R55/proj/README.md"
mkdir -p "$R55/proj/src/a"; printf 'ts\n' > "$R55/proj/src/a/b.tsx"
mk55() {  # session, tool, tool_input JSON [, tool_response JSON [, cwd]] -> the JSON the harness would put on stdin
  python3 -c 'import json,sys
d={"hook_event_name":"PostToolUse","session_id":sys.argv[1],"cwd":sys.argv[6] or sys.argv[2],"tool_name":sys.argv[3],"tool_input":json.loads(sys.argv[4])}
if sys.argv[5]: d["tool_response"]=json.loads(sys.argv[5])
print(json.dumps(d))' "$1" "$R55/proj" "$2" "$3" "${4:-}" "${5:-}"
}
# Every hook child runs under an alarm. A mutant that hangs is a legitimate
# mutant and bounding it is the harness's job: on 2026-09-04 the "regex-seg"
# mutant — a catastrophic pattern in the hook — was killed by its row and then
# ran for fifteen hours at 80 % CPU under parent 1, twice, because nothing here
# put a ceiling on the child (#101). The alarm survives exec, so python3 gets
# SIGALRM and exits 142, which every row reads as a failure. ALARM55 is
# overridable so the row that proves this can use a short one.
hook55() {  # the same arguments -> the hook's stdout, under the fixture's HOME and project
  mk55 "$@" | env HOME="$R55/home" XDG_CACHE_HOME="$R55/home/.cache" CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK"
}
closed55() {  # the same arguments -> the hook run with stdout genuinely closed
  # Closed AFTER env, by the shell that execs python3. Closing it before env
  # tests the host, not the hook: uutils coreutils 0.8.0 reopens a closed
  # standard descriptor to /dev/null before exec, so on such a machine the
  # hook wrote its envelope into /dev/null and kept the claim, and the row
  # below went red for a reason that had nothing to do with this repository
  # (found by review on a Linux host, 2026-09-04; GNU env leaves it closed,
  # which is why CI and darwin were both green).
  # The alarm sits OUTSIDE the closing shell. Inside it, perl reopened fd 1
  # before its exec — fstat(1) succeeded while write(1) still failed — and
  # the fixture no longer meant "stdout genuinely closed" (found by review,
  # 2026-09-05, perl 5.34 on darwin). The alarm survives both execs.
  mk55 "$@" | env HOME="$R55/home" XDG_CACHE_HOME="$R55/home/.cache" CLAUDE_PROJECT_DIR="$R55/proj" \
    perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" bash -c 'exec >&-; exec python3 "$1"' _ "$HOOK"
}
ctx55() {  # stdin: the hook's stdout -> the additionalContext, or "" ; exit 1 on a bad envelope
  python3 -c 'import json,sys
s=sys.stdin.read()
if not s: sys.exit(0)
d=json.loads(s); h=d["hookSpecificOutput"]
assert set(d)=={"hookSpecificOutput"} and set(h)=={"hookEventName","additionalContext"} and h["hookEventName"]=="PostToolUse", d
print(h["additionalContext"])'
}
ctx=$(hook55 s1 Bash '{"command":"mrw read docs/adr/x.md:1-3 README.md"}' | ctx55); rc=$?
want 0 "$rc" "a Bash mrw read comes back as the documented envelope and nothing else"
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "and delivers the rule whose glob matches the file the read served" \
  || bad "no rule delivered for docs/adr/x.md: $ctx"
grep -q 'PLAIN RULE BODY 55' <<<"$ctx" \
  && bad "an unconditional rule was delivered too, and the harness had already loaded it" \
  || ok "and not the unconditional rule, which the harness already loaded"
ctx=$(hook55 s1 Bash '{"command":"mrw read docs/adr/x.md:5"}' | ctx55)
[ -z "$ctx" ] && ok "a rule already delivered in this session is not delivered again" || bad "re-delivered: $ctx"
ctx=$(hook55 s2 Bash '{"command":"cat README.md"}' | ctx55)
grep -q 'INLINE RULE BODY 55' <<<"$ctx" && ! grep -q 'SCOPED' <<<"$ctx" \
  && ok "a slash-less pattern in an inline list matches a root file, and only that rule" \
  || bad "root-only inline glob: $ctx"
ctx=$(hook55 s2c Bash '{"command":"cat docs/adr/x.md"}' | ctx55)
! grep -q 'INLINE RULE BODY 55' <<<"$ctx" \
  && ok "and it matches nothing deeper: *.md is a file at the root, not every .md in the tree" \
  || bad "a slash-less glob reached a nested file: $ctx"
# The one exception, and it is the safe direction: a bare ** is the whole tree.
# Reading it as root-only would silently drop a rule whose author wrote the
# pattern that means everything.
printf -- '---\npaths: ["**"]\n---\n\nEVERY RULE BODY 55\n' > "$R55/proj/.claude/rules/every.md"
ctx=$(hook55 s2d Bash '{"command":"cat docs/adr/x.md"}' | ctx55)
grep -q 'EVERY RULE BODY 55' <<<"$ctx" && ok "a bare ** matches a file two directories deep" || bad "** did not match a nested file: $ctx"
ctx=$(hook55 s2e Bash '{"command":"cat README.md"}' | ctx55)
grep -q 'EVERY RULE BODY 55' <<<"$ctx" && ok "and one at the root" || bad "** did not match a root file: $ctx"
rm -f "$R55/proj/.claude/rules/every.md"
ctx=$(hook55 s2b Bash '{"command":"cat src/a/b.tsx"}' | ctx55)
grep -q 'INLINE RULE BODY 55' <<<"$ctx" \
  && ok "a brace group inside an inline list is one glob, not two" \
  || bad "src/**/*.{ts,tsx} did not match src/a/b.tsx: $ctx"
ctx=$(hook55 s3 mcp__mrw__mrw_read '{"specs":["pkg/a_test.go:/package/"]}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "an mrw_read spec with a pattern address is matched by its path, and **/*_test.go crosses directories" \
  || bad "mrw_read spec not matched: $ctx"
# `**` must match ZERO directories and TWO, not exactly one: a `**` that
# degraded to `*` still matches every one-deep fixture above.
printf 'package top\n' > "$R55/proj/top_test.go"; mkdir -p "$R55/proj/pkg/sub"; printf 'package sub\n' > "$R55/proj/pkg/sub/deep_test.go"
ctx=$(hook55 s3b Bash '{"command":"cat top_test.go"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "**/*_test.go matches a root file: ** stands for zero directories" || bad "** did not match zero directories: $ctx"
ctx=$(hook55 s3c Bash '{"command":"cat pkg/sub/deep_test.go"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "and a file two directories deep: ** stands for several" || bad "** did not cross two directories: $ctx"
printf 'package pkg\n' > "$R55/proj/pkg/new_test.go"
ctx=$(hook55 s4 mcp__mrw__mrw_write '{"plan":"@@ pkg/new_test.go - create\npackage pkg\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a file an mrw_write plan just created is matched: the hook runs after the write" \
  || bad "create op not matched: $ctx"
# Whether mrw ACCEPTS a plan is not mirrored. The mirror built in the third
# round could only add silence — every case where it was stricter than mrw (a
# pattern it would not compile, an integer Go rejects and Python accepts) was
# a successful write that delivered nothing — so the first field of every
# header-shaped line the tokeniser can split is a candidate, counted and raw
# bodies included, and a plan mrw
# refuses delivers early for the files it names. Early beats silent.
ctx=$(hook55 s5 mcp__mrw__mrw_write '{"plan":"@@ README.md 1 replace body=1 raw=true\n@@ docs/adr/x.md 1 replace\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "a header-shaped line inside a raw counted body delivers early for the file it names" || bad "a raw body line was suppressed: $ctx"
ctx=$(hook55 s5c mcp__mrw__mrw_write '{"plan":"@@ docs/adr/x.md 1 replace raw=true\nX\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "raw=true without body= is a plan mrw refuses, and it still delivers early for the file it names" || bad "a refused plan was suppressed: $ctx"
ctx=$(hook55 s5d mcp__mrw__mrw_write '{"plan":"﻿@@ \"docs/adr/my file.md\" 1 replace\nX\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a quoted path with a space behind a BOM is read as mrw reads it" \
  || bad "quoted header path not matched: $ctx"
ctx=$(hook55 s6 Read '{"file_path":"docs/adr/x.md"}' | ctx55)
[ -z "$ctx" ] && ok "the Read tool is left to the harness, so nothing arrives twice" || bad "the hook fired on Read: $ctx"
ctx=$(hook55 s7 Write "{\"file_path\":\"$R55/proj/docs/adr/x.md\"}" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "an absolute path inside the project is matched" || bad "absolute path not matched: $ctx"
# A tool_input of the wrong SHAPE costs the paths the input names, and nothing
# more: the result's own `==>` headers still deliver. Found in self-review —
# a `command` that arrived as a list raised inside shlex, and main()'s
# catch-all then delivered nothing at all, which is a silence rather than the
# early delivery this record prefers.
for bad_input in '{"command":["mrw","read"]}' '{"command":null}'; do
  ctx=$(hook55 "sh${RANDOM}" Bash "$bad_input" '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n"}' | ctx55)
  grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
    && ok "a Bash command of the wrong shape still delivers for the file its result served: ${bad_input:0:26}…" \
    || bad "a malformed command lost the served header: $ctx"
done
ctx=$(hook55 "sh${RANDOM}" mcp__mrw__mrw_read '{"specs":"docs/adr/x.md:1"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "and specs given as a string instead of a list" || bad "malformed specs lost the served header: $ctx"
ctx=$(hook55 "sh${RANDOM}" mcp__mrw__mrw_write '{"plan":{"not":"a string"}}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "and a plan given as an object instead of a string" || bad "malformed plan lost the served header: $ctx"
ctx=$(hook55 s8 Bash '{"command":"mrw read --grep record docs/"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a grep whose served files appear only in the result still delivers their rules" \
  || bad "grep result headers were not read: $ctx"
ctx=$(hook55 s9 Bash '{"command":"cd docs/adr && cat x.md"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "a leading cd moves the base the later tokens resolve against" || bad "cd-relative read not matched: $ctx"
ctx=$(hook55 s9b Bash '{"command":"cd docs && mrw read --grep record adr/"}' '{"stdout":"==> adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "and the same cd moves the base the served ==> headers resolve against, so a grep after a cd delivers" || bad "served headers after a cd were resolved from the wrong base: $ctx"
# mrw's own --root/-C moves the root its `==>` headers are relative to, and it
# comes BEFORE the subcommand (after `read`, -C is the integer context flag).
ctx=$(hook55 s9c Bash '{"command":"mrw -C .. read docs/adr/x.md:1"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a served header from an mrw run given an explicit --root resolves against that root, not against cwd" \
  || bad "an explicit mrw root lost the served header: $ctx"
# mrw can be reached behind assignments and a wrapper word, and a root given
# twice is not chosen between: both are tried, so neither reading loses a rule.
ctx=$(hook55 s9d Bash '{"command":"env FOO=1 mrw -C .. read docs/adr/x.md:1"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "and an mrw reached behind env and an assignment is still the mrw whose root moves the header" \
  || bad "a prefixed mrw lost the served header: $ctx"
# mrw is found by NAME, so no wrapper's own flags can hide it: an absolute
# path, a flag-bearing wrapper, `exec -a`, and the words no vocabulary would
# have listed — `sudo`, `timeout` — all still name mrw.
for w in '/usr/bin/env FOO=1' 'nice -n 5' 'command -p' 'time -p' 'exec -a custom' 'nohup' 'sudo -u nobody' 'timeout 30'; do
  ctx=$(hook55 "s9d${w//[^a-z]/}" Bash "{\"command\":\"$w mrw -C .. read docs/adr/x.md:1\"}" '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' "$R55/proj/pkg" | ctx55)
  grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
    && ok "and behind '$w', where a positional reading of the command loses it" \
    || bad "wrapper '$w' hid mrw from the root scan: $ctx"
done
# env's own --chdir moves where the command RAN, so it moves the operands too.
ctx=$(hook55 s9g Bash '{"command":"env -C docs cat adr/x.md"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "env --chdir moves the base the operands resolve against, as a leading cd does" || bad "env -C did not move the base: $ctx"
# One command may call mrw twice with two roots, and a token that merely
# SPELLS mrw may name a third that no call used. Every root found is tried,
# because picking one is how the header of the real read gets resolved against
# the wrong base and its rule goes missing.
ctx=$(hook55 s9h Bash '{"command":"printf \"%s\" \"mrw\" -C nowhere ; mrw -C .. read docs/adr/x.md:1"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a data token that spells mrw does not take the root away from the mrw that ran" \
  || bad "a false mrw token suppressed the served header: $ctx"
ctx=$(hook55 s9i Bash '{"command":"mrw -C .. read README.md:1 ; mrw -C ../docs read adr/x.md:1"}' '{"stdout":"==> README.md  1L  7B  sha 1234abcd\n==> adr/x.md  1L  9B  sha 5678efab\n"}' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && grep -q 'INLINE RULE BODY 55' <<<"$ctx" \
  && ok "two mrw calls with two roots both deliver: the headers of the second are not resolved against the first" \
  || bad "a second mrw root was lost: $ctx"
# A control operator needs no whitespace around it, and `;mrw` arrives as one
# token — which a scan by name misses unless the operators are split out.
ctx=$(hook55 s9k Bash '{"command":"mrw -C .. read README.md:1;mrw -C ../docs read adr/x.md:1"}' '{"stdout":"==> README.md  1L  7B  sha 1234abcd\n==> adr/x.md  1L  9B  sha 5678efab\n"}' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "and with no space before the operator, where the second call arrives as ';mrw'" \
  || bad "an adjacent control operator hid the second mrw: $ctx"
# A quoted operand may hold an operator of its own, so the composite token is
# still offered as a path even though the scan reads the split parts.
printf 'semi\n' > "$R55/proj/docs/adr/semi;colon.md"
ctx=$(hook55 s9l Bash '{"command":"cat \"docs/adr/semi;colon.md\""}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a quoted filename containing a control operator is still read whole" \
  || bad "splitting on the operator lost a quoted filename: $ctx"
# Real `env` takes the LAST --chdir relative to where it started, while a `cd`
# before it composes; both readings are offered, so neither loses the rule.
ctx=$(hook55 s9m Bash '{"command":"env -C /nowhere -C docs cat adr/x.md"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a repeated env --chdir delivers: the last value is tried, not only the two joined" \
  || bad "a repeated env --chdir lost the rule: $ctx"
# Each env takes the LAST of its own --chdir flags, so the candidate list grows
# by two per invocation and not by doubling: the first cut offered every value
# against every directory so far, which a review measured at 262,144 candidates
# from eighteen flags — enough to spend the 10 s budget and deliver nothing.
many55=$(for i in $(seq 1 40); do printf -- '-C d%s ' "$i"; done)
( perl -e 'alarm shift; exec @ARGV' 1 bash -c "$(declare -f mk55 hook55); R55='$R55'; HOOK='$HOOK'; hook55 s9n Bash '{\"command\":\"env $many55 -C docs cat adr/x.md\"}'" > "$R55/many.out" ); want 0 "$?" "forty repeated --chdir flags answer inside a 1 s alarm: the directory list is linear, not exponential"
grep -q 'SCOPED RULE BODY 55' <<<"$(ctx55 < "$R55/many.out")" && ok "and the last value still wins, so the rule is delivered" || bad "forty flags lost the rule: $(head -c 120 "$R55/many.out")"
# The boundary, asserted where it bites. The hook reads a command heuristically,
# so a subshell that changes directory prints a header it cannot place, and a
# read whose output is redirected prints no header into the result at all.
# Both deliver NOTHING, which is what the record now says rather than claiming
# the served header always makes up for an unparsed command.
ctx=$(hook55 s9o Bash '{"command":"(cd docs && mrw read adr/x.md:1)"}' '{"stdout":"==> adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' | ctx55)
[ -z "$ctx" ] && ok "a subshell that changes directory delivers nothing: its header names a path this reading cannot place" || bad "the subshell case delivered after all, so the boundary is wrong: $ctx"
ctx=$(hook55 s9p Bash '{"command":"mrw read docs/adr/x.md:1 > /tmp/out55"}' '{"stdout":""}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "a redirected read still delivers for the operand the command names" || bad "a redirected read lost its named operand: $ctx"
# A shell operand may itself contain a colon: `note:1` is a filename, and
# `x.md:1-3` is an mrw spec whose file is the part before it. Both forms are
# offered and only the one that exists survives.
printf 'colon\n' > "$R55/proj/docs/adr/note:1"
ctx=$(hook55 s9j Bash '{"command":"cat docs/adr/note:1"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "a filename containing a colon is read as itself, not cut at the colon" \
  || bad "a colon in a filename lost the rule: $ctx"
ctx=$(hook55 s9e Bash '{"command":"mrw -C /nowhere --root .. read docs/adr/x.md:1"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n    1| a record\n"}' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "and a root given twice delivers, because both are tried rather than one being chosen" \
  || bad "a repeated root lost the served header: $ctx"
# -C is mrw's root and nobody else's: git, make and tar all spell a different
# meaning the same way, so an unrecognised program moves no base.
ctx=$(hook55 s9f Bash '{"command":"git -C .. log --stat"}' '{"stdout":"==> docs/adr/x.md  1L  9B  sha 1234abcd\n"}' "$R55/proj/pkg" | ctx55)
[ -z "$ctx" ] && ok "another program's -C is not read as a root" || bad "a non-mrw -C moved the base: $ctx"
ctx=$(hook55 s10 Bash '{"command":"cat ../docs/adr/x.md"}' '' "$R55/proj/pkg" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" \
  && ok "with cwd in a subdirectory, the project root still comes from CLAUDE_PROJECT_DIR and ../ resolves" \
  || bad "a subdirectory cwd lost the rules: $ctx"
many55=$(printf 'README.md %.0s' $(seq 1 600))
ctx=$(hook55 s10c Bash "{\"command\":\"cat ${many55}docs/adr/x.md\"}" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "the 601st operand of a command is still a candidate: there is no token cap" || bad "a long command lost its last operand: $ctx"
out=$(printf 'not json' | env HOME="$R55/home" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK"); want 0 "$?" "malformed stdin exits 0: a broken hook must not take the turn down"
[ -z "$out" ] && ok "and prints nothing" || bad "printed on malformed stdin: $out"
( closed55 s11 Bash '{"command":"cat docs/adr/x.md"}' 2>/dev/null ); want 0 "$?" "a closed stdout still exits 0"
ctx=$(hook55 s11 Bash '{"command":"cat docs/adr/x.md"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "and the claim that call filed was withdrawn, so the next read in the same session still delivers" || bad "a claim outlived an envelope that never reached the harness: $ctx"
hook55 s12 Bash '{"command":"cat docs/adr/x.md"}' > "$R55/race1" & hook55 s12 Bash '{"command":"cat docs/adr/x.md"}' > "$R55/race2" & wait
n=$( { ctx55 < "$R55/race1"; echo; ctx55 < "$R55/race2"; } | grep -c 'SCOPED RULE BODY 55')
[ "$n" = 1 ] && ok "two hooks racing for one rule in one session deliver it exactly once" || bad "raced delivery count is $n, want 1"
printf -- '---\npaths:\n  - "%s*.md"\n---\n\nBOMB RULE BODY 55\n' "$(printf '**/%.0s' $(seq 1 24))" > "$R55/proj/.claude/rules/bomb.md"
( perl -e 'alarm shift; exec @ARGV' 5 bash -c "$(declare -f mk55 hook55); R55='$R55'; HOOK='$HOOK'; hook55 s13 Bash '{\"command\":\"cat docs/adr/x.md\"}'" > /dev/null ); want 0 "$?" "a rule with 24 globstars matches by segment and cannot stall the hook"
# The matcher's cost is the product of the two segment counts, so a rule of
# 300 globstars against a file 400 directories deep — inside macOS's 1024-byte
# path limit, so this row runs everywhere — must answer inside a 1 s alarm.
# The file must NOT match that rule: the memoised recursion this replaced
# short-circuited on a match and rescanned the rest of the path on every `**`
# for a non-match. Sized by measurement: at 200 globstars that recursion took
# 1.6 s here and a 5 s alarm let it SURVIVE as a mutant; at 300 it takes 2.3 s
# through the hook and the alarm kills it; past ~330 it dies of Python's
# recursion limit instead, which would kill the mutant for the wrong reason.
# The table answers in 40 ms through the hook.
printf -- '---\npaths:\n  - "%s*.md"\n---\n\nBOMB2 RULE BODY 55\n' "$(printf '**/%.0s' $(seq 1 300))" > "$R55/proj/.claude/rules/bomb2.md"
deep55=$(printf 'x/%.0s' $(seq 1 400))
mkdir -p "$R55/proj/docs/adr/$deep55" && printf 'deep\n' > "$R55/proj/docs/adr/${deep55}deep.txt"
( perl -e 'alarm shift; exec @ARGV' 1 bash -c "$(declare -f mk55 hook55); R55='$R55'; HOOK='$HOOK'; hook55 s13b Bash '{\"command\":\"cat docs/adr/${deep55}deep.txt\"}'" > "$R55/deep.out" ); want 0 "$?" "a 300-globstar rule against a file 400 directories deep that it does not match still answers inside a 1 s alarm"
ctx=$(ctx55 < "$R55/deep.out")
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ! grep -q 'BOMB' <<<"$ctx" && ok "and docs/adr/** matched it while the *.md rules did not: ** stands for 400 directories as it does for none" || bad "deep path: $(head -c 200 <<<"$ctx")"
# `*` inside one segment is matched by a two-pointer walk, not by a regex of
# `[^/]*` runs, which backtracks: sixteen stars in one segment took the regex
# over 2 s on a non-match. Twenty-four here, against a 200-character name,
# inside the same 1 s alarm.
printf -- '---\npaths:\n  - "docs/adr/%sz.md"\n---\n\nSTARS RULE BODY 55\n' "$(printf '*a%.0s' $(seq 1 24))" > "$R55/proj/.claude/rules/stars.md"
long55=$(printf 'a%.0s' $(seq 1 200)).md
printf 'long\n' > "$R55/proj/docs/adr/$long55"
( perl -e 'alarm shift; exec @ARGV' 1 bash -c "$(declare -f mk55 hook55); R55='$R55'; HOOK='$HOOK'; hook55 s13c Bash '{\"command\":\"cat docs/adr/$long55\"}'" > "$R55/stars.out" ); want 0 "$?" "a segment of 24 stars against a 200-character name it does not match answers inside a 1 s alarm"
ctx=$(ctx55 < "$R55/stars.out")
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ! grep -q 'STARS' <<<"$ctx" && ok "and docs/adr/** matched it while the star rule did not" || bad "star segment: $(head -c 200 <<<"$ctx")"
# The grammar's edges do what Decision 5 says: a trailing `dir/` names no
# file, and a pattern with nested braces is taken literally.
printf -- '---\npaths: ["README.md/"]\n---\n\nSLASH RULE BODY 55\n' > "$R55/proj/.claude/rules/slash.md"
ctx=$(hook55 s19 Bash '{"command":"cat README.md"}' | ctx55)
! grep -q 'SLASH RULE BODY 55' <<<"$ctx" && ok "a pattern ending in / names no file, so README.md/ does not match README.md" || bad "a trailing slash matched a file: $ctx"
printf -- '---\npaths: ["src/{a,{b,c}}/*.tsx"]\n---\n\nNESTED RULE BODY 55\n' > "$R55/proj/.claude/rules/nested.md"
mkdir -p "$R55/proj/src/{a,{b,c}}"; printf 'lit\n' > "$R55/proj/src/{a,{b,c}}/q.tsx"
ctx=$(hook55 s20 Bash '{"command":"cat src/a/b.tsx"}' | ctx55)
! grep -q 'NESTED RULE BODY 55' <<<"$ctx" && ok "nested braces are literal, so src/{a,{b,c}}/*.tsx does not match src/a/b.tsx" || bad "nested braces were expanded: $ctx"
ctx=$(hook55 s20 Bash '{"command":"cat \"src/{a,{b,c}}/q.tsx\""}' | ctx55)
grep -q 'NESTED RULE BODY 55' <<<"$ctx" && ok "and it does match the path spelled with those braces" || bad "literal nested braces did not match: $ctx"
# Nesting is decided for the WHOLE pattern before any group is expanded:
# expanding the flat group first would leave a half-expanded pattern whose
# later nested group had silently become an alternation.
printf -- '---\npaths: ["src/{a,b}/{c,{d,e}}/*.tsx"]\n---\n\nMIXED RULE BODY 55\n' > "$R55/proj/.claude/rules/mixed.md"
mkdir -p "$R55/proj/src/a/{c,{d,e}}"; printf 'lit\n' > "$R55/proj/src/a/{c,{d,e}}/q.tsx"
ctx=$(hook55 s21 Bash '{"command":"cat \"src/a/{c,{d,e}}/q.tsx\""}' | ctx55)
! grep -q 'MIXED RULE BODY 55' <<<"$ctx" \
  && ok "a flat group before a nested one does not expand either: the whole pattern is literal" \
  || bad "a flat group was expanded ahead of a nested one: $ctx"
# An inline list may carry a trailing comment, and the comment goes before the
# brackets do — stripping `]` first left the glob `docs/adr/**]`, which matches
# nothing and says nothing.
printf -- '---\npaths: ["docs/adr/**"] # the records\n---\n\nCOMMENT RULE BODY 55\n' > "$R55/proj/.claude/rules/comment.md"
ctx=$(hook55 s22 Bash '{"command":"cat docs/adr/x.md"}' | ctx55)
grep -q 'COMMENT RULE BODY 55' <<<"$ctx" \
  && ok "an inline paths list with a trailing comment still yields the glob inside it" \
  || bad "an inline list with a comment produced no usable glob: $ctx"
rm -f "$R55/proj/.claude/rules/comment.md" "$R55/proj/.claude/rules/mixed.md"
# A served path with a space in it. mrw's header is `==> path  NL  NB  sha …`,
# two spaces after the path, and for a grep the header is the only place the
# path appears at all.
ctx=$(hook55 s8b Bash '{"command":"mrw read --grep spaced docs/"}' '{"stdout":"==> docs/adr/my file.md  1L  7B  sha 1234abcd\n    1| spaced\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "a served path containing a space is read whole from its ==> header" || bad "spaced header path lost: $ctx"
printf 'twice\n' > "$R55/proj/docs/adr/my  file.md"
ctx=$(hook55 s8c Bash '{"command":"mrw read --grep twice docs/"}' '{"stdout":"==> docs/adr/my  file.md  1L  6B  sha 1234abcd\n    1| twice\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "and one holding two consecutive spaces: the path is read back from the NL  NB  sha suffix, not forward to the first gap" || bad "consecutive-space header path lost: $ctx"
# Plan headers are tokenised as internal/plan tokenises them, and this is the
# row that makes that DIFFERENTIAL rather than a Python opinion: for each
# header shape, the BUILT BINARY is given a one-hunk plan and reports the path
# it took (`files[].path` in --json, empty when it refuses to parse), and the
# hook is given the same header; the two must name the same string. Nothing
# else about the plan is compared, because acceptance is deliberately not
# mirrored — only which string is the path.
for hdr in 'docs/adr/x.md 1 replace' "'docs/adr/x.md' 1 replace" '"docs/adr/my file.md" 1 replace' 'docs/adr/x.md /^func (s *Store) Get/,/^}/ replace' 'docs/adr/x.md 1 replace anchor="unterminated'; do
  printf '@@ %s\nX\n' "$hdr" > "$R55/plan55"
  mrwpath=$("$MRW" -C "$R55/proj" write --json "$R55/plan55" 2>/dev/null | python3 -c 'import json,sys
try: d=json.load(sys.stdin)
except Exception: print(""); raise SystemExit
fs=d.get("files") or []
print(fs[0]["path"] if fs else "")')
  hookpath=$(printf '@@ %s\nX\n' "$hdr" | perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 -c 'import json,sys,importlib.util
spec=importlib.util.spec_from_file_location("h", sys.argv[1]); h=importlib.util.module_from_spec(spec); spec.loader.exec_module(h)
print("\n".join(h.plan_paths(sys.stdin.read())))' "$HOOK")
  [ "$mrwpath" = "$hookpath" ] \
    && ok "the hook and the built binary take the same path from the header: ${hdr:0:34}…" \
    || bad "path selection differs for '$hdr': mrw said '$mrwpath', the hook said '$hookpath'"
done
rm -f "$R55/plan55"
ctx=$(hook55 s5e mcp__mrw__mrw_write "{\"plan\":\"@@ 'docs/adr/x.md' 1 replace\\nX\\n\"}" | ctx55)
[ -z "$ctx" ] && ok "a single-quoted path is literal to mrw, names no file, and delivers nothing" || bad "single quotes were stripped, which mrw does not do: $ctx"
ctx=$(hook55 s5f mcp__mrw__mrw_write '{"plan":"@@ docs/adr/x.md /^func (s *Store) Get/,/^}/ replace\nX\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "a pattern address with spaces is one token, and the path before it delivers" || bad "a pattern address split the header: $ctx"
ctx=$(hook55 s5g mcp__mrw__mrw_write '{"plan":"@@ \"docs/adr/x.md 1 replace\nX\n@@ README.md 1 replace\nY\n"}' | ctx55)
! grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && grep -q 'INLINE RULE BODY 55' <<<"$ctx" && ok "an unterminated quote yields no path from that line, and the next header still delivers early" || bad "unterminated quote: $ctx"
ctx=$(hook55 s5h mcp__mrw__mrw_write '{"plan":"@@ docs/adr/x.md 1 replace anchor=\"a\" anchor=\"b\"\nX\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "a guard given twice is a plan mrw refuses, and it delivers early for the file it names" || bad "a refused plan (duplicate guard) was suppressed: $ctx"
ctx=$(hook55 s5i mcp__mrw__mrw_write '{"plan":"@@ docs/adr/x.md 1 frobnicate\nX\n"}' | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "an unknown op is a plan mrw refuses, and it delivers early for the file it names" || bad "a refused plan (unknown op) was suppressed: $ctx"
# A Bash operand is relative to where the command ran, and to nothing else:
# mrw's own --root defaults to ".", so a spelling that names no file from cwd
# read no file.
ctx=$(hook55 s10b Bash '{"command":"cat docs/adr/x.md"}' '' "$R55/proj/pkg" | ctx55)
[ -z "$ctx" ] && ok "from a subdirectory, a root-relative spelling that names nothing from cwd is not retried against the root" || bad "a Bash operand was resolved against the root: $ctx"
# Nothing lands in a checkout. A relative XDG_CACHE_HOME would land under
# whatever cwd the hook got; one inside the project would land in the tree.
# Both are refused, and the hook delivers WITHOUT a claim — a repeat beats a
# silence, and ADR-004's promise beats both.
mkdir -p "$R55/cwd"
ctx=$(cd "$R55/cwd" && mk55 s14 Bash '{"command":"cat docs/adr/x.md"}' | env HOME="$R55/home" XDG_CACHE_HOME=rel55 CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && [ ! -e "$R55/cwd/rel55" ] && [ ! -e "$R55/proj/rel55" ] \
  && ok "a relative XDG_CACHE_HOME is refused: nothing is created under cwd, and the rule is delivered anyway" \
  || bad "relative cache base: delivered=$([ -n "$ctx" ] && echo yes || echo no) created=$(ls -d "$R55/cwd/rel55" "$R55/proj/rel55" 2>/dev/null | tr '\n' ' ')"
ctx=$(mk55 s15 Bash '{"command":"cat docs/adr/x.md"}' | env HOME="$R55/home" XDG_CACHE_HOME="$R55/proj/.cache" CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && [ ! -e "$R55/proj/.cache" ] \
  && ok "a cache base inside the project is refused: nothing lands in the tree, and the rule is delivered anyway" \
  || bad "in-tree cache base: delivered=$([ -n "$ctx" ] && echo yes || echo no) created=$(ls -d "$R55/proj/.cache" 2>/dev/null)"
printf 'not a directory\n' > "$R55/notadir"
one=$(mk55 s16 Bash '{"command":"cat docs/adr/x.md"}' | env HOME="$R55/home" XDG_CACHE_HOME="$R55/notadir" CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
two=$(mk55 s16 Bash '{"command":"cat docs/adr/x.md"}' | env HOME="$R55/home" XDG_CACHE_HOME="$R55/notadir" CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$one" && grep -q 'SCOPED RULE BODY 55' <<<"$two" \
  && ok "an unusable state directory delivers on every call: once per session holds only while a claim can be filed" \
  || bad "an unusable state directory suppressed a delivery: first=$([ -n "$one" ] && echo yes || echo no) second=$([ -n "$two" ] && echo yes || echo no)"
# The state DIRECTORY's own path being a regular file raises FileExistsError,
# which is an OSError — and reading that as "another hook holds the claim"
# suppressed every rule for the session. Only the O_EXCL create may be read
# that way, so this pair must deliver twice.
mkdir -p "$R55/filecache"; printf 'not a directory\n' > "$R55/filecache/claude-rules-on-read"
one=$(mk55 s16b Bash '{"command":"cat docs/adr/x.md"}' | env HOME="$R55/home" XDG_CACHE_HOME="$R55/filecache" CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
two=$(mk55 s16b Bash '{"command":"cat docs/adr/x.md"}' | env HOME="$R55/home" XDG_CACHE_HOME="$R55/filecache" CLAUDE_PROJECT_DIR="$R55/proj" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$one" && grep -q 'SCOPED RULE BODY 55' <<<"$two" \
  && ok "a regular file where the claim directory belongs delivers on every call, not none: FileExistsError from makedirs is not a claim" \
  || bad "a file at the state directory's path suppressed a delivery: first=$([ -n "$one" ] && echo yes || echo no) second=$([ -n "$two" ] && echo yes || echo no)"
# Without CLAUDE_PROJECT_DIR the walk up from cwd takes the nearest
# .claude/rules, and stops at the first .git it meets: a nested repository
# does not inherit the enclosing one's rules.
ctx=$(mk55 s17 Bash '{"command":"cat ../docs/adr/x.md"}' '' "$R55/proj/pkg" | env -u CLAUDE_PROJECT_DIR HOME="$R55/home" XDG_CACHE_HOME="$R55/home/.cache" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
grep -q 'SCOPED RULE BODY 55' <<<"$ctx" && ok "without CLAUDE_PROJECT_DIR, the nearest .claude/rules above cwd is the project" || bad "the walk-up found no project: $ctx"
mkdir -p "$R55/proj/inner/.git"
ctx=$(mk55 s18 Bash '{"command":"cat ../docs/adr/x.md"}' '' "$R55/proj/inner" | env -u CLAUDE_PROJECT_DIR HOME="$R55/home" XDG_CACHE_HOME="$R55/home/.cache" perl -e 'alarm shift; exec @ARGV' "${ALARM55:-10}" python3 "$HOOK" | ctx55)
[ -z "$ctx" ] && ok "and it stops at a nested repository's .git, so the enclosing project's rules are not delivered into it" || bad "the walk-up crossed a .git boundary: $ctx"
[ -n "$(ls "$R55/home/.cache/claude-rules-on-read" 2>/dev/null)" ] && ok "dedup state lives under the caller's cache directory, not the shared temp" || bad "no state under HOME"
# A hook that never returns must be a FAILED row, not a process that outlives
# the run (#101). Every run of hook code above — hook55, closed55, and the
# plan_paths import — goes under the alarm hook55 describes. This row proves
# the alarm fires on the two entry points it can drive, by handing them a
# hook that sleeps longer than a 1 s alarm; the sleep is at module level, so
# it hangs an import exactly as it hangs an exec. Without the alarm the sleep
# completes, the exit is 0 and the row goes red — after thirty seconds
# instead of one. The 2>: bash reports the SIGALRM kill on stderr.
printf 'import time\ntime.sleep(30)\n' > "$R55/hang.py"
t0=$(date +%s)
( HOOK="$R55/hang.py"; ALARM55=1; hook55 s19 Bash '{"command":"cat docs/adr/x.md"}' >/dev/null ) 2>/dev/null; rc=$?
( HOOK="$R55/hang.py"; ALARM55=1; closed55 s19c Bash '{"command":"cat docs/adr/x.md"}' ) 2>/dev/null; rc2=$?
el=$(( $(date +%s) - t0 ))
[ "$rc" -ne 0 ] && [ "$rc2" -ne 0 ] && [ "$el" -lt 10 ] \
  && ok "a hook that never returns is killed by the alarm and its row fails, through hook55 and closed55 alike: exit $rc/$rc2 after ${el}s" \
  || bad "a hanging hook outlived its alarm: hook55 exit $rc, closed55 exit $rc2, after ${el}s"
rm -rf "$R55"

# 56. ADR-021: a plan names a file once, however it is spelled.
#
# Measured 2026-09-04: Same.txt and same.txt in one plan on a case-insensitive
# filesystem reported two hunks ok and two files written, and the file held
# only the second edit — both spellings staged a copy of the same bytes and
# the last rename won. A symlink and its target do the same on EVERY
# filesystem, which is the half this row can run on Linux CI. The fix asks
# os.SameFile at grouping time. The pair is the same two hunks under ONE
# spelling, which must still apply — a check that refused on resemblance
# would pass the first half and fail this one.
R56=$(mktemp -d)
printf 'one\ntwo\nthree\n' > "$R56/real.txt"; ln -s real.txt "$R56/link.txt"
( cd "$R56" && "$MRW" read real.txt link.txt >"$WORK/served.out" 2>&1 ); want 0 "$?" "both spellings are served first, so the refusal below is the identity check and not the ledger"
printf '@@ real.txt 1 replace\nX\n@@ link.txt 3 replace\nZ\n' > "$R56/two.plan"
out=$( cd "$R56" && "$MRW" write two.plan 2>&1 ); rc=$?
want 1 "$rc" "a plan naming one file as real.txt and as a symlink to it is refused"
[ "$(cat "$R56/real.txt")" = "$(printf 'one\ntwo\nthree')" ] \
  && ok "and nothing was written" \
  || bad "a refused plan wrote something: $(cat "$R56/real.txt")"
grep -q 'link.txt names the same file as real.txt' <<<"$out" \
  && ok "and the refusal names both spellings, so the plan can be fixed in one edit" \
  || bad "the refusal does not name both spellings: $out"
printf 'one\ntwo\nthree\n' > "$R56/Same.txt"
if [ -e "$R56/same.txt" ]; then
  ( cd "$R56" && "$MRW" read Same.txt same.txt >"$WORK/served.out" 2>&1 ); want 0 "$?" "both case spellings are served first"
  printf '@@ Same.txt 1 replace\nX\n@@ same.txt 3 replace\nZ\n' > "$R56/case.plan"
  ( cd "$R56" && "$MRW" write --quiet case.plan > /dev/null 2>&1 )
  want 1 "$?" "Same.txt and same.txt in one plan are refused where the filesystem folds case — the measured shape"
  [ "$(cat "$R56/Same.txt")" = "$(printf 'one\ntwo\nthree')" ] \
    && ok "and nothing was written" \
    || bad "the measured defect is back: $(cat "$R56/Same.txt")"
else
  skip "a case-sensitive filesystem here: the two-spelling half did not run; the symlink half above is its twin"
fi
printf '@@ real.txt 1 replace\nX\n@@ real.txt 3 replace\nZ\n' > "$R56/one.plan"
( cd "$R56" && "$MRW" write --quiet one.plan > /dev/null 2>&1 )
want 0 "$?" "the same two hunks under ONE spelling still apply"
[ "$(cat "$R56/real.txt")" = "$(printf 'X\ntwo\nZ')" ] \
  && ok "and both landed, through the symlink's target" \
  || bad "one spelling did not apply both hunks: $(cat "$R56/real.txt")"
rm -rf "$R56"

# 47. ADR-014 T1: an oversized read is a FIRST PAGE, and following it loses
# nothing.
#
# ADR-011-T3 bounded the result and left the caller a dead end: one suggested
# range and no page two, so a caller that followed it once had confidently read
# part of a file. This row pages a real file to completion through the built
# binary and reassembles it, because a continuation pointing at the wrong lines
# passes every check that only asks whether a continuation exists.
fixture
python3 -c "
with open('$R/huge.go','w') as f:
    f.write('package demo\n')
    for i in range(12000): f.write('// padding padding padding padding padding %06d\n' % i)
"
total=$(wc -l < "$R/huge.go" | tr -d ' ')

MRW_BIN="$MRW" python3 - "$R" "$total" <<'PY'
import json, subprocess, sys, os
root, total = sys.argv[1], int(sys.argv[2])
mrw = os.environ["MRW_BIN"]
spec, got, pages = "huge.go", [], 0
while True:
    pages += 1
    assert pages <= 30, "still paging after %d pages — the continuation is not advancing" % pages
    req = json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":
                      {"name":"mrw_read","arguments":{"specs":[spec]}}})
    out = subprocess.run([mrw, "-C", root, "mcp"], input=req+"\n",
                         capture_output=True, text=True).stdout
    r = json.loads(out)["result"]
    body = r["content"][0]["text"]
    for ln in body.split("\n"):
        i = ln.find("|")
        if i < 0: continue
        if not ln[:i].strip().isdigit(): continue
        got.append(ln[i+1:].removeprefix(" "))
    nxt = json.loads(r["content"][1]["text"]).get("next_read", "")
    if not nxt:
        break
    assert nxt != spec, "page %d handed back the spec it was given" % pages
    spec = nxt
assert pages > 1, "a file this size must page; the fixture is not exercising the row"
assert len(got) == total, "reassembled %d lines, want %d — paging lost or repeated content" % (len(got), total)
with open(os.path.join(root, "huge.go")) as f:
    want = f.read().split("\n")[:-1]
assert got == want, "the reassembly differs from the file on disk"
print("paged %d time(s), reassembled %d lines" % (pages, total))
PY
[ $? -eq 0 ] && ok "an oversized read pages to completion and reassembles byte for byte" \
             || bad "paging lost content, repeated it, or did not terminate"
# A page must license exactly what it served — no more, and no less. This needs
# a FRESH tree: the loop above read every page, so in that one the whole file is
# licensed and the row would pass for the wrong reason.
fixture
python3 -c "
with open('$R/huge.go','w') as f:
    f.write('package demo\n')
    for i in range(12000): f.write('// padding padding padding padding padding %06d\n' % i)
"
total=$(wc -l < "$R/huge.go" | tr -d ' ')
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["huge.go"]}}}\n' | m mcp 2>/dev/null)
# ⚠ A PAGE LICENSES NOTHING UNTIL IT IS ACKNOWLEDGED (ADR-031). This row used to
# read and then write, because being SENT a page was taken as receiving it —
# the belief a host's truncation falsified on 2026-09-05. Acknowledging every
# checkpoint is what a caller that received the whole page does, and the row's
# own claim is unchanged: the PAGE is licensed, the file is not.
printf '%s' "$out" > "$R/page.json"   # via a file: a page exceeds Linux's 128 KB argv limit
cks=$(python3 - "$R/page.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
print(",".join('"%s"' % c for c in re.findall(r"^-- ck ([0-9a-f]{16}) open ", r["content"][0]["text"], re.M)))
PY
)
[ -n "$cks" ] && ok "the page carries checkpoints to acknowledge" || bad "a paged read carries no checkpoint"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ huge.go 1 replace\\n// page one\\n","dry_run":true,"ack":[%s]}}}\n' "$cks" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 0, "a write to a line the acknowledged page served was refused: %s" % sc
PY
want 0 $? "a write to a line the first page served is licensed once acknowledged"
out=$(printf "@@ huge.go $total replace\n// last line\n" | m write --dry-run - 2>&1); rc=$?
want 1 "$rc" "and a write to a line that page did NOT serve is refused as unread"
# ⚠ Both judgements must sit ABOVE the next section. They were spliced apart by
# §48's `fixture`, which replaces $R — so the second write ran against a tree
# holding no huge.go at all, failed with "does not exist", and satisfied a row
# claiming to prove the ledger. A test that passes for the wrong reason is worse
# than one that fails. Caught in review of PR #76.

# 48. ADR-014 T2: what the wire TEACHES about paging is what the binary DOES.
#
# One row, both halves. ADR-012 taught an enum the engine never sent and ADR-013
# taught two examples that could not match anything; both were prose checked
# against nothing, and both were found by a reviewer rather than a gate.
fixture
python3 -c "
with open('$R/pager.go','w') as f:
    f.write('package demo\n')
    for i in range(9000): f.write('// padding padding padding padding padding %06d\n' % i)
"
# ⚠ THROUGH FILES, NOT ARGV. The page is ~156 KB, and Linux caps a single
# argument at 131,072 bytes (MAX_ARG_STRLEN), so passing it as argv dies with
# "Argument list too long" and the row reports a behaviour mismatch that is not
# one. macOS's limit is larger, which is why this was green here and red in CI.
# Caught in review of PR #76.
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null > "$WORK/init.json"
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["pager.go"]}}}\n' | m mcp 2>/dev/null > "$WORK/page.json"
python3 - "$WORK/init.json" "$WORK/page.json" <<'PY'
import json,sys
i=json.load(open(sys.argv[1]))["result"]["instructions"]
r=json.load(open(sys.argv[2]))["result"]
# What it teaches.
for w in ("next_read","PAGE","absent","part of a file"):
    assert w in i, "the instructions never mention %r" % w
# What it does — the same three claims, against a real oversized read.
assert "isError" not in r, "ADR-024: taught as a PAGE and not a failure, shipped carrying the isError key"
sc=json.loads(r["content"][1]["text"])
assert sc.get("next_read"), "taught next_read, shipped none"
assert "padding" in r["content"][0]["text"], "taught a PAGE of content, shipped no content"
assert "PARTIAL" in r["content"][0]["text"], "the page does not say it is partial to a human reader"
PY
[ $? -eq 0 ] && ok "the wire teaches paging, and a real oversized read does exactly that" \
             || bad "the taught paging behaviour is not the shipped one"
# 49. ADR-015 T1: the two mistakes this syntax invites name their own fix.
#
# This project's stated ethos is that a refusal is the tool working and "names
# the file, the plan line and the reason". These two did not. Both were hit
# repeatedly while writing this repository's own documentation, which is what
# writing plan syntax into a plan body gets you.
fixture

# D5: a body line beginning with @@ is read as a header.
out=$(printf '@@ a.go 1 replace\n@@ this is body text\nmore body\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a body line beginning with @@ is refused"
grep -q 'body=<n> raw=true' <<<"$out" \
  && ok "and the refusal names the escape, instead of only reporting a bogus op" \
  || bad "the refusal does not mention body=/raw=: $out"

# ...and the hint must NOT fire where it would be wrong.
out=$(printf '@@ a.go 1 mangle\nbody\n' | m write - 2>&1)
grep -q 'body=<n> raw=true' <<<"$out" \
  && bad "the body-line hint fired on an ordinary bad op on the first line: $out" \
  || ok "and it stays quiet on an ordinary bad op, so it does not become noise"

# D4: a glob the shell never expanded.
out=$(m read 'sub/*.go:1-3' 2>&1)
grep -q 'UNREADABLE' <<<"$out" || bad "expected an unreadable report: $out"
grep -q 'glob your shell did not expand' <<<"$out" \
  && ok "an unexpanded glob is named as one, not reported as a missing file" \
  || bad "the report reads as a missing file: $out"
grep -q -- '--grep' <<<"$out" \
  && ok "and it names the tools that do the job" \
  || bad "the hint does not say what to use instead: $out"

out=$(m read 'nope.go' 2>&1)
grep -q 'glob' <<<"$out" \
  && bad "the glob hint fired for a path with no metacharacter: $out" \
  || ok "and an ordinary missing file gets no glob hint"

# REPO HYGIENE, and it lives here because this is the PR that shipped the defect
# it catches. A merge of main into this branch committed <<<<<<< / ======= /
# >>>>>>> into docs/adr/BACKLOG.md and NOTHING NOTICED: CI does not read that
# file, adr-debt found every deferral it was looking for on both sides of the
# markers, and adr-lint treats BACKLOG.md as prose. A reviewer found it by
# eye. The class is "a merge artifact in a file no gate reads", so the gate has
# to be over the whole tree rather than over the files a gate happens to parse.
# git grep answers 0 for a match, 1 for none and anything else when it could
# not look — and `|| true` read all three as "clean", so a run whose SRC named
# a directory outside any checkout passed this row having searched nothing.
if git -C "$SRC" rev-parse --git-dir >/dev/null 2>&1; then
  markers=$(git -C "$SRC" grep -n -E '^(<<<<<<< |=======$|>>>>>>> )' -- . 2>&1); grc=$?
  case $grc in
    1) ok "no conflict markers are committed anywhere in the tree" ;;
    0) bad "conflict markers are committed: $markers" ;;
    *) bad "git grep could not look for conflict markers in $SRC (exit $grc): $markers" ;;
  esac
else
  bad "$SRC is not a git checkout, so the conflict-marker row cannot look"
fi

# 57. ADR-012 + ADR-021: the MCP surface teaches the refusal the engine added.
#
# §46 established the shape: a rule taught on the wire must be a rule the binary
# enforces, because ADR-012 once taught an enum the engine never sent. This is
# the converse and the gap v0.1.0 shipped with — the engine gained ADR-021's
# one-file-one-spelling refusal while the handshake said nothing about it. An
# MCP-only caller has no README and no AGENTS.md, so the wire is the ONLY place
# it can learn the rule before meeting it as a refusal. Both halves here, in one
# row, so the wire cannot drift from the binary in either direction.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
i=json.loads(sys.argv[1])["result"]["instructions"]
# The predicate is the WHOLE clause, not two words that co-occur. Codex found
# this: asserting "spellings of ONE file" and "refused" separately let a wire
# saying "a plan naming both is ALLOWED and the last wins" pass, because
# "refused" appears in six unrelated sentences. Reproduced here — the contract
# exited 0 while the wire contradicted the binary. A predicate loose enough to
# match a contradiction is not a predicate.
assert "spellings of ONE file" in i, "the wire never teaches that two spellings are one file"
assert "a plan naming both is refused" in i, "the wire names the case without refusing it"
b=len(i.encode())
assert b <= 4096, "the instructions are %d bytes; they are paid once per session" % b
PY
[ $? -eq 0 ] && ok "the wire teaches that two spellings of one file are refused, still within budget" \
             || bad "the one-spelling rule is missing from the handshake, or the budget is blown"
R57=$(mktemp -d)
printf 'one\ntwo\nthree\n' > "$R57/real.txt"; ln -s real.txt "$R57/link.txt"
( cd "$R57" && "$MRW" read real.txt link.txt >"$WORK/served.out" 2>&1 ); want 0 "$?" "both spellings are served, so the refusal below is the identity check"
printf '@@ real.txt 1 replace\nX\n@@ link.txt 3 replace\nZ\n' > "$R57/two.plan"
( cd "$R57" && "$MRW" write --quiet two.plan > /dev/null 2>&1 )
want 1 "$?" "and the binary refuses exactly what the wire just promised it would"
[ "$(cat "$R57/real.txt")" = "$(printf 'one\ntwo\nthree')" ] \
  && ok "with nothing written, so the taught rule and the enforced rule are one rule" \
  || bad "the wire teaches a refusal the binary did not perform: $(cat "$R57/real.txt")"

# 58. ADR-020-T2: the harder fixture names no service, and the two are two trials.
#
# The first reading returned 42 correct addresses in 45 trials against the named
# selector, flat across a hundredfold change in served bytes, with every client
# verified to have read the window whole. That is a ceiling, and a curve cannot
# bend against a task nobody fails. This row drives the BUILT binary because a
# selector can be correct in Generate and unreachable from the command line,
# which is exactly how a fixture mode nobody can generate ships green. The pair
# is the SAME parameters under the named selector, whose instruction must still
# name a service — a check that just looked for a short instruction would pass
# the first half and fail this one.
R58=$(mktemp -d)
"$CURVE" generate -out "$R58/rel" -bytes 3000 -position middle -distractors 3 -seed 58 -selector odd-retries > "$R58/rel.json" 2>&1
want 0 "$?" "the built binary generates a cell whose target is picked by relation"
"$CURVE" generate -out "$R58/named" -bytes 3000 -position middle -distractors 3 -seed 58 > "$R58/named.json" 2>&1
want 0 "$?" "and the same parameters still generate the named cell, unchanged"
python3 - "$R58" <<'PY'
import json,re,sys
d=sys.argv[1]
rel=json.load(open(d+"/rel/manifest.json")); named=json.load(open(d+"/named/manifest.json"))
body=open(d+"/rel/tree/services.conf").read()
names=set(re.findall(r"svc-[a-z]+", body))
assert names, "the fixture has no service names, so this row proves nothing"
leaked=[n for n in names if n in rel["instruction"]]
assert not leaked, "the relational instruction names %s, so the target is findable without reading" % leaked
nb=open(d+"/named/tree/services.conf").read()
assert any(n in named["instruction"] for n in set(re.findall(r"svc-[a-z]+", nb))), "the named instruction stopped naming a service"
vals=re.findall(r"^retries = (\d+)$", body, re.M)
odd=[v for v in set(vals) if vals.count(v)==1]
assert len(vals)>=3 and len(odd)==1, "want one odd retry budget over at least three blocks, got %r" % vals
ans=json.load(open(d+"/rel/answer.json"))
lines=body.split("\n")
assert lines[ans["line"]-1]=="timeout = 30", "the answer does not name a timeout line"
assert lines[ans["line"]-2]=="retries = "+odd[0], "the answer names a block that is not the odd one"
assert rel["trial_id"]!=named["trial_id"], "the two selectors share a trial id, so a result could be scored against the wrong cell"
PY
[ $? -eq 0 ] && ok "the relational instruction names no service, the answer is the odd block's timeout, and the two selectors are two trials" \
             || bad "the harder fixture is findable by name, points at the wrong block, or collides with the named cell"

# 59. ADR-020-T3: the relational fixture carries no constant signature.
#
# T2 removed the target's unique NAME and left a constant VALUE. Measured on
# e96504a: every relational cell at every seed rendered the target at
# "retries = 5", so a client that had seen one cell could search for it in every
# other at a cost independent of served size — the single-match shortcut the
# selector exists to remove, surviving inside its own fixture. Found by review of
# PR #93.
#
# FOUR relational seeds, and BOTH values checked separately. Review of PR #94
# found the first version of this row comparing whole multisets, which passes
# when the common value is fixed and only the odd one moves — the same shortcut
# one level removed, and exactly what the row claims to exclude. It also parsed
# with grep -oE, so "retries = 5x" matched "retries = 5", and an empty parse on
# both sides compared equal. The parsing is exact and in python now, which is
# this script's convention for anything with structure.
#
# The pair is two NAMED cells, whose retries lines must be IDENTICAL across
# seeds. That is checked here rather than left to the id: trialID hashes the
# PARAMETERS and never the rendered file, so drifted fixture bytes reuse the
# recorded id in silence. TestTheNamedFixtureMatchesItsGoldenBytes pins the
# bytes; this row pins the shape through the built binary.
R59=$(mktemp -d)
for s in 1 2 3 4; do
  "$CURVE" generate -out "$R59/rel$s" -bytes 2500 -position middle -distractors 3 -seed $s -selector odd-retries > /dev/null 2>&1
  want 0 "$?" "the built binary generates relational cell seed $s"
done
for s in 1 2; do
  "$CURVE" generate -out "$R59/named$s" -bytes 2500 -position middle -distractors 3 -seed $s > /dev/null 2>&1
  want 0 "$?" "and named cell seed $s"
done
python3 - "$R59" <<'PY'
import re,sys
d=sys.argv[1]
def retries(name, blocks=4):
    lines=open(d+"/"+name+"/tree/services.conf").read().split("\n")
    vals=[l for l in lines if re.fullmatch(r"retries = \d+", l)]
    assert len(vals)==blocks, "%s has %d retries lines, want %d" % (name, len(vals), blocks)
    return sorted(vals)
def relational(name, blocks=4):
    vals=retries(name, blocks)
    counts={v:vals.count(v) for v in set(vals)}
    assert sorted(counts.values())==[1,blocks-1], "%s has multiplicities %s, want [1, %d] — a relational cell with no singleton is unanswerable" % (name, sorted(counts.values()), blocks-1)
    odd=[v for v,n in counts.items() if n==1][0]
    common=[v for v,n in counts.items() if n==blocks-1][0]
    return odd, common
rel=[relational("rel%d"%s) for s in (1,2,3,4)]
odds={o for o,_ in rel}; commons={c for _,c in rel}
assert len(odds)>1, "the odd budget is %s at every seed: one look at any cell teaches a search that works on all of them" % odds
assert len(commons)>1, "the common budget is %s at every seed, which identifies the odd block by elimination" % commons
# The named fixture has NO singleton by design — every block carries the same
# budget — so it is compared as a whole, not through the relational shape.
n1, n2 = retries("named1"), retries("named2")
assert n1==n2, "the draw reached the named fixture: seed 1 is %s and seed 2 is %s" % (n1, n2)
assert len(set(n1))==1, "the named fixture no longer renders one constant budget: %s" % n1
PY
[ $? -eq 0 ] && ok "four relational seeds move BOTH budgets, and the named fixture renders the same constant at every seed" \
             || bad "the relational fixture carries a constant signature, or the draw reached the named fixture"
# 61. ADR-023 T1: a read's answer is the served text, and no envelope stands in
# for it.
#
# Measured 2026-09-05 on Claude Code 2.1.261 (issue #109): a non-error result
# that carried structuredContent reached the model AS the structuredContent, and
# the content blocks were dropped — so a successful mrw_read delivered the
# receipt and none of the lines, while the ledger had already recorded them as
# seen. Hence: no mrw_read result — served, paged, index — carries
# structuredContent; its receipt is content[1]; tools/list declares no
# outputSchema for it. Paired with the case that must differ: mrw_write keeps
# both, equal to each other, because its answer IS the receipt.
fixture
python3 -c "
with open('$R/big.txt','w') as f:
    for i in range(12000): f.write('padding line %05d that is long enough to make this file page\n' % i)
for i in range(60):
    with open('$R/document%05d.csv' % i,'w') as f:
        f.write('a line that matches nothing\n')
        for j in range(400): f.write('the NEEDLE is here\n')
"
# Results go to FILES, not argv: a page of a 12,000-line file is ~700 KB, and
# Linux refuses a single argument over 128 KB ("Argument list too long") —
# which is how the first cut of this row went red in CI and green on macOS.
call61() { printf '%s\n' "$2" | m mcp 2>/dev/null > "$WORK/61-$1.json"; }
call61 served '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go"]}}}'
call61 paged  '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["big.txt"]}}}'
call61 index  '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"grep":"NEEDLE"}}}'
call61 listed '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
call61 wrote  '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ a.go 3 replace\nfunc A() int { return 61 }\n"}}}'
python3 - "$WORK" <<'PY'
import json,sys
w=sys.argv[1]
def load(n):
    with open("%s/61-%s.json" % (w,n)) as f: return json.loads(f.readline())["result"]
served,paged,index,listed,wrote=[load(n) for n in ("served","paged","index","listed","wrote")]
for name,r in (("served",served),("paged",paged),("index",index)):
    assert "structuredContent" not in r, "%s read result carries structuredContent; a host that renders it in place of content hides the served text" % name
    c=r["content"]; assert len(c)>=2, "%s read result has %d content blocks, want the answer and the receipt" % (name,len(c))
    rec=json.loads(c[1]["text"]); assert "observed" in rec, "%s receipt at content[1] carries no observed" % name
assert served["content"][0]["text"].startswith("==> a.go"), "the served read's content[0] is not the served text"
assert served.get("isError") is not True, "a two-line served read read as an error"
assert "isError" not in paged and json.loads(paged["content"][1]["text"]).get("next_read"), "ADR-024: the page carries the isError key, or names no next_read at content[1]"
assert "isError" not in index and json.loads(index["content"][1]["text"]).get("index"), "ADR-024: the index carries the isError key, or carries no entries at content[1]"
tools={t["name"]:t for t in listed["tools"]}
assert "outputSchema" not in tools["mrw_read"], "mrw_read declares an outputSchema it never fulfils"
assert "outputSchema" in tools["mrw_write"], "mrw_write lost its outputSchema"
assert "structuredContent" in wrote, "mrw_write lost its structuredContent; its answer IS the receipt"
assert wrote["structuredContent"]==json.loads(wrote["content"][1]["text"]), "mrw_write's content[1] and structuredContent disagree"
assert wrote["structuredContent"]["applied"] is True, "the write did not apply; the pairing is not exercised"
PY
[ $? -eq 0 ] && ok "no mrw_read result carries structuredContent, its receipt is content[1], and mrw_write keeps both" \
             || bad "a read result still carries the envelope that hid the served text, or the write lost its own"

# 60. ADR-020-T4: a served window that does not begin at line one.
#
# Every miss in the 135 read-arm trials of readings 2, 3 and 4 sits at
# target+2, and every cell so far serves from line 1 — where a row count in the
# rendering, whose first two rows carry no line number, and the line number
# plus two are the same integer. This row drives the BUILT binary because a
# -from flag that parses and serves the whole file anyway would leave every
# test green and the discriminating reading measuring nothing. The pair is a
# window past the target, which must be REFUSED: a client served no answer is
# not a hard trial, it is an unanswerable one, and it would score as a miss.
R60=$(mktemp -d)
"$CURVE" generate -out "$R60/win" -bytes 20000 -position late -distractors 3 -seed 2 -selector odd-retries -from 120 > "$R60/win.json" 2>&1
"$CURVE" generate -out "$R60/twin" -bytes 20000 -position late -distractors 3 -seed 2 -selector odd-retries > "$R60/twin.json" 2>&1
want 0 "$?" "and its whole-file twin, same parameters, no window"
want 0 "$?" "the built binary generates a cell served from line 120"
python3 - "$R60" <<'PY'
import json,sys,re
d=sys.argv[1]
lines=open(d+"/win/served.txt").read().split("\n")
assert lines[1].startswith("@@ 120-"), "the served range header is %r, want it to begin at 120" % lines[1]
assert re.match(r"^\s*120\|", lines[2]), "the first served row is %r, want line 120" % lines[2]
a=json.load(open(d+"/win/answer.json")); m=json.load(open(d+"/win/manifest.json"))
assert a["line"]>=120, "the answer at line %d is outside a window that starts at 120" % a["line"]
import os
served=os.path.getsize(d+"/win/served.txt")
assert served==m["served_bytes"], "served.txt is %d bytes but the manifest says %d" % (served, m["served_bytes"])
assert served>=20000, "the window served %d bytes, below its target" % served
twin=json.load(open(d+"/twin/manifest.json"))
assert twin["trial_id"]!=m["trial_id"], "the windowed cell and its whole-file twin share trial id %s, so a result for one scores against the other" % m["trial_id"]
PY
[ $? -eq 0 ] && ok "the window starts where asked, the target is inside it, and the window is what was sized" \
             || bad "the served window does not start at 120, excludes the target, or was not what the fit sized"
"$CURVE" generate -out "$R60/past" -bytes 20000 -position early -distractors 3 -seed 2 -selector odd-retries -from 100000 > "$R60/past.json" 2>&1
want 2 "$?" "and a window that starts past every line is refused (exit 2, the usage-error code) rather than served with no answer in it"
grep -q 'excludes the target' "$R60/past.json" \
  && ok "with a refusal that says the window excludes the target" \
  || bad "the refusal does not say why: $(head -c 200 "$R60/past.json")"
# The case the one above does NOT reach: a window that EXISTS and starts after
# the target. An early target sits near line 76 of a ~374-line cell; a window
# from 200 is served in full and holds no answer. The fit-loop refusal never
# fires here, so this is the only row that reaches the post-fit check — and a
# mutant that deleted that check survived the fence until this row existed.
"$CURVE" generate -out "$R60/gap" -bytes 20000 -position early -distractors 3 -seed 2 -selector odd-retries -from 200 > "$R60/gap.json" 2>&1
want 2 "$?" "a window that exists but starts after an early target is refused too"
grep -q 'excludes the target' "$R60/gap.json" \
  && ok "and says so, rather than serving a window with no answer in it" \
  || bad "the existing-window refusal does not say why: $(head -c 200 "$R60/gap.json")"
# Nothing this run started may outlive it — including a descendant already
# re-parented away from its spawner, the shape #101 took. The runner is a
# fresh process group of its own (top of file) and re-parenting does not
# change a process's group, so `pgrep -g $$` is exact: every survivor of
# this run and nothing else — the wrapper, and any pipeline peer of the
# caller's, are in the caller's group. pgrep writes to a file, not a $( )
# substitution: the substitution's subshell and its pipeline members are
# group members too, and were listed (measured). This is the positive case — a
# deliberate orphan must be VISIBLE — because a detector that has never seen
# one is a detector that always says clean; the negative, that nothing of the
# run survives, is asserted after the last row. Its parent is asserted to be
# not this shell rather than literally 1: under a subreaper (an init in a
# container, a user manager) the adopter is not 1, and the group is what
# matters.
[ "$(ps -o pgid= -p $$ | tr -d ' ')" = "$$" ] && ok "the runner is its own process-group leader" || bad "the runner is not its group leader: pgid $(ps -o pgid= -p $$ | tr -d ' ')"
( sleep 300 & echo $! > "$WORK/orphan.pid" )   # its parent exits at once; it is re-parented away and keeps the group
sleep 0.2
pgrep -g $$ > "$WORK/kids"; prc=$?
orphan=$(cat "$WORK/orphan.pid" 2>/dev/null)
oppid=$(ps -o ppid= -p "${orphan:-0}" 2>/dev/null | tr -d ' ')
[ "$prc" -eq 0 ] && [ -n "$orphan" ] && grep -qx "$orphan" "$WORK/kids" && [ -n "$oppid" ] && [ "$oppid" != "$$" ] \
  && ok "an orphan no longer under this shell is visible to the group check: pid $orphan under $oppid" \
  || bad "the group check cannot see an orphan (pgrep exit $prc, planted '$orphan', saw '$(grep -vx "$$" "$WORK/kids" | tr '\n' ' ')', parent '$oppid')"
# Only the planted orphan is killed. Killing everything the check listed swept
# away whatever a row above had left running, unreported, so the check after
# the last row could not see it (review of #236).
kill "${orphan:-999999999}" 2>/dev/null
# The wrapper's and the trap's promises get their own failing case: a nested
# run of THIS FILE'S prologue — the committed text up to the trap line, not a
# copy — with a body that plants an orphan and exits 7. Run plain, the exit
# status must come back 7 and the orphan must be gone: that is the trap's
# group kill, and deleting it leaves the orphan alive. Run hanging, an INT
# to the wrapper must end the run 143 within seconds with the orphan gone:
# that is the forwarding, and deleting the handler leaves the runner asleep.
sed -n '1,/^trap .*EXIT$/p' "$SELF" > "$WORK/probe.sh"
# The cut is the first EXIT trap. Assert it is the prologue and nothing
# more or less, so a moved trap cannot silently change what the probe proves.
[ "$(grep -c '^trap ' "$WORK/probe.sh")" = 1 ] && grep -q 'CONTRACT_GROUP' "$WORK/probe.sh" && grep -q 'kill -- -\$\$' "$WORK/probe.sh" && ! grep -q '^ok()' "$WORK/probe.sh" \
  && ok "the probe is this file's prologue: one trap, the wrapper, the group kill, and no rows" \
  || bad "the probe cut is not the prologue: $(wc -l < "$WORK/probe.sh" | tr -d ' ') lines, $(grep -c '^trap ' "$WORK/probe.sh") trap line(s)"
cat >> "$WORK/probe.sh" <<'PROBE'
echo $$ > "$1.runner"
( sleep 300 & )
sleep 0.2
pgrep -g $$ > "$1.all"; grep -vx "$$" "$1.all" > "$1"
[ "${2:-}" = hang ] && sleep 60
exit 7
PROBE
chmod +x "$WORK/probe.sh"
"$WORK/probe.sh" "$WORK/probe1" > /dev/null; want 7 "$?" "a nested run of this file's own prologue repeats the runner's exit status through the wrapper"
sleep 0.3; opid=$(cat "$WORK/probe1" 2>/dev/null)
[ -n "$opid" ] && ! kill -0 "$opid" 2>/dev/null \
  && ok "and its EXIT trap reaped an orphan already re-parented away (pid $opid)" \
  || bad "the prologue's EXIT trap left the nested run's orphan '${opid:-?}' alive"
kill "${opid:-999999999}" 2>/dev/null
"$WORK/probe.sh" "$WORK/probe2" hang > /dev/null & wp=$!
for _ in $(seq 1 50); do [ -s "$WORK/probe2" ] && break; sleep 0.1; done
t0=$(date +%s); kill -INT "$wp"; wait "$wp"; rc=$?; el=$(( $(date +%s) - t0 ))
opid=$(cat "$WORK/probe2" 2>/dev/null)
[ "$rc" -eq 143 ] && [ "$el" -lt 10 ] && [ -n "$opid" ] && ! kill -0 "$opid" 2>/dev/null \
  && ok "INT to the wrapper is forwarded: the nested runner ended 143 in ${el}s and its orphan with it" \
  || bad "INT to the wrapper: exit $rc after ${el}s, orphan '${opid:-?}' $(kill -0 "${opid:-999999999}" 2>/dev/null && echo alive || echo gone)"
kill "${opid:-999999999}" 2>/dev/null; kill -- -"$(cat "$WORK/probe2.runner" 2>/dev/null || echo 999999999)" 2>/dev/null; true

# 62. ADR-024 T2: an answer that SERVED something is not flagged an error, and a
# refusal still is.
#
# A unit test proves the function; it cannot prove the SHIPPED server returns it.
# The pairing is the whole point: a row asserting only the ABSENCE of a flag
# would pass against a server that had stopped flagging everything, refusals
# included, which is a different defect wearing this fix's face.
#
# Measured 2026-09-06 on Claude Code 2.1.263 — the same 152,594-character page
# reached a consumer GAPPED with isError set (line 78, then line 2309 of 2,380)
# and CONTINUOUS without it. This row cannot see the host, only what the binary
# sends, so it pins the half that is ours.
fixture
python3 -c "
with open('$R/pager.go','w') as f:
    f.write('package demo\n')
    for i in range(9000): f.write('// padding padding padding padding padding %06d\n' % i)
"
# ⚠ THROUGH FILES, NOT ARGV — the page is ~156 KB and Linux caps one argument at
# 131,072 bytes (MAX_ARG_STRLEN). The same trap §48 records.
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["pager.go"]}}}\n' | m mcp 2>/dev/null > "$WORK/p62.json"
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go"],"exclude":["x"]}}}\n' | m mcp 2>/dev/null > "$WORK/r62.json"
printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null > "$WORK/l62.json"
python3 - "$WORK/p62.json" "$WORK/r62.json" "$WORK/l62.json" <<'PY'
import json,sys
page=json.load(open(sys.argv[1]))["result"]
ref=json.load(open(sys.argv[2]))["result"]
listed=json.load(open(sys.argv[3]))["result"]
assert "isError" not in page, "the built server flags a PAGE as an error; a host then discards its middle, and the key must be ABSENT rather than false"
txt=page["content"][0]["text"]
assert "-- PARTIAL:" in txt, "the page does not say it is partial in the served text, which is now the only place it says so"
assert "line(s) remain" in txt, "the page's notice does not say how much remains"
assert json.loads(page["content"][1]["text"]).get("next_read"), "the page names no next_read, so a caller cannot continue"
assert ref.get("isError") is True, "a refusal that served NOTHING lost its flag; without this pair the row would pass against a server that flags nothing at all"
# ⚠ AND THE SURFACE MUST NOT STILL TEACH THE RETIRED PROMISE. The tools/list
# description is what a host reads before it ever calls anything, and it went on
# promising `isError true` for a page after the server had stopped sending it —
# neither §48 nor the first version of this row looked at it. Found by the Codex
# review of #118.
desc=[t for t in listed["tools"] if t["name"]=="mrw_read"][0]["description"]
assert "isError true" not in desc, "mrw_read's tools/list description still teaches the flag ADR-024 retired"
assert "-- PARTIAL:" in desc, "mrw_read's tools/list description does not teach what replaced the flag"
PY
[ $? -eq 0 ] && ok "the built server sends a page unflagged and saying so in its own text, and still flags a refusal" \
             || bad "the shipped paging or refusal shape is not what ADR-024 decided"

# 63. ADR-025: a read that served NOTHING is an error, on the path that serves.
#
# ADR-024 stopped an answer that SERVED something from claiming to be a failure,
# and left this return passing an unconditional false — which also covered the case
# where nothing was served at all. Two calls with the same outcome then disagreed
# on the flag, decided only by whether `grep` was passed.
#
# THE PAIRING IS THE WHOLE POINT, TWICE OVER. A row asserting only the flag would
# pass against a server that had started flagging everything; a row asserting only
# its absence would pass against the shipped v1.2.0 binary this record corrects.
# The third call is the one that pins WHICH count decides: a range that matches no
# line is still OBSERVED, so it must stay unflagged even though it counts a problem.
fixture
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["nope_dir"]}}}\n' | m mcp 2>/dev/null > "$WORK/n63.json"
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go","nope_dir"]}}}\n' | m mcp 2>/dev/null > "$WORK/s63.json"
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go:99"]}}}\n' | m mcp 2>/dev/null > "$WORK/m63.json"
python3 - "$WORK/n63.json" "$WORK/s63.json" "$WORK/m63.json" <<'PY'
import json,sys
nothing=json.load(open(sys.argv[1]))["result"]
sib=json.load(open(sys.argv[2]))["result"]
miss=json.load(open(sys.argv[3]))["result"]
assert nothing.get("isError") is True, "the built server does not flag a read that served NOTHING; the caller got none of what it asked for and cannot tell that from the envelope"
txt=nothing["content"][0]["text"]
import re
assert re.search(r"(?m)^==> nope_dir\s+UNREADABLE\s+\S", txt), "the flagged result does not carry an UNREADABLE record with a reason; flagging an answer must not be traded for dropping the only thing it carries, and a substring check would accept a response naming the path and nothing else"
assert "isError" not in sib, "the built server flags a read that served a good sibling; ADR-024 removed the flag from answers that delivered something and ADR-025 does not restore it"
assert "isError" not in miss, "the built server flags a range that matched no line in a file it OBSERVED; the observation count is the test, never the problem count"
PY
[ $? -eq 0 ] && ok "the built server flags a read that served nothing and names the path, and leaves a served answer unflagged" \
             || bad "the shipped served-nothing shape is not what ADR-025 decided"

# 64. ADR-026: an address may say how many lines follow, on BOTH paths.
#
# `A,+N` is the line A resolves to plus the N lines after it. Before it, the read
# path parsed the comma as a separator and `+N` as the ABSOLUTE line N — so
# `/func A/,+2` served the match and line 2 at exit 0 — while the plan path
# refused the same string as a bad line number. One grammar, two behaviours.
#
# THE PAIRING, AND WHY IT IS FOUR-WAY. A row asserting only that the form is
# served would pass against a parser that accepts anything, and a row asserting
# only the refusals would pass against the binary this record corrects. Both
# refusals are asserted on BOTH paths, because the two parsers are separate code
# and this section is the only place their agreement is checked: each package's
# own tests stay green while the two drift.
fixture
out=$(m read 'a.go:3,+1' 2>&1); rc=$?
want 0 "$rc" "a read with a relative end exits 0"
grep -q '@@ 3-4' <<<"$out" && ok "a.go:3,+1 serves the start plus one line" || bad "a.go:3,+1 did not serve 3-4: $out"
grep -q 'func B' <<<"$out" && ok "the line after the start is in the answer" || bad "the line after the start is missing"
out=$(m read 'a.go:/func A/,+2' 2>&1); rc=$?
want 0 "$rc" "a pattern with a relative end exits 0"
grep -q '@@ 3-5' <<<"$out" && ok "a.go:/func A/,+2 serves the match plus two lines" || bad "the pattern form did not serve 3-5: $out"
out=$(m read 'a.go:4,+10' 2>&1); rc=$?
want 0 "$rc" "a relative end past the last line clamps rather than failing"
grep -q '@@ 4-5' <<<"$out" && ok "the clamp stops at the last line" || bad "the clamp did not stop at the last line: $out"
out=$(m read 'a.go:+3' 2>&1); rc=$?
want 2 "$rc" "a read's relative end with no start is refused"
grep -q 'A,+3' <<<"$out" && ok "the read refusal names the fix" || bad "the read refusal does not name the fix: $out"
out=$(m read 'a.go:2,+0' 2>&1); rc=$?
want 2 "$rc" "a read's ,+0 is refused"
grep -q '+0' <<<"$out" && ok "the read refusal names what was written" || bad "the read refusal does not name +0: $out"
out=$(printf '@@ a.go 3,+1 replace anchor="func A()"\nfunc A() int { return 10 }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a plan hunk with a relative end applies"
grep -q 'func B' "$R/a.go" && bad "the relative end did not reach the line after the start" || ok "a plan's relative end replaced the start plus one line"
grep -q 'func C' "$R/a.go" && ok "the relative end stopped where it said" || bad "the relative end ran past the lines it named"
out=$(printf '@@ a.go +3 replace\nx\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a plan's relative end with no start is refused"
grep -q 'A,+3' <<<"$out" && ok "the plan refusal names the fix, in the same words as the read" || bad "the plan refusal does not name the fix: $out"
out=$(printf '@@ a.go 3,+0 replace\nx\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a plan's ,+0 is refused"
grep -q '+0' <<<"$out" && ok "the plan refusal names what was written" || bad "the plan refusal does not name +0: $out"
# The MCP write path carries the field through a SECOND wiring site, in
# internal/mcp/tools.go. The CLI probes above cannot see it: delete that line and
# every unit test stays green while an MCP caller's relative end is silently
# dropped, which is the shape of defect this whole record is about.
fixture
req=$(printf '@@ a.go 3,+1 replace anchor="func A()"\nfunc A() int { return 11 }\n' | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read()}}}))')
printf '%s\n' "$req" | m mcp 2>/dev/null > "$WORK/w64.json"
python3 - "$WORK/w64.json" "$R/a.go" <<'PY'
import json,sys
res=json.load(open(sys.argv[1]))["result"]
txt=res["content"][0]["text"]
after=open(sys.argv[2]).read()
assert "isError" not in res, "the MCP server refused a plan with a relative end: "+txt
assert "func B" not in after, "the MCP write did not reach the line after the start, so plan.Addr.RelEnd is not wired through internal/mcp/tools.go"
assert "func C" in after, "the MCP write ran past the lines the relative end named"
PY
[ $? -eq 0 ] && ok "the MCP write path carries a relative end too" \
             || bad "the MCP write path drops the relative end"
# ONE BASE MATRIX, DRIVEN THROUGH BOTH PATHS AND COMPARED WORD FOR WORD.
#
# The first cut of this section asserted fragments — that a refusal mentioned
# `A,+3` somewhere — which is what let the two parsers ship already diverged:
# `f.txt:,+3` served lines 1-4 at exit 0 on the read path while the plan path
# refused the identical string as an empty address. Found by the Codex review of
# PR #125. Both paths now call internal/addr, so the row compares the MESSAGE,
# not a substring of it: a shared function cannot drift, and this is what says
# so if someone re-splits it.
fixture
for a in ',+3' '5-7,+3' '3-,+3' '$-5,+3' '0,+3' '-,+3' '+3' '3,+0' '/func A/,/func B/,+2'; do
  rout=$(m read "a.go:$a" 2>&1); rrc=$?
  wout=$(printf '@@ a.go %s replace\nX\n' "$a" | m write - 2>&1); wrc=$?
  want 2 "$rrc" "read refuses the base $a"
  want 2 "$wrc" "a plan refuses the base $a"
  python3 - "$a" "$rout" "$wout" <<'PY'
import sys
a, rout, wout = sys.argv[1], sys.argv[2], sys.argv[3]
# The shared refusal always opens by quoting what the caller wrote. Take from
# that quote to the end of the line on each side and require them equal.
def core(txt):
    for line in txt.splitlines():
        i = line.find('"%s' % a)
        if i >= 0:
            return line[i:].strip()
    return None
r, w = core(rout), core(wout)
assert r is not None, "the read refusal of %s does not quote what the caller wrote:\n%s" % (a, rout)
assert w is not None, "the plan refusal of %s does not quote what the caller wrote:\n%s" % (a, wout)
assert r == w, "the two paths refuse %s in DIFFERENT words, which is the drift internal/addr exists to make impossible:\n  read: %s\n  plan: %s" % (a, r, w)
PY
  [ $? -eq 0 ] && ok "both paths refuse $a in the same words" || bad "the refusal of $a differs between the paths"
done

# An op that cannot honour a relative end must REFUSE it, not ignore it. Before
# the review, `2,+3 insert-after` applied and reported `ok a.go 2 insert-after`.
fixture
for op in insert-after insert-before; do
  out=$(printf '@@ a.go 3,+2 %s\nX\n' "$op" | m write - 2>&1); rc=$?
  want 2 "$rc" "a plan refuses a relative end on $op"
  grep -q 'single line' <<<"$out" && ok "the $op refusal says it takes a single line" || bad "the $op refusal does not name the fix: $out"
done
out=$(printf '@@ new.go 0,+2 create\nX\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a plan refuses a relative end on create"
# ...and the ops that DO take a range still work, or the rule above is just a ban.
fixture
out=$(printf '@@ a.go 3,+1 delete\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "delete still takes a relative end"
grep -q 'func B' "$R/a.go" && bad "delete with a relative end did not remove the second line" || ok "delete with a relative end removed both lines"

# THE RECEIPT ECHOES THE ADDRESS THE CALLER WROTE. It reported `3,+1` as `3`,
# which hides the span the hunk consumed from the one line a caller reads.
fixture
out=$(printf '@@ a.go 3,+1 replace anchor="func A()"\nX\n' | m write - 2>&1)
grep -q 'ok   a.go 3,+1 replace' <<<"$out" && ok "the receipt keeps the relative end" || bad "the receipt drops the relative end: $out"
fixture
out=$(printf '@@ a.go /func A/,+1 replace anchor="func A()"\nX\n' | m write - 2>&1)
grep -q 'ok   a.go /func A/,+1 replace' <<<"$out" && ok "the receipt keeps a pattern's relative end" || bad "the receipt mangles a pattern's relative end: $out"

# A WRITE REFUSES TO RUN PAST THE LAST LINE WHERE A READ CLAMPS, and that pairing
# is the assertion: each is the path's own existing rule — `2-99` serves what
# exists, `5-9999` is refused as out of range — so the relative form must not
# invent a third behaviour on either side.
fixture
out=$(m read 'a.go:4,+99' 2>&1); rc=$?
want 0 "$rc" "a read clamps a relative end at the last line"
grep -q '@@ 4-5' <<<"$out" && ok "the read clamp stops at the last line" || bad "the read clamp is wrong: $out"
out=$(printf '@@ a.go 4,+99 replace\nX\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a plan refuses a relative end that runs past the last line"
grep -q 'out of range' <<<"$out" && ok "the write refusal says out of range, as the explicit form does" || bad "the write refusal does not match the explicit form's wording: $out"
grep -q '4,+99' <<<"$out" && ok "the write refusal names the address the caller wrote" || bad "the write refusal does not name the caller's address: $out"

# EACH DOCUMENTED LOCATION IS GATED SEPARATELY, and the CLI help is gated against
# the BUILT BINARY rather than against the source string. A single
# `grep -q ',+N' README.md` passed from the Write passage alone while the Read
# passage and `mrw read --help` did not mention the form at all — one grep for
# several places is a gate that reports the best of them (Codex review of #125).
#
# README tutorial-phrase greps retired — ADR-053. Keep AGENTS.md, --help, and
# the MCP wire. A tidy of a README sentence is not a product break.
grep -q 'Addresses are line numbers, .*`A,+N` for the' AGENTS.md \
  && ok "AGENTS.md section 1 carries the relative end" \
  || bad "AGENTS.md section 1 does not carry the relative end"
grep -q 'a relative end$' AGENTS.md && grep -q '^`A,+N` — the line `A` plus the `N` lines after it' AGENTS.md \
  && ok "AGENTS.md section 2 carries the relative end" \
  || bad "AGENTS.md section 2 does not carry the relative end"
# ⚠ CAPTURE FIRST, THEN GREP. `"$MRW" read --help | grep -q` fails under this
# script's `set -o pipefail`: grep -q exits at the first match and closes the
# pipe, mrw takes SIGPIPE, and the PIPELINE reports 141 even though the text was
# there. Same family as the exit-code-through-a-pipe rule in CONTRIBUTING.md,
# and it cost a red row that was telling the truth about the shell rather than
# about the help text.
help=$("$MRW" read --help 2>&1)
grep -q 'A,+N' <<<"$help" \
  && ok "mrw read --help names the relative end" \
  || bad "mrw read --help does not name the relative end"
# README CLAMPS / false-clamp greps retired — ADR-053.
# AGENTS.md is the plan grammar every non-Claude agent reads, and it said a
# relative end "clamps at the last line" after the write path started refusing
# one — advice that makes a caller build a plan the tool rejects. Gated on the
# ASYMMETRY, not on the form's presence, because the form was present and the
# rule was wrong (second Codex review of #125).
grep -q 'A READ CLAMPS a relative end at the last line; a WRITE' AGENTS.md \
  && ok "AGENTS.md says which path clamps and which refuses" \
  || bad "AGENTS.md does not carry the read-clamps/write-refuses split"
grep -qi 'relative end clamps at the last line, has no backwards' AGENTS.md \
  && bad "AGENTS.md still tells agents a relative end clamps on a write" \
  || ok "the stale clamp claim is gone from AGENTS.md"

# THE MCP WIRE TEXT IS THE MCP CALLER'S ONLY GRAMMAR. A model reaching mrw over
# MCP never reads README.md or --help: it gets the initialize instructions and
# the two tool descriptions. All three omitted the form while the server
# accepted it — the defect issues #51 and #73 recorded, one surface over. Driven
# through the BUILT server so the assertion is about what goes on the wire.
fixture
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null > "$WORK/w64b.json"
python3 - "$WORK/w64b.json" <<'PY'
import json,sys
init=None; tools=None
for line in open(sys.argv[1]):
    line=line.strip()
    if not line: continue
    m=json.loads(line)
    if m.get("id")==1: init=m["result"]
    if m.get("id")==2: tools=m["result"]
assert init is not None, "the built server sent no initialize result"
assert tools is not None, "the built server sent no tools/list result"
instr=init.get("instructions","")
# BOTH passages, not "somewhere in the text": the reading grammar and the
# writing grammar are two answers a caller needs, and a mutant that broke
# only the reading one survived a check for the form anywhere.
assert "path:A,+N" in instr, "the READING grammar never mentions path:A,+N, so an MCP caller reading files cannot know the form exists"
assert "an N-M range, A,+N" in instr, "the WRITING grammar never mentions A,+N, so an MCP caller authoring a plan cannot know the form exists"
assert "clamps" in instr and "refuses" in instr, "the instructions do not tell an MCP caller that a read clamps where a write refuses"
byname={t["name"]: json.dumps(t) for t in tools["tools"]}
for n in ("mrw_read","mrw_write"):
    assert n in byname, "the built server does not declare %s" % n
    assert "A,+N" in byname[n], "%s's wire description never mentions A,+N" % n
PY
[ $? -eq 0 ] && ok "the MCP wire text teaches the relative end on both tools" \
             || bad "the MCP wire text does not teach the relative end"

# THE INTEGER BOUNDARY, THROUGH THE BUILT BINARY. `i + 1 + N` wraps negative at a
# large count and printed an INVERTED span — `@@ 2--9223372036854775807` — while
# serving nothing and exiting 0. Both the relative end and -C reach that
# arithmetic, and -C only refuses a NEGATIVE value, so a caller can type this.
# The assertion is the absence of an inverted span AND the presence of real
# lines: either alone would pass on a tool that had stopped serving anything.
fixture
MAXINT=9223372036854775807
out=$(m read "a.go:/func A/,+$MAXINT" 2>&1); rc=$?
want 0 "$rc" "a pattern with a relative end at the integer boundary clamps"
grep -q '@@ 3-5' <<<"$out" && ok "the pattern branch clamps at the last line" || bad "the pattern branch did not clamp: $out"
grep -q -- '--' <<<"$out" && bad "the pattern branch produced an inverted span, so the end wrapped" || ok "the pattern branch produced no inverted span"
out=$(m read "a.go:3,+$MAXINT" 2>&1); rc=$?
want 0 "$rc" "a numeric relative end at the integer boundary clamps"
grep -q '@@ 3-5' <<<"$out" && ok "the numeric branch clamps at the last line" || bad "the numeric branch did not clamp: $out"
out=$(m read -C "$MAXINT" 'a.go:/func A/' 2>&1); rc=$?
want 0 "$rc" "a context at the integer boundary clamps"
grep -q '@@ 1-5' <<<"$out" && ok "-C clamps to the whole file" || bad "-C did not clamp: $out"
grep -q -- '--' <<<"$out" && bad "-C produced an inverted span, so the end wrapped" || ok "-C produced no inverted span"
out=$(printf '@@ a.go 3,+%s replace\nX\n' "$MAXINT" | m write - 2>&1); rc=$?
want 1 "$rc" "a write refuses a relative end at the integer boundary"
grep -q 'out of range' <<<"$out" && ok "the write refusal says out of range" || bad "the write refusal is wrong at the boundary: $out"

# A PATTERN ENDING IN A LITERAL BACKSLASH is closed by its next slash, because
# the backslash before it is itself escaped. All three delimiter scanners tested
# only the preceding byte and read it as unclosed, refusing a legal address.
fixture
printf 'a\\b\nplain\n' > "$R/bs.txt"
m read bs.txt >"$WORK/served.out" 2>&1
out=$(m read 'bs.txt:/\\/,+1' 2>&1); rc=$?
want 0 "$rc" "a pattern ending in a backslash is closed on the read path"
grep -q '@@ 1-2' <<<"$out" && ok "the backslash pattern resolves to its match plus one" || bad "the backslash pattern did not resolve: $out"
# anchor="a" is line 1 of bs.txt (`a\b`), deliberately the plain half of it: the
# row is about the DELIMITER scanner closing a pattern that ends in a backslash,
# and an anchor carrying its own backslash would put two escaping questions in
# one assertion.
out=$(printf '@@ bs.txt /\\\\/,+1 replace anchor="a"\nX\nY\n' | m write - 2>&1); rc=$?
want 0 "$rc" "a pattern ending in a backslash is closed on the plan path too"

# THE TWO GRAMMARS AGREE ON EVERY SHAPE NAMED BELOW, which is narrower than
# "agree on what is malformed" and is what this row can actually assert. Seven
# shapes the read path once accepted and the plan path refused: `/` as an
# empty regexp matching every line at exit 0, `//`, `/a/garbage` compiled as the
# pattern a/garbage, `/a/,/b/,/c/` reduced to its first endpoint, `/a/,//` with
# an empty END pattern, and the trailing-comma forms `5,+2,` and `/a/,/b/,`
# whose empty component splitRanges silently dropped.
for a in '/' '//' '/a/garbage' '/a/,/b/,/c/' '/a/,//' '5,+2,' '/a/,/b/,'; do
  m read "a.go:$a" >"$WORK/served.out" 2>&1; rr=$?
  printf '@@ a.go %s replace\nX\n' "$a" | m write - > /dev/null 2>&1; wr=$?
  want 2 "$rr" "a read refuses the malformed pattern $a"
  want 2 "$wr" "a plan refuses the malformed pattern $a"
done
# ...and the legal forms still parse on both, or the rule above is just a ban.
out=$(m read 'a.go:/func A/,/func C/' 2>&1); want 0 "$?" "a two-pattern read still parses"
grep -q '@@ 3-5' <<<"$out" && ok "the two-pattern read serves its range" || bad "the two-pattern read is wrong: $out"

# A QUOTE IS AN ORDINARY REGEXP CHARACTER. The header splitter toggled on it
# inside a pattern and CONSUMED it, so /^"x"$/ reached the parser as /^x$/ — a
# different expression, a different line, and a receipt echoing the mutation.
fixture
printf 'package demo\n\nfunc A() int { return 1 }\nconst Q = "quoted"\n' > "$R/q.go"
m read q.go >"$WORK/served.out" 2>&1
out=$(printf '@@ q.go /"quoted"/ replace\nconst Q = "changed"\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a plan pattern containing quotes applies"
grep -q 'ok   q.go /"quoted"/ replace' <<<"$out" && ok "the receipt echoes the pattern with its quotes" || bad "the receipt shows a mutated address: $out"
grep -q 'changed' "$R/q.go" && ok "the quoted pattern reached the line it named" || bad "the quoted pattern edited the wrong line"
out=$(printf '@@ q.go 3 replace anchor="func A"\nfunc A() int { return 9 }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a quoted anchor still works, so the toggle was narrowed and not removed"

# 65. ADR-027: an empty file is created on purpose, or not at all.
#
# A create carrying no body made an empty file and reported `ok`. ADR-006 refuses
# the same shape for replace, and the reasoning is not about deletion: a body
# lost in transit is indistinguishable from a body never written, and a create is
# very often the LAST hunk of a plan, which is where a truncation loses one.
#
# THE PAIRING IS THE POINT, AND SO IS THE FILE'S ABSENCE. A row asserting only
# the refusal would pass against a tool that had stopped creating files at all,
# so the deliberate `body=0` case is asserted beside it. And a refusal that still
# left the file behind would be worse than the behaviour being removed (ADR-004),
# so the row checks the file is not there rather than only the exit code.
fixture
out=$(printf '@@ new.txt 0 create\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a create carrying no body is refused"
grep -q 'body=0' <<<"$out" && ok "the refusal names the fix" || bad "the refusal does not name body=0: $out"
[ -e "$R/new.txt" ] && bad "the refused create left the file behind" || ok "the refused create left nothing behind"
out=$(printf '@@ empty.txt 0 create body=0\n' | m write - 2>&1); rc=$?
want 0 "$rc" "create body=0 is the deliberate empty file"
[ -f "$R/empty.txt" ] && [ ! -s "$R/empty.txt" ] && ok "body=0 created a file with no content" || bad "body=0 did not create an empty file: $out"
out=$(printf '@@ full.txt 0 create\nhello\n' | m write - 2>&1); rc=$?
want 0 "$rc" "an ordinary create still applies"
grep -q hello "$R/full.txt" && ok "the ordinary create wrote its body" || bad "the ordinary create lost its body"
out=$(printf '@@ a.go 3 replace\n' | m write - 2>&1); rc=$?
want 2 "$rc" "an empty-bodied replace is still refused, in its own words"
grep -q 'would delete' <<<"$out" && ok "the replace refusal is unchanged" || bad "the replace refusal changed: $out"

# 66. ADR-028: a guard does not read back what was not served.
#
# `anchor=` was checked ABOVE the ledger, so a FAILED anchor quoted the line the
# file holds — on a line the caller had never been served. ADR-002 and ADR-005
# say mrw does not tell you what it has not shown you, and this was the one path
# that did. ADR-008 had already moved its own guard below `covered()` for the
# same reason and left this sibling; the asymmetry is closed now.
#
# BOTH HALVES, OR THE ROW PROVES NOTHING. A section asserting only the absence
# would pass against a binary that had stopped checking anchors altogether, so
# the served-line half is asserted beside it. And the fixture serves a NARROW
# range: a file served NOT AT ALL is refused by the whole-file gate before
# either guard runs, which is green with the ordering reversed (BACKLOG:226).
#
# ALL FOUR ANCHORED OPS. `replace` and `delete` compare the anchor inline; the
# two insertions reach it through a guard closure invoked beside `covered()`.
# The first cut of ADR-028 moved one pair and claimed every guard, and the first
# version of this section drove three of the four while the record above it said
# four — neither was caught by a gate, because a section that runs three ops
# passes exactly like one that runs four (PR #128, reviews two and three).
#
# ⚠ A SERVED-LINE CONTROL MUST ASSERT THE REFUSAL, NOT ONLY THE QUOTE. Grepping
# the output for the line's text passes against a binary that has stopped
# checking anchors, whenever that op's SUCCESS receipt happens to print the line
# — which `delete`'s does, ADR-008 having made it say what it removed (review
# four). `insert-after` had the same shape and survived only because its receipt
# prints no content, which is an accident of receipt shape and not a property of
# the check.
#
# ⚠ AND ONE FRESH FIXTURE PER CASE. A case that wrongly SUCCEEDS writes the file
# AND records the ledger as WHOLE, so every later case in the section silently
# becomes a different test than the one it is named for — an unserved line that
# is no longer unserved, a sentinel that is no longer there (review five). The
# isolation is what makes each assertion evidence about its own op.
anchored_fixture() {
  fixture
  printf 'public line\nUNSERVED-SENTINEL-42\nthird\n' > "$R/s.txt"
  m read 's.txt:1' >"$WORK/served.out" 2>&1
}
anchored_plan() {  # $1 = op, $2 = line; `delete` is the one op that carries no body
  if [ "$1" = delete ]; then
    printf '@@ s.txt %s delete anchor="no-such-text"\n' "$2"
  else
    printf '@@ s.txt %s %s anchor="no-such-text"\nX\n' "$2" "$1"
  fi
}
for op in replace delete insert-after insert-before; do
  anchored_fixture
  out=$(anchored_plan "$op" 2 | m write - 2>&1); rc=$?
  want 1 "$rc" "an anchored $op on an unserved line is refused"
  grep -q 'UNSERVED-SENTINEL-42' <<<"$out" \
    && bad "the $op refusal read back a line the caller was never served: $out" \
    || ok "the $op refusal reads back no line the caller was not served"
  # Paired with the absence above on purpose: an absence assertion alone is
  # satisfied by EMPTY output, so the ledger's own message is asserted present.
  grep -q 'has not been read' <<<"$out" \
    && ok "the $op refusal is the ledger's, naming what was served" \
    || bad "the $op refusal is not the ledger's: $out"

  anchored_fixture
  out=$(anchored_plan "$op" 1 | m write - 2>&1); rc=$?
  want 1 "$rc" "an anchored $op on a SERVED line is still anchor-checked"
  grep -q 'anchor "no-such-text" not in line 1' <<<"$out" \
    && ok "a served line's $op anchor failure is the anchor's own" \
    || bad "$op stopped checking anchors: $out"
  grep -q 'public line' <<<"$out" \
    && ok "a served line's $op anchor failure still quotes it" \
    || bad "the $op anchor no longer quotes a line the caller was served: $out"
done
# 67. ADR-029: one file is one observation, whatever the plan calls it.
#
# The ledger is keyed on the path a caller TYPED, and a file has more than one
# valid name. The file-level check recovered an aliased observation with
# os.SameFile (issue #47); the per-line gate looked the ledger up AGAIN by exact
# key, missed, and read the miss as "this caller has read nothing" — which the
# file-level check has already refused. So for an alias every per-line check
# passed and a write to lines never served APPLIED, at exit 0. Not a weakened
# guard: an absent one.
#
# BOTH HALVES. A row asserting only the refusal is green against a binary that
# refuses EVERY alias, which is issue #47 undone — a file that HAS been read
# must not be refused as unread because the caller typed another valid name for
# it. The whole-read case is asserted beside it.
#
# ⚠ THIS ROW CARRIES THE SYMLINK HALF ONLY. The other alias is a case-only
# spelling, which needs a case-INsensitive filesystem; this script runs on Linux
# in CI, where real.txt and REAL.txt are two different files, so the case is not
# skipped here — it cannot be written here at all. It is covered by
# TestAnAliasSpellingIsTheSameFileToThePerLineLedger, which probes the
# filesystem at runtime and runs on the Windows CI job.
#
# ⚠ And one fresh fixture per case, for §66's reason: a case that wrongly
# SUCCEEDS writes the file AND records the ledger whole, so every later case
# silently becomes a different test than its name says.
alias_fixture() {
  fixture
  printf 'first\nsecond\nthird\nUNSERVED-SENTINEL-29\n' > "$R/real.txt"
  ln -s real.txt "$R/link.txt"
}
alias_fixture
m read 'real.txt:1' >"$WORK/served.out" 2>&1
before=$(cksum < "$R/real.txt")
out=$(printf '@@ link.txt 4 replace\nPWNED\n' | m write - 2>&1); rc=$?
want 1 "$rc" "an alias-spelled write to a line never served is refused"
# The PER-LINE message, not merely "has not been read": the file-level check
# says that too, for a file no spelling of which is in the ledger, and matching
# the shorter string would let this case pass on the wrong refusal.
grep -q 'has not been read: mrw served' <<<"$out" \
  && ok "the alias refusal is the per-line ledger's, naming what was served" \
  || bad "the alias refusal is not the per-line ledger's: $out"
# Byte identity by digest, not the absence of one string: a write that landed
# anywhere else in the file would leave PWNED absent and the file changed.
[ "$(cksum < "$R/real.txt")" = "$before" ] \
  && ok "the file the alias names is byte-identical" \
  || bad "the alias spelling changed the file it names: $(cat "$R/real.txt")"

# Issue #47's half, and the reason this is a resolution rather than a ban.
alias_fixture
m read 'real.txt' >"$WORK/served.out" 2>&1
out=$(printf '@@ link.txt 4 replace\nrewritten\n' | m write - 2>&1); rc=$?
want 0 "$rc" "a WHOLE read still licenses a write spelled as the alias"
grep -q 'rewritten' "$R/real.txt" && ok "the licensed alias write reached the file" || bad "the alias write did not reach the file: $out"

# ADR-028's property reaching the alias: with the per-line gate absent, a failed
# anchor printed the line as it always had.
alias_fixture
m read 'real.txt:1' >"$WORK/served.out" 2>&1
out=$(printf '@@ link.txt 4 replace anchor="no-such-text"\nX\n' | m write - 2>&1); rc=$?
want 1 "$rc" "an alias-spelled anchored hunk on an unserved line is refused"
grep -q 'UNSERVED-SENTINEL-29' <<<"$out" \
  && bad "the alias refusal read back a line the caller was never served: $out" \
  || ok "the alias refusal reads back no line the caller was not served"
# 68. ADR-031: a page licenses only what came back.
#
# A page mrw SENDS is not a page the caller RECEIVED. Measured 2026-09-05: the
# host cut the middle out of a 2,727-line page, the model saw lines 1-90 and
# 2644-2727, mrw recorded that page whole (1-2727), and a write to line 1500
# applied at exit 0.
# So a served span is held pending against checkpoints woven through the text
# and reaches the ledger only when the caller echoes them.
#
# BOTH HALVES. A row asserting only the refusal passes against a server that
# licenses NOTHING, which would be a ban rather than a narrowing — so the
# acknowledged write must succeed in the same section.
#
# ⚠ AND THE UNACKNOWLEDGED HALF IS THE MIDDLE, NOT THE TAIL. The measured cut
# kept both ends, so a fixture that omits the LAST checkpoint is green against
# the one-token-at-the-end design ADR-031 rejects, and proves nothing.
fixture
python3 - "$R" <<'PY'
import sys, pathlib
pathlib.Path(sys.argv[1], "big.txt").write_text("".join("line %d\n" % i for i in range(1, 12001)))
PY
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["big.txt"]}}}\n' | m mcp 2>/dev/null)
want 0 $? "the server answers a read that must page"
# the page is kept in $R/page.json below: $out is reused by every write
# ⚠ THE PAGE GOES THROUGH A FILE, NOT THROUGH ARGV. Linux caps a single
# argument at 128 KB (MAX_ARG_STRLEN) while macOS does not, so passing a
# 200 KB page as sys.argv[1] worked locally and died on CI with "Argument
# list too long" — and the row then reported the FEATURE missing rather
# than the harness broken.
printf '%s' "$out" > "$R/page.json"
cks=$(python3 - "$R/page.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
print(" ".join(re.findall(r"^-- ck ([0-9a-f]{16}) open ", r["content"][0]["text"], re.M)))
PY
)
[ -n "$cks" ] && ok "a paged read carries checkpoints a caller can echo" || bad "a paged read carries no checkpoint, so nothing can ever be acknowledged"
# ⚠ COVERAGE, not just the markers that happen to exist. Validating only the
# blocks present passes against a server that brackets lines 1-200 and leaves
# the rest unmarked: every assertion would be satisfied and the page would
# license nothing anyone could acknowledge. The review of PR #132 demonstrated
# exactly that. So parse every block, require them to TILE the served range with
# no gap and no overlap, require more than one, and check the negative probe
# lands inside a real block that is NOT acknowledged.
python3 - "$R/page.json" <<'PY'
import json,re,sys
t=json.load(open(sys.argv[1]))["result"]["content"][0]["text"].split(chr(10))
blocks=[]
for i,l in enumerate(t):
    m=re.match(r"^-- ck ([0-9a-f]{16}) open lines (\d+)-(\d+) \((\d+) lines follow\)$", l)
    if m:
        blocks.append({"ck":m.group(1),"a":int(m.group(2)),"b":int(m.group(3)),"n":int(m.group(4)),"at":i})
assert len(blocks) > 1, "a page with one block cannot show that acknowledging is per-region"
served=[int(re.match(r"^ *(\d+)\|", l).group(1)) for l in t if re.match(r"^ *\d+\|", l)]
assert served, "the page served no numbered lines"
for blk in blocks:
    close=[i for i,l in enumerate(t) if l == "-- ck " + blk["ck"] + " close"]
    assert close, "checkpoint has no close marker: " + blk["ck"]
    body=[l for l in t[blk["at"]+1:close[0]] if re.match(r"^ *\d+\|", l)]
    assert len(body) == blk["n"], "checkpoint " + blk["ck"] + " miscounts its span"
    nums=[int(re.match(r"^ *(\d+)\|", l).group(1)) for l in body]
    assert nums[0] == blk["a"] and nums[-1] == blk["b"], "checkpoint " + blk["ck"] + " brackets lines it does not claim"
blocks.sort(key=lambda x: x["a"])
assert blocks[0]["a"] == min(served) and blocks[-1]["b"] == max(served), "the blocks do not span the served range"
for x, y in zip(blocks, blocks[1:]):
    assert y["a"] == x["b"] + 1, "blocks leave a gap or overlap, so some served lines can never be acknowledged"
assert sum(b["n"] for b in blocks) == len(served), "served line count and bracketed line count differ"
first, last = blocks[0]["ck"], blocks[-1]["ck"]
inside=[b for b in blocks if b["a"] <= 900 <= b["b"]]
assert inside, "line 900 is in no block, so refusing it says nothing about acknowledgement"
assert inside[0]["ck"] not in (first, last), "line 900 sits in an acknowledged block, so the negative probe cannot fail"
assert first != last, "the first and last checkpoints are the same block"
PY
want 0 $? "the page tiles its served lines and the negative probe sits in a real unacknowledged block"
# ⚠ Guarded, because  is on and this section must REPORT rather than
# abort. Against a server without ADR-031 there are no checkpoints at all, and
# an unguarded `shift` on an empty list took the whole script down with it —
# so the row that proves the feature missing also silenced every row after it.
first=""; last=""
for c in $cks; do [ -z "$first" ] && first="$c"; last="$c"; done
if [ -z "$first" ]; then first="none"; last="none"; fi

# Unacknowledged: the page licenses nothing at all.
# ⚠ EVERY BLOCK BEFORE ANY ACKNOWLEDGEMENT, not just line 1. Probing one line
# lets a server that records the LAST span while serving still pass: the
# post-ack assertions cannot tell a licence that came from acknowledgement from
# one that was there already (eighth review of PR #132).
python3 - "$R/page.json" > "$R/preack.plan" <<'PY'
import json,re,sys
t=json.load(open(sys.argv[1]))["result"]["content"][0]["text"]
for m in re.finditer(r"^-- ck [0-9a-f]{16} open lines (\d+)-", t, re.M):
    print("@@ big.txt %s replace" % m.group(1)); print("X")
PY
plan_preack=$(python3 -c "
import sys,json
print(json.dumps(open(sys.argv[1]).read())[1:-1])" "$R/preack.plan")
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"%s","dry_run":true}}}\n' "$plan_preack" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == sc["failed"] and sc["applied"] == 0, "a freshly served page licensed something before any acknowledgement: %s" % sc
PY
want 0 $? "a freshly served page licenses NO block until it is acknowledged"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ big.txt 1 replace\\nX\\n","dry_run":true}}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 1, "a write against an UNACKNOWLEDGED page was allowed: %s" % sc
PY
want 0 $? "an unacknowledged page licenses nothing"

# Acknowledged FIRST and LAST, and the middle deliberately not — the shape the
# host's cut actually produced.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ big.txt 1 replace\\nX\\n","dry_run":true,"ack":["%s","%s"]}}}\n' "$first" "$last" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 0, "an acknowledged segment did not license its own lines: %s" % sc
PY
want 0 $? "an acknowledged segment licenses its own lines"

# ⚠ AND A WRITE INSIDE THE **LAST** ACKNOWLEDGED SPAN. The row above sends two
# ids and writes only line 1, so the last id could be ignored entirely and the
# section would still pass — found by the review of PR #132. The last span's
# range is read out of its own open marker rather than assumed.
lastspan=$(python3 - "$R/page.json" "$last" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
m=re.search(r"^-- ck %s open lines (\d+)-(\d+) " % re.escape(sys.argv[2]), r["content"][0]["text"], re.M)
print(m.group(1) if m else "")
PY
)
[ -n "$lastspan" ] && ok "the last checkpoint names the span it opens" || bad "the last checkpoint's open marker carries no range"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ big.txt %s replace\\nX\\n","dry_run":true,"ack":["%s","%s"]}}}\n' "$lastspan" "$first" "$last" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 0, "the LAST acknowledged segment licensed nothing, so acking it is unexercised: %s" % sc
PY
want 0 $? "the last acknowledged segment licenses its own lines too"

# And a line covered only by a checkpoint nobody echoed stays unwritable.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ big.txt 900 replace\\nX\\n","dry_run":true,"ack":["%s","%s"]}}}\n' "$first" "$last" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 1, "a line in the UNACKNOWLEDGED middle was writable: %s" % sc
assert "ack" in json.dumps(sc), "the refusal does not name the remedy: %s" % sc
PY
want 0 $? "an unacknowledged middle stays unwritable, and the refusal names ack"

# ⚠ AND EVERY RENDERED BLOCK MUST BE PROMOTABLE, not just the two ends. The
# tiling check proves the marker TEXT covers the page; a server could render
# every interior block and store only the first and last mappings, and every
# assertion above would still pass while no caller could acknowledge anything
# between them. The fourth review of PR #132 named that hole. So: acknowledge
# EVERY id, and write one line from EVERY declared block in a single plan.
plan=$(python3 - "$R/page.json" <<'PY'
import json,re,sys
t=json.load(open(sys.argv[1]))["result"]["content"][0]["text"].split(chr(10))
out=[]
for l in t:
    m=re.match(r"^-- ck ([0-9a-f]{16}) open lines (\d+)-(\d+) ", l)
    if m: out.append("@@ big.txt %s replace\\nX\\n" % m.group(2))
print("".join(out))
PY
)
allcks=$(python3 - "$R/page.json" <<'PY'
import json,re,sys
t=json.load(open(sys.argv[1]))["result"]["content"][0]["text"]
print(",".join('"%s"' % c for c in re.findall(r"^-- ck ([0-9a-f]{16}) open ", t, re.M)))
PY
)
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"%s","dry_run":true,"ack":[%s]}}}\n' "$plan" "$allcks" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 0, "acknowledging every rendered id did not license one line from every block: %s" % sc["hunks"]
PY
want 0 $? "every rendered checkpoint is promotable, not only the two ends"
# 69. ADR-033: a cap of zero is a cap.
#
# --max-lines 0 meant UNLIMITED: both guards asked `> 0`, so a cap of zero was
# indistinguishable from no cap — and NOTHING was reported withheld, though the
# README promises whatever is withheld always is. body=0 (ADR-027) and lines=0
# had already been decided the other way.
#
# BOTH SPELLINGS. A row asserting only the zero case passes against a binary
# that serves nothing at all, which is a ban rather than a narrowing.
fixture
printf 'alpha\nbravo\ncharlie\n' > "$R/cap.txt"   # DISTINCT lines: a count passes on duplicates
out=$(m read --max-lines 0 cap.txt 2>&1); rc=$?
want 1 "$rc" "a cap of zero serves nothing, and a read that served nothing is an error"
grep -q 'WITHHELD 3 line(s)' <<<"$out" \
  && ok "a cap of zero reports what it withheld, with the count" \
  || bad "a cap of zero withheld silently: $out"
grep -qE '^ +[0-9]+\| ' <<<"$out" && bad "a cap of zero served content: $out" || ok "a cap of zero served no content line"
# The other spelling: no flag at all is how a caller asks for no cap.
out=$(m read cap.txt 2>&1); rc=$?
want 0 "$rc" "a read with no cap is served"
# ⚠ THE SEQUENCE, COMPARED EXACTLY. Three independent greps accept the same
# lines reordered or repeated, which the second review of PR #133 reproduced —
# and counting numbered lines accepts substituted content. The served sequence
# is extracted and compared to the fixture's, in order.
seq=$(sed -nE 's/^ *[0-9]+\| (.*)$/\1/p' <<<"$out" | tr '\n' ',')
[ "$seq" = "alpha,bravo,charlie," ] \
  && ok "an ABSENT --max-lines serves the file whole, in order, without repeats" \
  || bad "omitting the flag no longer serves the file whole: got [$seq] from: $out"
# 70. ADR-032: the ceiling is the caller's, and it bounds the whole answer.
#
# Both tools advertise `anthropic/maxResultSizeChars` and only the read path
# kept it. Measured 2026-09-07: a 4,000-hunk dry-run came back at 453,632
# characters against an advertised 200,000, and a host that trusts the number
# truncates — which is the answer ADR-031 exists because mrw cannot see.
#
# ⚠ THE ENCODED RESULT IS WHAT IS MEASURED, and it is measured as the SERVER'S
# OWN BYTES — raw_decode hands back the exact substring of the response line
# rather than a re-serialization of a parsed object. ADR-031 shipped a size
# check on a buffer that was not what got sent, twice, and a row that
# re-encodes what it parsed repeats that at the gate instead of in the code.
#
# ⚠ BOTH TOOLS AND BOTH BUDGETS. A row covering only reads passes against the
# pre-ADR-032 tree, and a row covering only the default proves nothing about a
# number the caller set.
fixture
python3 - "$R" <<'PY'
import sys, pathlib
d = pathlib.Path(sys.argv[1])
# ⚠ NON-ASCII ON PURPOSE. The size assertion below counts UTF-8 BYTES, and an
# all-ASCII fixture makes bytes and code points identical — so the correction
# from len(str) to len(bytes) would pass either way and assert nothing. Each
# line carries a multi-byte character, which is exactly the content that made
# the two counts diverge. Found by the second Codex review of #135.
d.joinpath("wide.txt").write_text("".join("línė %05d — ok\n" % i for i in range(1, 4001)), encoding="utf-8")
d.joinpath("plan.txt").write_text("".join("@@ wide.txt %d replace\nlínė %05d — ok\n" % (i, i) for i in range(1, 4001)), encoding="utf-8")
d.joinpath("small.txt").write_text("alpha\nbravo\n", encoding="utf-8")
PY
# The read is what licenses the write, so the receipt below carries 4,000
# VERDICTS rather than 4,000 refusals — the shape an elision may shorten.
m read wide.txt small.txt >"$WORK/served.out" 2>&1
want 0 $? "the ceiling fixture is served"
python3 - "$R" > "$R/calls.jsonl" <<'PY'
import json, pathlib, sys
d = pathlib.Path(sys.argv[1])
for args in ({"name": "mrw_read", "arguments": {"specs": ["wide.txt"]}},
             {"name": "mrw_write", "arguments": {"plan": d.joinpath("plan.txt").read_text(), "dry_run": True}},
             {"name": "mrw_write", "arguments": {"plan": "@@ small.txt 1 replace\nALPHA\n", "dry_run": True}}):
    print(json.dumps({"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": args}))
PY
for ceiling in 200000 20000; do
  flags=""
  [ "$ceiling" = 200000 ] || flags="--max-result-chars $ceiling"
  adv=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list"}\n' | m mcp $flags 2>/dev/null \
        | python3 -c 'import sys,json; d=json.load(sys.stdin); print(" ".join("%s=%s" % (t["name"], t["_meta"]["anthropic/maxResultSizeChars"]) for t in sorted(d["result"]["tools"], key=lambda t: t["name"])))')
  [ "$adv" = "mrw_read=$ceiling mrw_write=$ceiling" ] \
    && ok "both tools advertise the ceiling in force ($ceiling)" \
    || bad "the advertised ceiling is not the one in force ($ceiling): $adv"
  m mcp $flags < "$R/calls.jsonl" > "$R/answers.jsonl" 2>/dev/null
  out=$(python3 - "$R/answers.jsonl" "$ceiling" <<'PY'
import json, sys
limit, dec, over, seen = int(sys.argv[2]), json.JSONDecoder(), [], 0
for line in open(sys.argv[1]):
    line = line.rstrip("\n")
    if not line:
        continue
    i = line.index('"result":') + len('"result":')
    _, end = dec.raw_decode(line, i)
    # ⚠ AND THIS CORRECTION CANNOT BE MADE TO BIND FROM A GREEN RUN — said here
    # rather than left for the next reader to discover. A correct server enforces
    # the ceiling in BYTES, so every answer it delivers has bytes <= ceiling, and
    # therefore code points <= bytes <= ceiling as well: both counts pass. The
    # unit only diverges against a server that has ALREADY shipped an oversized
    # answer, which is the case this row exists to catch and the one no passing
    # suite can stage. Reverting to len() leaves the contract green; that is a
    # property of the check, not evidence the check is idle.
    # ⚠ BYTES, NOT CHARACTERS. json.load hands back a decoded str, so len()
    # counts code points; the ceiling is enforced in bytes (schema.go says so
    # and says why). Equal for an ASCII fixture, which is what made this look
    # right, and wrong the moment a served line is not ASCII. Codex, #135.
    n = len(line[i:end].encode("utf-8"))   # the server's own bytes, not a re-encoding
    seen += 1
    if n > limit:
        over.append(n)
print("answers=%d over=%s" % (seen, ",".join(str(n) for n in over) or "none"))
PY
)
  [ "$out" = "answers=3 over=none" ] \
    && ok "no answer exceeds the ceiling in force ($ceiling)" \
    || bad "an answer exceeded the ceiling in force ($ceiling): $out"
done
# ⚠ THE PAIR THAT MUST STILL WORK. A row asserting only the bound passes
# against a server that elides everything, which would be a ban rather than a
# narrowing. The big receipt says what it dropped; the small one drops nothing
# and says nothing.
m mcp < "$R/calls.jsonl" > "$R/answers.jsonl" 2>/dev/null
out=$(python3 - "$R/answers.jsonl" <<'PY'
import json, sys
rows = [json.loads(l)["result"] for l in open(sys.argv[1]) if l.strip()]
big, small = rows[1]["structuredContent"], rows[2]["structuredContent"]
print("big=%s small=%s failed=%s dry=%s hunks=%s" % (
    "elided" if big.get("elided") else "whole",
    "elided" if small.get("elided") else "whole",
    big["failed"], big["dry_run"], len(small["hunks"])))
PY
)
[ "$out" = "big=elided small=whole failed=0 dry=True hunks=1" ] \
  && ok "an oversized receipt says what it dropped and a small one drops nothing" \
  || bad "the elision fired on the wrong receipt, or silently: $out"

# ⚠ AND A REAL WRITE AT A CEILING TOO SMALL TO REPORT ONE. Measured on the
# built binary before this guard existed: `--max-result-chars 0` applied a
# licensed one-hunk write, recorded the ledger, and answered "0 of 1 hunk(s)
# failed and nothing was written". The file is the assertion, not the message —
# a fix that only corrected the wording would leave the write applying.
fixture
printf 'alpha\nbravo\n' > "$R/tiny.txt"
python3 - "$R" > "$R/one.jsonl" <<'PY'
import json, sys
print(json.dumps({"jsonrpc": "2.0", "id": 1, "method": "tools/call",
                  "params": {"name": "mrw_read", "arguments": {"specs": ["tiny.txt:1"]}}}))
PY
python3 - "$R" > "$R/onewrite.jsonl" <<'PY'
import json, sys
print(json.dumps({"jsonrpc": "2.0", "id": 2, "method": "tools/call",
                  "params": {"name": "mrw_write",
                             "arguments": {"plan": "@@ tiny.txt 1 replace\nMUTATED\n"}}}))
PY
m mcp < "$R/one.jsonl" >/dev/null 2>&1
out=$(m mcp --max-result-chars 0 < "$R/onewrite.jsonl" 2>/dev/null | python3 -c '
import json, sys
d = json.load(sys.stdin)
print("error" if "error" in d else "result")')
[ "$out" = "error" ] \
  && ok "a ceiling too small to report a write refuses it instead" \
  || bad "a write under an unreportable ceiling returned a tool result: $out"
grep -q '^alpha$' "$R/tiny.txt" \
  && ok "and the tree is untouched, which is the assertion the message is not" \
  || bad "the write APPLIED under a ceiling that cannot report it: $(cat "$R/tiny.txt")"


# 71. ADR-034: an entry whose checkout is gone is removable, and only when asked.
#
# THE ROW BUILDS ITS OWN BASE. Every other row shares whatever XDG_STATE_HOME is
# in force, and this one counts directories, so it needs a base holding only
# what it planted. That sub-base sits under $WORK and goes with the EXIT trap.
#
# AND EVERY CASE IS PAIRED. A row that scores only "the dead one is gone" is
# green against a binary that removes the whole base — which is exactly the
# defect a prune has, and the one this section exists to refuse.
P71=$(mktemp -d "$WORK/prune-XXXXXX")
export XDG_STATE_HOME="$P71/state"
P71LIVE=$(mktemp -d "$WORK/p71live-XXXXXX")
P71DEAD=$(mktemp -d "$WORK/p71dead-XXXXXX")
P71SELF=$(mktemp -d "$WORK/p71self-XXXXXX")
for d in "$P71LIVE" "$P71DEAD" "$P71SELF"; do
  printf 'alpha\n' > "$d/f.txt"
  # Through the BINARY, so the `root` markers under test are the ones mrw writes.
  "$MRW" -C "$d" read f.txt >"$WORK/served.out" 2>&1
done
rm -rf "$P71DEAD"
P71UNKNOWN="$XDG_STATE_HOME/mrw/dddddddddddddddd"
mkdir -p "$P71UNKNOWN"                       # no `root` marker: mrw did not write it
p71dir() { "$MRW" -C "$1" seen 2>/dev/null | head -1; }
P71SELFDIR=$(p71dir "$P71SELF")
P71LIVEDIR=$(p71dir "$P71LIVE")
# The dead entry's directory is whichever one neither of the live roots claims.
# ⚠ Iterate the full paths. The first cut ran `ls -d .../mrw/*/ | tr -d /`,
# which strips EVERY slash rather than the trailing one and turns each path into
# one mangled token — the row then "found" a directory that never existed and
# all four of its comparisons missed.
P71DEADDIR=""
for d in "$XDG_STATE_HOME"/mrw/*/; do
  d=${d%/}
  case "$d" in "$P71SELFDIR" | "$P71LIVEDIR" | "$P71UNKNOWN") continue ;; esac
  P71DEADDIR="$d"
  break
done

[ -n "$P71DEADDIR" ] && [ -d "$P71DEADDIR" ] \
  && ok "the fixture planted a dead entry" \
  || bad "the fixture planted no dead entry: $(ls "$XDG_STATE_HOME/mrw" | tr '\n' ' ')"

# THE FIRST LINE IS STILL THE DIRECTORY. Read the status from the binary, never
# through the pipe: `$MRW seen | head -1` returns head's exit code.
"$MRW" -C "$P71SELF" seen > "$P71/seen.out" 2>&1
want 0 $? "mrw seen still exits 0 with the count line added"
[ "$(head -1 "$P71/seen.out")" = "$P71SELFDIR" ] \
  && ok "the state directory is still the FIRST line of mrw seen" \
  || bad "line 1 of mrw seen is not the state directory: $(head -1 "$P71/seen.out")"
grep -q '^# 4 state directories under ' "$P71/seen.out" \
  && ok "mrw seen counts the entries in the base" \
  || bad "mrw seen printed no count: $(tr '\n' '|' < "$P71/seen.out")"
grep -q -- '--prune' "$P71/seen.out" \
  && ok "and points at the flag that removes the dead ones" \
  || bad "mrw seen names the growth and not the remedy"

# A DRY RUN REMOVES NOTHING.
out=$("$MRW" -C "$P71SELF" seen --prune --dry-run 2>&1); rc=$?
want 0 "$rc" "mrw seen --prune --dry-run exits 0"
grep -q "$P71DEADDIR" <<<"$out" && ok "a dry run names the entry it would remove" || bad "a dry run named nothing: $out"
[ -d "$P71DEADDIR" ] && ok "and removes nothing" || bad "a dry run removed $P71DEADDIR"

# --dry-run WITHOUT --prune IS A USAGE ERROR, not a silent no-op.
"$MRW" -C "$P71SELF" seen --dry-run >/dev/null 2>&1
want 2 $? "mrw seen --dry-run without --prune is a usage error"

# THE REAL RUN. One goes; three stay.
out=$("$MRW" -C "$P71SELF" seen --prune 2>&1); rc=$?
want 0 "$rc" "mrw seen --prune exits 0"
[ -d "$P71DEADDIR" ] && bad "the dead entry survived --prune" || ok "the entry whose checkout is gone is removed"
[ -d "$P71LIVEDIR" ] && ok "the entry whose checkout still exists is KEPT" || bad "--prune removed a live entry"
[ -d "$P71SELFDIR" ] && ok "the running root's own entry is KEPT" || bad "--prune removed the entry it is running from"
[ -d "$P71UNKNOWN" ] && ok "an entry with no root marker is KEPT" || bad "--prune removed an entry of unknown provenance"
grep -q "$P71DEADDIR" <<<"$out" && ok "and the delete says what it removed (ADR-008)" || bad "--prune removed silently: $out"
grep -q '1 kept' <<<"$out" && ok "and says the unidentifiable entry was kept" || bad "--prune did not report the kept entry: $out"

# A SECOND RUN IS A NO-OP THAT STILL SAYS SO.
out=$("$MRW" -C "$P71SELF" seen --prune 2>&1); rc=$?
want 0 "$rc" "a second --prune exits 0"
grep -q '^0 of 3 state directories removed' <<<"$out" \
  && ok "a prune that removed nothing says so rather than printing nothing" \
  || bad "a no-op prune reported: $out"

# 72. ADR-034: the prune refuses a state base mrw did not create.
#
# PAIRED, because a row that scores only the refusal is green against a binary
# that refuses everything. The same shape is driven twice: once against a real
# base holding a dead entry, which must GO, and once against a base whose
# `mrw` directory is a symlink to somebody else's directory holding an
# identical-looking dead entry, which must STAY.
#
# ⚠ THE BASE, NOT AN ENTRY. The symlinked ENTRY under a real base is covered by
# a Go test, not by §71 — an earlier version of this comment claimed §71 plants
# one, and it does not. This row is the level ABOVE that, and it is the one
# os.RemoveAll followed: the walk and the removal each re-resolved the base by
# path, so replacing it put a directory nobody gave mrw inside the prune's
# reach.
P72=$(mktemp -d "$WORK/prune72-XXXXXX")
P72ROOT=$(mktemp -d "$WORK/p72root-XXXXXX")
printf 'alpha\n' > "$P72ROOT/f.txt"

# --- THE GOOD CASE: a base mrw made, holding one entry whose checkout is gone.
export XDG_STATE_HOME="$P72/good"
P72DEAD=$(mktemp -d "$WORK/p72dead-XXXXXX")
printf 'alpha\n' > "$P72DEAD/f.txt"
"$MRW" -C "$P72DEAD" read f.txt >"$WORK/served.out" 2>&1
"$MRW" -C "$P72ROOT" read f.txt >"$WORK/served.out" 2>&1
P72DEADDIR=$("$MRW" -C "$P72DEAD" seen 2>/dev/null | head -1)
rm -rf "$P72DEAD"
[ -n "$P72DEADDIR" ] && [ -d "$P72DEADDIR" ] \
  && ok "the fixture planted a dead entry under a real base" \
  || bad "the fixture planted no dead entry: $P72DEADDIR"
"$MRW" -C "$P72ROOT" seen --prune > "$P72/good.out" 2>&1
want 0 $? "mrw seen --prune exits 0 on a base mrw created"
[ -d "$P72DEADDIR" ] \
  && bad "the dead entry survived a prune of a legitimate base" \
  || ok "and the entry whose checkout is gone is removed"

# --- THE BAD CASE: <base>/mrw is a symlink out to a directory that is not ours.
export XDG_STATE_HOME="$P72/bad"
mkdir -p "$XDG_STATE_HOME"
P72VICTIM=$(mktemp -d "$WORK/p72victim-XXXXXX")
mkdir -p "$P72VICTIM/eeeeeeeeeeeeeeee"
printf '/no/such/checkout\n' > "$P72VICTIM/eeeeeeeeeeeeeeee/root"
printf 'not mrw own\n' > "$P72VICTIM/precious"
if ln -s "$P72VICTIM" "$XDG_STATE_HOME/mrw" 2>/dev/null; then
  "$MRW" -C "$P72ROOT" seen --prune > "$P72/bad.out" 2>&1
  rc=$?
  want 2 "$rc" "a prune through a symlinked base is refused, not attempted"
  grep -qi 'symlink' "$P72/bad.out" \
    && ok "and the refusal says the base is a symlink" \
    || bad "the refusal does not name what is wrong: $(cat "$P72/bad.out")"
  [ -d "$P72VICTIM/eeeeeeeeeeeeeeee" ] \
    && ok "the entry on the far side of the link is untouched" \
    || bad "--prune removed $P72VICTIM/eeeeeeeeeeeeeeee through a symlinked base"
  [ -f "$P72VICTIM/precious" ] \
    && ok "and so is everything beside it" \
    || bad "--prune removed $P72VICTIM/precious through a symlinked base"

  # AND A SYMLINKED ENTRY UNDER A REAL BASE IS COUNTED AND REPORTED, not
  # silently dropped. ADR-034 promises "reported and skipped"; skipping it
  # without reporting makes it indistinguishable from an entry nothing looked
  # at, which is the same argument the record makes about unidentifiable ones.
  export XDG_STATE_HOME="$P72/good"
  P72LINK=$(mktemp -d "$WORK/p72link-XXXXXX")
  printf 'mine\n' > "$P72LINK/precious"
  ln -s "$P72LINK" "$XDG_STATE_HOME/mrw/7777777777777777"
  out=$("$MRW" -C "$P72ROOT" seen 2>&1); rc=$?
  want 0 "$rc" "mrw seen exits 0 with a symlinked entry under the base"
  grep -qE '^# 2 state directories under ' <<<"$out" \
    && ok "a symlinked entry is COUNTED rather than dropped from the walk" \
    || bad "the count does not include the symlinked entry: $(grep '^#' <<<"$out" | tr '\n' '|')"
  out=$("$MRW" -C "$P72ROOT" seen --prune 2>&1); rc=$?
  want 0 "$rc" "mrw seen --prune exits 0 with a symlinked entry under the base"
  grep -q '1 kept' <<<"$out" \
    && ok "and it is REPORTED as kept, not silently skipped" \
    || bad "--prune did not report the symlinked entry as kept: $out"
  [ -L "$XDG_STATE_HOME/mrw/7777777777777777" ] \
    && ok "the symlinked entry itself is left alone" \
    || bad "--prune removed the symlinked entry"
  [ -f "$P72LINK/precious" ] \
    && ok "and nothing on the far side of it is touched" \
    || bad "--prune followed the symlinked entry and removed $P72LINK/precious"
  export XDG_STATE_HOME="$P72/bad"
else
  skip "this filesystem does not do symlinks, so the symlinked-base case cannot be planted"
fi
# RESTORE the file-wide pin, do not unset it. This row swapped XDG_STATE_HOME for
# a sub-base of its own because it COUNTS entries; unsetting it here would send
# any row added after this one back to the real state base, which is the leak
# ADR-034 T4 closed. It is last today, and that is not a reason to leave a trap.
export XDG_STATE_HOME="$WORK/state"

# 73. ADR-035: a multi-line replace declares what it replaces.
#
# The unit tests prove the guard; they cannot prove the SHIPPED binary runs it.
# Both spellings are driven here, because a row that only shows the refusal is
# satisfied by a binary that refuses every replace.
#
# The message is asserted, not just the exit code: `mrw write` exits 1 for a
# dozen reasons — an unread line, a bad path, a failed guard — so `want 1` alone
# scores the binary's error handling rather than this guard.
fixture
out=$(printf '@@ a.go 3-4 replace\nfunc A() int { return 10 }\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a multi-line replace with no anchor= is refused"
grep -q 'carries no anchor=' <<<"$out" \
  && ok "and the refusal names the guard that fired" \
  || bad "the refusal does not name the missing anchor: $out"
# THE REMEDY IS ITS OWN ASSERTION. This row used to grep the DIAGNOSIS only,
# while the mutation log credited it with covering the remedy — so deleting
# `say anchor="…"` left every gate green. Reported on PR #146 and reproduced
# before fixing: a refusal that diagnoses without prescribing is the failure a
# refusal exists to avoid.
grep -q 'say anchor="<text from the first line>"' <<<"$out" \
  && ok "and it prescribes the remedy, not just the diagnosis" \
  || bad "the refusal does not tell the caller what to write: $out"
grep -q 'func B' "$R/a.go" \
  && ok "and ADR-001 holds: the refused plan wrote nothing" \
  || bad "the refused plan changed the file: $(cat "$R/a.go")"
# The same plan with the anchor the message asked for. Without this pair the row
# above passes on a binary that refuses everything.
out=$(printf '@@ a.go 3-4 replace anchor="func A()"\nfunc A() int { return 10 }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "the same plan applies once it carries an anchor"
grep -q 'return 10' "$R/a.go" \
  && ok "and the anchored replace wrote what it said" \
  || bad "the anchored replace did not apply: $(cat "$R/a.go")"
# A SINGLE-line replace still needs none. This is what makes the row a narrowing
# rather than a ban, and it is the assertion a guard keyed on the op instead of
# the span would fail.
fixture
out=$(printf '@@ a.go 3 replace\nfunc A() int { return 99 }\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a single-line replace needs no anchor"
grep -q 'return 99' "$R/a.go" \
  && ok "and it wrote what it said" \
  || bad "the single-line replace did not apply: $(cat "$R/a.go")"
# A multi-line DELETE is untouched: ADR-035 is scoped to replace, because every
# measured incident is one and ADR-008 already gives delete an expected body.
fixture
out=$(printf '@@ a.go 3-4 delete\n' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a multi-line delete still needs no anchor"
grep -q 'func C' "$R/a.go" \
  && ok "and it removed only the lines it named" \
  || bad "the delete removed the wrong span: $(cat "$R/a.go")"
# THE DOCUMENTED PLANS MUST BE PLANS mrw WOULD ACCEPT. A grammar change that
# leaves the examples behind teaches the old shape to every reader, and nothing
# else here reads README.md or AGENTS.md. Anchored to column 0, so the two
# PROSE mentions of `@@ f.go 5-9999 replace` — which discuss the out-of-range
# refusal, and which mrw still refuses for that reason first — are not matched.
# CONSERVATIVE BY CONSTRUCTION: the safe forms are enumerated and everything
# else is flagged, rather than the reverse — and this row no longer tries. THE
# DOCUMENTATION CHECK MOVED TO A GO TEST, because it needs the PARSER and a
# contract row cannot have one. Four shell cuts were each defeated by a header
# `splitHeader` accepts and a regex does not see: a quoted path read as the op,
# a greedy match backtracking into one, tab separators, and finally a QUOTED OP
# (`@@ f.go 2-3 "replace"`) plus a BOM-prefixed header. Every one was found by a
# review round rather than by reading, which is the signal that the approach was
# wrong rather than incomplete.
#
# internal/adversarial.TestEveryDocumentedReplaceCarriesItsAnchor now calls
# plan.Parse on every `@@` line in README.md and AGENTS.md, so the TOKENIZATION
# is the parser's rather than a regex's — the part every shell cut got wrong. It
# is not equivalent in every respect and its own comment says where: the body
# length is searched to a finite bound. Its
# companion TestTheDocumentedPlanCheckRejectsWhatItMustReject drives a table of
# must-flag headers — the four that defeated the shell among them — and a table
# that must pass. The tables are the count; naming a number here would go stale
# in the commit that grows them, and this comment's did.
#
# What stays HERE is what only a contract row can do: drive the built binary.

# 74. ADR-036: a paired pattern means one thing on both paths.
#
# The unit tests prove the resolver; they cannot prove the shipped binary uses
# it. Both changed behaviours are driven here, each paired with the case that
# must still work, because a row that only shows the refusal is satisfied by a
# binary that refuses every paired pattern.
fixture
printf 'alpha\nSTART\nbody one\nbody two\nomega\n' > "$R/p.txt"
out=$(m read 'p.txt:/^START$/,/^NEVER$/' 2>&1); rc=$?
want 1 "$rc" "a paired pattern whose end never matches is refused, not served to EOF"
grep -q 'end pattern' <<<"$out" \
  && ok "and the report says which HALF of the pattern missed" \
  || bad "the report does not name the end as the half that failed: $out"
# ⚠ THE ABSENCE IS THE ASSERTION. A build that reports the range AND serves it
# passes a message-only check, and serving is the defect this row exists for.
grep -q 'body one' <<<"$out" \
  && bad "content past the start was served anyway: $out" \
  || ok "and nothing past the start was served"
# The end is the first match AT OR AFTER the start, so an end on the start line
# closes the span there — one line, exactly as the write path resolves it.
printf 'alpha\nMARK here\nmiddle\nMARK again\nomega\n' > "$R/q.txt"
out=$(m read 'q.txt:/^MARK here$/,/MARK/' 2>&1); rc=$?
want 0 "$rc" "an end matching the start line is accepted"
grep -q '@@ 2-2' <<<"$out" \
  && ok "and it closes the span on that line" \
  || bad "an end on the start line did not close the span there: $out"
# The pair: an ordinary later end still spans to it, so this is a narrowing
# rather than a ban.
out=$(m read 'q.txt:/^alpha$/,/^middle$/' 2>&1); rc=$?
want 0 "$rc" "an ordinary paired pattern still resolves"
grep -q '@@ 1-3' <<<"$out" \
  && ok "and still spans to its end" \
  || bad "an ordinary paired pattern no longer spans to its end: $out"
# The difference ADR-036 KEEPS: a read serves a span for every match of the start
# THAT IS NOT ALREADY INSIDE A SPAN IT SERVED — the resolver advances past each
# span it emits — where a write refuses unless the start matches once. The spans
# need a GAP between them, because contiguous spans are coalesced into one.
printf 'START\nx\nEND\ngap\nSTART\ny\nEND\n' > "$R/r.txt"
out=$(m read 'r.txt:/^START$/,/^END$/' 2>&1); rc=$?
want 0 "$rc" "a start matching twice is still served by a read"
grep -q '@@ 1-3' <<<"$out" && grep -q '@@ 5-7' <<<"$out" \
  && ok "and both spans come back, which the write path deliberately refuses" \
  || bad "a start matching twice no longer serves both spans: $out"
# A RESOLVED SPAN MUST NOT SUPPRESS A LATER MISSING END. `found` is range-wide,
# so the first cut of this record reported nothing for `START … END … START … EOF`
# — one span, no problem, exit 0, a start the address named silently dropped.
# That is the same silent-drop ADR-036 removes, moved one case along. Reported by
# review of PR #146.
printf 'START\nx\nEND\ngap\nSTART\ny\nz\n' > "$R/s.txt"
out=$(m read 's.txt:/^START$/,/^END$/' 2>&1); rc=$?
want 1 "$rc" "a resolved span does not suppress a later missing end"
grep -q 'end pattern' <<<"$out" \
  && ok "and the report names the END as the half that missed" \
  || bad "the mixed case reports nothing about the unresolved start: $out"
# BOTH, not either: the answer mrw could resolve is still served.
grep -q '@@ 1-3' <<<"$out" \
  && ok "and the span that DID resolve is still served" \
  || bad "reporting the problem threw away the resolved span: $out"
# 75. ADR-037: the binary teaches the format it demands.
#
# A unit test on guide.CLI cannot prove the shipped binary prints it. Drive
# $MRW: the good case is exit 0 and every Shared sentence plus the pipe trap,
# and the pair is an extra argument, which is usage — not a file to append to.
out=$(m instructions 2>&1); rc=$?
want 0 "$rc" "mrw instructions exits 0"
python3 - "$out" <<'PY'
import sys
out = sys.argv[1]
shared = (
    "Use mrw always: plan the activity as one read of every site, then one plan, then one write.",
    "A plan applies whole or not at all: if any hunk fails validation, nothing is written; a failed commit reports what reached disk (CLI: PARTIALLY APPLIED).",
    "Read before you write, per line, not per file.",
    "mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.",
    "A refusal names the file, the plan line, and the reason.",
)
missing = [s for s in shared if s not in out]
assert not missing, "instructions omitted Shared sentences: %r" % missing
assert "through a pipe" in out, "instructions omitted the pipe trap"
PY
[ $? -eq 0 ] && ok "and it prints Shared verbatim plus the pipe trap" \
             || bad "instructions omitted Shared or the pipe trap"
out=$(m instructions nope 2>&1); rc=$?
want 2 "$rc" "an extra argument is a usage error"

# 76. ADR-038: a ledger write is one writer, even across processes.
#
# §24 keeps the safety half (writability follows the ledger). This row is the
# keep-40 half, driven through $MRW, so a package-only lock that cmd/mrw does
# not call cannot pass. Pair: 40 concurrent reads keep 40; then every one of
# those files is writable, and no extra file changed.
fixture
N=40
for i in $(seq 1 $N); do printf 'a\nb\nc\n' > "$R/p$i.txt"; done
for i in $(seq 1 $N); do m read "p$i.txt" >"$WORK/served.out" 2>&1 & done
wait
kept=$(m seen 2>/dev/null | grep -cE '(^| )p[0-9]+\.txt$')
want "$N" "$kept" "40 concurrent reads keep 40 ledger entries"
applied=0
for i in $(seq 1 $N); do
  printf '@@ p%s.txt 2 replace\nB\n' "$i" | m write - >/dev/null 2>&1 && applied=$((applied + 1))
done
want "$N" "$applied" "and every kept entry is writable"
changed=$(grep -lx 'B' "$R"/p*.txt 2>/dev/null | wc -l | tr -d ' ')
want "$applied" "$changed" "and no file changed that was not applied"

# 77. ADR-039: a fitting read licenses only what came back.
#
# A unit test on hold cannot prove the built server calls it. Drive $MRW mcp:
# a three-line file must carry checkpoints, an unacked write is refused and
# leaves the tree unchanged, and the same write with ack applies. Pair both
# halves so a server that licenses nothing cannot pass.
fixture
printf 'one\ntwo\nthree\n' > "$R/small.txt"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["small.txt"]}}}\n' | m mcp 2>/dev/null)
want 0 $? "the server answers a fitting read"
printf '%s' "$out" > "$R/fit.json"
python3 - "$R/fit.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
t=r["content"][0]["text"]
assert "-- PARTIAL:" not in t, "the fixture paged; §77 must drive a fitting serve"
cks=re.findall(r"^-- ck ([0-9a-f]{16}) open ", t, re.M)
assert cks, "a fitting serve carried no checkpoints"
open(sys.argv[1]+".cks","w").write(" ".join(cks))
PY
want 0 $? "a fitting serve carries checkpoints and did not page"
cks=$(cat "$R/fit.json.cks")
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ small.txt 2 replace\\nTWO\\n"}}}\n' | m mcp 2>/dev/null)
python3 - "$out" "$R/small.txt" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 1 and sc.get("applied") is False, "an unacked fitting write applied: %s" % sc
reason=(sc["hunks"][0].get("reason") or "")
assert "ack" in reason.lower() or "Send an id in ack only if you hold BOTH" in reason, "the refusal does not name ack / AckRule: %s" % reason
assert open(sys.argv[2]).read() == "one\ntwo\nthree\n", "an unacked write changed the tree"
PY
want 0 $? "an unacked fitting write is refused and the tree is unchanged"
first=$(printf '%s' "$cks" | awk '{print $1}')
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ small.txt 2 replace\\nTWO\\n","ack":["%s"]}}}\n' "$first" | m mcp 2>/dev/null)
python3 - "$out" "$R/small.txt" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 0, "an acked fitting write was refused: %s" % sc
assert open(sys.argv[2]).read() == "one\nTWO\nthree\n", "an acked write did not apply"
PY
want 0 $? "an acked fitting write applies"

# 78. ADR-019: a write is licensed only by the ledger of the --root that served the read.
#
# m() is "$MRW" -C "$R" and cannot name two roots. Drive --root on mcp.
# Shared XDG_STATE_HOME (file-wide). Same relative path, same body. A's ack
# must not license B; the same ack must license A. Pair both halves.
A78=$(mktemp -d "$WORK/a78-XXXXXX")
B78=$(mktemp -d "$WORK/b78-XXXXXX")
printf 'one\ntwo\nthree\n' > "$A78/f.txt"
printf 'one\ntwo\nthree\n' > "$B78/f.txt"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["f.txt"]}}}\n' | "$MRW" --root "$A78" mcp 2>/dev/null)
want 0 $? "root A answers a fitting read"
printf '%s' "$out" > "$A78/read.json"
python3 - "$A78/read.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
t=r["content"][0]["text"]
assert "-- PARTIAL:" not in t, "the fixture paged; §78 must drive a fitting serve"
cks=re.findall(r"^-- ck ([0-9a-f]{16}) open ", t, re.M)
assert cks, "A's fitting serve carried no checkpoints"
open(sys.argv[1]+".cks","w").write(" ".join(cks))
PY
want 0 $? "A's fitting serve carries checkpoints"
cks=$(cat "$A78/read.json.cks")
first=$(printf '%s' "$cks" | awk '{print $1}')
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ f.txt 2 replace\\nTWO\\n","ack":["%s"]}}}\n' "$first" | "$MRW" --root "$B78" mcp 2>/dev/null)
python3 - "$out" "$B78/f.txt" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 1 and sc.get("applied") is False, "A's ack licensed a write under B: %s" % sc
assert open(sys.argv[2]).read() == "one\ntwo\nthree\n", "B's tree changed"
PY
want 0 $? "A's ack does not license a write under B"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ f.txt 2 replace\\nTWO\\n","ack":["%s"]}}}\n' "$first" | "$MRW" --root "$A78" mcp 2>/dev/null)
python3 - "$out" "$A78/f.txt" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc["failed"] == 0, "A's ack did not license A: %s" % sc
assert open(sys.argv[2]).read() == "one\nTWO\nthree\n", "acked write on A did not apply"
PY
want 0 $? "A's ack licenses a write under A"

# 79. ADR-040: write --help names how to quote a header option.
#
# A unit test on Description cannot prove the binary prints it. Drive $MRW.
# Pair: write --help names quoting / body= / -C / --root; an unknown write
# flag is usage (exit 2), so a binary that always exits 0 cannot pass.
help79=$(m write --help)
want 0 $? "write --help exits 0"
printf '%s' "$help79" | grep -q 'anchor="' && ok "write --help names double-quoted anchor=" || bad "write --help names double-quoted anchor="
printf '%s' "$help79" | grep -q 'body=' && ok "write --help names body=" || bad "write --help names body="
printf '%s' "$help79" | grep -q -- '-C' && ok "write --help names -C" || bad "write --help names -C"
printf '%s' "$help79" | grep -q -- '--root' && ok "write --help names --root" || bad "write --help names --root"
m write --not-a-flag >/dev/null 2>&1
want 2 $? "an unknown write flag is usage, not silent help"

# 80. ADR-040: unquoted and single-quoted anchor= parse; a leftover token is still usage.
fixture
printf 'func openTestStore\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
printf '@@ f.go 1 replace anchor=func openTestStore\nNEW\n' | m write --dry-run - >/dev/null
want 0 $? "unquoted spaced anchor= parses"
printf "@@ f.go 1 replace anchor='func openTestStore'\nNEW\n" | m write --dry-run - >/dev/null
want 0 $? "single-quoted anchor= parses"
printf '@@ f.go 1 replace leftover\nNEW\n' | m write --dry-run - >/dev/null 2>&1
want 2 $? "a trailing token that is not key=value is still usage"

# 81. ADR-040: mrw version prints the same string -v already prints.
v81=$("$MRW" version)
want 0 $? "mrw version exits 0"
[ -n "$v81" ] && ok "mrw version prints a string" || bad "mrw version prints a string"
"$MRW" -v 2>/dev/null | grep -F -q "$(printf '%s' "$v81" | tr -d '\n')" && ok "-v names the same string" || bad "-v names the same string"
"$MRW" version extra >/dev/null 2>&1
want 2 $? "version extra is usage"

# 82. ADR-051: apply_patch compiles to the native plan; an unread sibling
# writes nothing. Pair: unread (exit 1, FAIL+skip, ledger reason, unchanged)
# / served (exit 0, both replacements) / no flag (exit 2) / git-shaped under
# the flag (exit 2). A mutant that ignores --format fails the unread half
# via exit 2 rather than 1.
patch82=$(printf '%s\n' \
	'*** Begin Patch' \
	'*** Update File: a.go' \
	'@@' \
	'-func A() int { return 1 }' \
	'+func A() int { return 10 }' \
	'@@' \
	'-func C() int { return 3 }' \
	'+func C() int { return 30 }' \
	'*** End Patch')
R=$(mktemp -d "$WORK/r-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
m read 'a.go:3' >"$WORK/served.out"
out=$(printf '%s\n' "$patch82" | m write --format=apply_patch - 2>&1); rc=$?
want 1 "$rc" "two-hunk apply_patch, one unread line -> exit 1"
grep -q 'FAIL' <<<"$out" && ok "unread apply_patch names FAIL" || bad "unread apply_patch names FAIL"
grep -q '^skip' <<<"$out" && ok "unread apply_patch siblings skip" || bad "unread apply_patch siblings skip"
grep -q 'has not been read' <<<"$out" && ok "unread apply_patch is the ledger refusal" || bad "unread apply_patch is the ledger refusal"
grep -q 'return 1 }' "$R/a.go" && ok "unread apply_patch wrote nothing" || bad "unread apply_patch wrote"

fixture
out=$(printf '%s\n' "$patch82" | m write --format=apply_patch --no-check - 2>&1); rc=$?
want 0 "$rc" "served two-hunk apply_patch -> exit 0"
grep -q 'return 10' "$R/a.go" && ok "served apply_patch rewrote A" || bad "served apply_patch rewrote A"
grep -q 'return 30' "$R/a.go" && ok "served apply_patch rewrote C" || bad "served apply_patch rewrote C"

R=$(mktemp -d "$WORK/r-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
out=$(printf '%s\n' "$patch82" | m write - 2>&1); rc=$?
want 2 "$rc" "apply_patch without --format is a bad native plan"
grep -q 'return 1 }' "$R/a.go" && ok "no-flag apply_patch wrote nothing" || bad "no-flag apply_patch wrote"

git82=$(printf '%s\n' \
	'diff --git a/a.go b/a.go' \
	'--- a/a.go' \
	'+++ b/a.go' \
	'@@ -3,1 +3,1 @@' \
	'-func A() int { return 1 }' \
	'+func A() int { return 10 }')
out=$(printf '%s\n' "$git82" | m write --format=apply_patch - 2>&1); rc=$?
want 2 "$rc" "a git patch under --format=apply_patch is refused"
printf '' | m write --format=git - >/dev/null 2>&1
want 2 $? "--format=git is usage"

# 83. ADR-051 F-27: mrw_write.format=apply_patch is the MCP served path.
# Pair: unread (failed+skipped, ledger, unchanged) / served (both
# replacements) / no format (parse refuse) / format=git (usage) / cargo
# stays two tools and declares format. A mutant that ignores format fails
# the unread half as a parse error rather than failed+skipped.
patch83=$(printf '%s\n' \
	'*** Begin Patch' \
	'*** Update File: a.go' \
	'@@' \
	'-func A() int { return 1 }' \
	'+func A() int { return 10 }' \
	'@@' \
	'-func C() int { return 3 }' \
	'+func C() int { return 30 }' \
	'*** End Patch')
printf '%s\n' "$patch83" > "$WORK/p83.apply"
R=$(mktemp -d "$WORK/r83-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go:3"]}}}\n' | m mcp 2>/dev/null)
want 0 $? "MCP serves a.go:3"
printf '%s' "$out" > "$R/read83.json"
python3 - "$R/read83.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
t=r["content"][0]["text"]
cks=re.findall(r"^-- ck ([0-9a-f]{16}) open ", t, re.M)
assert cks, "a.go:3 carry no checkpoints"
open(sys.argv[1]+".cks","w").write("\n".join(cks))
PY
want 0 $? "a.go:3 carry checkpoints"
req=$(python3 - "$WORK/p83.apply" "$R/read83.json.cks" <<'PY'
import json,sys
plan=open(sys.argv[1]).read()
acks=open(sys.argv[2]).read().split()
print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":plan,"format":"apply_patch","ack":acks}}}))
PY
)
out=$(printf '%s\n' "$req" | m mcp 2>/dev/null)
python3 - "$out" "$R/a.go" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
assert r.get("isError") is True, "unread apply_patch was not a tool error: %s" % r
sc=r.get("structuredContent") or {}
assert sc.get("applied") is False, "unread apply_patch applied: %s" % sc
assert sc.get("failed") == 1, "unread apply_patch failed=%s want 1: %s" % (sc.get("failed"), sc)
st=[h.get("status") for h in sc.get("hunks") or []]
assert "failed" in st and "skipped" in st, "want failed+skipped, got %s" % st
reason=" ".join((h.get("reason") or "") for h in sc.get("hunks") or [])
assert "has not been read" in reason, "not the ledger refusal: %s" % reason
assert "return 1 }" in open(sys.argv[2]).read(), "unread apply_patch wrote"
PY
want 0 $? "MCP format apply_patch, one unread line writes nothing"

fixture
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go"]}}}\n' | m mcp 2>/dev/null)
printf '%s' "$out" > "$R/read83s.json"
python3 - "$R/read83s.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
cks=re.findall(r"^-- ck ([0-9a-f]{16}) open ", r["content"][0]["text"], re.M)
assert cks, "served a.go carry no checkpoints"
open(sys.argv[1]+".cks","w").write("\n".join(cks))
PY
want 0 $? "served a.go carry checkpoints"
req=$(python3 - "$WORK/p83.apply" "$R/read83s.json.cks" <<'PY'
import json,sys
plan=open(sys.argv[1]).read()
acks=open(sys.argv[2]).read().split()
print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":plan,"format":"apply_patch","ack":acks}}}))
PY
)
out=$(printf '%s\n' "$req" | m mcp 2>/dev/null)
python3 - "$out" "$R/a.go" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
assert sc.get("applied") is True and sc.get("failed") == 0, "served apply_patch refused: %s" % sc
body=open(sys.argv[2]).read()
assert "return 10" in body and "return 30" in body, "served apply_patch missed a replacement: %r" % body
PY
want 0 $? "MCP format apply_patch served writes both replacements"

R=$(mktemp -d "$WORK/r83n-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
req=$(python3 - "$WORK/p83.apply" <<'PY'
import json,sys
print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":open(sys.argv[1]).read()}}}))
PY
)
out=$(printf '%s\n' "$req" | m mcp 2>/dev/null)
python3 - "$out" "$R/a.go" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
assert r.get("isError") is True, "no-format apply_patch was not a parse refusal: %s" % r
assert "structuredContent" not in r, "no-format apply_patch compiled: %s" % r
assert "return 1 }" in open(sys.argv[2]).read(), "no-format apply_patch wrote"
PY
want 0 $? "MCP apply_patch without format is a bad native plan"

req=$(python3 - "$WORK/p83.apply" <<'PY'
import json,sys
print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":open(sys.argv[1]).read(),"format":"git"}}}))
PY
)
out=$(printf '%s\n' "$req" | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
r=json.loads(sys.argv[1])
r=r["result"]
assert r.get("isError") is True, "format=git was not an isError result: %s" % r
assert "git patch is not" in r["content"][0]["text"], "git refuse does not name the grammar: %s" % r
PY
want 0 $? "MCP format=git is refused, naming the grammar"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
tools={t["name"]:t for t in json.loads(sys.argv[1])["result"]["tools"]}
assert set(tools)=={"mrw_read","mrw_write"}, "cargo grew: %s" % sorted(tools)
fmt=(tools["mrw_write"].get("inputSchema") or {}).get("properties") or {}
assert "format" in fmt, "mrw_write does not declare format"
assert "plan" in (fmt["format"].get("enum") or []) or "plan" in str(fmt["format"])
assert "apply_patch" in str(fmt["format"])
PY
want 0 $? "tools/list still two tools and mrw_write declares format"

# 84. ADR-051 F-26: --format=search_replace compiles Aider SEARCH/REPLACE.
# Pair: unread (exit 1, FAIL+skip, ledger, unchanged) / served (exit 0, both
# replacements) / no flag (exit 2) / MCP unread writes nothing / cargo still
# two tools and declares search_replace. A mutant that ignores the flag fails
# the unread half as exit 2 rather than 1. Exact match only — no fuzzy.
sr84=$(printf '%s\n' \
	'a.go' \
	'<<<<<<< SEARCH' \
	'func A() int { return 1 }' \
	'=======' \
	'func A() int { return 10 }' \
	'>>>>>>> REPLACE' \
	'<<<<<<< SEARCH' \
	'func C() int { return 3 }' \
	'=======' \
	'func C() int { return 30 }' \
	'>>>>>>> REPLACE')
R=$(mktemp -d "$WORK/r84-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
m read 'a.go:3' >"$WORK/served.out"
out=$(printf '%s\n' "$sr84" | m write --format=search_replace - 2>&1); rc=$?
want 1 "$rc" "two-hunk SEARCH/REPLACE, one unread line -> exit 1"
grep -q 'FAIL' <<<"$out" && ok "unread SEARCH/REPLACE names FAIL" || bad "unread SEARCH/REPLACE names FAIL"
grep -q '^skip' <<<"$out" && ok "unread SEARCH/REPLACE siblings skip" || bad "unread SEARCH/REPLACE siblings skip"
grep -q 'has not been read' <<<"$out" && ok "unread SEARCH/REPLACE is the ledger refusal" || bad "unread SEARCH/REPLACE is the ledger refusal"
grep -q 'return 1 }' "$R/a.go" && ok "unread SEARCH/REPLACE wrote nothing" || bad "unread SEARCH/REPLACE wrote"

fixture
out=$(printf '%s\n' "$sr84" | m write --format=search_replace --no-check - 2>&1); rc=$?
want 0 "$rc" "served two-hunk SEARCH/REPLACE -> exit 0"
grep -q 'return 10' "$R/a.go" && ok "served SEARCH/REPLACE rewrote A" || bad "served SEARCH/REPLACE rewrote A"
grep -q 'return 30' "$R/a.go" && ok "served SEARCH/REPLACE rewrote C" || bad "served SEARCH/REPLACE rewrote C"

R=$(mktemp -d "$WORK/r84n-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
out=$(printf '%s\n' "$sr84" | m write - 2>&1); rc=$?
want 2 "$rc" "SEARCH/REPLACE without --format is a bad native plan"
grep -q 'return 1 }' "$R/a.go" && ok "no-flag SEARCH/REPLACE wrote nothing" || bad "no-flag SEARCH/REPLACE wrote"

near84=$(printf '%s\n' \
	'a.go' \
	'<<<<<<< SEARCH' \
	'func A() int { return  1 }' \
	'=======' \
	'func A() int { return 10 }' \
	'>>>>>>> REPLACE')
R=$(mktemp -d "$WORK/r84f-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
out=$(printf '%s\n' "$near84" | m write --format=search_replace - 2>&1); rc=$?
want 2 "$rc" "a near-miss SEARCH is a compile refusal, not fuzzy apply"
grep -q 'return 1 }' "$R/a.go" && ok "near-miss SEARCH wrote nothing" || bad "near-miss SEARCH wrote"

printf '%s\n' "$sr84" > "$WORK/p84.sr"
R=$(mktemp -d "$WORK/r84m-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\nfunc B() int { return 2 }\nfunc C() int { return 3 }\n' > "$R/a.go"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go:3"]}}}\n' | m mcp 2>/dev/null)
want 0 $? "MCP serves a.go:3 for SEARCH/REPLACE"
printf '%s' "$out" > "$R/read84.json"
python3 - "$R/read84.json" <<'PY'
import json,re,sys
r=json.load(open(sys.argv[1]))["result"]
cks=re.findall(r"^-- ck ([0-9a-f]{16}) open ", r["content"][0]["text"], re.M)
assert cks, "a.go:3 carry no checkpoints"
open(sys.argv[1]+".cks","w").write("\n".join(cks))
PY
want 0 $? "a.go:3 carry checkpoints for SEARCH/REPLACE"
req=$(python3 - "$WORK/p84.sr" "$R/read84.json.cks" <<'PY'
import json,sys
plan=open(sys.argv[1]).read()
acks=open(sys.argv[2]).read().split()
print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":plan,"format":"search_replace","ack":acks}}}))
PY
)
out=$(printf '%s\n' "$req" | m mcp 2>/dev/null)
python3 - "$out" "$R/a.go" <<'PY'
import json,sys
r=json.loads(sys.argv[1])["result"]
assert r.get("isError") is True, "unread SEARCH/REPLACE was not a tool error: %s" % r
sc=r.get("structuredContent") or {}
assert sc.get("applied") is False, "unread SEARCH/REPLACE applied: %s" % sc
assert sc.get("failed") == 1, "unread SEARCH/REPLACE failed=%s want 1: %s" % (sc.get("failed"), sc)
st=[h.get("status") for h in sc.get("hunks") or []]
assert "failed" in st and "skipped" in st, "want failed+skipped, got %s" % st
reason=" ".join((h.get("reason") or "") for h in sc.get("hunks") or [])
assert "has not been read" in reason, "not the ledger refusal: %s" % reason
assert "return 1 }" in open(sys.argv[2]).read(), "unread SEARCH/REPLACE wrote"
PY
want 0 $? "MCP format search_replace, one unread line writes nothing"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | m mcp 2>/dev/null)
python3 - "$out" <<'PY'
import json,sys
tools={t["name"]:t for t in json.loads(sys.argv[1])["result"]["tools"]}
assert set(tools)=={"mrw_read","mrw_write"}, "cargo grew: %s" % sorted(tools)
fmt=(tools["mrw_write"].get("inputSchema") or {}).get("properties") or {}
assert "search_replace" in str(fmt.get("format")), "mrw_write does not declare search_replace"
PY
want 0 $? "tools/list still two tools and mrw_write declares search_replace"

# 85. A plan that fails validation writes nothing because a write that changed nothing is
# invisible. A unit test on CLI() cannot prove the shipped binary prints it.
# Drive $MRW: instructions and initialize both carry the why; Shared's five
# sentences remain (the why is extra, not a rewrite); pair: handshake stays
# at most 4096 bytes — funding the sentence by raising the bound fails.
why85='A plan that fails validation writes nothing because a write that changed nothing is invisible.'
out=$(m instructions 2>&1); rc=$?
want 0 "$rc" "mrw instructions still exits 0"
python3 - "$out" "$why85" <<'PY'
import sys
out, why = sys.argv[1], sys.argv[2]
assert why in out, "instructions omitted the why: %r" % why
shared = (
    "Use mrw always: plan the activity as one read of every site, then one plan, then one write.",
    "A plan applies whole or not at all: if any hunk fails validation, nothing is written; a failed commit reports what reached disk (CLI: PARTIALLY APPLIED).",
    "Read before you write, per line, not per file.",
    "mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.",
    "A refusal names the file, the plan line, and the reason.",
)
missing = [s for s in shared if s not in out]
assert not missing, "instructions dropped Shared sentences: %r" % missing
# The why is extra: Shared's first sentence is still the trigger, not the why.
assert not out.startswith(why), "the why replaced Shared at the front of instructions"
PY
[ $? -eq 0 ] && ok "and the binary prints the why after Shared" \
             || bad "instructions omitted the why or dropped Shared"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
want 0 $? "initialize still answers"
python3 - "$out" "$why85" <<'PY'
import json,sys
why = sys.argv[2]
i=json.loads(sys.argv[1])["result"].get("instructions")
assert isinstance(i,str) and i.strip(), "initialize carries no instructions"
assert why in i, "handshake omitted the why"
shared = (
    "Use mrw always: plan the activity as one read of every site, then one plan, then one write.",
    "A plan applies whole or not at all: if any hunk fails validation, nothing is written; a failed commit reports what reached disk (CLI: PARTIALLY APPLIED).",
    "Read before you write, per line, not per file.",
    "mrw models no target syntax: after a multi-line body, read on past the range until the enclosing structure closes.",
    "A refusal names the file, the plan line, and the reason.",
)
for s in shared:
    assert s in i, "handshake dropped Shared sentence: %r" % s
assert not i.startswith(why), "the why replaced Shared at the front of the handshake"
assert len(i.encode()) <= 4096, "handshake is %d bytes; do not raise 4096 to fund the why" % len(i.encode())
PY
[ $? -eq 0 ] && ok "and the handshake teaches the why without raising 4096" \
             || bad "handshake omitted the why, dropped Shared, or overflowed 4096"

# 86. ADR-042: an in-root miss is refused, not a silent whole-project PASS.
#     Pair: a real package still scopes (exit 0); a miss is exit 2 and emits
#     no result document. Prose and testdata still fall back — those paths
#     are there. 4096 stays; ADR-019 A stands.
fixture
mkdir -p "$R/pkg" "$R/docs"
printf 'package pkg\n' > "$R/pkg/p.go"
printf '# prose\n' > "$R/docs/guide.md"
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check pkg 2>&1); rc=$?
want 0 "$rc" "an existing package still scopes"
grep -qF 'SCOPED ./pkg/...' <<<"$out" && ok "and the scoped form ran" || bad "not scoped: $out"
out=$(m check chek.go 2>&1); rc=$?
want 2 "$rc" "a missing in-root path is refused"
grep -qF 'chek.go is not there' <<<"$out" && ok "and the reason names the miss" || bad "reason: $out"
grep -q 'FULL' <<<"$out" && bad "fell back and answered about the root: $out" \
  || ok "and nothing ran under the miss"
out=$(m check --json nosuchdir 2>/dev/null); rc=$?
want 2 "$rc" "--json refuses the same miss"
grep -q 'exit_code' <<<"$out" && bad "a refusal emitted a result document: $out" \
  || ok "and emits no result document to read a verdict out of"
out=$(m check docs 2>&1); rc=$?
want 0 "$rc" "a present prose directory still falls back"
grep -q 'echo FULL' <<<"$out" && ok "and the full command ran" || bad "not full: $out"

# 87. ADR-052: a multi-line replace without a served line after End writes
# nothing. Pair: ranged read of Start-End (exit 1, FAIL+skip, both files
# unchanged) / ranged read through End+1 (exit 0, body applied).
# fixture() whole-reads and would license End+1 — do not use it here.
R=$(mktemp -d "$WORK/r87-XXXXXX")
printf '1\n2\n3\n4\n5\n' > "$R/f.txt"
printf 'keep\n' > "$R/g.txt"
m read 'f.txt:2-3' 'g.txt:1' >"$WORK/served.out"
plan87=$(printf '%s\n' \
	'@@ g.txt 1 replace' \
	'KEEP' \
	'@@ f.txt 2-3 replace anchor="2"' \
	'X' \
	'Y')
out=$(printf '%s\n' "$plan87" | m write - 2>&1); rc=$?
want 1 "$rc" "multi-line replace without End+1 -> exit 1"
grep -q 'FAIL' <<<"$out" && ok "unread neighbour names FAIL" || bad "unread neighbour names FAIL"
grep -q '^skip' <<<"$out" && ok "unread neighbour siblings skip" || bad "unread neighbour siblings skip"
grep -qE 'after 3|line 4' <<<"$out" && ok "unread neighbour names the missing line" || bad "unread neighbour names the missing line"
grep -qx 'keep' "$R/g.txt" && ok "unread neighbour sibling unchanged" || bad "unread neighbour sibling unchanged"
grep -q '^2$' "$R/f.txt" && ok "unread neighbour wrote nothing" || bad "unread neighbour wrote"

R=$(mktemp -d "$WORK/r87b-XXXXXX")
printf '1\n2\n3\n4\n5\n' > "$R/f.txt"
m read 'f.txt:2-4' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.txt 2-3 replace anchor="2"' \
	'X' \
	'Y' | m write - 2>&1); rc=$?
want 0 "$rc" "served End+1 -> exit 0"
grep -q '^X$' "$R/f.txt" && ok "served End+1 applied" || bad "served End+1 applied"

# 88. ADR-052: --echo-pad N prints N lines after the body; a closer there
# stays ok. Pair: --echo-pad 1 shows the line after the body and ok /
# default 0 prints no pad.
R=$(mktemp -d "$WORK/r88-XXXXXX")
printf '1\n2\n3\n</div>\n5\n' > "$R/f.txt"
m read 'f.txt:2-4' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.txt 2-3 replace anchor="2"' \
	'X' \
	'Y' | m write --echo-pad 1 - 2>&1); rc=$?
want 0 "$rc" "--echo-pad 1 applies"
grep -q '^ok' <<<"$out" && ok "pad write stays ok" || bad "pad write stays ok"
grep -q '</div>' <<<"$out" && ok "pad shows the closer" || bad "pad shows the closer"

R=$(mktemp -d "$WORK/r88b-XXXXXX")
printf '1\n2\n3\n</div>\n5\n' > "$R/f.txt"
m read 'f.txt:2-4' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.txt 2-3 replace anchor="2"' \
	'X' \
	'Y' | m write - 2>&1); rc=$?
want 0 "$rc" "default echo-pad 0 applies"
if grep -q '</div>' <<<"$out"; then
  bad "default 0 printed a pad"
else
  ok "default 0 prints no pad"
fi

# 89. ADR-054: a write to a non-prose path runs the project's check by default;
# --no-check opts out; a prose-only plan does not spawn it; a tree with no
# check command still exits 0. The declared check is `exit 3` so a run that
# happened (3) and one that did not (0) are one code apart and nothing else
# produces either. Pair: .go without --check -> 3 / .md without --check -> 0 /
# .go --no-check -> 0 / no harness, no go.mod -> 0 / --check --no-check -> 2.
R=$(mktemp -d "$WORK/r89-XXXXXX")
printf '{"check":"exit 3"}\n' > "$R/.quality-harness.json"
printf 'package a\nfunc A() {}\n' > "$R/a.go"
printf '# notes\nline two\n' > "$R/notes.md"
m read 'a.go:2' 'notes.md:2' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 1 }' | m write - 2>&1); rc=$?
want 3 "$rc" "a .go write without --check runs the declared check (exit 3)"
grep -q 'check FAIL' <<<"$out" && ok "and the receipt shows the failing check" || bad "receipt: $out"

out=$(printf '%s\n' \
	'@@ notes.md 2 replace anchor="line two"' \
	'line 2' | m write - 2>&1); rc=$?
want 0 "$rc" "a markdown-only write does not spawn the check (exit 0)"
if grep -q 'check' <<<"$out"; then
  bad "prose write mentioned the check: $out"
else
  ok "prose write never mentions the check"
fi

m read 'a.go:2' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 2 }' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "--no-check opts out of the default check (exit 0)"

out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 3 }' | m write --check --no-check - 2>&1); rc=$?
want 2 "$rc" "--check --no-check is usage (exit 2)"

R=$(mktemp -d "$WORK/r89b-XXXXXX")
printf 'package a\nfunc A() {}\n' > "$R/a.go"
m read 'a.go:2' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 1 }' | m write - 2>&1); rc=$?
want 0 "$rc" "no harness and no go.mod: the default does not invent exit 2"
if grep -q 'SKIPPED' <<<"$out"; then
  bad "the default reported a skipped check nobody demanded: $out"
else
  ok "no skipped-check line where none was demanded"
fi

# 90. ADR-054: a non-prose hunk whose delimiter nets differ between the
# replaced lines and the body prints a balance row under ok and stays ok; the
# same replace on a .md file prints none. Pair: .go replace of a `{`-only line
# with a balanced body -> ok + balance row, exit 0 / .md -> ok, no balance row /
# .go replace with matching nets -> no balance row.
R=$(mktemp -d "$WORK/r90-XXXXXX")
printf 'package f\nfunc A() {\n\treturn\n}\n' > "$R/f.go"
printf '# t\nfunc A() {\n\treturn\n}\n' > "$R/n.md"
m read 'f.go:2' 'n.md:2' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 2 replace anchor="func A"' \
	'func A() { return }' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a .go replace that unbalances braces still exits 0"
grep -q '^ok' <<<"$out" && ok "and the hunk stays ok" || bad "hunk not ok: $out"
grep -q 'balance {' <<<"$out" && ok "and the receipt carries the brace delta" || bad "no balance row: $out"

out=$(printf '%s\n' \
	'@@ n.md 2 replace anchor="func A"' \
	'func A() { return }' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "the same replace on a .md file exits 0"
if grep -q 'balance' <<<"$out"; then
  bad "a prose hunk printed a balance row: $out"
else
  ok "a prose hunk prints no balance row"
fi

R=$(mktemp -d "$WORK/r90b-XXXXXX")
printf 'package f\nfunc A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:2' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 2 replace anchor="func A"' \
	'func B() {' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a .go replace with matching nets exits 0"
if grep -q 'balance' <<<"$out"; then
  bad "matching nets printed a balance row: $out"
else
  ok "matching nets print no balance row"
fi

# 91. ADR-054: stats prints every vocabulary name at zero, and one landed line
# whose denominator is applied + failed_check + check_not_run. Pair: a checkout
# with one --no-check apply prints `failed_check 0 of 1` and `landed writes: 1`
# / after a default-check write that fails (exit 3) the line reads
# `landed writes: 2; failed_check 1 of those` / --json carries all five keys
# plus landed and failed_check_of_landed.
R=$(mktemp -d "$WORK/r91-XXXXXX")
printf '{"check":"exit 3"}\n' > "$R/.quality-harness.json"
printf 'package a\nfunc A() {}\nfunc B() {}\n' > "$R/a.go"
m read 'a.go:2-3' >"$WORK/served.out"
printf '@@ a.go 2 replace\nfunc A() { _ = 1 }\n' | m write --no-check - >/dev/null 2>&1
out=$(m stats 2>&1); rc=$?
want 0 "$rc" "stats after one applied plan exits 0"
grep -qE 'failed_check +0 of 1' <<<"$out" && ok "failed_check prints at zero" || bad "failed_check hidden at zero: $out"
grep -qE 'check_not_run +0 of 1' <<<"$out" && ok "check_not_run prints at zero" || bad "check_not_run hidden at zero: $out"
grep -q 'landed writes: 1; failed_check 0 of those' <<<"$out" && ok "the landed line names its denominator" || bad "no landed line: $out"

m read 'a.go:3' >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc B() { _ = 1 }\n' | m write - >/dev/null 2>&1; rc=$?
want 3 "$rc" "a default-check write against a failing check exits 3"
out=$(m stats 2>&1); rc=$?
want 0 "$rc" "stats after the failed check exits 0"
grep -q 'landed writes: 2; failed_check 1 of those (50.0%)' <<<"$out" && ok "landed counts the failed check as a landed write" || bad "landed line wrong: $out"
grep -qE 'failed_check +1 of 2' <<<"$out" && ok "failed_check row is 1 of 2" || bad "failed_check row wrong: $out"

jout=$(m stats --json 2>&1); rc=$?
want 0 "$rc" "stats --json exits 0"
for k in applied refused_parse refused_apply partially_applied check_not_run failed_check; do
  grep -q "\"$k\":" <<<"$jout" && ok "--json carries $k" || bad "--json omits $k: $jout"
done
grep -q '"landed": 2' <<<"$jout" && ok "--json landed is 2" || bad "--json landed wrong: $jout"
grep -q '"failed_check_of_landed": 1' <<<"$jout" && ok "--json failed_check_of_landed is 1" || bad "--json failed_check_of_landed wrong: $jout"

# 92. ADR-055: the summary line carries the advisory count, zero included, and
# the JSON receipt carries `advisories`. Pair: a `{`-only replace with a
# balanced body -> `1 advisory` on the summary and "advisories": 1 in JSON /
# a clean replace -> `0 advisories` (the clause is never omitted) / --quiet
# still carries it.
R=$(mktemp -d "$WORK/r92-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a delta replace exits 0"
grep -q '0 failed, 1 advisory — applied' <<<"$out" && ok "the summary says 1 advisory" || bad "summary lacks the count: $out"

R=$(mktemp -d "$WORK/r92b-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func B() {' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "a clean replace exits 0"
grep -q '0 failed, 0 advisories — applied' <<<"$out" && ok "the summary says 0 advisories, not nothing" || bad "clean summary omits the clause: $out"

R=$(mktemp -d "$WORK/r92c-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check --quiet - 2>&1); rc=$?
want 0 "$rc" "--quiet delta replace exits 0"
grep -q '1 advisory' <<<"$out" && ok "--quiet keeps the advisory count" || bad "--quiet dropped it: $out"

R=$(mktemp -d "$WORK/r92d-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
jout=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check --json - 2>/dev/null); rc=$?
want 0 "$rc" "--json delta replace exits 0"
[ "$(jq -r .advisories <<<"$jout")" = "1" ] && ok "--json carries advisories: 1" || bad "--json advisories: $(jq -c .advisories <<<"$jout")"

# 93. ADR-055: a recent-window ring beside the tally; the receipt prints a
# pattern line when three of the last ten landed writes carried an advisory,
# and stats shows the window. Pair: two delta writes -> no `pattern:` / the
# third -> `pattern: 3 of your last 3` / stats -> `recent: 3 write(s)` and the
# line / the ring file holds no path (strings finds no `/` and no `.go`).
R=$(mktemp -d "$WORK/r93-XXXXXX")
for i in 1 2 3; do
  printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
  m read 'f.go:1' >"$WORK/served.out"
  out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check - 2>&1); rc=$?
  want 0 "$rc" "delta write $i exits 0"
  if [ "$i" -lt 3 ]; then
    if grep -q '^pattern:' <<<"$out"; then bad "write $i already printed a pattern line: $out"; else ok "write $i prints no pattern line"; fi
  else
    grep -q '^pattern: 3 of your last 3 writes carried a balance advisory' <<<"$out" && ok "the third advisory prints the pattern line" || bad "no pattern line on the third: $out"
  fi
done
out=$(m stats 2>&1); rc=$?
want 0 "$rc" "stats exits 0"
grep -q 'recent: 3 write(s) in the window' <<<"$out" && ok "stats shows the window" || bad "stats window missing: $out"
grep -q '^pattern: 3 of your last 3' <<<"$out" && ok "stats repeats the pattern line" || bad "stats lacks the pattern: $out"
ring="$(m seen | head -1)/recent"
if [ -f "$ring" ]; then
  ok "the ring exists beside the tally"
  [ "$(wc -l < "$ring" | tr -d ' ')" = "3" ] && ok "ring holds three lines" || bad "ring lines: $(wc -l < "$ring")"
  if grep -qE '/|\.go|f\.go|func' "$ring"; then bad "ring carries a path or plan text: $(cat "$ring")"; else ok "ring carries no path and no plan text"; fi
else
  bad "no ring at $ring"
fi

# 94. ADR-055: --strict-balance refuses the wrap-tail signature and nothing
# else. Pair: flag + single-line replace of a `{`-only line with a balanced
# body -> exit 1, file unchanged, reason names `{ +1` and the flag / same
# plan without the flag -> exit 0 with a balance row / flag + balanced body
# -> exit 0 / flag + the same shape in a .md -> exit 0.
R=$(mktemp -d "$WORK/r94-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
printf 'func A() {\n\treturn\n}\n' > "$R/n.md"
m read 'f.go:1' 'n.md:1' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check --strict-balance - 2>&1); rc=$?
want 1 "$rc" "--strict-balance refuses the wrap-tail signature (exit 1)"
grep -q 'strict-balance' <<<"$out" && grep -q '{ +1' <<<"$out" && ok "the reason names the flag and the net" || bad "reason: $out"
grep -q '^func A() {$' "$R/f.go" && ok "nothing was written" || bad "the refused plan wrote: $(head -1 "$R/f.go")"

out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "the same plan without the flag applies (exit 0)"
grep -q 'balance {' <<<"$out" && ok "and carries the balance row" || bad "no balance row without the flag: $out"

R=$(mktemp -d "$WORK/r94b-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func B() {' | m write --no-check --strict-balance - 2>&1); rc=$?
want 0 "$rc" "--strict-balance leaves a balanced single-line replace alone"

R=$(mktemp -d "$WORK/r94c-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/n.md"
m read 'n.md:1' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ n.md 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check --strict-balance - 2>&1); rc=$?
want 0 "$rc" "--strict-balance leaves prose alone"

# 95. ADR-056: both JSON receipts carry `pattern` on every write. Pair: the
# CLI --json receipt after one delta write -> pattern {1,1,false} / the
# mrw_write receipt on the same checkout after the third -> {3,3,true} /
# the object is present, fires or not.
R=$(mktemp -d "$WORK/r95-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
jout=$(printf '%s\n' \
	'@@ f.go 1 replace anchor="func A"' \
	'func A() { return }' | m write --no-check --json - 2>/dev/null); rc=$?
want 0 "$rc" "delta write 1 exits 0"
[ "$(jq -c '.pattern' <<<"$jout")" = '{"advisory_writes":1,"window":1,"fires":false}' ] \
  && ok "the CLI --json receipt carries pattern {1,1,false}" || bad "CLI pattern: $(jq -c .pattern <<<"$jout")"
for i in 2 3; do
  printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
  m read 'f.go:1' >"$WORK/served.out"
  req=$(printf '@@ f.go 1 replace anchor="func A"\nfunc A() { return }\n' | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read()}}}))')
  mcpout=$(printf '%s\n' "$req" | "$MRW" -C "$R" mcp 2>/dev/null); rc=$?
  want 0 "$rc" "mrw_write $i over a pipe exits 0"
done
pat=$(python3 -c 'import json,sys; print(json.dumps(json.loads(sys.argv[1].splitlines()[0])["result"]["structuredContent"]["pattern"],sort_keys=True))' "$mcpout")
[ "$pat" = '{"advisory_writes": 3, "fires": true, "window": 3}' ] \
  && ok "the mrw_write receipt carries pattern {3,3,true} on the third" || bad "MCP pattern: $pat"

# 96. ADR-056: the tally prices --strict-balance as writes land, counts only.
# Pair: a --no-check delta write -> stats shows 1 candidate, 1 would-refuse,
# unchecked 1 / a clean single-line write -> candidates 2, would-refuse still
# 1 / a --strict-balance delta write is refused and NOT priced (unchanged) /
# --json carries the five keys / the pricing file is `strict_<name> N` lines.
R=$(mktemp -d "$WORK/r96-XXXXXX")
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
printf '%s\n' '@@ f.go 1 replace anchor="func A"' 'func A() { return }' | m write --no-check - >/dev/null 2>&1; rc=$?
want 0 "$rc" "priced delta write exits 0"
out=$(m stats 2>&1)
grep -q 'strict-balance pricing: 1 landed writes with a single-line code replace; the flag would have refused 1 — broke 0, held 0, unchecked 1' <<<"$out" \
  && ok "stats prices the delta write as would-refuse, unchecked" || bad "pricing after one write: $out"
grep -q 'no checked refusals yet' <<<"$out" && ok "no rate is reported on no checked evidence" || bad "rate on no evidence: $out"

printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
printf '%s\n' '@@ f.go 1 replace anchor="func A"' 'func B() {' | m write --no-check - >/dev/null 2>&1; rc=$?
want 0 "$rc" "clean single-line write exits 0"
printf 'func A() {\n\treturn\n}\n' > "$R/f.go"
m read 'f.go:1' >"$WORK/served.out"
printf '%s\n' '@@ f.go 1 replace anchor="func A"' 'func A() { return }' | m write --no-check --strict-balance - >/dev/null 2>&1; rc=$?
want 1 "$rc" "--strict-balance write is refused"
jout=$(m stats --json 2>/dev/null)
[ "$(jq -c '[.pricing.strict_candidates,.pricing.strict_would_refuse,.pricing.strict_would_refuse_broke,.pricing.strict_would_refuse_held,.pricing.strict_would_refuse_unchecked]' <<<"$jout")" = '[2,1,0,0,1]' ] \
  && ok "--json pricing is [2,1,0,0,1]: the clean write is a candidate, the flagged write is not priced" || bad "pricing json: $(jq -c .pricing <<<"$jout")"
pf="$(m seen | head -1)/pricing"
[ -f "$pf" ] && [ "$(grep -cE '^strict_[a-z_]+ [0-9]+$' "$pf")" = "5" ] && [ "$(wc -l < "$pf" | tr -d ' ')" = "5" ] \
  && ok "the pricing file is five strict_<name> N lines" || bad "pricing file: $(cat "$pf" 2>&1)"

# 97. ADR-059: fenceTimeout bounds the check the same way timeout_seconds does.
# Pair: Zeus-shaped fenceTimeout 1 times out sleep 5 on a .go write (exit 3) /
# timeout_seconds and fenceTimeout set differently refuse mrw check at exit 2.
R=$(mktemp -d "$WORK/r97-XXXXXX")
printf '{"check":"sleep 5","fenceTimeout":1}\n' > "$R/.quality-harness.json"
printf 'package a\nfunc A() {}\n' > "$R/a.go"
m read 'a.go:2' >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 1 }' | m write - 2>&1); rc=$?
want 3 "$rc" "fenceTimeout 1 times out the check (exit 3)"
grep -q 'timed out' <<<"$out" && ok "receipt names the timeout" || bad "timeout text: $out"

R=$(mktemp -d "$WORK/r97b-XXXXXX")
printf '{"check":"true","timeout_seconds":300,"fenceTimeout":1}\n' > "$R/.quality-harness.json"
out=$(m check 2>&1); rc=$?
want 2 "$rc" "disagreeing timeout keys refuse mrw check (exit 2)"
echo "$out" | grep -q '300' && echo "$out" | grep -q '1' \
  && ok "refusal names both values" || bad "disagree text: $out"

# 98. ADR-057: native unlink/rename. Pair: whole-file read then unlink
# (exit 0, path gone, receipt `removed`) / unlink plus an unread sibling
# (exit 1, path stays). Rename of a served file lands the dest.
R=$(mktemp -d "$WORK/r98-XXXXXX")
printf 'gone\n' > "$R/gone.txt"
m read gone.txt >"$WORK/served.out"
out=$(printf '%s\n' '@@ gone.txt - unlink' | m write - 2>&1); rc=$?
want 0 "$rc" "whole-file read then unlink -> exit 0"
grep -q 'removed gone.txt' <<<"$out" && ok "receipt names removed" || bad "receipt names removed: $out"
[ ! -e "$R/gone.txt" ] && ok "unlinked path is gone" || bad "gone.txt still exists"

R=$(mktemp -d "$WORK/r98b-XXXXXX")
printf 'gone\n' > "$R/gone.txt"
printf 'keep\n' > "$R/keep.txt"
m read gone.txt >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ gone.txt - unlink' \
	'@@ keep.txt 1 replace anchor="keep"' \
	'nope' | m write - 2>&1); rc=$?
want 1 "$rc" "unlink plus unread sibling -> exit 1"
grep -q '^skip' <<<"$out" && ok "unread sibling skips the unlink" || bad "sibling skip: $out"
[ -f "$R/gone.txt" ] && ok "sibling fail restores gone.txt" || bad "gone.txt was removed on sibling fail"

R=$(mktemp -d "$WORK/r98c-XXXXXX")
printf 'moved\n' > "$R/old.txt"
m read old.txt >"$WORK/served.out"
out=$(printf '%s\n' '@@ old.txt - rename' 'new.txt' | m write - 2>&1); rc=$?
want 0 "$rc" "whole-file read then rename -> exit 0"
[ ! -e "$R/old.txt" ] && [ -f "$R/new.txt" ] && grep -qx 'moved' "$R/new.txt" \
  && ok "rename lands dest and removes source" || bad "rename tree: $(ls -l "$R")"

# 99. ADR-057: apply_patch Delete File / Move to. Pair: Delete File after a
# whole-file read applies / Move to with @@ hunks over an unread file is exit 1
# and the tree is unchanged (ADR-114 compiles it; the ledger refuses it).
R=$(mktemp -d "$WORK/r99-XXXXXX")
printf 'gone\n' > "$R/gone.txt"
m read gone.txt >"$WORK/served.out"
patch99=$(printf '%s\n' \
	'*** Begin Patch' \
	'*** Delete File: gone.txt' \
	'*** End Patch')
out=$(printf '%s\n' "$patch99" | m write --format=apply_patch - 2>&1); rc=$?
want 0 "$rc" "Delete File after whole-file read -> exit 0"
[ ! -e "$R/gone.txt" ] && ok "Delete File removes the path" || bad "Delete File left gone.txt"

R=$(mktemp -d "$WORK/r99b-XXXXXX")
printf 'stay\n' > "$R/a.txt"
patch99b=$(printf '%s\n' \
	'*** Begin Patch' \
	'*** Update File: a.txt' \
	'*** Move to: b.txt' \
	'@@' \
	'-stay' \
	'+gone' \
	'*** End Patch')
# ADR-114: Move to with hunks compiles now; over a file never read it is the
# ledger that refuses it, exit 1, and the tree is unchanged.
out=$(printf '%s\n' "$patch99b" | m write --format=apply_patch - 2>&1); rc=$?
want 1 "$rc" "Move to with hunks over an unread file -> exit 1"
grep -q 'has not been read' <<<"$out" && ok "and the refusal names the unread file" || bad "Move to with hunks text: $out"
grep -qx 'stay' "$R/a.txt" && [ ! -e "$R/b.txt" ] \
  && ok "Move to with hunks wrote nothing" || bad "Move to with hunks wrote: $(ls -l "$R")"

# 100. ADR-060 T1: leftover body= names declared vs extra; dry-run prints parsed.
# Pair: body=1 plus two extras -> exit 2, names body=1 and 2 extra /
# a valid --dry-run prints parsed: body=1.
R=$(mktemp -d "$WORK/r100-XXXXXX")
printf 'package a\nfunc A() {}\n' > "$R/a.go"
out=$(printf '%s\n' \
	'@@ a.go 1 replace body=1' \
	'the body' \
	'extra one' \
	'extra two' | m write --no-check - 2>&1); rc=$?
want 2 "$rc" "leftover body=1 is exit 2"
echo "$out" | grep -q 'body=1' && echo "$out" | grep -q '2 extra' \
	&& ok "leftover names declared vs extra" || bad "leftover: $out"

m read a.go >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A" body=1' \
	'func A() { _ = 1 }' | m write --dry-run --no-check - 2>&1); rc=$?
want 0 "$rc" "dry-run of a counted hunk exits 0"
echo "$out" | grep -q 'parsed:' && echo "$out" | grep -q 'body=1' \
	&& ok "dry-run prints parsed body=1" || bad "parsed: $out"

# 101. ADR-060 T2: a ranged multi-line read hints the neighbour line.
# Pair: read f:1-2 on a 3-line file greps after 2 / whole-file read has no note.
R=$(mktemp -d "$WORK/r101-XXXXXX")
printf 'a\nb\nc\n' > "$R/f.txt"
out=$(m read 'f.txt:1-2' 2>&1); rc=$?
want 0 "$rc" "ranged multi-line read exits 0"
echo "$out" | grep -q 'after 2' \
	&& ok "ranged multi-line read hints the neighbour" || bad "hint: $out"
out=$(m read f.txt 2>&1); rc=$?
want 0 "$rc" "whole-file read exits 0"
echo "$out" | grep -q 'needs a served line after' \
	&& bad "whole-file printed a neighbour hint: $out" \
	|| ok "whole-file read is quiet"

# 102. ADR-060 T3: unquoted anchor= containing " is refused.
# Pair: unquoted vitest-style header exit 2 with anchor=" / quoted form dry-run ok.
R=$(mktemp -d "$WORK/r102-XXXXXX")
printf 'import { inject, vi } from "vitest";\n' > "$R/f.ts"
out=$(printf '%s\n' \
	'@@ f.ts 1 replace anchor=import { inject, vi } from "vitest"; body=1' \
	'X' | m write --no-check - 2>&1); rc=$?
want 2 "$rc" "unquoted anchor with embedded quotes is exit 2"
echo "$out" | grep -q 'anchor="' \
	&& ok "refusal names the quoted form" || bad "unquoted: $out"

m read f.ts >"$WORK/served.out"
cat > "$R/quoted.mrw" <<'EOF'
@@ f.ts 1 replace anchor="import { inject, vi } from \"vitest\";" body=1
X
EOF
out=$(m write --dry-run --no-check "$R/quoted.mrw" 2>&1); rc=$?
want 0 "$rc" "quoted anchor with embedded quotes dry-runs"

# 103. ADR-060 T4: body=@path loads the body from a rooted file.
# Pair: create body=@src.txt writes dest / body=@/etc/hosts refuses naming the path.
R=$(mktemp -d "$WORK/r103-XXXXXX")
printf 'alpha\nbeta\n' > "$R/src.txt"
out=$(printf '%s\n' '@@ dest.txt 0 create body=@src.txt' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "create body=@src.txt exits 0"
if [ -f "$R/dest.txt" ] && grep -qx 'alpha' "$R/dest.txt" && grep -qx 'beta' "$R/dest.txt"; then
	ok "body=@src.txt wrote dest from the file"
else
	bad "dest.txt: $(cat "$R/dest.txt" 2>&1)"
fi
out=$(printf '%s\n' '@@ dest2.txt 0 create body=@/etc/hosts' | m write --no-check - 2>&1); rc=$?
want 2 "$rc" "rooted body=@/etc/hosts is exit 2"
echo "$out" | grep -q '/etc/hosts' \
	&& ok "rooted body=@ names the path" || bad "rooted: $out"
[ ! -e "$R/dest2.txt" ] && ok "rooted body=@ wrote nothing" || bad "dest2.txt exists"

# 104. ADR-060 T5: a failing check prints its last error above the log path.
# Pair: failing check greps check last: unique-last-error-line before full output: /
# check: true has no check last:.
R=$(mktemp -d "$WORK/r104-XXXXXX")
printf '%s\n' '{"check":"sh -c '"'"'echo unique-last-error-line; exit 1'"'"'"}' > "$R/.quality-harness.json"
printf 'package a\nfunc A() {}\n' > "$R/a.go"
m read a.go >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 1 }' | m write - 2>&1); rc=$?
want 3 "$rc" "failing check is exit 3"
last=$(printf '%s\n' "$out" | grep -n 'check last: unique-last-error-line' | head -1 | cut -d: -f1)
full=$(printf '%s\n' "$out" | grep -n 'full output:' | head -1 | cut -d: -f1)
[ -n "$last" ] && [ -n "$full" ] && [ "$last" -lt "$full" ] \
	&& ok "check last: sits above full output:" || bad "check last order: $out"

R=$(mktemp -d "$WORK/r104b-XXXXXX")
printf '%s\n' '{"check":"true"}' > "$R/.quality-harness.json"
printf 'package a\nfunc A() {}\n' > "$R/a.go"
m read a.go >"$WORK/served.out"
out=$(printf '%s\n' \
	'@@ a.go 2 replace anchor="func A"' \
	'func A() { _ = 1 }' | m write - 2>&1); rc=$?
want 0 "$rc" "passing check is exit 0"
echo "$out" | grep -q 'check last:' \
	&& bad "PASS printed check last: $out" || ok "PASS has no check last:"

# 105. ADR-060 T7: replace/insert body=@ parse and load after parse.
# Pair: replace body=@src.txt writes those bytes / empty replace (no body=@) still exit 2.
R=$(mktemp -d "$WORK/r105-XXXXXX")
printf 'package a\nfunc A() {}\n' > "$R/a.go"
printf 'func A() { _ = 1 }\n' > "$R/src.txt"
m read a.go >"$WORK/served.out"
out=$(printf '%s\n' '@@ a.go 2 replace anchor="func A" body=@src.txt' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "replace body=@src.txt exits 0"
if grep -Fq 'func A() { _ = 1 }' "$R/a.go"; then
	ok "replace body=@ wrote src.txt into a.go"
else
	bad "a.go after replace body=@: $(cat "$R/a.go" 2>&1)"
fi

R=$(mktemp -d "$WORK/r105b-XXXXXX")
printf 'alpha\nbeta\n' > "$R/notes.md"
printf 'INSERTED\n' > "$R/src.txt"
m read notes.md >"$WORK/served.out"
out=$(printf '%s\n' '@@ notes.md 1 insert-after body=@src.txt' | m write --no-check - 2>&1); rc=$?
want 0 "$rc" "insert-after body=@src.txt exits 0"
if grep -qx 'INSERTED' "$R/notes.md"; then
	ok "insert-after body=@ wrote src.txt"
else
	bad "notes.md after insert-after body=@: $(cat "$R/notes.md" 2>&1)"
fi

R=$(mktemp -d "$WORK/r105c-XXXXXX")
printf 'package a\nfunc A() {}\n' > "$R/a.go"
m read a.go >"$WORK/served.out"
out=$(printf '%s\n' '@@ a.go 2 replace anchor="func A"' | m write --no-check - 2>&1); rc=$?
want 2 "$rc" "empty replace without body=@ is still exit 2"
echo "$out" | grep -q 'would delete' \
	&& ok "empty replace still names delete" || bad "empty replace: $out"

# 106. ADR-015 T2: several English-word UNREADABLE paths name quoting; one miss stays silent.
fixture
out=$(m read rules that will 2>&1); rc=$?
want 1 "$rc" "three English-word misses exit 1"
n=$(grep -c UNREADABLE <<<"$out")
[ "$n" = 3 ] && ok "three UNREADABLE reports" || bad "UNREADABLE count $n: $out"
grep -q quot <<<"$out" && grep -q -- '--grep' <<<"$out" \
	&& ok "and the hint names quoting and --grep" || bad "English-word hint: $out"
grep -q 'glob your shell did not expand' <<<"$out" \
	&& bad "glob hint fired on English-word paths: $out" \
	|| ok "and it is not the glob hint"

out=$(m read nope.go 2>&1)
grep -q quot <<<"$out" \
	&& bad "a single missing nope.go got the English-word hint: $out" \
	|| ok "a single missing nope.go stays silent"

# 107. ADR-058: missing ast-grep is exit 2 and names the binary, not an unknown flag.
fixture
empty=$(mktemp -d)
out=$(PATH="$empty" m read --ast-grep 'fmt.Println' 2>&1); rc=$?
want 2 "$rc" "missing ast-grep is usage"
grep -q 'ast-grep' <<<"$out" && ok "and the reason names ast-grep" || bad "missing binary: $out"
grep -qE 'flag provided|unknown flag' <<<"$out" \
	&& bad "missing binary looks like an unknown flag: $out" \
	|| ok "and it is not an unknown-flag refusal"

# 108. ADR-058: --grep and --ast-grep together are two sources of specs.
fixture
out=$(m read --grep package --ast-grep 'fmt.Println' 2>&1); rc=$?
want 2 "$rc" "grep plus ast-grep is usage"
grep -q 'two sources' <<<"$out" && ok "and the refusal says two sources" || bad "two sources: $out"

# 109. ADR-058: a present ast-grep with zero hits names the pattern, not PATH.
fixture
d=$(mktemp -d)
printf '#!/bin/sh\necho []\n' > "$d/ast-grep"
chmod +x "$d/ast-grep"
out=$(PATH="$d" m read --ast-grep zzz-absent 2>&1); rc=$?
want 1 "$rc" "zero ast-grep hits exit 1"
grep -q zzz-absent <<<"$out" && ok "and the reason names the pattern" || bad "zero hits: $out"
grep -qE 'not found|PATH' <<<"$out" \
	&& bad "zero hits reported as a missing binary: $out" \
	|| ok "and it is not the missing-binary path"

# 110. ADR-022 T2: the hook's own 2 s wall-clock bound still exits 0.
# Outer perl alarm is the same idiom §55 uses so a missing bound cannot orphan.
HANG110=$(mktemp)
python3 -c '
import pathlib, sys
src = pathlib.Path(sys.argv[1]).read_text()
mut = src.replace("def run(data):", "def run(data):\n    import time\n    time.sleep(30)\n", 1)
assert mut != src, "could not inject hang"
pathlib.Path(sys.argv[2]).write_text(mut)
' "$PWD/.claude/hooks/rules-on-read.py" "$HANG110"
t0=$(date +%s)
printf '%s\n' '{"hook_event_name":"PostToolUse","session_id":"s110","cwd":"/tmp","tool_name":"Bash","tool_input":{"command":"mrw read x.md:1"}}' \
	| perl -e 'alarm shift; exec @ARGV' 5 python3 "$HANG110" >/dev/null 2>/dev/null
rc=$?
t1=$(date +%s)
want 0 "$rc" "a hanging matcher still exits 0"
dur=$((t1 - t0))
[ "$dur" -le 4 ] && ok "and returns within 4 s (bound is 2 s; whole-second clock plus start-up)" || bad "hook hung ${dur}s"

# 111. ADR-058 T3: a hanging ast-grep on PATH is killed at 2 s.
# Bounded by `bounded`, which kills mrw at 5 s: Go ignores SIGALRM, so the
# perl alarm this row used could never have stopped it.
fixture
d111=$(mktemp -d)
printf '%s\n' '#!/bin/sh' "echo \$\$ > '$d111/gc.pid'" 'exec sleep 30' > "$d111/ast-grep"
chmod +x "$d111/ast-grep"
t0=$(date +%s)
bounded 5 "$WORK/111.out" env PATH="$d111:$PATH" "$MRW" -C "$R" read --ast-grep zzz-absent; rc=$?; out=$(cat "$WORK/111.out")
t1=$(date +%s)
# The fake runs in a process group of its own, so if mrw's 2 s kill regressed
# and `bounded` killed mrw instead, the sleep would outlive the run: end it here.
kill -9 "$(cat "$d111/gc.pid" 2>/dev/null || echo 999999999)" 2>/dev/null
want 2 "$rc" "hanging ast-grep is usage"
grep -q 'timed out' <<<"$out" && ok "and the reason says timed out" || bad "timeout: $out"
grep -q 'ast-grep' <<<"$out" && ok "and names ast-grep" || bad "timeout name: $out"
grep -qE 'not found|PATH' <<<"$out" \
	&& bad "timeout reported as a missing binary: $out" \
	|| ok "and it is not the missing-binary path"
grep -q 'no file matched' <<<"$out" \
	&& bad "timeout reported as zero hits: $out" \
	|| ok "and it is not zero hits"
dur=$((t1 - t0))
[ "$dur" -le 4 ] && ok "and returns within 4 s (bound is 2 s; whole-second clock plus start-up)" || bad "ast-grep hung ${dur}s"

# 112. ADR-057 teaching from the 2026-09-15 Zeus field report.
# Pair: write --help names rename dest as the one-line body (not to=) /
# unlink of an unread file is exit 1, says "takes no line address", and does
# not say "a line address means nothing".
help112=$(m write --help)
want 0 $? "write --help exits 0"
printf '%s' "$help112" | grep -q 'to=' && ok "write --help names to=" || bad "write --help names to="
printf '%s' "$help112" | grep -q 'one-line body' && ok "write --help names dest as one-line body" || bad "write --help dest body: $help112"
printf '%s' "$help112" | grep -q 'destination' && ok "write --help names destination" || bad "write --help destination"

R=$(mktemp -d "$WORK/r112-XXXXXX")
printf 'gone\n' > "$R/gone.txt"
out=$(printf '%s\n' '@@ gone.txt - unlink' | m write - 2>&1); rc=$?
want 1 "$rc" "unread unlink is exit 1"
echo "$out" | grep -q 'takes no line address' \
	&& ok "unread unlink names no line address" || bad "unread unlink wording: $out"
echo "$out" | grep -q 'a line address means nothing' \
	&& bad "unread unlink still talks about a line address: $out" \
	|| ok "unread unlink does not say a line address means nothing"
[ -f "$R/gone.txt" ] && ok "unread unlink wrote nothing" || bad "gone.txt was removed unread"

out=$(printf '%s\n' '@@ gone.txt - rename to=new.txt' | m write - 2>&1); rc=$?
want 2 "$rc" "rename to= is usage"
echo "$out" | grep -q 'unknown option' \
	&& ok "to= is an unknown option" || bad "to= refusal: $out"

# 113. ADR-061: a {files}-only scoped_check runs when packages() cannot map.
# Pair: mrw check a.rs with scoped_check echo SCOPED {files} prints SCOPED a.rs /
# the same path with {packages}-only (and mixed) still prints the whole-project
# command. No go.mod: an inferred go test would hide the fallback.
R=$(mktemp -d "$WORK/r113-XXXXXX")
printf 'fn main() {}\n' > "$R/a.rs"
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {files}"}\n' > "$R/.quality-harness.json"
out=$(m check a.rs 2>&1); rc=$?
want 0 "$rc" "{files}-only scoped_check on .rs runs"
grep -qF 'SCOPED a.rs' <<<"$out" && ok "and scopes to the named file" || bad "not files-scoped: $out"
grep -q 'FULL' <<<"$out" && bad "fell back to the whole-project check: $out" \
  || ok "and did not run FULL"

printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check a.rs 2>&1); rc=$?
want 0 "$rc" "{packages}-only on .rs still runs"
grep -q 'echo FULL' <<<"$out" && ok "and falls back to the whole-project check" || bad "scoped an empty packages map: $out"
grep -qF 'SCOPED' <<<"$out" && bad "substituted an empty {packages}: $out" \
  || ok "and did not print SCOPED"

printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages} {files}"}\n' > "$R/.quality-harness.json"
out=$(m check a.rs 2>&1); rc=$?
want 0 "$rc" "mixed placeholders on .rs still run"
grep -q 'echo FULL' <<<"$out" && ok "and mixed still falls back" || bad "mixed scoped with empty packages: $out"

# 114. ADR-062: mrw instructions teaches always and a plan, not a 3+ threshold.
# Pair: CLI stdout carries @@ path 0 create / initialize does not (cookbook
# stays off the handshake) and still names Shared's first sentence, ≤ 4096.
out=$(m instructions 2>&1); rc=$?
want 0 "$rc" "mrw instructions still exits 0"
python3 - "$out" <<'PY'
import sys
out = sys.argv[1]
assert "Use mrw always" in out, "instructions omitted always: %r" % out[:200]
assert "one read of every site, then one plan, then one write" in out, "instructions omitted the plan: %r" % out[:200]
assert "@@ path 0 create" in out, "instructions omitted the create op: %r" % out[:400]
assert "3 or more edits" not in out, "instructions still teaches the 3+ threshold"
PY
[ $? -eq 0 ] && ok "and it teaches always + plan plus @@ path 0 create" \
             || bad "instructions still teach 3+ or omit the cookbook"

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
want 0 $? "initialize still answers"
python3 - "$out" <<'PY'
import json,sys
i=json.loads(sys.argv[1])["result"].get("instructions")
assert isinstance(i,str) and i.strip(), "initialize carries no instructions"
assert "Use mrw always: plan the activity as one read of every site, then one plan, then one write." in i, "handshake omitted Shared first sentence"
assert "@@ path 0 create" not in i, "handshake absorbed the CLI cookbook"
assert "3 or more edits" not in i, "handshake still teaches the 3+ threshold"
assert len(i.encode()) <= 4096, "handshake is %d bytes; do not raise 4096" % len(i.encode())
PY
[ $? -eq 0 ] && ok "and the handshake carries always + plan without the cookbook or a 4096 raise" \
             || bad "handshake omitted always, carried the cookbook, or overflowed 4096"

# 115. ADR-063: mrw instructions teaches the read side, and the forms it
# teaches behave as taught on the built binary.
#
# Good half: the address list (written literally here, never parsed out of
# help) and every flag in the OPTIONS block of `read --help` except --help are
# in the instructions, the instructions name no read flag the binary lacks, and
# --ast-grep's placeholder is PATTERN; the quoted example spec, a -M read, a
# clamped relative read and --files-from - all exit 0. The pair, each of which
# must fail and leave the file alone: a plan start pattern matching twice, a
# plan relative end past the last line, and a plan -M address.
out=$(m instructions 2>&1); rc=$?
want 0 "$rc" "mrw instructions exits 0 with the read side"
help=$(m read --help 2>&1)
python3 - "$out" "$help" <<'PY'
import re, sys
out, help = sys.argv[1], sys.argv[2]
forms = ("PATH:RANGE[,RANGE...]", "N-M", "N- (to the end)", "-M (from the start)",
         "A,+N", "$ (the last line)", "/regexp/", "/from/,/to/",
         "but not -M or a comma list", "must match exactly once",
         "'b.go:/func Start/,+12'")
missing = [f for f in forms if f not in out]
assert not missing, "instructions omit read forms: %r" % missing
opts = help.split("OPTIONS:", 1)[1].split("GLOBAL OPTIONS:", 1)[0]
flags = [f for f in re.findall(r"^\s+--([a-z][a-z-]*)", opts, re.M) if f != "help"]
assert len(flags) >= 8, "read --help lists only %r" % flags
untaught = [f for f in flags if not re.search(r"--%s(?![A-Za-z0-9-])" % re.escape(f), out)]
assert not untaught, "instructions omit read flags: %r" % untaught
# --then and --then-sh are write and check flags the instructions teach too (ADR-092),
# and --no-check is the write flag they name for a depth past the limit (ADR-095).
unknown = set(re.findall(r"--([a-z][a-z-]*)", out)) - set(flags) - {"root", "help", "then", "then-sh", "no-check"}
assert not unknown, "instructions teach flags read does not have: %r" % sorted(unknown)
assert re.search(r"^\s+--ast-grep PATTERN\b", opts, re.M), "read --help does not show --ast-grep PATTERN"
PY
[ $? -eq 0 ] && ok "and it names every read form and read flag, and --ast-grep shows PATTERN" \
             || bad "instructions omit a read form or flag, or --ast-grep shows the wrong placeholder"

fixture
printf 'func Start\nX one\ny\nX two\nz\n' > "$R/t.txt"
m read t.txt >"$WORK/served.out" 2>&1
m read 't.txt:/func Start/,+12' >"$WORK/served.out" 2>&1
want 0 $? "the quoted example spec serves"
m read t.txt:-2 >"$WORK/served.out" 2>&1
want 0 $? "a read takes -M"
out=$(m read 't.txt:4,+9' 2>&1); rc=$?
want 0 "$rc" "a read clamps a relative end past the last line"
grep -q '^@@ 4-5' <<<"$out" && ok "and serves @@ 4-5" || bad "the clamped read served: $out"
printf 't.txt:/X/\n' | m read --files-from - >"$WORK/served.out" 2>&1
want 0 $? "--files-from - takes specs on stdin"
before=$(cat "$R/t.txt")
printf '@@ t.txt /X/ replace\nQ\n' | m write --no-check - >/dev/null 2>&1
want 1 $? "a plan start pattern matching twice is refused"
printf '@@ t.txt 4,+9 delete\n' | m write --no-check - >/dev/null 2>&1
want 1 $? "a plan relative end past the last line is refused"
printf '@@ t.txt -2 delete\n' | m write --no-check - >/dev/null 2>&1
want 2 $? "a plan -M address is refused"
[ "$(cat "$R/t.txt")" = "$before" ] && ok "and none of the three refusals changed the file" \
                                     || bad "a refused plan changed t.txt"

# 116. ADR-064: both finders apply ADR-007's exclusion rule. A file the caller
# NAMES is never pruned by a glob that matches it, and a directory the search
# walks into is pruned when the glob matches it. The fake ast-grep is §111's
# shape — a script that prints a fixed JSON hit whatever it is asked — so each
# case installs the hit it needs. Every $MRW call runs under `bounded`, which
# kills it at 5 s; the alarm it used could not, since Go ignores SIGALRM.
fixture
d116="$WORK/fake116"; mkdir -p "$d116"
fake116() { printf '%s\n' "$1" > "$d116/hit.json"; printf '#!/bin/sh\ncat "%s"\n' "$d116/hit.json" > "$d116/ast-grep"; chmod +x "$d116/ast-grep"; }
m116() { bounded 5 "$WORK/116.out" env PATH="$d116:$PATH" "$MRW" -C "$R" "$@"; local rc=$?; cat "$WORK/116.out"; return $rc; }
fake116 '[{"file":"b.go","range":{"start":{"line":2},"end":{"line":2}}}]'
out=$(m116 read --ast-grep D --exclude b.go b.go 2>&1); rc=$?
want 0 "$rc" "ast-grep with b.go named and excluded exits 0"
grep -q '^==> b.go' <<<"$out" && ok "ast-grep serves a named file the glob matches" \
                              || bad "ast-grep dropped a file the caller named: $out"
m116 read --ast-grep D --exclude b.go >"$WORK/served.out" 2>&1
want 1 $? "with nothing named, ast-grep's hit on b.go meets the glob"
out=$(m116 read --grep 'func D' --exclude b.go b.go 2>&1); rc=$?
want 0 "$rc" "grep with b.go named and excluded exits 0"
grep -q '^==> b.go' <<<"$out" && ok "grep serves a named file the glob matches" \
                              || bad "grep dropped a file the caller named: $out"
m116 read --grep 'func D' --exclude b.go >"$WORK/served.out" 2>&1
want 1 $? "with nothing named, grep's hit on b.go meets the glob"
mkdir -p "$R/vendor"; printf 'package v\nfunc D() {}\n' > "$R/vendor/v.go"
fake116 '[{"file":"vendor/v.go","range":{"start":{"line":1},"end":{"line":1}}}]'
out=$(m116 read --ast-grep D --exclude vendor 2>&1); rc=$?
want 1 "$rc" "ast-grep's only hit sits under an excluded walked directory"
grep -q 'vendor/v.go' <<<"$out" && bad "ast-grep served a hit under an excluded walked directory: $out" \
                                || ok "ast-grep prunes a walked directory the glob matches"
out=$(m116 read --ast-grep D --exclude vendor vendor 2>&1); rc=$?
want 0 "$rc" "ast-grep with vendor named and excluded exits 0"
grep -q '^==> vendor/v.go' <<<"$out" && ok "and a directory the caller names is not pruned" \
                                     || bad "ast-grep pruned a directory the caller named: $out"

# 117. ADR-064: every pipeline mrw teaches for agents runs as an agent runs it —
# stdin an open pipe, under a 5 s bound. rg given no path searches a piped stdin
# and waits (found 2026-09-24 by the ADR-063 chaos pass), which is why the taught
# line names `.`. This row executes the line the BINARY prints and the one
# AGENTS.md carries, and pairs them with the path-less line, which the bound must
# kill — the case that proves the row can see the hang at all. The open stdin is
# a FIFO held by a sleep this row kills. The alarm kills bash, not the pipeline's
# children; they exit when the FIFO's writer goes, and the runner's group kill
# on EXIT reaps anything left.
if command -v rg >/dev/null 2>&1; then
  fixture
  printf 'package demo\n\nfunc Handle() {}\n' > "$R/h.go"
  rm -f "$WORK/117.fifo"; mkfifo "$WORK/117.fifo"; sleep 60 > "$WORK/117.fifo" & hold117=$!
  run117() { ( cd "$R" && PATH="$(dirname "$MRW"):$PATH" perl -e 'alarm shift; exec @ARGV' 5 bash -c "$1" < "$WORK/117.fifo" > "$WORK/117.out" 2>&1 ); }
  line=$("$MRW" instructions | grep -oE "rg -l X \. \| sed '[^']*' \| mrw read --files-from -")
  if [ -z "$line" ]; then
    bad "the binary teaches no rg pipeline that names its path"
  else
    cmd=${line//X/Handle}
    run117 "$cmd"; rc=$?
    { [ "$rc" -eq 0 ] && grep -q '^==> ' "$WORK/117.out"; } \
      && ok "the taught rg pipeline completes under an open stdin" \
      || bad "the taught rg pipeline did not complete under an open stdin (exit $rc): $(head -2 "$WORK/117.out" | tr '\n' ' ')"
    hung=${cmd/ . |/ |}
    t0=$(date +%s); run117 "$hung"; rc=$?; t1=$(date +%s)
    { [ "$rc" -gt 128 ] && [ $((t1 - t0)) -ge 4 ]; } \
      && ok "without its path, rg reads the pipe and the bound kills it" \
      || bad "the path-less pipeline was not held by stdin (exit $rc after $((t1 - t0)) s): the row cannot see the hang"
  fi
  aline=$(grep -m1 -E "^rg -l .* \| mrw read .*--files-from -$" "$SRC/AGENTS.md")
  if [ -z "$aline" ]; then
    bad "AGENTS.md carries no rg | mrw read --files-from - pipeline"
  else
    run117 "$aline"; rc=$?
    { [ "$rc" -eq 0 ] && grep -q '^==> ' "$WORK/117.out"; } \
      && ok "AGENTS.md's rg pipeline completes under an open stdin" \
      || bad "AGENTS.md's rg pipeline did not complete under an open stdin (exit $rc): $(head -2 "$WORK/117.out" | tr '\n' ' ')"
  fi
  kill "$hold117" 2>/dev/null; wait "$hold117" 2>/dev/null
else
  skip "rg absent — taught pipeline not executed"
fi

# 118. ADR-066: a rename's destination is checked before anything is written.
# A destination directory that cannot be created used to be discovered at
# COMMIT, after the plan's other files were already renamed into place, while
# every hunk read `ok` and the summary said `applied`. A 300-byte component is
# ENAMETOOLONG on every OS mrw runs on, so the row needs no permissions and
# works as root. The good case is the same plan with a short name.
fixture
L118=$(printf '%0300d' 0 | tr 0 x)
printf 'sib\n' > "$R/s.txt"; printf 'bee\n' > "$R/b.txt"; m read s.txt b.txt >"$WORK/served.out"
printf '@@ s.txt 1 replace\nSIB\n@@ b.txt - rename\nd/short/f.txt\n' | m write --no-check - >"$WORK/118.out" 2>&1
want 0 $? "a sibling edit and a rename into a new directory exit 0"
{ grep -q -- '— applied' "$WORK/118.out" && [ -f "$R/d/short/f.txt" ]; } \
  && ok "a rename into a new directory applies" \
  || bad "a rename into a new directory did not apply: $(tr '\n' ' ' < "$WORK/118.out")"
fixture
printf 'sib\n' > "$R/s.txt"; printf 'bee\n' > "$R/b.txt"; m read s.txt b.txt >"$WORK/served.out"
plan118="$(printf '@@ s.txt 1 replace\nSIB\n@@ b.txt - rename\nn/%s/f.txt\n' "$L118")"
printf '%s\n' "$plan118" | m write --no-check - >"$WORK/118.out" 2>&1
want 1 $? "a rename whose destination directory cannot be made exits 1 (ADR-132: a name too long is the target's)"
{ [ "$(cat "$R/s.txt")" = "sib" ] && [ -f "$R/b.txt" ] && [ ! -e "$R/n" ] \
    && grep -q '^FAIL' "$WORK/118.out" && grep -q '^skip' "$WORK/118.out" \
    && ! grep -q -- '— applied' "$WORK/118.out"; } \
  && ok "a rename whose directory cannot be made writes nothing" \
  || bad "a rename whose directory cannot be made changed the tree or claimed success: $(tr '\n' ' ' < "$WORK/118.out" | cut -c1-300)"
if command -v jq >/dev/null 2>&1; then
  # Into a file first: under this file's pipefail, mrw's exit 2 would fail the
  # pipeline whatever jq decided.
  printf '%s\n' "$plan118" | m write --no-check --json - >"$WORK/118.json" 2>/dev/null
  jq -e '.failed==1 and .applied==false and ([.files[]|select(.written)]|length==0)' "$WORK/118.json" >/dev/null \
    && ok "and its --json receipt says one failed, nothing written" \
    || bad "the --json receipt of an uncreatable rename directory is not failed==1, applied==false, nothing written"
else
  skip "jq absent — the --json half of §118 not checked"
fi
mkdir -p "$R/ok"
printf '@@ s.txt 1 replace\nSIB\n@@ b.txt - rename\nok/%s\n' "$L118" | m write --no-check - >"$WORK/118.out" 2>&1
want 1 $? "a rename whose name the filesystem rejects fails validation (exit 1)"
{ [ "$(cat "$R/s.txt")" = "sib" ] && [ -f "$R/b.txt" ]; } \
  && ok "a rename whose name the filesystem rejects is refused" \
  || bad "a rejected rename name reached the tree: $(tr '\n' ' ' < "$WORK/118.out" | cut -c1-300)"
printf '@@ s.txt 1 replace\nSIB\n@@ b.txt - rename\nn2/%s\n' "$L118" | m write --no-check - >"$WORK/118.out" 2>&1
want 1 $? "a rename whose leaf is rejected under a new parent exits 1 (ADR-132)"
{ [ "$(cat "$R/s.txt")" = "sib" ] && [ -f "$R/b.txt" ] && [ ! -e "$R/n2" ]; } \
  && ok "a rename whose leaf is rejected under a new parent writes nothing" \
  || bad "a rejected leaf under a new parent reached the tree: $(tr '\n' ' ' < "$WORK/118.out" | cut -c1-300)"

# 119. ADR-066: a failed unlink/rename commit is undone as a unit, and a commit
# that wrote some files before failing says PARTIALLY APPLIED. The failure is a
# rename into a directory made read-only, which only bites for a non-root user,
# so the row probes it first and SKIPs visibly when the write goes through, as
# §21 does; the Go tests drive the same paths through a seam as root too. The
# must-fail replacing plan is the shape that lost data on v1.22.3: unlink c,
# rename b onto c, then a rename that fails — the old restore moved c back over
# b and B-CONTENT existed nowhere.
# ADR-086 moved the read-only case to staging: a rename's destination is now
# created and removed there, so the directory's refusal fails the plan before
# its content edit lands, and the mixed plan below writes NOTHING. The
# PARTIALLY APPLIED report for a commit failure no probe can foresee is driven
# through the commitRenameFn seam by
# TestAFailedContentCommitReportsWrittenHunksOkAndTheRestSkipped; its wording by
# TestAPartialCommitIsNotSummarisedAsApplied and
# TestAPartialWriteNamesWhatWasAlreadyWritten.
fixture
mkdir -p "$R/ro" "$R/w"; chmod 555 "$R/ro"
if ( : > "$R/ro/.probe" ) 2>/dev/null; then
  rm -f "$R/ro/.probe"; chmod 755 "$R/ro"
  skip "a read-only directory is writable here — §119 not driven through the binary"
else
  setup119() { printf 'OLD-C\n' > "$R/c.txt"; printf 'B-CONTENT\n' > "$R/b.txt"; printf 'D\n' > "$R/d.txt"; printf 'sib\n' > "$R/s.txt"; m read c.txt b.txt d.txt s.txt >"$WORK/served.out"; }
  setup119
  printf '@@ c.txt - unlink\n@@ b.txt - rename\nc.txt\n@@ d.txt - rename\nw/d.txt\n' | m write --no-check - >"$WORK/119.out" 2>&1
  want 0 $? "a replacing rename plan into a writable directory exits 0"
  rm -f "$R/w/d.txt"; setup119
  printf '@@ c.txt - unlink\n@@ b.txt - rename\nc.txt\n@@ d.txt - rename\nro/d.txt\n' | m write --no-check - >"$WORK/119.out" 2>&1
  want 1 $? "a replacing rename plan whose last rename cannot land exits 1 (ADR-132: a permission is the target's)"
  { [ "$(cat "$R/c.txt")" = "OLD-C" ] && [ "$(cat "$R/b.txt" 2>/dev/null)" = "B-CONTENT" ] && [ "$(cat "$R/d.txt")" = "D" ] \
      && ! grep -q -- '— applied' "$WORK/119.out"; } \
    && ok "a failed rename after a replacing rename loses no file" \
    || bad "a failed rename after a replacing rename lost or moved a file: c=$(cat "$R/c.txt" 2>&1) b=$(cat "$R/b.txt" 2>&1) :: $(tr '\n' ' ' < "$WORK/119.out" | cut -c1-300)"
  setup119
  printf '@@ s.txt 1 replace\nSIB\n@@ d.txt - rename\nro/d.txt\n' | m write --no-check - >"$WORK/119.out" 2>&1
  want 1 $? "a mixed plan whose rename cannot land exits 1 (ADR-132)"
  { [ "$(cat "$R/s.txt")" = "sib" ] && grep -q '^FAIL' "$WORK/119.out" \
      && ! grep -q 'PARTIALLY APPLIED' "$WORK/119.out"; } \
    && ok "a mixed plan whose rename cannot land writes nothing: ADR-086 refuses it at staging" \
    || bad "a mixed plan whose rename cannot land reached the tree: $(tr '\n' ' ' < "$WORK/119.out" | cut -c1-300)"
  if command -v jq >/dev/null 2>&1; then
    setup119
    printf '@@ s.txt 1 replace\nSIB\n@@ d.txt - rename\nro/d.txt\n' | m write --no-check --json - >"$WORK/119.json" 2>/dev/null
    jq -e '.failed==1 and .applied==false and ([.files[]|select(.written)]|length==0)' "$WORK/119.json" >/dev/null \
      && ok "and its --json receipt says one failed and no file written" \
      || bad "the --json receipt of a staging refusal is not failed==1, applied==false, no file written"
  else
    skip "jq absent — the --json half of §119 not checked"
  fi
  chmod 755 "$R/ro"
fi

# 120. ADR-065: read and write number a file's lines the same way. A CR-only
# file used to be served as ONE line while the write engine addressed three, so
# a write to its line 2 applied though read had never served it; a CRLF line was
# served with its \r, so a read's /one$/ missed where the write's matched.
fixture
printf 'one\rtwo\rthree\r' > "$R/cr.txt"
out=$(m read cr.txt 2>&1)
{ grep -q '^==> cr.txt  3L' <<<"$out" && ! grep -q '^==> cr.txt  1L' <<<"$out"; } \
  && ok "a CR-only file is served as its three lines" \
  || bad "a CR-only file was not served as 3L: $(head -1 <<<"$out")"
fixture
printf 'one\rtwo\rthree\r' > "$R/cr.txt"
m read cr.txt:2 >"$WORK/served.out"
printf '@@ cr.txt 2 replace\nTWO\n' | m write --no-check - >"$WORK/120.out" 2>&1
want 0 $? "a write to the served line 2 of a CR-only file exits 0"
[ "$(od -An -c "$R/cr.txt" | tr -d ' \n')" = 'one\rTWO\rthree\r' ] \
  && ok "a CR-only line read is the line a write addresses" \
  || bad "the CR-only write did not replace exactly line 2: $(od -An -c "$R/cr.txt" | tr -d '\n')"
printf 'one\r\ntwo\r\nthree\r\n' > "$R/crlf.txt"
m read crlf.txt >"$WORK/served.out"
printf '@@ crlf.txt 1 replace\nONE\n' | m write --no-check - >"$WORK/120.out" 2>&1
want 0 $? "a write after a whole read of a CRLF file exits 0"
[ "$(od -An -c "$R/crlf.txt" | tr -d ' \n')" = 'ONE\r\ntwo\r\nthree\r\n' ] \
  && ok "a CRLF file read whole is written without changing its endings" \
  || bad "the CRLF write changed its endings: $(od -An -c "$R/crlf.txt" | tr -d '\n')"
m read 'crlf.txt:/two$/' >"$WORK/served.out" 2>&1
want 0 $? "a CRLF line is served without its \\r, so /two$/ matches line 2"
m read 'crlf.txt:/two\r$/' >"$WORK/served.out" 2>&1
want 1 $? "a read pattern that needs the \\r a CRLF line no longer carries matches nothing"

# 121. ADR-065: the foreign plan formats compile against the lines the write
# engine numbers. The document is LF-normalised (ADR-051 F-9), so on a CRLF or
# CR-only target an old side used to meet "two\r" (or one whole-file line) and
# the edit was refused with "matched no lines", exit 2. The must-fail case is an
# old side the file does not hold, which must still refuse and write nothing.
fixture
printf 'one\r\ntwo\r\nthree\r\n' > "$R/crlf.txt"
m read crlf.txt >"$WORK/served.out"
printf '*** Begin Patch\n*** Update File: crlf.txt\n@@\n one\n-two\n+TWO\n three\n*** End Patch\n' \
  | m write --no-check --format=apply_patch - >"$WORK/121.out" 2>&1
want 0 $? "an apply_patch edit of a CRLF file exits 0"
[ "$(od -An -c "$R/crlf.txt" | tr -d ' \n')" = 'one\r\nTWO\r\nthree\r\n' ] \
  && ok "apply_patch edits a CRLF file" \
  || bad "apply_patch did not edit the CRLF file: $(tr '\n' ' ' < "$WORK/121.out" | cut -c1-200)"
printf 'one\rtwo\rthree\r' > "$R/cr.txt"
m read cr.txt >"$WORK/served.out"
printf 'cr.txt\n<<<<<<< SEARCH\ntwo\n=======\nTWO\n>>>>>>> REPLACE\n' \
  | m write --no-check --format=search_replace - >"$WORK/121.out" 2>&1
want 0 $? "a search_replace edit of a CR-only file exits 0"
[ "$(od -An -c "$R/cr.txt" | tr -d ' \n')" = 'one\rTWO\rthree\r' ] \
  && ok "search_replace edits a CR-only file" \
  || bad "search_replace did not edit the CR-only file: $(tr '\n' ' ' < "$WORK/121.out" | cut -c1-200)"
printf '*** Begin Patch\n*** Update File: crlf.txt\n@@\n one\n-absent\n+X\n*** End Patch\n' \
  | m write --no-check --format=apply_patch - >"$WORK/121.out" 2>&1
want 2 $? "an apply_patch old side the CRLF file lacks still refuses"
[ "$(od -An -c "$R/crlf.txt" | tr -d ' \n')" = 'one\r\nTWO\r\nthree\r\n' ] \
  && ok "and the refused apply_patch wrote nothing" \
  || bad "a refused apply_patch changed the CRLF file"

# 122. ADR-067 T1: an mrw_read grep too large to serve degrades to an index,
# and the index now names every path the walk could not use, as a served answer
# does. It used to carry only their count. The pair: the same grep without the
# missing path names nothing, so the line comes from the problem.
fixture
mkdir -p "$R/g"
for i in $(seq 1 60); do
  awk 'BEGIN { print "none"; for (j = 0; j < 400; j++) print "the NEEDLE is here" }' > "$R/g/d$i.csv"
done
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["no-such-dir","g"],"grep":"NEEDLE"}}}' \
  | m mcp > "$WORK/122.out" 2>/dev/null
python3 - "$WORK/122.out" <<'PY'
import json,sys
t=json.load(open(sys.argv[1]))["result"]["content"][0]["text"]
assert "-- INDEX:" in t, "not an index: %s" % t[:300]
assert "-- no-such-dir:" in t, "the index does not name the missing path: %s" % t[:600]
PY
want 0 $? "an oversized grep index names the path it could not use"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["g"],"grep":"NEEDLE"}}}' \
  | m mcp > "$WORK/122b.out" 2>/dev/null
python3 - "$WORK/122b.out" <<'PY'
import json,sys
t=json.load(open(sys.argv[1]))["result"]["content"][0]["text"]
assert "-- INDEX:" in t and "no-such-dir" not in t, "the clean grep index names a missing path: %s" % t[:600]
PY
want 0 $? "and the same grep without the missing path names nothing"

# 123. ADR-067 T2: initialize answers the version the host asked for when mrw
# speaks it. Claude Code 2.1.281 asks for 2025-11-25 (wire capture, 2026-09-24),
# and mrw used to answer 2025-06-18 whatever was asked. The pair: a host asking
# 2025-06-18 still gets 2025-06-18.
fixture
for v in 2025-11-25 2025-06-18; do
  printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"%s"}}\n' "$v" \
    | m mcp > "$WORK/123-$v.out" 2>/dev/null
done
python3 - "$WORK/123-2025-11-25.out" "$WORK/123-2025-06-18.out" <<'PY'
import json,sys
for path, want in zip(sys.argv[1:], ["2025-11-25", "2025-06-18"]):
    got = json.load(open(path))["result"]["protocolVersion"]
    assert got == want, "asked for %s, answered %s" % (want, got)
PY
want 0 $? "initialize answers the version the host asked for"

# 124. ADR-067 T3: a caller's argument mistake is a tool execution error — a
# result with isError that a host hands the model (SEP-1303) — not a JSON-RPC
# -32602. The pair: a call whose params are not even an object is still one.
fixture
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"x","echo_pad":-1}}}' \
  | m mcp > "$WORK/124.out" 2>/dev/null
python3 - "$WORK/124.out" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]))
assert "error" not in r, "an argument mistake was a JSON-RPC error: %s" % r
r=r["result"]
assert r.get("isError") is True and "echo_pad" in r["content"][0]["text"], "not an isError result naming echo_pad: %s" % r
PY
want 0 $? "an argument mistake is a tool result the model reads"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":"x"}' | m mcp > "$WORK/124b.out" 2>/dev/null
python3 - "$WORK/124b.out" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]))
assert r.get("error",{}).get("code") == -32602, "a malformed call was not -32602: %s" % r
PY
want 0 $? "and a malformed call is still a protocol error"

# 125. ADR-067 T4: mrw is dual-era. A request carrying the 2026-07-28 per-request
# _meta is served per request — server/discover answers, a tools/call result
# carries resultType — and a request naming a version mrw does not speak gets
# -32022 with the supported list. No measured host sends this era yet (Claude
# Code 2.1.281 opens with initialize, 2026-09-24), so this is a replayed probe.
fixture
M='"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}'
{
  printf '{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{%s}}\n' "$M"
  printf '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["b.go"]},%s}}\n' "$M"
  printf '%s\n' '{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01","io.modelcontextprotocol/clientCapabilities":{}}}}'
} | m mcp > "$WORK/125.out" 2>/dev/null
python3 - "$WORK/125.out" <<'PY'
import json,sys
lines=[json.loads(l) for l in open(sys.argv[1]) if l.strip()]
d=lines[0]["result"]
assert d["supportedVersions"]==["2026-07-28","2025-11-25","2025-06-18"], "discover: %s" % d
c=lines[1]["result"]
assert c.get("resultType")=="complete" and "func D()" in c["content"][0]["text"], "modern tools/call: %s" % c
PY
want 0 $? "a modern request is served per request"
python3 - "$WORK/125.out" <<'PY'
import json,sys
lines=[json.loads(l) for l in open(sys.argv[1]) if l.strip()]
e=lines[2].get("error",{})
assert e.get("code")==-32022 and e.get("data",{}).get("supported")==["2026-07-28","2025-11-25","2025-06-18"], "unknown version: %s" % lines[2]
PY
want 0 $? "a modern request naming an unknown version is refused with the supported list"

# 126. An MCP read of a file whose name holds a space is served, with the
# checkpoints that license a write. From ADR-039 (v1.11.0) to v1.24.0 a read
# that fit on one page was refused with -32603 "holding checkpoints: no
# observation for x": the served header's path was cut at its first space. Found by the chaos harness's MCP
# suite, 2026-09-24. The pair: a spaced path that does not exist is reported as
# a problem naming it, not as an internal error.
fixture
printf 'hello\n' > "$R/x y.txt"
{
  printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["x y.txt"]}}}'
  printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["no such.txt"]}}}'
} | m mcp > "$WORK/126.out" 2>/dev/null
python3 - "$WORK/126.out" <<'PY'
import json,sys
r=[json.loads(l) for l in open(sys.argv[1]) if l.strip()][0]
assert "error" not in r, "a spaced file was refused: %s" % r
t=r["result"]["content"][0]["text"]
assert "    1| hello" in t and "-- ck " in t, "a spaced file was not served with checkpoints: %s" % t[:300]
PY
want 0 $? "an MCP read of a file with a space in its name is served"
python3 - "$WORK/126.out" <<'PY'
import json,sys
r=[json.loads(l) for l in open(sys.argv[1]) if l.strip()][1]
assert "error" not in r and "no such.txt" in r["result"]["content"][0]["text"], "a missing spaced path: %s" % r
PY
want 0 $? "and a missing spaced path is reported by name, not as an internal error"

# 127. A read of a file whose name ends in a space or a carriage return licenses
# that file, not the one whose name is it trimmed (ADR-068). Through v1.24.0 the
# ledger loaded the observation of "x " under the key "x" (parseLine trimmed the
# line) and of "x\r" under "x" (bufio.ScanLines dropped the \r), and the SHA
# guard could not tell them apart because they held the same bytes, so a write
# to the unread "x" applied, exit 0. The read passes `--` because urfave/cli
# trims a positional argument before it (BACKLOG, From ADR-068). The pair: the
# file that WAS read is still writable.
fixture
printf 'same\n' > "$R/x"
printf 'same\n' > "$R/x "
m read -- "x " >"$WORK/served.out" 2>&1
printf '@@ x 1 replace\nWROTE\n' > "$WORK/127a.mrw"
m write --no-check "$WORK/127a.mrw" >/dev/null 2>&1
want 1 $? "a read of a trailing-space path does not license its trimmed sibling"
[ "$(cat "$R/x")" = same ] && ok "and the trimmed sibling is unchanged" || bad "x now holds: $(cat "$R/x")"
printf '@@ "x " 1 replace\nWROTE\n' > "$WORK/127b.mrw"
m write --no-check "$WORK/127b.mrw" >/dev/null 2>&1
want 0 $? "and the file that was read is writable"
[ "$(cat "$R/x ")" = WROTE ] && ok "and the write to it landed" || bad "x-space holds: $(cat "$R/x ")"
fixture
printf 'same\n' > "$R/x"
printf 'same\n' > "$R/x"$'\r'
m read -- "x"$'\r' >"$WORK/served.out" 2>&1
want 0 $? "a read of a trailing-CR path is served"
m write --no-check "$WORK/127a.mrw" >/dev/null 2>&1
want 1 $? "a read of a trailing-CR path does not license its trimmed sibling"
[ "$(cat "$R/x")" = same ] && ok "and that sibling is unchanged" || bad "x now holds: $(cat "$R/x")"
printf '@@ "x\r" 1 replace\nWROTE\n' > "$WORK/127c.mrw"
m write --no-check "$WORK/127c.mrw" >/dev/null 2>&1
want 0 $? "and the trailing-CR file that was read is writable"
[ "$(cat "$R/x"$'\r')" = WROTE ] && ok "and the write to it landed" || bad "x-CR holds: $(cat "$R/x"$'\r')"


# 128. ADR-069 T1: urfave/cli trims a positional before `--`, so `read 'x '`
# served x, a file the caller did not name. It is refused, exit 2, naming the
# `--` that reaches it; the pair: `read -- 'x '` still serves it.
fixture
printf 'padded\n' > "$R/x "
out=$(m read 'x ' 2>&1); rc=$?
want 2 "$rc" "a padded positional without -- is refused and names --"
grep -qF "put -- before the path" <<<"$out" && ok "and the refusal says where -- goes" || bad "refusal: $out"
out=$(m read -- 'x ' 2>&1); rc=$?
want 0 "$rc" "and read -- 'x ' is served"
grep -q 'padded' <<<"$out" && ok "and it serves the file named" || bad "served: $out"

# 129. ADR-069 T2: a --files-from line was TrimSpaced, so a list naming "x "
# served x. The pair: the list's comment and blank line are still skipped.
fixture
printf 'plain\n' > "$R/x"
printf 'padded\n' > "$R/x "
out=$(printf '# note\n\nx \n' | m read --files-from - 2>&1); rc=$?
want 0 "$rc" "a --files-from list with a comment and a blank line reads"
grep -q 'padded' <<<"$out" && ! grep -q 'plain' <<<"$out" \
	&& ok "a --files-from line keeps its trailing space" || bad "files-from served: $out"

# 130. ADR-069 T3: a rename's destination was TrimSpaced, so a rename to "d "
# landed at d. The pair: the rename still applies.
fixture
printf 'src\n' > "$R/a.txt"
m read a.txt >"$WORK/served.out" 2>&1
printf '@@ a.txt - rename\nd \n' > "$WORK/130.mrw"
m write --no-check "$WORK/130.mrw" >/dev/null 2>&1
want 0 $? "a rename to a trailing-space name applies"
[ -e "$R/d " ] && [ ! -e "$R/d" ] && ok "a rename destination keeps its trailing space" || bad "rename landed: $(ls "$R")"

# 131. ADR-069 T4: apply_patch TrimSpaced the path after its marker, so a
# patch naming "x " edited x. The pair: x, holding the same bytes, is untouched.
fixture
printf 'one\ntwo\n' > "$R/x"
printf 'one\ntwo\n' > "$R/x "
m read -- 'x ' >"$WORK/served.out" 2>&1
printf '*** Begin Patch\n*** Update File: x \n@@\n one\n-two\n+TWO\n*** End Patch\n' > "$WORK/131.patch"
m write --no-check --format=apply_patch "$WORK/131.patch" >/dev/null 2>&1
want 0 $? "an apply_patch update of a trailing-space path applies"
[ "$(cat "$R/x ")" = "$(printf 'one\nTWO')" ] && [ "$(cat "$R/x")" = "$(printf 'one\ntwo')" ] \
	&& ok "an apply_patch path keeps its trailing space" || bad "x-space: $(cat "$R/x "); x: $(cat "$R/x")"

# 132. ADR-070 T1: blind reading 04's agents saw body= described and never on
# a header, and four of seven wrote it on a line of its own. mrw instructions
# carries a worked replace header that counts its body.
fixture
out=$(m instructions 2>&1)
grep -qE '@@ [^ ]+ [0-9]+(-[0-9]+)? replace [^ ].* body=[1-9]' <<<"$out" \
	&& ok "mrw instructions shows body= on a header" || bad "no worked body= header in: $out"

# 133. ADR-070 T2: a hunk with no body= count whose first body line begins
# body= is refused, exit 2 (the plan does not parse), nothing written; reading 04 had it written into
# s2's docs/meta.yaml. The pair: counting the body writes that line.
fixture
printf 'title: x\n' > "$R/meta.yaml"
m read meta.yaml >"$WORK/served.out" 2>&1
printf '@@ meta.yaml 1 replace\nbody=1\ntitle: y\n' > "$WORK/133a.mrw"
out=$(m write --no-check "$WORK/133a.mrw" 2>&1); rc=$?
want 2 "$rc" "a body= line under the header is refused"
[ "$(cat "$R/meta.yaml")" = 'title: x' ] && ok "and nothing was written" || bad "meta.yaml: $(cat "$R/meta.yaml")"
grep -q 'belongs ON' <<<"$out" && ok "and the refusal says body= belongs on the header" || bad "refusal: $out"
printf '@@ meta.yaml 1 replace body=1\nbody=1\n' > "$WORK/133b.mrw"
m write --no-check "$WORK/133b.mrw" >/dev/null 2>&1
want 0 $? "and a counted body writes a body= line as content"
[ "$(cat "$R/meta.yaml")" = 'body=1' ] && ok "and the file holds it" || bad "meta.yaml: $(cat "$R/meta.yaml")"

# 134. ADR-069 T5 (the Codex review of v1.25.0): a "--" consumed as a flag
# value ended the padded-path guard, and urfave trims an attached flag value
# with its token, so `read --grep -- 'x '` and `--files-from='list '` still
# reached the trimmed name. Both are refused, exit 2, and so is a padded
# attached root flag. The pair: the separate spelling serves the path.
fixture
printf 'padded\n' > "$R/x "
m read --grep -- 'x ' >"$WORK/served.out" 2>&1
want 2 $? "a -- consumed as a flag value does not end the guard"
printf 'x \n' > "$WORK/134 list "
m read --files-from="$WORK/134 list " >"$WORK/served.out" 2>&1
want 2 $? "an attached flag value ending in a space is refused"
"$MRW" --root="$R " read -- 'x ' >"$WORK/served.out" 2>&1
want 2 $? "and so is a padded attached root flag"
out=$(m read --files-from "$WORK/134 list " 2>&1); rc=$?
want 0 "$rc" "and the separate spelling is served"
grep -q 'padded' <<<"$out" && ok "and it serves the padded path" || bad "served: $out"

# 135. ADR-069 T6 (the Codex review of PR #222): the guard looked a flag name
# up as typed, so a padded boolean name ('--no-numbers ') read as value-taking
# and hid the padded path after it; the whole-argv check read a separate value
# as a flag and stopped at a "--" a root flag had consumed; and an attached
# value was checked for space and tab while the parser trims every whitespace.
# Each is paired with the spelling the parser keeps.
fixture
printf 'plain\n' > "$R/x"; printf 'padded\n' > "$R/x "
m read '--no-numbers ' 'x ' >"$WORK/served.out" 2>&1
want 2 $? "a padded boolean flag name does not hide a padded path"
out=$(m read --no-numbers x 2>&1); rc=$?
want 0 "$rc" "and the trimmed spelling serves x"
printf 'x \n' > "$R/--list= "
out=$(cd "$R" && "$MRW" -C "$R" read --files-from '--list= ' 2>&1); rc=$?
want 0 "$rc" "a separate value that looks like an attached flag is served"
grep -q 'padded' <<<"$out" && ok "and it reaches the padded path" || bad "served: $out"
"$MRW" --root -- --root="$R " read x >"$WORK/served.out" 2>&1
want 2 $? "a -- consumed by a root flag does not end the whole-argv guard"
printf 'x \n' > "$WORK/135 list"$'\n'
out=$(m read --files-from="$WORK/135 list"$'\n' 2>&1); rc=$?
want 2 "$rc" "an attached value ending in a newline is refused"
grep -q 'own argument' <<<"$out" && ok "and the newline refusal names the separate spelling" || bad "refusal: $out"
out=$(m read --files-from "$WORK/135 list"$'\n' 2>&1); rc=$?
want 0 "$rc" "and the separate spelling serves the newline-named list"

# 136. ADR-069 T7 (the Codex review of PR #222, second round): a parent's
# persistent flag is accepted by a subcommand that has no flag of the same
# name (--root after write or iter; not after read, whose -C is context), and
# neither guard knew it; and a single dash before a non-letter stops the
# parser, which keeps the rest as given, while the guard read it as a flag.
fixture
printf 'padded\n' > "$R/x "
mkdir "$R/d "; printf 'in d\n' > "$R/d /x"; printf '@@ x 1 replace\nnew\n' > "$WORK/136.mrw"
out=$(m write --no-check --root -- --root="$R " "$WORK/136.mrw" 2>&1); rc=$?
want 2 "$rc" "an inherited root flag after the verb is read by the whole-argv guard"
grep -q 'own argument' <<<"$out" && ok "and its refusal names the separate spelling" || bad "refusal: $out"
"$MRW" -C "$R/d " read x >"$WORK/served.out"
"$MRW" --root -- --root "$R/d " write --no-check --dry-run "$WORK/136.mrw" >/dev/null 2>&1
want 0 $? "and a -- the root flag consumed, then a separate padded root, is accepted"
mkdir "$R/ x"; printf 'q\n' > "$R/ x/x"
(cd "$R" && "$MRW" -C "$R" iter --root ' x' add x) >/dev/null 2>&1
want 0 $? "an inherited root value equal to a positional after its trim is not refused"
(cd "$R" && "$MRW" -C "$R" iter --root ' x' add 'x ') >/dev/null 2>&1
want 2 $? "and a padded positional after it still is"
printf 'dashfile\n' > "$R/ -1= "
out=$(cd "$R" && "$MRW" -C "$R" read ' -1= ' 2>&1); rc=$?
want 0 "$rc" "a single dash before a non-letter stops the parser and the guard"
grep -q 'dashfile' <<<"$out" && ok "and the path is served as given" || bad "served: $out"

# 137. ADR-069 T8 (the Codex review of PR #222, third round): a "--" before
# the subcommand ends the ROOT's options only — the parser still dispatches
# the subcommand, which parses its own flags — and both guards went quiet
# there, so `-- iter note --root='dir '` and `-- stats --root='dir '` reached
# the trimmed root. And a lone "-" ends the parse (the parser keeps it and
# drops the rest) while the guard read on and refused what the parser never
# saw. Each is paired with the shape the parser keeps.
fixture
mkdir "$R/d "; printf 'in d\n' > "$R/d /x"
out=$("$MRW" -C "$R" -- iter note --root="$R/d " revised 2>&1); rc=$?
want 2 "$rc" "a -- before the subcommand does not end the guard for its flags"
grep -q 'own argument' <<<"$out" && ok "and that refusal names the separate spelling" || bad "refusal: $out"
"$MRW" -C "$R" -- iter note --root "$R/d " revised >/dev/null 2>&1
want 0 $? "and the separate spelling is accepted after the --"
"$MRW" -C "$R" -- stats --root="$R/d " >/dev/null 2>&1
want 2 $? "a subcommand with no guard of its own is covered by the whole-argv guard"
"$MRW" -C "$R" -- stats --root "$R/d " >/dev/null 2>&1
want 0 $? "and its separate spelling is accepted"
printf '@@ a.go 1 replace\npackage demo\n' | m write --no-check --dry-run - '--format=plan ' >/dev/null 2>&1
want 0 "${PIPESTATUS[1]}" "a lone - ends the parse and the guard: a token the parser drops is not refused"
printf '@@ a.go 1 replace\npackage demo\n' | m write --no-check --dry-run '--format=plan ' - >/dev/null 2>&1
want 2 "${PIPESTATUS[1]}" "and the same token before the - is refused"

# 138. ADR-069 T9 (the Codex review of PR #222, fourth round): the parser
# trims a lone "-" and KEEPS it as the positional that ends its parse, so
# `write ' - '` read stdin once T8 stopped the guard there. The token is
# judged first; after a "--" it is kept as given, and a bare "-" is stdin.
fixture
out=$(printf '@@ a.go 1 replace\npackage demo\n' | m write --no-check --dry-run ' - ' 2>&1); rc=$?
want 2 "$rc" "a padded lone - is refused before the guard stops"
grep -q 'edge whitespace' <<<"$out" && ok "and the refusal names the --" || bad "refusal: $out"
printf '@@ a.go 1 replace\npackage demo\n' | m write --no-check --dry-run -- - >/dev/null 2>&1
want 0 "${PIPESTATUS[1]}" "and a bare - after -- is stdin"

# 139. ADR-069 T10 (the Codex review of PR #222, fifth round): a single dash
# before a non-letter stops the parser, which keeps that token and the rest
# as given; T9 judged it before stopping, so a padded name beside its trimmed
# twin was refused. Only the lone "-", which the parser trims and keeps, is
# judged before the stop.
fixture
printf 'padded\n' > "$R/ -1= "; printf 'plain\n' > "$R/-1="
out=$(cd "$R" && "$MRW" -C "$R" read ' -1= ' '-1=' 2>&1); rc=$?
want 0 "$rc" "a preserved stop token is not judged against a sibling"
grep -q 'padded' <<<"$out" && grep -q 'plain' <<<"$out" && ok "and both names are served as given" || bad "served: $out"
m read ' - ' a.go >"$WORK/served.out" 2>&1
want 2 $? "and the padded lone - is still refused"

# 140. ADR-069 T12: `mrw instructions` teaches the padded-path rule. A caller
# with only the binary learned it at the refusal (the v1.25.1 field test of
# the downloaded Windows asset found the gap). The pair: the same text must
# not teach the trim as the rule.
out=$("$MRW" instructions 2>&1); rc=$?
want 0 "$rc" "mrw instructions exits 0"
grep -q "goes after --" <<<"$out" && grep -q "mrw read -- 'x '" <<<"$out" && ok "instructions teach that a padded path goes after --" || bad "instructions: no -- rule"
grep -q "as its own argument" <<<"$out" && ok "and that an attached padded value is passed separately" || bad "instructions: no attached-value rule"
grep -q "would reach x, so mrw reads x" <<<"$out" && bad "instructions teach the trim as the rule" || ok "and they do not teach the trim as the rule"
# 141. ADR-071 T2: two creates that could be one file are refused before
# anything is written. The v1.25.1 round created n.txt and N.TXT in one plan
# on APFS and NTFS: both said "created", one file remained and the first body
# was lost, exit 0; and one path created twice got both bodies. The pair: two
# creates of different names still land.
fixture
printf '@@ c1.txt - create\none\n@@ c2.txt - create\ntwo\n' > "$R/p141a.mrw"
m write --no-check "$R/p141a.mrw" >/dev/null 2>&1; rc=$?
want 0 "$rc" "two creates of different names apply"
[ -f "$R/c1.txt" ] && [ -f "$R/c2.txt" ] && ok "and both files are on disk" || bad "a create of a different name was lost"
printf '@@ n.txt - create\none\n@@ N.TXT - create\ntwo\n' > "$R/p141b.mrw"
out=$(m write --no-check "$R/p141b.mrw" 2>&1); rc=$?
want 1 "$rc" "n.txt and N.TXT created in one plan are refused"
grep -q 'N.TXT may name the same file as n.txt' <<<"$out" && ok "and the refusal names both spellings" || bad "refusal: $out"
[ ! -e "$R/n.txt" ] && [ ! -e "$R/N.TXT" ] && ok "and neither was written" || bad "a refused plan created a file"
printf '@@ d.txt - create\none\n@@ d.txt - create\ntwo\n' > "$R/p141c.mrw"
out=$(m write --no-check "$R/p141c.mrw" 2>&1); rc=$?
want 1 "$rc" "one path created twice is refused"
grep -q 'd.txt is created twice in this plan (plan lines 1 and 3)' <<<"$out" && ok "and the refusal names both plan lines" || bad "refusal: $out"
[ ! -e "$R/d.txt" ] && ok "and nothing was written" || bad "a path created twice was written: $(cat "$R/d.txt")"
# The folds no name comparison can see (ß and ss on APFS): the commit stops at
# the second create instead of renaming over the first — PARTIALLY APPLIED,
# exit 2, the first body kept. Skipped where the filesystem keeps them apart.
fixture
printf 'x' > "$R/$(printf '\303\237').probe"
if [ -e "$R/ss.probe" ]; then
  printf '@@ \303\237.txt - create\none\n@@ ss.txt - create\ntwo\n' > "$R/p141d.mrw"
  out=$(m write --no-check "$R/p141d.mrw" 2>&1); rc=$?
  want 2 "$rc" "a create the filesystem folds into an earlier one stops the commit"
  grep -q 'appeared before commit' <<<"$out" && grep -q 'PARTIALLY APPLIED' <<<"$out" && ok "and says so, naming what landed" || bad "fold at commit: $out"
  [ "$(cat "$R/ss.txt")" = "one" ] && ok "and the first body survives" || bad "the first body was lost: $(cat "$R/ss.txt")"
else
  skip "this filesystem keeps ß and ss apart"
fi
# 142. ADR-072 T1: a malformed .quality-harness.json refuses the write before
# anything is written. It was read after the commit: the tree changed and mrw
# exited 2 with only the JSON error. The pair: --no-check never reads it.
fixture
printf '{' > "$R/.quality-harness.json"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 9 }\n' > "$R/p142.mrw"
before=$(cat "$R/a.go")
out=$(m write "$R/p142.mrw" 2>&1); rc=$?
want 2 "$rc" "a malformed harness refuses the write"
grep -q 'quality-harness.json' <<<"$out" && grep -q 'nothing was written' <<<"$out" && ok "and names the harness and says nothing was written" || bad "refusal: $out"
[ "$(cat "$R/a.go")" = "$before" ] && ok "and a.go is untouched" || bad "a.go changed under a refused write"
m write --no-check "$R/p142.mrw" >/dev/null 2>&1; rc=$?
want 0 "$rc" "--no-check never reads the harness and applies"
# 143. ADR-072 T2: a write killed while its check runs has already printed its
# receipt, and stats counts the landing; both used to come after the check.
# The check records its pid and sleeps; the row kills mrw with SIGKILL once the
# check has started — a Go binary ignores SIGALRM, so an alarm would not — and
# then kills the check, which runs in a process group of its own.
fixture
printf '{"check":"echo $$ > gc.pid; exec sleep 30"}' > "$R/.quality-harness.json"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 7 }\n' > "$R/p143.mrw"
"$MRW" -C "$R" write "$R/p143.mrw" > "$R/out143" 2>&1 & pid=$!
for i in $(seq 1 50); do [ -s "$R/gc.pid" ] && break; sleep 0.1; done
kill -9 "$pid"; wait "$pid"; rc=$?
[ -f "$R/gc.pid" ] && kill -9 "$(cat "$R/gc.pid")" 2>/dev/null
want 137 "$rc" "mrw was killed during its check"
grep -q -- '— applied' "$R/out143" && ok "and its receipt was already printed" || bad "a write killed during its check printed: $(cat "$R/out143")"
out=$(m stats 2>&1); grep -q 'landed writes: 1;' <<<"$out" && ok "and stats counts the landing" || bad "stats: $out"
# 144. ADR-072 T3: under --json a plan that does not parse is one JSON
# document with an error field, not text. The pair: a filesystem failure
# already was one (§21).
fixture
out=$(printf 'garbage\n' | m write --json - 2>/dev/null); rc=$?
want 2 "$rc" "an unparseable plan under --json exits 2"
jq -e '.applied == false and (.error | test("stdin")) and .hunks == [] and .files == []' <<<"$out" >/dev/null && ok "and stdout is one JSON document naming the cause" || bad "--json printed: $out"
# 145. ADR-072 T4: a check that times out takes its process group with it, and
# an interrupt sent to mrw while its check runs stops the check and says so.
# The timeout used to kill only sh, and sleep outlived it.
fixture
printf '{"check":"sleep 30 & echo $! > gc.pid; wait","timeout_seconds":1}' > "$R/.quality-harness.json"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 5 }\n' > "$R/p145.mrw"
out=$(m write "$R/p145.mrw" 2>&1); rc=$?
want 3 "$rc" "a check that times out is exit 3"
grep -q 'timed out' <<<"$out" && ok "and says it timed out" || bad "timeout: $out"
gone=0; for i in 1 2 3 4 5 6 7 8 9 10; do kill -0 "$(cat "$R/gc.pid")" 2>/dev/null || { gone=1; break; }; sleep 0.3; done
[ "$gone" = 1 ] && ok "and the check's grandchild is gone" || { kill -9 "$(cat "$R/gc.pid")" 2>/dev/null; bad "the check's grandchild outlived its timeout"; }
fixture
printf '{"check":"echo $$ > gc.pid; exec sleep 30"}' > "$R/.quality-harness.json"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 6 }\n' > "$R/p145b.mrw"
"$MRW" -C "$R" write "$R/p145b.mrw" > "$R/out145" 2>&1 & pid=$!
for i in $(seq 1 50); do [ -s "$R/gc.pid" ] && break; sleep 0.1; done
kill -TERM "$pid"; wait "$pid"; rc=$?
want 3 "$rc" "a terminate during the check is exit 3"
grep -q 'interrupted' "$R/out145" && ok "and the receipt says interrupted" || bad "terminate: $(cat "$R/out145")"
gone=0; for i in 1 2 3 4 5 6 7 8 9 10; do kill -0 "$(cat "$R/gc.pid")" 2>/dev/null || { gone=1; break; }; sleep 0.3; done
[ "$gone" = 1 ] && ok "and the check is gone" || { kill -9 "$(cat "$R/gc.pid")" 2>/dev/null; bad "the check outlived the interrupt"; }
# A hangup too: it ends mrw, and the check in its own group would not hear it.
# (TERM and HUP, not INT: a shell starts a background job with SIGINT ignored,
# and mrw leaves an ignored signal ignored.)
# perl sets SIGHUP back to default first: under `nohup ./scripts/contract.sh`
# mrw would inherit it ignored, rightly keep ignoring it, and this row would
# read that as a failure (second review of #229).
fixture
printf '{"check":"echo $$ > gc.pid; exec sleep 30"}' > "$R/.quality-harness.json"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 4 }\n' > "$R/p145c.mrw"
perl -e '$SIG{HUP}="DEFAULT"; exec @ARGV' "$MRW" -C "$R" write "$R/p145c.mrw" > "$R/out145c" 2>&1 & pid=$!
for i in $(seq 1 50); do [ -s "$R/gc.pid" ] && break; sleep 0.1; done
kill -HUP "$pid"; wait "$pid"; rc=$?
want 3 "$rc" "a hangup during the check is exit 3"
gone=0; for i in 1 2 3 4 5 6 7 8 9 10; do kill -0 "$(cat "$R/gc.pid")" 2>/dev/null || { gone=1; break; }; sleep 0.3; done
[ "$gone" = 1 ] && ok "and the check is gone" || { kill -9 "$(cat "$R/gc.pid")" 2>/dev/null; bad "the check outlived the hangup"; }
# The pair: under nohup a hangup is ignored, as it always was, and the check
# runs on — mrw leaves a signal it was started with ignored alone.
fixture
printf '{"check":"echo $$ > gc.pid; exec sleep 30"}' > "$R/.quality-harness.json"
printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 3 }\n' > "$R/p145d.mrw"
nohup "$MRW" -C "$R" write "$R/p145d.mrw" > "$R/out145d" 2>&1 & pid=$!
for i in $(seq 1 50); do [ -s "$R/gc.pid" ] && break; sleep 0.1; done
kill -HUP "$pid"; sleep 0.5
kill -0 "$(cat "$R/gc.pid")" 2>/dev/null && ok "under nohup a hangup leaves the check running" || bad "under nohup a hangup stopped the check"
kill -9 "$(cat "$R/gc.pid")" 2>/dev/null; wait "$pid"

# 146. ADR-073: a line edit to a UTF-16 file is refused and its bytes are
# unchanged; the v1.25.1 round rewrote one at exit 0 with the BOM gone. read
# serves it with a note. The pairs: the same edit to a UTF-8 file applies, and
# an unlink of the UTF-16 file applies.
fixture
printf '\xff\xfea\x00\n\x00b\x00\n\x00' > "$R/u16.txt"; cp "$R/u16.txt" "$R/u16.bak"
out=$(m read u16.txt 2>&1); rc=$?
want 0 "$rc" "a UTF-16 file is read"
grep -q -- '-- note: u16.txt is UTF-16 (BOM FF FE)' <<<"$out" && ok "with a note naming its encoding" || bad "read: $out"
printf '@@ u16.txt 1 replace\nX\n' > "$R/p146.mrw"
out=$(m write --no-check "$R/p146.mrw" 2>&1); rc=$?
want 1 "$rc" "a line edit to it is refused"
grep -q 'UTF-16 (BOM FF FE)' <<<"$out" && ok "and the refusal names the encoding" || bad "refusal: $out"
cmp -s "$R/u16.txt" "$R/u16.bak" && ok "and its bytes are unchanged" || bad "u16.txt changed under a refused write"
printf 'one\ntwo\n' > "$R/a8.txt"; m read a8.txt >"$WORK/served.out"
printf '@@ a8.txt 1 replace\nX\n' > "$R/p146b.mrw"
m write --no-check "$R/p146b.mrw" >/dev/null 2>&1; rc=$?
want 0 "$rc" "the same edit to a UTF-8 file applies"
printf '@@ u16.txt - unlink\n' > "$R/p146c.mrw"
m write --no-check "$R/p146c.mrw" >/dev/null 2>&1; rc=$?
want 0 "$rc" "and an unlink of the UTF-16 file applies"
[ ! -e "$R/u16.txt" ] && ok "and removes it" || bad "u16.txt is still there"
# --force overrides the read ledger, not the file's encoding; and a NUL in the
# first 8 KiB refuses as a byte-order mark does (review of #230).
cp "$R/u16.bak" "$R/u16.txt"; m read u16.txt >"$WORK/served.out"
m write --no-check --force "$R/p146.mrw" >/dev/null 2>&1; rc=$?
want 1 "$rc" "--force does not bypass the encoding refusal"
cmp -s "$R/u16.txt" "$R/u16.bak" && ok "and the bytes are unchanged" || bad "u16.txt changed under --force"
printf 'a\000b\nc\n' > "$R/nul.txt"; m read nul.txt >"$WORK/served.out"
printf '@@ nul.txt 2 replace\nX\n' > "$R/p146d.mrw"
out=$(m write --no-check "$R/p146d.mrw" 2>&1); rc=$?
want 1 "$rc" "a line edit to a file with a NUL in its first 8 KiB is refused"
grep -q 'holds a NUL byte at offset 1' <<<"$out" && ok "and the refusal names the offset" || bad "nul: $out"
# A FIFO named in a native plan or an apply_patch document is refused before it
# is opened. Bounded by polling and a kill, not an alarm: Go ignores SIGALRM.
mkfifo "$R/p146"
printf '@@ p146 1 replace\nX\n' > "$R/p146e.mrw"
printf '*** Begin Patch\n*** Update File: p146\n@@\n-x\n+y\n*** End Patch\n' > "$R/p146f.patch"
for how in plan apply_patch; do
	src="$R/p146e.mrw"; [ "$how" = apply_patch ] && src="$R/p146f.patch"
	"$MRW" -C "$R" write --no-check --format="$how" "$src" > "$R/out146" 2>&1 & pid=$!
	done146=0; for i in $(seq 1 30); do kill -0 "$pid" 2>/dev/null || { done146=1; break; }; sleep 0.1; done
	if [ "$done146" = 1 ]; then
		wait "$pid"; rc=$?
		want146=1; [ "$how" = apply_patch ] && want146=2
		[ "$rc" = "$want146" ] && grep -q 'not a regular file' "$R/out146" && ok "a FIFO named in the $how write is refused by name, exit $want146" || bad "$how fifo: rc $rc (want $want146) $(cat "$R/out146")"
	else
		kill -9 "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
		bad "a FIFO in a $how write blocked for 3 s"
	fi
done
# 147. ADR-074 T1: an ast-grep behind a wrapper whose grandchild holds its
# stdout returns at the 2 s bound, and the grandchild dies with it. It used to
# wait out the grandchild's 30 s. §111 is the pair: a hang with no grandchild.
# No perl alarm: Go ignores SIGALRM, and the wrapper's own sleep bounds a
# regression at 30 s.
fixture
d147=$(mktemp -d)
printf '%s\n' '#!/bin/sh' 'sleep 30 &' "echo \$! > '$d147/gc.pid'" 'wait' > "$d147/ast-grep"
chmod +x "$d147/ast-grep"
t0=$(date +%s)
out=$(PATH="$d147:$PATH" "$MRW" -C "$R" read --ast-grep zzz 2>&1); rc=$?
t1=$(date +%s)
want 2 "$rc" "a forking ast-grep that hangs is usage"
grep -q 'timed out' <<<"$out" && ok "and the reason says timed out" || bad "forking timeout: $out"
dur=$((t1 - t0))
[ "$dur" -le 4 ] && ok "and returns within 4 s though a grandchild holds its stdout" || bad "a held stdout kept the read ${dur}s"
gc=$(cat "$d147/gc.pid" 2>/dev/null)
if [ -n "$gc" ]; then
	gone=0; for i in 1 2 3 4 5 6 7 8 9 10; do kill -0 "$gc" 2>/dev/null || { gone=1; break; }; sleep 0.3; done
	[ "$gone" = 1 ] && ok "and the grandchild is gone" || { kill -9 "$gc" 2>/dev/null; bad "ast-grep's grandchild outlived the bound"; }
else
	bad "the forking wrapper never recorded its grandchild"
fi

# 148. ADR-074 T2: a FIFO named to `read` is reported by name, and the file
# named beside it is served. It used to block until something wrote to the pipe.
fixture
mkfifo "$R/p148"
"$MRW" -C "$R" read p148 a.go > "$WORK/out148" 2>&1 & pid=$!
done148=0; for i in $(seq 1 30); do kill -0 "$pid" 2>/dev/null || { done148=1; break; }; sleep 0.1; done
if [ "$done148" = 1 ]; then
	wait "$pid"; rc=$?
	want 1 "$rc" "a FIFO spec is a problem, exit 1"
	grep -q 'p148  UNREADABLE  not a regular file' "$WORK/out148" && ok "and it is named as not a regular file" || bad "fifo: $(cat "$WORK/out148")"
	grep -q '^==> a.go ' "$WORK/out148" && ok "and the file named beside it is served" || bad "fifo sibling: $(cat "$WORK/out148")"
else
	kill -9 "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
	bad "read blocked on a FIFO for 3 s"
fi

# 149. ADR-074 T3: over MCP, a markup file whose JSON-escaped text overflows the
# ceiling is served as a first page with next_read, not refused whole. The page
# was sized from raw bytes while the ceiling measures the encoded answer.
fixture
python3 -c "import sys; sys.stdout.write('<div class=\"a\">&amp;</div>\n' * 5688)" > "$R/m.tsx"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["m.tsx"]}}}' | m mcp > "$R/out149" 2>/dev/null
# The result is bounded at 200,000 encoded bytes; the JSON-RPC envelope around
# it adds under a hundred. Read from a file, not argv: Linux caps one argument
# at 128 KiB, and this answer is near 200,000 bytes (CI on #232).
python3 - "$R/out149" <<'PY' && ok "a markup file over MCP is a first page with next_read, within the ceiling" || bad "markup page: $(head -c 300 "$R/out149")"
import json, sys
line = open(sys.argv[1]).read().strip().splitlines()[-1]
res = json.loads(line)["result"]
assert not res.get("isError"), "refused"
assert "-- PARTIAL:" in res["content"][0]["text"], "not a page"
assert "next_read" in json.dumps(res), "no next_read"
assert len(line) <= 200100, "over the ceiling"
PY
# The pair: a CLOSED range over the same file is not paged (ADR-014), so it is
# refused, and the refusal names the encoding, not the per-file receipt.
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["m.tsx:1-5000"]}}}' | m mcp > "$R/out149b" 2>/dev/null
python3 - "$R/out149b" <<'PY' && ok "a closed markup range over the ceiling is refused, naming the encoding" || bad "closed markup range: $(head -c 300 "$R/out149b")"
import json, sys
res = json.loads(open(sys.argv[1]).read().strip().splitlines()[-1])["result"]
assert res.get("isError"), "served"
t = res["content"][0]["text"]
assert "encoded" in t and "per-file receipt" not in t, t
PY

# 150. ADR-075: eight writers off one read, each replacing a different line of
# one file. Every writer that exits 0 has its edit in the file, and every other
# is refused as stale, exit 1. Before the write lock a later rename discarded
# earlier edits while every writer printed "applied" (the v1.25.1 round: 45-53%
# lost). The pair is every single-writer row above: one writer lands.
fixture
seq -f 'line %g' 1 16 > "$R/f150.txt"
m read f150.txt >"$WORK/served.out"
for j in 0 1 2 3 4 5 6 7; do
	printf '@@ f150.txt %d replace\nwriter %d\n' $((j + 1)) "$j" | m write --no-check - > "$R/out150.$j" 2>&1 &
	pids150[$j]=$!
done
for j in 0 1 2 3 4 5 6 7; do
	wait "${pids150[$j]}"; rcs150[$j]=$?
done
lost=0; odd=0
for j in 0 1 2 3 4 5 6 7; do
	line=$(sed -n "$((j + 1))p" "$R/f150.txt")
	case "${rcs150[$j]}" in
		0) [ "$line" = "writer $j" ] || lost=$((lost + 1)) ;;
		1) grep -q 'changed since' "$R/out150.$j" || odd=$((odd + 1)) ;;
		*) odd=$((odd + 1)) ;;
	esac
done
[ "$lost" = 0 ] && ok "no writer exits 0 and loses its edit" || bad "$lost writer(s) exited 0 and lost their edit"
[ "$odd" = 0 ] && ok "and every other writer is refused as stale" || bad "$odd writer(s) ended oddly: $(cat "$R"/out150.*)"

# 151. ADR-076 T1: a path that ends in a separator names a directory. `a.go/`
# was cleaned to a.go — a read served it and a plan edited it — and a rename
# to `d/` made a FILE named d. A root that does not exist was reported as a
# path "outside the root", and a create under it made the root. Each is
# refused and named; a directory keeps its slash.
fixture
mkdir -p "$R/sub"; printf 'package sub\n' > "$R/sub/s.go"
out=$(m read a.go/ 2>&1); rc=$?
want 1 "$rc" "a read of a file spelled with a trailing slash is a problem"
grep -q 'a.go/  UNREADABLE  a.go/ names a directory' <<<"$out" && ok "and it is named, not served" || bad "trailing-slash read: $out"
grep -q '| package' <<<"$out" && bad "a line was served through a directory spelling: $out" || ok "and no line of it is served"
out=$(m read a.go/. 2>&1); rc=$?
{ [ "$rc" = 1 ] && grep -q 'names a directory' <<<"$out"; } && ok "and so is a.go/., which the OS refuses as it refuses a.go/" || bad "a.go/. read: exit $rc: $out"
m read --grep package sub/ >"$WORK/served.out" 2>&1; want 0 $? "a directory named with its slash is still walked"
m read a.go b.go >"$WORK/served.out"
printf '@@ a.go/ 3 replace\nfunc A() int { return 9 }\n' > "$R/p151a.mrw"
before=$(cat "$R/a.go"); out=$(m write --no-check "$R/p151a.mrw" 2>&1); rc=$?
want 1 "$rc" "a plan path with a trailing slash is refused"
[ "$(cat "$R/a.go")" = "$before" ] && ok "and a.go is unchanged" || bad "a.go/ edited a.go: $out"
printf '@@ b.go - rename\nd/\n' > "$R/p151b.mrw"
out=$(m write --no-check "$R/p151b.mrw" 2>&1); rc=$?
want 1 "$rc" "a rename to d/ is refused"
[ ! -e "$R/d" ] && ok "and no file d is made" || bad "a rename to d/ made $(ls -ld "$R/d")"
grep -q 'd/b.go' <<<"$out" && ok "and the refusal names the file to write instead" || bad "rename d/: $out"
printf '@@ b.go - rename\nd/b.go\n' > "$R/p151c.mrw"
m write --no-check "$R/p151c.mrw" >/dev/null 2>&1; want 0 $? "a rename that names the file lands"
[ -f "$R/d/b.go" ] && ok "at d/b.go" || bad "rename to d/b.go: $(ls -R "$R" 2>&1 | head -20)"
out=$("$MRW" -C "$WORK/no-such-root" read a.go 2>&1); rc=$?
want 1 "$rc" "a read under a root that does not exist is a problem"
{ grep -q 'does not exist' <<<"$out" && ! grep -q 'outside the root' <<<"$out"; } \
  && ok "and the root is named as missing, not the path as outside it" || bad "missing root: $out"
printf '@@ x.txt 0 create\nx\n' > "$WORK/p151d.mrw"
out=$("$MRW" -C "$WORK/no-such-root" write --no-check "$WORK/p151d.mrw" 2>&1); rc=$?
{ [ "$rc" -ne 0 ] && [ ! -e "$WORK/no-such-root" ]; } \
  && ok "and a create under it is refused without making the root" || bad "a create under a missing root: exit $rc, $(ls -ld "$WORK/no-such-root" 2>&1): $out"

# 152. ADR-076 T4 and T5: the receipt names every path a write touched, and an
# insert into an empty file ends with a newline. A write through an in-root
# symlink named only the link; the directories a create or a rename made were
# not named; a removed path printed `sha ` and nothing after it.
fixture
printf 'r\n' > "$R/real.txt"; ln -s real.txt "$R/link.txt"; printf 'm\n' > "$R/mv.txt"
m read link.txt mv.txt >"$WORK/served.out"
printf '@@ link.txt 1 replace\nR\n@@ n/deep/c.txt 0 create\nc\n@@ mv.txt - rename\nm2/mv.txt\n' > "$R/p152.mrw"
out=$(m write --no-check --json "$R/p152.mrw" 2>&1); rc=$?
want 0 "$rc" "a plan through a link, into new directories, with a rename applies"
jq -e '.files[] | select(.path=="link.txt") | .target=="real.txt"' <<<"$out" >/dev/null && ok "the receipt names the link's target" || bad "no target: $out"
jq -e '[.files[] | select(.path=="mv.txt" or .path=="m2/mv.txt") | has("target")] | any | not' <<<"$out" >/dev/null && ok "and a file reached by its own name carries none" || bad "a plain file named a target: $out"
jq -e '.dirs_created==["m2","n","n/deep"]' <<<"$out" >/dev/null && ok "and the directories the plan made, parents first" || bad "dirs_created: $(jq -c .dirs_created <<<"$out" 2>&1)"
{ [ "$(cat "$R/real.txt")" = R ] && [ -L "$R/link.txt" ]; } && ok "and the write reached the target through the link it kept" || bad "link write: $(ls -l "$R")"
fixture
printf 'g\n' > "$R/gone.txt"; : > "$R/e.txt"; printf 'a' > "$R/nn.txt"
m read gone.txt e.txt nn.txt >"$WORK/served.out"
printf '@@ gone.txt - unlink\n@@ e.txt 0 insert-after\nx\n@@ nn.txt 1 replace\nb\n@@ n/c.txt 0 create\nc\n' > "$R/p152b.mrw"
out=$(m write --no-check "$R/p152b.mrw" 2>&1); rc=$?
want 0 "$rc" "an unlink, an insert into an empty file, an edit and a create apply"
grep -qE '^removed gone.txt  1L -> 0L  was sha [0-9a-f]{8}$' <<<"$out" && ok "a removed file's line says what it was" || bad "removed line: $out"
grep -qE 'sha $' <<<"$out" && bad "a receipt line ends in an empty sha: $out" || ok "and no line ends in an empty sha"
grep -q '^created n/$' <<<"$out" && ok "the human receipt names the directory made" || bad "no created n/: $out"
[ "$(od -An -c "$R/e.txt" | tr -d ' ')" = 'x\n' ] && ok "an insert into an empty file ends with a newline" || bad "e.txt is $(od -c "$R/e.txt")"
[ "$(od -An -c "$R/nn.txt" | tr -d ' ')" = 'b' ] && ok "and an edit of a file with no final newline adds none" || bad "nn.txt is $(od -c "$R/nn.txt")"

# 153. ADR-076 T6: a read-only file is refused, not replaced or removed. A
# replace renames a new file over it and an unlink removes the entry, and
# neither needs the file to be writable, so both applied at exit 0. The check
# reads the mode bits, so it holds under uid 0 as well.
fixture
printf 'a\n' > "$R/ro.txt"; chmod 444 "$R/ro.txt"; m read ro.txt >"$WORK/served.out"
for plan in '@@ ro.txt 1 replace\nX\n' '@@ ro.txt - unlink\n' '@@ ro.txt - rename\nmoved.txt\n'; do
  printf "$plan" > "$R/p153.mrw"
  out=$(m write --no-check "$R/p153.mrw" 2>&1); rc=$?
  { [ "$rc" = 1 ] && grep -q 'ro.txt is read-only' <<<"$out" && grep -q 'chmod u+w ro.txt' <<<"$out"; } \
    && ok "$(head -1 "$R/p153.mrw" | cut -d' ' -f3-) on a read-only file is refused, naming chmod" || bad "read-only $(head -1 "$R/p153.mrw"): exit $rc: $out"
done
{ [ "$(cat "$R/ro.txt")" = a ] && [ ! -e "$R/moved.txt" ]; } && ok "and ro.txt is as it was" || bad "ro.txt changed: $(ls -l "$R")"
chmod 644 "$R/ro.txt"
printf '@@ ro.txt 1 replace\nX\n' > "$R/p153.mrw"
m write --no-check "$R/p153.mrw" >/dev/null 2>&1; want 0 $? "once writable, the same replace applies"

# 154. ADR-077: mrw's own state is never served as the caller's file. With
# XDG_STATE_HOME inside the root, a --grep walked the state directory, a read
# served the ledger and the ack store (whose checkpoint ids could then be acked
# without the lines ever being read, ADR-031), and a plan could edit the
# ledger. Each is refused; a file beside mrw/ is still the caller's.
fixture
st154() { XDG_STATE_HOME="$R/.st" "$MRW" -C "$R" "$@"; }
st154 read a.go >"$WORK/served.out"
led=$(cd "$R" && ls .st/mrw/*/seen 2>/dev/null | head -1)
[ -n "$led" ] && ok "the ledger lives inside the root for this row" || bad "no ledger under $R/.st: $(ls -R "$R/.st" 2>&1 | head)"
printf 'func A planted\n' > "$R/.st/mrw/planted.txt"; printf 'func A notes\n' > "$R/.st/notes.txt"
out=$(st154 read --grep 'func A' 2>&1); rc=$?
want 0 "$rc" "a grep over a root that holds mrw's state answers"
grep -q '\.st/mrw' <<<"$out" && bad "the grep served mrw's own state: $out" || ok "and serves nothing of mrw's state"
grep -q '^==> .st/notes.txt' <<<"$out" && ok "while a file beside mrw/ is served" || bad "the sibling was not served: $out"
out=$(st154 read "$led" 2>&1); rc=$?
want 1 "$rc" "a read of the ledger is refused"
grep -q "own state" <<<"$out" && ok "and names mrw's own state" || bad "ledger read: $out"
before=$(cat "$R/$led")
printf '@@ %s 1 replace\nX\n' "$led" > "$R/p154.mrw"
out=$(st154 write --no-check --force "$R/p154.mrw" 2>&1); rc=$?
want 1 "$rc" "a plan editing the ledger is refused"
grep -q "own state" <<<"$out" && ok "refused as mrw's own state, even past the read ledger (--force)" || bad "the ledger write was refused for another reason: $out"
[ "$(cat "$R/$led")" = "$before" ] && ok "and the ledger is unchanged" || bad "the ledger changed: $out"

# 155. ADR-078 T1: a usage error writes nothing to stdout. urfave/cli printed a
# command's whole help to stdout on a flag it rejected, and stdout is `mrw mcp`'s
# protocol stream; `mrw mcp stray` started a server past its argument. --help
# still prints to stdout.
fixture
for argv in 'mcp --bogus' '--rot . mcp' 'read --bogus' 'mcp stray'; do
  "$MRW" -C "$R" $argv </dev/null >"$WORK/o155" 2>"$WORK/e155"; rc=$?
  { [ "$rc" = 2 ] && [ ! -s "$WORK/o155" ] && grep -q -- '--help\|no arguments' "$WORK/e155"; } \
    && ok "mrw $argv: exit 2, nothing on stdout, the reason on stderr" || bad "mrw $argv: exit $rc, stdout: $(head -3 "$WORK/o155")"
done
"$MRW" mcp --help >"$WORK/o155" 2>&1; rc=$?
{ [ "$rc" = 0 ] && grep -q USAGE "$WORK/o155"; } && ok "mrw mcp --help still prints its help" || bad "mcp --help: exit $rc"

# 156. ADR-078 T2: the MCP server refuses what it cannot represent. An id that
# is neither a string nor an integer was dispatched and echoed back; invalid
# UTF-8 in an argument became U+FFFD and named a path nobody sent; and
# `exclude: ["["]` was ignored where the CLI refuses `--exclude '['`. An integer
# is a value, not a spelling (1.0 is served), and an argument named "" is not a
# way past the UTF-8 check (the review of #239).
fixture
python3 - "$WORK/in156" <<'PY'
import sys
lines=[b'{"jsonrpc":"2.0","id":{},"method":"ping"}',
       b'{"jsonrpc":"2.0","id":1.5,"method":"ping"}',
       b'{"jsonrpc":"2.0","id":"s","method":"ping"}',
       b'{"jsonrpc":"2.0","id":2.0,"method":"ping"}',
       b'{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["\xff.go"]}}}',
       b'{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"mrw_read","arguments":{"grep":"A","exclude":["["]}}}',
       b'{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"mrw_read","arguments":{"grep":"A","exclude":["vendor"]}}}',
       b'{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"mrw_read","arguments":{"":"\xff","specs":["a.go"]}}}',
       b'{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"mrw_read","arguments":{"grep":"A","exclude":["vendor/"]}}}']
open(sys.argv[1],"wb").write(b"\n".join(lines)+b"\n")
PY
bounded 20 "$WORK/out156" sh -c '"$0" -C "$1" mcp < "$2" 2>/dev/null' "$MRW" "$R" "$WORK/in156"; want 0 $? "the server answers every line and exits at EOF"
python3 - "$WORK/out156" > "$WORK/v156" <<'PY'
import json,sys
rs=[json.loads(l) for l in open(sys.argv[1]) if l.startswith("{")]
bad=[r for r in rs if r.get("id") is None]
print("ids-refused", len(bad)==2 and all(r["error"]["code"]==-32600 for r in bad))
by={r.get("id"):r for r in rs}
print("string-id-served", "result" in by.get("s",{}))
print("integral-id-served", "result" in by.get(2.0,{}))
t=lambda i: by[i]["result"]["content"][0]["text"] if "result" in by.get(i,{}) else ""
print("utf8-refused", by.get(7,{}).get("result",{}).get("isError") is True and "specs is not valid UTF-8" in t(7))
print("empty-key-refused", by.get(10,{}).get("result",{}).get("isError") is True and "not valid UTF-8" in t(10))
print("exclude-refused", by.get(8,{}).get("result",{}).get("isError") is True and 'exclude "["' in t(8))
print("exclude-slash-refused", by.get(11,{}).get("result",{}).get("isError") is True and "ends in /" in t(11))
print("exclude-kept", by.get(9,{}).get("result",{}).get("isError") is not True)
PY
for k in ids-refused string-id-served integral-id-served utf8-refused empty-key-refused exclude-refused exclude-slash-refused exclude-kept; do
  grep -q "^$k True$" "$WORK/v156" && ok "mcp: $k" || bad "mcp: $k: $(cat "$WORK/v156") $(head -c 600 "$WORK/out156")"
done
out=$(m read --grep A --exclude vendor/ 2>&1); want 2 $? "--exclude vendor/ is refused on the CLI too"
grep -q 'ends in /' <<<"$out" && ok "and the refusal says why" || bad "--exclude vendor/: $out"

# 157. ADR-078 T3: a named MCP read of many specs stops once it is refused. It
# read every spec to the end first: 100,000 held the server past 120 s in the
# v1.25.1 round for a refusal it knew after a few thousand.
fixture
python3 - "$WORK/in157" <<'PY'
import json,sys
req={"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go"]*100000}}}
open(sys.argv[1],"w").write(json.dumps(req)+"\n")
PY
t0=$(date +%s)
bounded 60 "$WORK/out157" sh -c '"$0" -C "$1" mcp < "$2" 2>/dev/null' "$MRW" "$R" "$WORK/in157"; rc=$?
t1=$(date +%s)
want 0 "$rc" "a read of 100,000 specs is answered"
# The size, not the wording, shows the read stopped: a read that ran on to the
# end said "more than 24900000" and passed the grep (the review of #239).
n157=$(grep -o 'would have returned more than [0-9]* bytes' "$WORK/out157" | grep -o '[0-9][0-9]*' | head -1)
[ -n "$n157" ] && [ "$n157" -lt 400000 ] && ok "and refused as soon as it overflowed: more than $n157 bytes, under twice the limit" || bad "no early stop (more than ${n157:-?} bytes): $(head -c 400 "$WORK/out157")"
[ $((t1 - t0)) -le 30 ] && ok "within 30 s" || bad "100,000 specs took $((t1 - t0)) s"

# 158. ADR-078 T3: a line past the MCP ceiling is named, with the ranges around
# it, and the open range offered is served — for a line escaping pushes past the
# ceiling, and for a plain one longer than it, which the capped sample never
# holds whole (the review of #239). The refusal used to advise a range that held
# the line (the waiver on #232).
fixture
python3 -c 'import sys; open(sys.argv[1],"w").write("<"*150000+"\n"+("y"*70+"\n")*1000); open(sys.argv[2],"w").write("x"*201000+"\nnext\n")' "$R/f.svg" "$R/h.js"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["f.svg"]}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["f.svg:2-"]}}}' \
  '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["h.js"]}}}' \
  '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["h.js:2-"]}}}' > "$WORK/in158"
bounded 20 "$WORK/out158" sh -c '"$0" -C "$1" mcp < "$2" 2>/dev/null' "$MRW" "$R" "$WORK/in158"; want 0 $? "the server answers every request"
python3 - "$WORK/out158" > "$WORK/v158" <<'PY'
import json,sys
by={r["id"]:r for r in (json.loads(l) for l in open(sys.argv[1]) if l.startswith("{"))}
t=lambda i: by[i]["result"]["content"][0]["text"]
print("named", "Line 1 of f.svg" in t(1) and "`f.svg:2-`" in t(1) and ":1-1" not in t(1))
print("served", by[2]["result"].get("isError") is not True)
print("plain-named", "Line 1 of h.js" in t(3) and "`h.js:2-`" in t(3) and "no narrower range" not in t(3))
print("plain-served", by[4]["result"].get("isError") is not True)
PY
grep -q '^named True$' "$WORK/v158" && ok "the refusal names line 1 and the range after it" || bad "not named: $(head -c 800 "$WORK/out158")"
grep -q '^served True$' "$WORK/v158" && ok "and the range it offers is served" || bad "the offered range was refused"
grep -q '^plain-named True$' "$WORK/v158" && ok "a plain line past the ceiling is named the same way" || bad "plain line not named: $(sed -n 3p "$WORK/out158" | head -c 800)"
grep -q '^plain-served True$' "$WORK/v158" && ok "and the range after it is served" || bad "the range after the plain line was refused"

# 159. ADR-078 T1: a bad header is one error, not one per body line; a
# tab after @@ is named; and -C on a call with no /pattern/ is refused rather
# than ignored in silence. Each is paired with the input still accepted.
fixture
printf '@@ a.go 3 replac\nX\nY\n' > "$R/p159a.mrw"
out=$(m write --no-check "$R/p159a.mrw" 2>&1); rc=$?
want 2 "$rc" "a plan with a bad op is refused"
{ grep -q 'plan has 1 error' <<<"$out" && ! grep -q 'text before' <<<"$out"; } && ok "as one error, its body not reported as stray text" || bad "bad op: $out"
printf '@@\ta.go\t3\treplace\nX\n' > "$R/p159b.mrw"
out=$(m write --no-check "$R/p159b.mrw" 2>&1); rc=$?
want 2 "$rc" "a header with a tab after @@ is refused"
grep -q 'tab after @@' <<<"$out" && ok "and the tab is named" || bad "tab header: $out"
out=$(m read -C 1 a.go:3 2>&1); rc=$?
want 2 "$rc" "-C on a line range is refused"
grep -q -- '-C widens a /pattern/ match' <<<"$out" && ok "and says -C needs a pattern" || bad "-C on a range: $out"
m read -C 1 'a.go:/func B/' >"$WORK/served.out" 2>&1; want 0 $? "-C on a pattern still serves its context"
m read --grep 'func B' -C 1 >"$WORK/served.out" 2>&1; want 0 $? "and on a grep"
# The reviews of #239: a tab header under a header that did not parse was
# swallowed as its body; -C with --ast-grep was told to name a /pattern/; and a
# noted working set printed its note to stdout before the -C refusal.
printf '@@ a.go 3 replac\nX\n@@\ta.go\t4\treplace\nY\n' > "$R/p159c.mrw"
out=$(m write --no-check "$R/p159c.mrw" 2>&1); rc=$?
want 2 "$rc" "a tab header after a broken one is refused"
{ grep -q 'plan has 2 error' <<<"$out" && grep -q 'line 3: a header is' <<<"$out"; } && ok "and named as its own error" || bad "tab after a broken header: $out"
out=$(m read --ast-grep 'func $A() int' -C 1 2>"$WORK/e159"); rc=$?
want 2 "$rc" "-C with --ast-grep is refused"
{ [ -z "$out" ] && grep -q -- '--ast-grep hit' "$WORK/e159"; } && ok "by name, before the finder runs, with nothing on stdout" || bad "-C with --ast-grep: stdout [$out] stderr $(cat "$WORK/e159")"
m iter add a.go:3 >/dev/null 2>&1 && m iter note probe note >/dev/null 2>&1
out=$(m read -C 1 2>/dev/null); rc=$?
want 2 "$rc" "-C on a noted working set is refused"
[ -z "$out" ] && ok "with nothing on stdout, the note included" || bad "-C on a noted set wrote: $out"
out=$(m read 2>/dev/null); want 0 $? "the noted working set still reads"
grep -q '^# iteration: probe note$' <<<"$out" && ok "and prints its note" || bad "note missing: $out"
# A padded name holding an apostrophe: the offered fix is quoted for a POSIX
# shell, and running it serves the file (the review of #239 found the cmd.exe
# form rewriting it; the POSIX form had the same flaw).
printf 'apostrophe\n' > "$R/ O'Brien"
out=$(m read " O'Brien" 2>&1); want 2 $? "a padded name holding an apostrophe is refused"
fix=${out##*put -- before the path: mrw }
out=$(eval "m $fix" 2>&1); want 0 $? "and the fix it offers, run by a POSIX shell, serves the file"
grep -q '| apostrophe' <<<"$out" && ok "the file it names" || bad "the offered fix read something else: $out"

# 160. ADR-079: the working set and the tally survive racing processes. Each
# was rewritten by truncate-and-write with no lock, so racing processes lost an
# entry or a count — or read the file emptied mid-rewrite and wiped the lot.
fixture
for i in $(seq 1 12); do printf 'f\n' > "$R/f$i.txt"; done
pids=()
for i in $(seq 1 12); do m iter add "f$i.txt" >/dev/null 2>&1 & pids+=($!); done
for p in "${pids[@]}"; do wait "$p"; done
n=$(m iter 2>/dev/null | grep -c '^@')
[ "$n" = 12 ] && ok "12 racing iter adds keep 12 entries" || bad "12 racing iter adds kept $n: $(m iter 2>&1)"
m read f1.txt f2.txt f3.txt f4.txt f5.txt f6.txt f7.txt f8.txt >"$WORK/served.out"
pids=()
for i in $(seq 1 8); do printf '@@ f%d.txt 1 replace\nw\n' "$i" > "$R/p160.$i"; m write --no-check "$R/p160.$i" >/dev/null 2>&1 & pids+=($!); done
for p in "${pids[@]}"; do wait "$p"; done
applied=$(m stats --json 2>/dev/null | jq -r '.counts.applied')
[ "$applied" = 8 ] && ok "8 racing writes count 8 applied" || bad "8 racing writes counted applied=$applied: $(m stats --json 2>&1 | head -c 300)"
# The race rows above can pass with no lock at all when the timing never opens
# the window: the tally row went red in 1 run of 18 with the lock removed, since
# the write lock staggers the writers (the review of #240). So each lock is also
# held on purpose, by a process that is not mrw: a command must wait while it is
# held, and finish once it is released.
sd=$(m seen 2>/dev/null | head -1)
hold160() { # LOCKFILE: hold it for 2 s, printing "held" once it is taken
  perl -MFcntl=:flock -e 'open(my $f, ">>", $ARGV[0]) or die "$!"; flock($f, LOCK_EX) or die "$!"; $| = 1; print "held\n"; sleep 2' "$1" > "$WORK/h160" 2>&1 &
  hp=$!
  for _ in $(seq 1 50); do grep -q held "$WORK/h160" && return 0; sleep 0.1; done
  return 1
}
waited160() { # LABEL CMD...: CMD is still running 0.5 s in, and done by 10 s
  "${@:2}" >/dev/null 2>&1 & wp=$!
  sleep 0.5
  if kill -0 "$wp" 2>/dev/null; then ok "$1 waits while its lock is held"; else bad "$1 finished while its lock was held"; fi
  for _ in $(seq 1 100); do kill -0 "$wp" 2>/dev/null || break; sleep 0.1; done
  if kill -0 "$wp" 2>/dev/null; then kill -9 "$wp"; bad "$1 never finished after its lock was released"; fi
  wait "$wp"; want 0 $? "$1 finishes once its lock is released"
  wait "$hp"
}
printf '@@ f1.txt 1 replace\nheld\n' > "$R/p160.h"
hold160 "$sd/authoring.lock" && waited160 "a write's tally" m write --no-check "$R/p160.h" || bad "could not hold authoring.lock: $(cat "$WORK/h160")"
[ "$(m stats --json 2>/dev/null | jq -r '.counts.applied')" = 9 ] && ok "and its count lands" || bad "after the held write: $(m stats --json 2>&1 | head -c 300)"
hold160 "$sd/iteration.lock" && waited160 "an iter add" m iter add f9.txt || bad "could not hold iteration.lock: $(cat "$WORK/h160")"

# 161. ADR-079: a dry run is not a landing. A clean --dry-run was tallied as
# applied, and counted among the landed writes though nothing landed; a
# refused one is still one refusal.
fixture
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' > "$R/p161a.mrw"
m write --no-check --dry-run "$R/p161a.mrw" >/dev/null 2>&1; want 0 $? "a clean dry run applies nothing and exits 0"
landed=$(m stats --json 2>/dev/null | jq -r '.landed, .plans' | tr '\n' ' ')
[ "$landed" = "0 0 " ] && ok "and records nothing: no plan, no landed write" || bad "a clean dry run was counted (landed, plans: $landed)"
printf '@@ a.go 99 replace\nx\n' > "$R/p161b.mrw"
m write --no-check --dry-run "$R/p161b.mrw" >/dev/null 2>&1; want 1 $? "a dry run addressing a line that does not exist is refused"
[ "$(m stats --json 2>/dev/null | jq -r '.counts.refused_apply, .plans' | tr '\n' ' ')" = "1 1 " ] && ok "and is one refusal" || bad "a refused dry run: $(m stats --json 2>&1 | head -c 300)"

# 164. ADR-083: a plan refused after it parsed is counted on the CLI, as mrw_write
# counts it. A plan naming a directory exited 2 and the tally stayed empty,
# --dry-run or not; a clean write beside it is still counted as applied.
fixture
mkdir "$R/d164"
printf '@@ d164 1 replace\nx\n' > "$R/p164a.mrw"
m write --no-check "$R/p164a.mrw" >/dev/null 2>&1; want 2 $? "a plan naming a directory is refused, exit 2"
[ "$(m stats --json 2>/dev/null | jq -r '.counts.refused_apply, .plans' | tr '\n' ' ')" = "1 1 " ] && ok "and is one refusal" || bad "a filesystem refusal: $(m stats --json 2>&1 | head -c 300)"
m write --no-check --dry-run "$R/p164a.mrw" >/dev/null 2>&1; want 2 $? "the same plan under --dry-run is refused too"
[ "$(m stats --json 2>/dev/null | jq -r '.counts.refused_apply, .plans' | tr '\n' ' ')" = "2 2 " ] && ok "and is a second refusal, dry run or not" || bad "a refused dry run: $(m stats --json 2>&1 | head -c 300)"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' > "$R/p164b.mrw"
m write --no-check "$R/p164b.mrw" >/dev/null 2>&1; want 0 $? "a clean write beside them lands"
[ "$(m stats --json 2>/dev/null | jq -r '.counts.refused_apply, .counts.applied, .plans' | tr '\n' ' ')" = "2 1 3 " ] && ok "and is counted as applied, not as a refusal" || bad "the clean write: $(m stats --json 2>&1 | head -c 300)"

# 165. ADR-083 T2: mrw_write counts a plan refused after it parsed, as the CLI
# does. A pointer hunk path naming two working-set entries returned before the
# MCP tally; a pointer that resolves still lands and is counted as applied.
fixture
m iter add a.go b.go >/dev/null
# ADR-113: check: false — this row counts the landing, and the fixture's go.mod would infer a check.
rq165() { printf '%s' "$1" | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read(),"check":False}}}))'; }
printf '%s\n' "$(rq165 "$(printf '@@ @1-2 1 replace\nx\n')")" | "$MRW" -C "$R" mcp >/dev/null 2>"$WORK/mcp165.err"; want 0 $? "mrw mcp answers a pointer that names two entries"
[ "$(m stats --json 2>/dev/null | jq -r '.counts.refused_apply, .plans' | tr '\n' ' ')" = "1 1 " ] && ok "and mrw_write counts it as one refusal" || bad "an MCP pointer refusal: $(m stats --json 2>&1 | head -c 300)"
m read a.go >"$WORK/served.out"
printf '%s\n' "$(rq165 "$(printf '@@ @1 3 replace\nfunc A() int { return 8 }\n')")" | "$MRW" -C "$R" mcp >/dev/null 2>>"$WORK/mcp165.err"; want 0 $? "mrw mcp applies a pointer that resolves to one file"
grep -q 'return 8' "$R/a.go" && [ "$(m stats --json 2>/dev/null | jq -r '.counts.refused_apply, .counts.applied, .plans' | tr '\n' ' ')" = "1 1 2 " ] \
  && ok "and it landed and is counted as applied, not as a refusal" || bad "the MCP pointer write: $(head -c 200 "$R/a.go"); $(m stats --json 2>&1 | head -c 300)"

# 166. ADR-084: what blind reading 03 taught. A write address `-2` failed as
# `bad line number ""`; it is refused naming the write form, while `1-2` still
# applies. The instructions say a write's exit 1 and 2, and --exclude pruning.
fixture
m read a.go >"$WORK/served.out"
printf '@@ a.go -2 delete\n' > "$R/p166a.mrw"
out=$(m write --no-check "$R/p166a.mrw" 2>&1); want 2 $? "a write address -2 is refused, exit 2"
grep -q '1-2' <<<"$out" && grep -q 'read range' <<<"$out" && ok "and the refusal names the write form 1-2" || bad "the -2 refusal: $out"
printf '@@ a.go 1-2 delete\n' > "$R/p166b.mrw"
m write --no-check --dry-run "$R/p166b.mrw" >/dev/null 2>&1; want 0 $? "while 1-2 still applies"
ins166=$(m instructions 2>&1) && grep -q 'A write exits 1 when a hunk fails validation' <<<"$ins166" && ok "mrw instructions teaches a write's exit 1 and 2" || bad "mrw instructions does not teach a write's exits"
help166=$(m read --help 2>&1) && grep -q 'prunes that whole subtree' <<<"$help166" && ok "read --help says a bare directory name prunes" || bad "read --help does not say --exclude prunes"

# 167. ADR-085: eight mrw mcp servers each serve a read of their own file at
# once, and each acknowledgement then licenses its write. hold and promote
# rewrote pending.json with no lock across processes, so a later save dropped
# another server's checkpoint ids and their acks matched nothing. The pair is
# every single-server ack row above: one read, one ack, one write lands.
fixture
rq167r() { python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":[sys.argv[1]]}}}))' "$1"; }
rq167w() { python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.argv[1],"ack":[sys.argv[2]]}}}))' "$1" "$2"; }
for j in 0 1 2 3 4 5 6 7; do
	printf 'x%d\n' "$j" > "$R/p167_$j.txt"
done
for j in 0 1 2 3 4 5 6 7; do
	printf '%s\n' "$(rq167r "p167_$j.txt")" | "$MRW" -C "$R" mcp > "$WORK/r167.$j" 2>&1 &
	pids167[$j]=$!
done
for j in 0 1 2 3 4 5 6 7; do wait "${pids167[$j]}"; done
landed=0; noid=0
for j in 0 1 2 3 4 5 6 7; do
	ck=$(grep -oE 'ck [^ ]+ open' "$WORK/r167.$j" | head -1 | cut -d' ' -f2)
	[ -n "$ck" ] || { noid=$((noid + 1)); continue; }
	printf '%s\n' "$(rq167w "$(printf '@@ p167_%d.txt 1 replace\ny%d\n' "$j" "$j")" "$ck")" | "$MRW" -C "$R" mcp > "$WORK/w167.$j" 2>&1
	[ "$(cat "$R/p167_$j.txt")" = "y$j" ] && landed=$((landed + 1))
done
[ "$noid" = 0 ] && ok "each of eight concurrent reads served a checkpoint id" || bad "$noid read(s) served no checkpoint: $(head -c 200 "$WORK/r167.0")"
[ "$landed" = 8 ] && ok "and every acknowledgement licensed its write: no server's pending span was lost" || bad "only $landed of 8 acknowledged writes landed: $(head -c 300 "$WORK/w167.0")"

# 168. ADR-086: a name is written as the plan spells it, or refused with nothing
# written — never under another name. The header walk turned a byte that is not
# valid UTF-8 into U+FFFD, so `@@ bad\xffname.txt 0 create` created
# bad�name.txt at exit 0. ext4 (Linux CI) holds the byte; APFS refuses it,
# and the staging probe then fails the plan before its content edit lands.
fixture
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 6 }\n@@ bad\377name.txt 0 create\nx\n' > "$R/p168.mrw"
m write --no-check "$R/p168.mrw" >"$WORK/out168" 2>&1; rc=$?
state=$(python3 -c 'import os,sys; ns=set(os.listdir(os.fsencode(sys.argv[1]))); print("exact" if b"bad\xffname.txt" in ns else "replaced" if "bad�name.txt".encode() in ns else "none")' "$R")
case "$rc:$state" in
	0:exact) ok "a name the filesystem holds lands with the bytes the plan wrote" ;;
	1:none) grep -q 'return 6' "$R/a.go" && bad "the refused plan's content edit landed" || ok "a name the filesystem refuses fails the plan at staging, exit 1 (ADR-132), and nothing is written" ;;
	*) bad "exit $rc, name $state: $(head -c 300 "$WORK/out168")" ;;
esac
[ "$state" != replaced ] && ok "and the name is never rewritten to U+FFFD" || bad "the plan's name was rewritten to U+FFFD"

# 169. ADR-086 T4: a --json receipt names every file the plan gives as written.
# encoding/json turns a byte that is not valid UTF-8 into U+FFFD, so a --json plan
# with such a name is refused before anything lands; the same plan with a valid
# name applies. A name the filesystem supplies is deferred (BACKLOG).
fixture
printf '@@ bad\377name.txt 0 create\nx\n' > "$R/p169a.mrw"
out=$(m write --no-check --json "$R/p169a.mrw" 2>&1); want 2 $? "a --json plan whose name is not valid UTF-8 is refused, exit 2"
state=$(python3 -c 'import os,sys; print("none" if not [n for n in os.listdir(os.fsencode(sys.argv[1])) if n.startswith(b"bad")] else "some")' "$R")
{ [ "$state" = none ] && grep -q 'not valid UTF-8' <<<"$out"; } && ok "and nothing is written, and the refusal says why" || bad "the --json refusal: $state :: $(head -c 300 <<<"$out")"
printf '@@ good169.txt 0 create\nx\n' > "$R/p169b.mrw"
m write --no-check --json "$R/p169b.mrw" >/dev/null 2>&1; want 0 $? "while a --json create with a valid name applies"

# 170. ADR-087: the acknowledgement remedy is found by the refusal's kind, not
# its words. Over the built server, a write to a served and unacknowledged line
# is refused with the remedy appended; a write to a file never served is refused
# as not read, without it. Pair both, so a remedy on every refusal cannot pass.
fixture
printf 'one\ntwo\nthree\n' > "$R/s170.txt"; printf 'uno\ndos\n' > "$R/n170.txt"
printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["s170.txt"]}}}\n' | m mcp >/dev/null 2>&1
want 0 $? "the server serves s170.txt"
for f in s170 n170; do
  printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ %s.txt 2 replace\\nX\\n"}}}\n' "$f" | m mcp > "$WORK/w170.$f" 2>/dev/null
done
python3 - "$WORK/w170.s170" "$WORK/w170.n170" "$R" <<'PY'
import json,sys
def reason(p):
    r=json.load(open(p))["result"]
    sc=r.get("structuredContent") or json.loads(r["content"][1]["text"])
    assert sc["failed"] == 1 and sc.get("applied") is False, "a write with no licence applied: %s" % sc
    return sc["hunks"][0].get("reason") or ""
s, n = reason(sys.argv[1]), reason(sys.argv[2])
assert "has not been read" in s and "never acknowledged" in s, "the served, unacknowledged write lacks the remedy: %s" % s
assert "has not been read" in n and "never acknowledged" not in n, "the never-served write names a remedy that cannot work: %s" % n
assert open(sys.argv[3]+"/s170.txt").read() == "one\ntwo\nthree\n" and open(sys.argv[3]+"/n170.txt").read() == "uno\ndos\n", "a refused write changed the tree"
PY
want 0 $? "the remedy follows the not-read refusal of a served line, and only that one"

# 171. ADR-088 T4: a read whose answer could not be written records nothing. The
# answer was buffered and flushed after the ledger had recorded it, unchecked,
# so a read to a full device licensed lines nobody saw. /dev/full is Linux's;
# elsewhere TestAReadWhoseAnswerCannotBeWrittenRecordsNothing covers it.
fixture
if [ -w /dev/full ]; then
  printf 'one\ntwo\n' > "$R/c171.txt"
  m read c171.txt > /dev/full 2> "$WORK/e171"; want 2 $? "a read whose answer cannot be written exits 2"
  grep -q 'nothing was recorded' "$WORK/e171" && ok "and says nothing was recorded" || bad "the refusal: $(head -c 300 "$WORK/e171")"
  printf '@@ c171.txt 2 replace\nTWO\n' | m write --no-check - > /dev/null 2>&1; want 1 $? "and a write to its lines is refused as unread"
  m read c171.txt >"$WORK/served.out"; want 0 $? "while a read that reached its caller exits 0"
  printf '@@ c171.txt 2 replace\nTWO\n' | m write --no-check - > /dev/null 2>&1; want 0 $? "and licenses the same write"
else
  skip "§171 needs /dev/full (Linux); TestAReadWhoseAnswerCannotBeWrittenRecordsNothing covers it here"
fi

# 172. ADR-092: each step after a write gets its own verdict. A sequence whose
# second step fails exits 3 and never runs the third — its marker is absent —
# while passing steps exit 0; a declared step runs; an unknown --then is
# refused before anything is written; a failed hunk runs no step.
fixture
printf '{"check":"exit 0","steps":{"mark":"touch m172-declared"}}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 172 }\n' > "$R/p172.mrw"
m write --no-check --then-sh 'touch m172-1' --then-sh 'exit 1' --then-sh 'touch m172-3' "$R/p172.mrw" > "$WORK/out172" 2>&1; want 3 $? "a sequence whose second step fails exits 3"
{ [ -e "$R/m172-1" ] && [ ! -e "$R/m172-3" ]; } && ok "step 1 ran and step 3 never did" || bad "the markers: $(ls "$R" | grep m172 | tr '\n' ' ')"
grep -q 'NOT RUN' "$WORK/out172" && ok "the receipt names the step not run" || bad "the receipt: $(head -c 400 "$WORK/out172")"
m write --no-check --then-sh 'exit 0' --then mark "$R/p172.mrw" > /dev/null 2>&1; want 0 $? "the same write with passing steps exits 0"
[ -e "$R/m172-declared" ] && ok "a declared step runs" || bad "the declared step did not run"
printf '@@ a.go 3 replace\nfunc A() int { return 99 }\n' | m write --then nope - > "$WORK/out172u" 2>&1; want 2 $? "an unknown --then is refused"
{ ! grep -q 'return 99' "$R/a.go" && grep -q 'declared: mark' "$WORK/out172u"; } && ok "nothing is written and the declared steps are named" \
  || bad "an unknown step: $(head -c 300 "$WORK/out172u")"
printf '@@ a.go 3 replace anchor="not there"\nx\n' | m write --then-sh 'touch m172-hunk' - > /dev/null 2>&1; want 1 $? "a failed hunk exits 1"
[ ! -e "$R/m172-hunk" ] && ok "and runs no step" || bad "a step ran after a failed hunk"

# 173. ADR-092 end to end: the steps as a caller meets them, through a real
# shell and the built binary, in a real Go module. The harness declares steps
# and no check, so the check is the INFERRED `go test`, scoped to the written package; a real `go vet`
# step follows it. Pairs: the check passing lets the steps run and the check
# failing runs none; a passing step exits 0 and a step past its timeout exits
# 3 with its grandchild gone; a TERM mid-step exits 3 (not killed) and the step
# after it never runs. The attached padded value is refused by main() alone,
# which no in-process test reaches.
fixture
printf '{"timeout_seconds":120,"steps":{"vet":"go vet ./...","one":"echo 1 >> order173","two":"echo 2 >> order173"}}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 0 + 1 }\n' > "$R/p173.mrw"
m write --json --then vet --then-sh 'go build ./...' "$R/p173.mrw" > "$WORK/j173" 2> "$WORK/e173"; want 0 $? "a real check and two real Go steps pass: exit 0"
python3 - "$WORK/j173" <<'PY' && ok "the receipt: the check ran and passed, then both steps pass, adhoc marks only --then-sh, pruned_logs present" || bad "the --json receipt: $(head -c 400 "$WORK/j173")"
import json, sys
r = json.load(open(sys.argv[1]))
c, t = r["check"], r["then"]
assert c["ran"] and c["exit_code"] == 0 and c["command"].startswith("go test "), c
assert [s["status"] for s in t["steps"]] == ["pass", "pass"], t
assert [s["adhoc"] for s in t["steps"]] == [False, True], t
assert t["steps"][0]["command"] == "go vet ./..." and "pruned_logs" in t, t
PY
m write --no-check --then two --then one "$R/p173.mrw" > /dev/null 2>&1; want 0 $? "declared steps named out of file order pass"
[ "$(cat "$R/order173" 2>/dev/null)" = "$(printf '2\n1')" ] && ok "and ran in the order the caller named" || bad "the order: $(cat "$R/order173" 2>/dev/null | tr '\n' ' ')"
m check --full --json --then one > "$WORK/j173c" 2>&1; want 0 $? "mrw check --then runs after a passing check"
python3 -c 'import json,sys; r=json.load(open(sys.argv[1])); assert r["ran"] and r["command"]=="go test ./..." and r["then"]["steps"][0]["status"]=="pass", r' "$WORK/j173c" \
  && ok "and keeps the flat check fields beside then" || bad "check --json: $(head -c 300 "$WORK/j173c")"
m write --then-sh 'printf "%s" "a,b" > comma173' "$R/p173.mrw" > /dev/null 2>&1; want 0 $? "a --then-sh holding quotes and a comma, through a real shell"
[ "$(cat "$R/comma173" 2>/dev/null)" = "a,b" ] && ok "is one step that wrote a,b" || bad "comma173 holds '$(cat "$R/comma173" 2>/dev/null)'"
m write --then-sh='echo x ' "$R/p173.mrw" > "$WORK/o173p" 2>&1; want 2 $? "an attached --then-sh value ending in whitespace is refused by the binary"
grep -q "\-\-then-sh 'echo x '" "$WORK/o173p" && ok "and names the separate form" || bad "the padded refusal: $(head -c 300 "$WORK/o173p")"
printf '@@ a.go 3 replace\nfunc A() int { return 5 }\n' > "$R/p173bad.mrw"
m write --then-sh 'touch m173-afterfail' "$R/p173bad.mrw" > "$WORK/o173f" 2>&1; want 3 $? "a real go test that fails: exit 3"
{ [ ! -e "$R/m173-afterfail" ] && grep -q 'NOT RUN' "$WORK/o173f"; } && ok "and the step after it never ran, named NOT RUN" || bad "after a failed check: $(head -c 400 "$WORK/o173f")"
m write --no-check --then-sh 'exit 1' "$R/p173.mrw" > /dev/null 2>&1; want 3 $? "a failing step on a --no-check write exits 3"
m stats --json > "$WORK/j173s" 2>&1
python3 -c 'import json,sys; c=json.load(open(sys.argv[1]))["counts"]; assert c["failed_check"]==2, c' "$WORK/j173s" \
  && ok "stats counts the failed check and the failed step: failed_check 2" || bad "stats: $(head -c 300 "$WORK/j173s")"
printf '{"timeout_seconds":2,"steps":{}}\n' > "$R/.quality-harness.json"
t0=$(date +%s)
bounded 30 "$WORK/o173t" "$MRW" -C "$R" write --no-check --then-sh "sleep 60 & echo \$! > $R/gc173.pid; wait" --then-sh 'touch m173-aftertimeout' "$R/p173.mrw"; rc=$?; el=$(( $(date +%s) - t0 ))
{ [ "$rc" = 3 ] && [ "$el" -lt 20 ] && grep -q 'TIMED OUT' "$WORK/o173t" && [ ! -e "$R/m173-aftertimeout" ]; } \
  && ok "a step past its timeout exits 3 in ${el}s, says TIMED OUT, and the next step never runs" || bad "timeout: exit $rc after ${el}s: $(head -c 400 "$WORK/o173t")"
gc=$(cat "$R/gc173.pid" 2>/dev/null); alive=1
for _ in $(seq 1 30); do kill -0 "${gc:-999999999}" 2>/dev/null || { alive=0; break; }; sleep 0.1; done
{ [ -n "$gc" ] && [ "$alive" = 0 ]; } && ok "and the timed-out step's grandchild is gone" || { kill -9 "${gc:-999999999}" 2>/dev/null; bad "the timed-out step's grandchild '$gc' outlived it"; }
# The TERM row gets its own generous timeout (the row above left 2 s), waits
# for the step to say it started before signalling, bounds mrw's exit, and
# records the step's pid: the step runs in a process group of its own, which
# neither this file's EXIT trap nor its survivors check reaches (review of #267).
printf '{"timeout_seconds":300,"steps":{}}\n' > "$R/.quality-harness.json"
"$MRW" -C "$R" write --no-check --then-sh "echo \$\$ > $R/step173.pid; touch $R/started173; exec sleep 60" --then-sh 'touch m173-afterterm' "$R/p173.mrw" > "$WORK/o173i" 2>&1 < /dev/null & wp=$!
ready=0; for _ in $(seq 1 200); do [ -e "$R/started173" ] && { ready=1; break; }; sleep 0.1; done
if [ "$ready" = 1 ]; then
  kill -TERM "$wp"
  ended=0; for _ in $(seq 1 200); do kill -0 "$wp" 2>/dev/null || { ended=1; break; }; sleep 0.1; done
  [ "$ended" = 1 ] || kill -9 "$wp" 2>/dev/null
  wait "$wp"; rc=$?
  { [ "$ended" = 1 ] && [ "$rc" = 3 ] && [ ! -e "$R/m173-afterterm" ] && grep -q 'INTERRUPTED' "$WORK/o173i"; } \
    && ok "a TERM while a step runs: mrw reports it INTERRUPTED, exits 3, and the next step never runs" || bad "TERM mid-step: exit $rc, ended $ended: $(head -c 400 "$WORK/o173i")"
else
  kill -9 "$wp" 2>/dev/null; wait "$wp"
  bad "the TERM row's step never said it started: $(head -c 300 "$WORK/o173i")"
fi
sp=$(cat "$R/step173.pid" 2>/dev/null); alive=1
for _ in $(seq 1 30); do kill -0 "${sp:-999999999}" 2>/dev/null || { alive=0; break; }; sleep 0.1; done
{ [ -n "$sp" ] && [ "$alive" = 0 ]; } && ok "and the interrupted step's process is gone" \
  || { kill -9 "${sp:-999999999}" 2>/dev/null; bad "the interrupted step '${sp:-?}' outlived mrw"; }

# 174. ADR-092 T5, from the stress round of 2026-09-28. A malformed "steps"
# block does not stop a write that asks for no step (it refused every write),
# and asking for one refuses it with nothing written. A step that re-runs mrw
# with steps recurses to MRW_STEP_DEPTH 8 and stops there, the whole chain
# returning with nothing left running (one peer's grew to ~107 processes).
fixture
printf '{"check":"true","steps":{"my step":"true"}}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 0 + 1 }\n' > "$R/p174.mrw"
m write "$R/p174.mrw" > /dev/null 2>&1; want 0 $? "a write asking for no step lands beside a malformed steps block"
printf '@@ a.go 3 replace\nfunc A() int { return 1 + 0 }\n' > "$R/p174b.mrw"
m write --then x "$R/p174b.mrw" > "$WORK/o174" 2>&1; want 2 $? "asking for a step reads the block and refuses it"
{ grep -q 'return 0 + 1' "$R/a.go" && grep -q '"my step"' "$WORK/o174"; } && ok "with nothing written, naming the step" || bad "the malformed block: $(head -c 300 "$WORK/o174")"
printf '{"check":"true","steps":{}}\n' > "$R/.quality-harness.json"
printf '%s\n' 'echo "$MRW_STEP_DEPTH" >> depth174' "exec \"$MRW\" check --then-sh 'sh \"$R/rec174.sh\"'" > "$R/rec174.sh"
t0=$(date +%s)
bounded 90 "$WORK/o174r" "$MRW" -C "$R" check --then-sh "sh \"$R/rec174.sh\""; rc=$?; el=$(( $(date +%s) - t0 ))
deepest=$(tail -1 "$R/depth174" 2>/dev/null)
{ [ "$rc" = 3 ] && [ "$deepest" = 8 ] && [ "$el" -lt 60 ]; } && ok "a step that re-runs mrw with steps stops at MRW_STEP_DEPTH 8: exit 3 in ${el}s" \
  || bad "recursion: exit $rc after ${el}s, deepest '$deepest': $(head -c 300 "$WORK/o174r")"
# Matched by this run's own fixture path, never the bare script name: another
# contract run on this machine has a rec174.sh of its own (review of #269).
left=$(pgrep -f "$R/rec174.sh" | wc -l | tr -d ' ')
[ "$left" = 0 ] && ok "and nothing of the recursion is left running" || { pkill -9 -f "$R/rec174.sh"; bad "$left recursion processes outlived the row"; }

# 175. ADR-093 T1: an MCP tool refuses an argument it does not declare. A
# `check` sent to mrw_write, or a `max_lines` sent to mrw_read, was ignored and
# the call answered as if it had done what was asked (the 2026-09-29 gap
# survey, C1). ADR-113 declared `check`, so the write sends `no_check`, which
# no mrw declares, and ADR-117 declared `max_lines`, so the read sends `context`.
# The pair: the same call without the key is applied or served.
fixture
printf 'one\ntwo\n' > "$R/a.txt"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ b.txt 0 create\nX\n","no_check":true}}}' | m mcp 2>/dev/null > "$WORK/w175.json"
{ jq -e '.result.isError == true and (.result.content[0].text | contains("\"no_check\"") and contains("\"plan\""))' "$WORK/w175.json" > /dev/null && [ ! -e "$R/b.txt" ]; } \
  && ok "an undeclared argument is refused by name, and nothing is written" || bad "the refused write: $(head -c 400 "$WORK/w175.json")"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ b.txt 0 create\nX\n"}}}' | m mcp 2>/dev/null > "$WORK/w175b.json"
{ jq -e '.result and (.result.isError | not)' "$WORK/w175b.json" > /dev/null && [ "$(cat "$R/b.txt" 2>/dev/null)" = X ]; } \
  && ok "and the same call without it is served and applied: b.txt holds X" || bad "the declared write: $(head -c 400 "$WORK/w175b.json")"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.txt"],"context":1}}}' | m mcp 2>/dev/null > "$WORK/r175.json"
jq -e '.result.isError == true and (.result.content[0].text | contains("\"context\"") and (contains("-- ck ") | not))' "$WORK/r175.json" > /dev/null \
  && ok "an undeclared argument is refused by name on mrw_read, and nothing is served" || bad "the refused read: $(head -c 400 "$WORK/r175.json")"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.txt"]}}}' | m mcp 2>/dev/null > "$WORK/r175b.json"
jq -e '(.result.isError | not) and (.result.content[0].text | contains("1| one"))' "$WORK/r175b.json" > /dev/null \
  && ok "and the same call without it is served and applied: a.txt is served" || bad "the declared read: $(head -c 400 "$WORK/r175b.json")"

# 176. ADR-093 T2: tools/list says what T1 enforces. Neither input schema said
# additionalProperties false, so an extra key was valid against both. The pair:
# a call with declared arguments is still served.
fixture
printf 'one\ntwo\n' > "$R/a.txt"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | m mcp 2>/dev/null > "$WORK/l176.json"
jq -e '(.result.tools | length) >= 2 and all(.result.tools[]; .inputSchema.additionalProperties == false)' "$WORK/l176.json" > /dev/null \
  && ok "every input schema is closed: additionalProperties false on both tools" || bad "tools/list: $(head -c 400 "$WORK/l176.json")"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.txt"]}}}' | m mcp 2>/dev/null > "$WORK/r176.json"
jq -e '(.result.isError | not) and (.result.content[0].text | contains("1| one"))' "$WORK/r176.json" > /dev/null \
  && ok "and a call with declared arguments is still served" || bad "the declared read: $(head -c 400 "$WORK/r176.json")"

# 184. ADR-096: a path the caller names is never dropped by a finder. A named
# in-root link to a directory, spelled relative to the root, answered "no file
# matched" with no line about it (the 2026-09-29 gap survey, loops-L10); it is
# refused naming the directory to name. The pair: spelled absolutely the same
# link still walks the directory it resolves to, as v1.31.0 does (M's choice at
# acceptance), a path through the link is walked, and a relative link to the
# root walks the root.
fixture
mkdir -p "$R/d/sub"; printf 'N184\n' > "$R/d/f.go"; printf 'N184\n' > "$R/d/sub/g.go"
ln -s d "$R/dlink"; ln -s . "$R/self"
for sp in dlink dlink/ dlink/.; do
  m read --grep N184 "$sp" > "$WORK/o184" 2>&1; want 1 $? "--grep over a named $sp exits 1"
  grep -qF "==> $sp  REFUSED  is a link to the directory d, and a walk does not follow a link: name d" "$WORK/o184" \
    && ok "and its REFUSED line names $sp and the directory d" || bad "$sp: $(head -c 300 "$WORK/o184")"
done
m read --grep N184 d dlink > "$WORK/o184b" 2>&1; want 1 $? "--grep over d and dlink exits 1"
{ grep -q '^==> d/f.go' "$WORK/o184b" && grep -q '^==> dlink  REFUSED' "$WORK/o184b"; } \
  && ok "serving d/f.go beside dlink's REFUSED line" || bad "d dlink: $(head -c 300 "$WORK/o184b")"
m read --grep N184 "$R/dlink" > "$WORK/o184c" 2>&1; want 0 $? "the pair: --grep over the absolute \$R/dlink exits 0"
{ grep -q '^==> d/f.go' "$WORK/o184c" && ! grep -q 'REFUSED' "$WORK/o184c"; } \
  && ok "and walks d under its own names, as v1.31.0 does" || bad "\$R/dlink: $(head -c 300 "$WORK/o184c")"
m read --grep N184 d dlink/sub > "$WORK/o184d" 2>&1; want 0 $? "--grep over d and dlink/sub exits 0"
{ grep -q '^==> d/f.go' "$WORK/o184d" && grep -q '^==> dlink/sub/g.go' "$WORK/o184d"; } \
  && ok "serving d/f.go and dlink/sub/g.go" || bad "d dlink/sub: $(head -c 300 "$WORK/o184d")"
m read --grep N184 self > "$WORK/o184e" 2>&1; want 0 $? "--grep over a relative link to . exits 0"
{ grep -q '^==> d/f.go' "$WORK/o184e" && grep -q '^==> d/sub/g.go' "$WORK/o184e"; } \
  && ok "and walks the root" || bad "self: $(head -c 300 "$WORK/o184e")"
# §184, ast-grep (ADR-096 T2): --ast-grep judges every named path as --grep does
# before the binary starts. A refused path never reaches it, so a path outside
# the root is not searched; an accepted one reaches it as the absolute path mrw
# judged. The fake records its arguments in argv184, which exists only if it ran.
d184=$(mktemp -d "$WORK/d184-XXXXXX")
cat > "$d184/ast-grep" <<EOF184
#!/bin/sh
for a in "\$@"; do printf '%s\n' "\$a" >> "$d184/argv184"; done
printf '%s' '[{"file":"d/f.go","range":{"start":{"line":0},"end":{"line":0}}}]'
EOF184
chmod +x "$d184/ast-grep"
RP=$(cd "$R" && pwd -P)
mkdir -p "$WORK/out184"; printf 'N184\n' > "$WORK/out184/x.go"
bounded 10 "$WORK/a184" env PATH="$d184:$PATH" "$MRW" -C "$R" read --ast-grep N184 "$WORK/out184"; want 1 $? "--ast-grep over a path outside the root exits 1"
{ grep -q 'REFUSED.*outside the root' "$WORK/a184" && [ ! -e "$d184/argv184" ]; } \
  && ok "with a REFUSED line, and ast-grep never ran" || bad "outside: $(head -c 300 "$WORK/a184")"
bounded 10 "$WORK/a184b" env PATH="$d184:$PATH" "$MRW" -C "$R" read --ast-grep N184 dlink; want 1 $? "--ast-grep over a named dlink exits 1"
{ grep -q '^==> dlink  REFUSED.*name d$' "$WORK/a184b" && [ ! -e "$d184/argv184" ]; } \
  && ok "naming the directory d, and ast-grep never ran" || bad "dlink: $(head -c 300 "$WORK/a184b")"
printf 'N184\n' > "$R/locked184.go"; chmod 000 "$R/locked184.go"
if [ -r "$R/locked184.go" ]; then
  skip "--ast-grep refuses a named file it cannot open (permission bits not enforced here — running as root?)"
else
  bounded 10 "$WORK/a184c" env PATH="$d184:$PATH" "$MRW" -C "$R" read --ast-grep N184 a.go locked184.go; want 1 $? "--ast-grep over a.go and a mode-000 file exits 1"
  { grep -q '^==> locked184.go  REFUSED.*permission denied' "$WORK/a184c" && grep -qxF "$RP/a.go" "$d184/argv184" && ! grep -q 'locked184' "$d184/argv184"; } \
    && ok "refusing the file with the OS's error, while ast-grep ran on a.go alone" || bad "mode 000: $(head -c 300 "$WORK/a184c") argv: $(tr '\n' ' ' < "$d184/argv184" 2>/dev/null)"
fi
chmod 644 "$R/locked184.go"; rm -f "$d184/argv184"
bounded 10 "$WORK/a184d" env PATH="$d184:$PATH" "$MRW" -C "$R" read --ast-grep N184 d; want 0 $? "the pair: --ast-grep over d exits 0"
{ grep -q '^==> d/f.go' "$WORK/a184d" && grep -qxF "$RP/d" "$d184/argv184"; } \
  && ok "serving the hit, and ast-grep was handed d's absolute path" || bad "d: $(head -c 300 "$WORK/a184d") argv: $(cat "$d184/argv184" 2>/dev/null)"
printf 'N184\n' > "$R/-p"; rm -f "$d184/argv184"
bounded 10 "$WORK/a184e" env PATH="$d184:$PATH" "$MRW" -C "$R" read --ast-grep N184 -- -p; want 0 $? "--ast-grep over a named file -p exits 0"
[ "$(tail -1 "$d184/argv184")" = "$RP/-p" ] && ok "and ast-grep was handed it as an absolute path, not as an option" \
  || bad "-p: argv $(tr '\n' ' ' < "$d184/argv184" 2>/dev/null)"
printf 'N184\n' > "$R/f.go"; rm -f "$d184/argv184"
bounded 10 "$WORK/a184f" env PATH="$d184:$PATH" "$MRW" -C "$R" read --ast-grep N184 self/../f.go; want 0 $? "--ast-grep over self/../f.go exits 0"
{ grep -qxF "$RP/f.go" "$d184/argv184" && ! grep -qF 'self/../f.go' "$d184/argv184"; } \
  && ok "and ast-grep was handed the root's f.go as mrw judged it, not the spelling" || bad "self/../f.go: argv $(tr '\n' ' ' < "$d184/argv184" 2>/dev/null)"

# 185. ADR-096 T3: `iter add` names the error the OS gave for a path it cannot
# stat. A link loop answered "no such file … (quote a spec containing spaces)",
# a hint about word splitting (the 2026-09-29 gap survey). The pair: a path
# that is not there keeps that sentence, and a reachable one is added.
fixture
ln -s loop185 "$R/loop185"
m iter add loop185 > "$WORK/o185" 2>&1; want 2 $? "iter add over a link loop exits 2"
{ grep -q 'too many levels of symbolic links' "$WORK/o185" && ! grep -q 'no such file' "$WORK/o185"; } \
  && ok "naming the loop, not \"no such file\"" || bad "loop: $(head -c 300 "$WORK/o185")"
m iter add nosuch185 > "$WORK/o185b" 2>&1; want 2 $? "the pair: iter add over a missing path exits 2"
grep -q 'no such file: nosuch185' "$WORK/o185b" && ok "and keeps the missing-file sentence" || bad "nosuch: $(head -c 300 "$WORK/o185b")"
m iter add a.go > "$WORK/o185c" 2>&1; want 0 $? "iter add a.go exits 0"
grep -q '^@1 *a.go$' "$WORK/o185c" && ok "and lists a.go" || bad "a.go: $(head -c 300 "$WORK/o185c")"

# 178. ADR-094 T1: a step whose command holds {files} or {packages} is refused,
# exit 2, before anything is written or run: mrw expands the tokens only in
# scoped_check, and a step given one ran the literal text and passed. The
# declared check touches m178-check, so a refusal moved below the check leaves
# it. The pair: the same declared step and --then-sh, with a path in place of
# the token, run and leave their markers.
fixture
printf '{"check":"touch m178-check","steps":{"fmt":"echo fmt {files} >> m178-step"}}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 178 }\n' > "$R/p178.mrw"
m write --then fmt "$R/p178.mrw" > "$WORK/o178a" 2>&1; want 2 $? "write --then naming a declared step that holds {files} is refused"
{ ! grep -q 'return 178' "$R/a.go" && [ ! -e "$R/m178-step" ] && [ ! -e "$R/m178-check" ] && grep -q 'step "fmt": its command holds {files}' "$WORK/o178a"; } \
  && ok "nothing is written or run, and the refusal names the step and the token" || bad "declared, on write: $(head -c 300 "$WORK/o178a")"
m check --then fmt a.go > "$WORK/o178b" 2>&1; want 2 $? "check --then naming the same step is refused"
{ [ ! -e "$R/m178-check" ] && [ ! -e "$R/m178-step" ] && grep -q '{files}' "$WORK/o178b"; } \
  && ok "before the check runs, naming the token" || bad "declared, on check: $(head -c 300 "$WORK/o178b")"
m write --then-sh 'echo sh {packages} >> m178-step' "$R/p178.mrw" > "$WORK/o178c" 2>&1; want 2 $? "write --then-sh holding {packages} is refused"
{ ! grep -q 'return 178' "$R/a.go" && [ ! -e "$R/m178-step" ] && [ ! -e "$R/m178-check" ] && grep -q -- '--then-sh .*its command holds {packages}' "$WORK/o178c"; } \
  && ok "nothing is written or run, and the refusal names --then-sh and the token" || bad "ad hoc, on write: $(head -c 300 "$WORK/o178c")"
m check --then-sh 'echo sh {packages} >> m178-step' a.go > "$WORK/o178d" 2>&1; want 2 $? "check --then-sh holding {packages} is refused"
{ [ ! -e "$R/m178-check" ] && [ ! -e "$R/m178-step" ] && grep -q '{packages}' "$WORK/o178d"; } \
  && ok "before the check runs, naming the token" || bad "ad hoc, on check: $(head -c 300 "$WORK/o178d")"
printf '{"check":"touch m178-check","steps":{"fmt":"echo fmt a.go >> m178-step"}}\n' > "$R/.quality-harness.json"
m write --then fmt --then-sh 'echo sh a.go >> m178-step' "$R/p178.mrw" > "$WORK/o178e" 2>&1; want 0 $? "the pair: the same steps with a path in place of the token pass on write"
{ grep -q 'return 178' "$R/a.go" && [ -e "$R/m178-check" ] && [ "$(cat "$R/m178-step" 2>/dev/null)" = "$(printf 'fmt a.go\nsh a.go')" ]; } \
  && ok "the write landed, the check ran and both steps left their marker" || bad "the pair, on write: $(head -c 300 "$WORK/o178e")"
rm -f "$R/m178-check" "$R/m178-step"
m check --then fmt --then-sh 'echo sh a.go >> m178-step' a.go > "$WORK/o178f" 2>&1; want 0 $? "the pair on check passes"
{ [ -e "$R/m178-check" ] && [ "$(cat "$R/m178-step" 2>/dev/null)" = "$(printf 'fmt a.go\nsh a.go')" ]; } \
  && ok "the check ran and both steps left their marker" || bad "the pair, on check: $(head -c 300 "$WORK/o178f")"

# 179. ADR-094 T2: a passing step shows the last non-empty line of its output
# under its PASS head, so a masked failure shows what it printed: a step that
# printed and then exited 1 behind `|| true` passed with nothing under it. The
# pair: a silent pass prints no tail line, and a failing step still exits 3
# with then last:.
fixture
printf '{"check":"exit 0"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 179 }\n' > "$R/p179.mrw"
m write --no-check --then-sh 'echo LASTLINE; false || true' "$R/p179.mrw" > "$WORK/o179a" 2>&1; want 0 $? "a masked failure that printed passes: exit 0"
[ "$(grep -A1 -- '— PASS' "$WORK/o179a" | sed -n 2p)" = "  | LASTLINE" ] \
  && ok "and the line under its PASS head is its last line" || bad "the pass: $(head -c 300 "$WORK/o179a")"
m write --no-check --then-sh true "$R/p179.mrw" > "$WORK/o179b" 2>&1; want 0 $? "a silent pass exits 0"
{ grep -q -- '— PASS' "$WORK/o179b" && ! grep -A1 -- '— PASS' "$WORK/o179b" | grep -q '^  | '; } \
  && ok "and prints no tail line under its head" || bad "the silent pass: $(head -c 300 "$WORK/o179b")"
m write --no-check --then-sh 'echo X; false' "$R/p179.mrw" > "$WORK/o179c" 2>&1; want 3 $? "a failing step still exits 3"
grep -q '^then last: X$' "$WORK/o179c" && ok "with then last: X" || bad "the failing step: $(head -c 300 "$WORK/o179c")"

# 181. ADR-095 T1: the check counts a level, and at MRW_STEP_DEPTH 8 mrw starts
# no project command. A check that runs `mrw check` again ran with its caller's
# depth — unset at every level — so nothing bounded the chain. Here it records
# levels 1 to 8 and stops, exit 3; the script stops itself past 12 levels, so
# a missing guard fails the row instead of running away. At 8 a .go write whose
# check is due is refused with nothing written and no check run, beside the
# --no-check write that lands; mrw check is refused at 8 and runs at 7 and
# under a value mrw did not write.
fixture
printf '{"check":"sh %s/rec181.sh"}\n' "$R" > "$R/.quality-harness.json"
printf '%s\n' 'echo "$MRW_STEP_DEPTH" >> depth181' '[ "$(wc -l < depth181)" -gt 12 ] && exit 97' "exec \"$MRW\" check --full" > "$R/rec181.sh"
t0=$(date +%s)
bounded 90 "$WORK/o181r" env -u MRW_STEP_DEPTH "$MRW" -C "$R" check --full; rc=$?; el=$(( $(date +%s) - t0 ))
levels=$(tr '\n' ' ' < "$R/depth181" 2>/dev/null)
{ [ "$rc" = 3 ] && [ "$levels" = "1 2 3 4 5 6 7 8 " ] && [ "$el" -lt 60 ]; } && ok "a check that re-runs mrw check stops at MRW_STEP_DEPTH 8: exit 3 in ${el}s, levels 1 to 8" \
  || bad "recursion through the check: exit $rc after ${el}s, levels '$levels': $(head -c 300 "$WORK/o181r")"
left=$(pgrep -f "$R/rec181.sh" | wc -l | tr -d ' ')
[ "$left" = 0 ] && ok "and nothing of the recursion is left running" || { pkill -9 -f "$R/rec181.sh"; bad "$left recursion processes outlived the row"; }
fixture
printf '{"check":"echo x >> m181"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 181 }\n' > "$R/p181.mrw"
MRW_STEP_DEPTH=8 "$MRW" -C "$R" write "$R/p181.mrw" > "$WORK/o181w" 2>&1; want 2 $? "at depth 8 a .go write whose check is due is refused"
{ ! grep -q 'return 181' "$R/a.go" && [ ! -e "$R/m181" ] && grep -q 'MRW_STEP_DEPTH=8' "$WORK/o181w" && grep -q -- '--no-check' "$WORK/o181w"; } \
  && ok "nothing is written, no check runs, and the refusal names the depth and --no-check" || bad "the refused write: $(head -c 300 "$WORK/o181w")"
MRW_STEP_DEPTH=8 "$MRW" -C "$R" write --no-check "$R/p181.mrw" > /dev/null 2>&1; want 0 $? "the pair: at depth 8 the same write with --no-check lands"
{ grep -q 'return 181' "$R/a.go" && [ ! -e "$R/m181" ]; } && ok "and runs no check" || bad "the --no-check write did not land, or a check ran"
MRW_STEP_DEPTH=8 "$MRW" -C "$R" check --full > "$WORK/o181c" 2>&1; want 2 $? "mrw check at depth 8 is refused"
[ ! -e "$R/m181" ] && ok "before its check runs" || bad "the check ran at depth 8"
MRW_STEP_DEPTH=7 "$MRW" -C "$R" check --full > /dev/null 2>&1; want 0 $? "mrw check at depth 7 runs"
[ -e "$R/m181" ] && ok "and its check ran" || bad "the check did not run at depth 7"
MRW_STEP_DEPTH=x "$MRW" -C "$R" check --full > /dev/null 2>&1; want 0 $? "mrw check under MRW_STEP_DEPTH=x runs: a value mrw did not write counts as zero"

# 182. ADR-095 T2: a timeout reaches the check of a nested mrw. The outer check
# runs mrw on an inner tree — `; true` keeps sh the leader, so the TERM kills sh
# at once and only the grace keeps the nested mrw alive — and the inner check
# sleeps. SIGKILL to the outer group killed the nested mrw and its check's sleep
# ran on in a group of its own, which the run-wide survivors check cannot see;
# TERM first lets the nested mrw stop it. The pair: a check that ignores TERM is
# still killed once the grace has passed.
fixture
mkdir -p "$R/inner"
printf '{"check":"%s -C inner check --full; true","timeout_seconds":3}\n' "$MRW" > "$R/.quality-harness.json"
printf '{"check":"echo $$ > pid182; exec sleep 300"}\n' > "$R/inner/.quality-harness.json"
bounded 30 "$WORK/o182a" "$MRW" -C "$R" check --full; want 3 $? "an outer check that times out while a nested mrw runs its check exits 3"
p182=$(cat "$R/inner/pid182" 2>/dev/null); alive=1
for _ in $(seq 1 30); do kill -0 "${p182:-999999999}" 2>/dev/null || { alive=0; break; }; sleep 0.1; done
{ [ -n "$p182" ] && [ "$alive" = 0 ]; } && ok "and the nested mrw's check is gone within 3 s" \
  || { kill -9 "${p182:-999999999}" 2>/dev/null; bad "the nested check '$p182' outlived the outer timeout: $(head -c 300 "$WORK/o182a")"; }
fixture
printf '%s\n' "{\"check\":\"trap '' TERM; echo \$\$ > pid182b; sleep 300\",\"timeout_seconds\":1}" > "$R/.quality-harness.json"
t0=$(date +%s)
bounded 30 "$WORK/o182b" "$MRW" -C "$R" check --full; rc=$?; el=$(( $(date +%s) - t0 ))
p182b=$(cat "$R/pid182b" 2>/dev/null); alive=1
for _ in $(seq 1 30); do kill -0 "${p182b:-999999999}" 2>/dev/null || { alive=0; break; }; sleep 0.1; done
{ [ "$rc" = 3 ] && [ "$el" -lt 10 ] && [ -n "$p182b" ] && [ "$alive" = 0 ]; } && ok "the pair: a check that ignores TERM exits 3 in ${el}s and is killed" \
  || { kill -9 "${p182b:-999999999}" 2>/dev/null; bad "a check ignoring TERM: exit $rc after ${el}s, pid '$p182b' alive $alive: $(head -c 300 "$WORK/o182b")"; }

# 183. ADR-095 T2: the reap after a clean exit sends TERM first. A check that
# passes leaves a straggler that traps TERM; it hears TERM, writes its marker
# and is gone when mrw returns. SIGKILL at once gave it no chance to clean up.
fixture
printf '%s\n' "{\"check\":\"(trap 'echo term > marker183; exit 0' TERM; sleep 300 & echo \$! > sleep183; touch ready183; wait) >/dev/null 2>&1 & while [ ! -e ready183 ]; do sleep 0.05; done; exit 0\"}" > "$R/.quality-harness.json"
bounded 30 "$WORK/o183" "$MRW" -C "$R" check --full; want 0 $? "a check that passes leaving a straggler exits 0"
s183=$(cat "$R/sleep183" 2>/dev/null); alive=1
for _ in $(seq 1 30); do kill -0 "${s183:-999999999}" 2>/dev/null || { alive=0; break; }; sleep 0.1; done
{ [ -e "$R/marker183" ] && [ -n "$s183" ] && [ "$alive" = 0 ]; } && ok "the straggler heard TERM first, wrote its marker, and is gone" \
  || { kill -9 "${s183:-999999999}" 2>/dev/null; bad "the straggler: marker $([ -e "$R/marker183" ] && echo yes || echo no), sleep '$s183' alive $alive: $(head -c 300 "$WORK/o183")"; }

# 186. ADR-092 Decision 5: a write that landed and then could not save its
# ledger still names every step it asked for, not_run, in both receipts; it
# dropped them (the 2026-09-29 gap survey, C3). The tally stays ADR-083's: a
# --no-check landing is applied. The pair: with the ledger writable again the
# same write runs its step. The ledger is made read-only, which uid 0 ignores.
fixture
printf '@@ a.go 3 replace\nfunc A() int { return 186 }\n' > "$R/p186.mrw"
printf '@@ b.go 3 replace\nfunc D() int { return 186 }\n' > "$R/p186b.mrw"
led186="$(m seen | head -1)/seen"
chmod 444 "$led186"
if [ -w "$led186" ]; then
  chmod 600 "$led186"
  skip "a landing whose ledger cannot be saved names its steps not run (permission bits not enforced here — running as root?)"
else
  m write --no-check --json --then-sh 'touch m186' "$R/p186.mrw" > "$WORK/j186" 2> /dev/null; want 2 $? "a landing whose ledger cannot be saved exits 2"
  { grep -q 'return 186' "$R/a.go" && [ ! -e "$R/m186" ] && jq -se 'length == 1 and (.[0] | (.error | length > 0) and (.then.steps | length == 1 and .[0].status == "not_run"))' "$WORK/j186" > /dev/null; } \
    && ok "it landed, the step never ran, and the --json receipt names it not_run beside the error" || bad "the --json receipt: $(head -c 400 "$WORK/j186")"
  m write --no-check --then-sh 'touch m186' "$R/p186b.mrw" > "$WORK/o186" 2>&1; want 2 $? "the same landing in human form exits 2"
  { grep -q 'return 186' "$R/b.go" && [ ! -e "$R/m186" ] && grep -q 'then 1/1 --then-sh: touch m186 — NOT RUN' "$WORK/o186"; } \
    && ok "and the human receipt names the step NOT RUN" || bad "the human receipt: $(head -c 400 "$WORK/o186")"
  m stats --json > "$WORK/s186" 2>&1
  jq -e '.counts.applied == 2 and .counts.check_not_run == 0' "$WORK/s186" > /dev/null \
    && ok "stats counts both --no-check landings applied, as ADR-083 does" || bad "stats: $(head -c 300 "$WORK/s186")"
  chmod 600 "$led186"
  m read a.go >"$WORK/served.out"
  printf '@@ a.go 3 replace\nfunc A() int { return 1860 }\n' > "$R/p186c.mrw"
  m write --no-check --json --then-sh 'touch m186' "$R/p186c.mrw" > "$WORK/j186c" 2> /dev/null; want 0 $? "the pair: with the ledger writable the same write exits 0"
  { [ -e "$R/m186" ] && jq -e '.then.steps[0].status == "pass"' "$WORK/j186c" > /dev/null; } \
    && ok "and its step runs and passes" || bad "the pair: $(head -c 400 "$WORK/j186c")"
fi

# 187. ADR-092 Decision 5, "the check could not start": such a check names every
# step asked for, not_run, under --json in one document. With TMPDIR pointing
# nowhere the check cannot create its log, and `check --json` printed nothing at
# all (the 2026-09-29 gap survey, C2). Without a step asked it is one document
# holding the error and no then block (ADR-100; before it, no document at all).
# The pair: with this run's TMPDIR the same check runs its step.
fixture
printf '{"check":"exit 0"}\n' > "$R/.quality-harness.json"
TMPDIR="$WORK/gone187" "$MRW" -C "$R" check --full --json --then-sh 'touch m187' > "$WORK/j187" 2> /dev/null; want 2 $? "a check whose log cannot be created exits 2"
{ [ ! -e "$R/m187" ] && jq -se 'length == 1 and (.[0] | (.error | length > 0) and (.then.steps | length == 1 and .[0].status == "not_run"))' "$WORK/j187" > /dev/null; } \
  && ok "stdout is one JSON document naming the error and the step not_run, which never ran" || bad "check --json: $(head -c 400 "$WORK/j187")"
TMPDIR="$WORK/gone187" "$MRW" -C "$R" check --full --json > "$WORK/n187" 2> /dev/null; want 2 $? "the same check with no step asked exits 2"
jq -se 'length == 1 and (.[0] | type == "object" and keys == ["error"] and (.error | length > 0))' "$WORK/n187" > /dev/null && ok "and, with no step asked, prints exactly one document holding only the error (ADR-100)" || bad "no step asked: $(head -c 300 "$WORK/n187")"
TMPDIR="$WORK/gone187" "$MRW" -C "$R" check --full --then-sh 'touch m187' > "$WORK/o187" 2>&1; want 2 $? "the same check in human form exits 2"
{ [ ! -e "$R/m187" ] && grep -q 'then 1/1 --then-sh: touch m187 — NOT RUN' "$WORK/o187"; } \
  && ok "and names the step NOT RUN" || bad "the human report: $(head -c 400 "$WORK/o187")"
m check --full --json --then-sh 'touch m187' > "$WORK/j187p" 2> /dev/null; want 0 $? "the pair: with a temp directory the same check exits 0"
{ [ -e "$R/m187" ] && jq -e '.then.steps[0].status == "pass"' "$WORK/j187p" > /dev/null; } \
  && ok "and its step runs and passes" || bad "the pair: $(head -c 400 "$WORK/j187p")"

# 188. ADR-097: an undefined flag that is exactly the name of a subcommand of the
# command it was given to names that subcommand. `mrw --instructions` answered
# "flag provided but not defined: -instructions", true and no help to a caller
# who had met `mrw --version`. The pairs: an unknown name, and a sibling's name
# given to another command, keep their v1.31.0 wording; a padded attached value
# is still refused first (ADR-069); and the subcommand named is the one that works.
fixture
"$MRW" --instructions > "$WORK/o188" 2> "$WORK/e188"; want 2 $? "mrw --instructions exits 2"
{ [ ! -s "$WORK/o188" ] && grep -qxF 'mrw: --instructions is not a flag; the subcommand is `mrw instructions` (see: mrw --help)' "$WORK/e188"; } \
  && ok "and names the subcommand on stderr, with nothing on stdout" || bad "--instructions: stdout $(head -c 200 "$WORK/o188") stderr $(head -c 300 "$WORK/e188")"
"$MRW" --bogus188 > /dev/null 2> "$WORK/e188b"; want 2 $? "an unknown flag naming no subcommand exits 2"
{ grep -qxF 'mrw: flag provided but not defined: -bogus188 (see: mrw --help)' "$WORK/e188b" && ! grep -q 'is not a flag' "$WORK/e188b"; } \
  && ok "and keeps its v1.31.0 wording" || bad "--bogus188: $(head -c 300 "$WORK/e188b")"
"$MRW" -C "$R" check --stats > /dev/null 2> "$WORK/e188c"; want 2 $? "a sibling's name given to check exits 2"
grep -qxF 'mrw: flag provided but not defined: -stats (see: mrw check --help)' "$WORK/e188c" \
  && ok "and is worded as before: a sibling is not named" || bad "check --stats: $(head -c 300 "$WORK/e188c")"
"$MRW" '--instructions= ' > /dev/null 2> "$WORK/e188d"; want 2 $? "a padded attached value exits 2"
grep -q 'ends in whitespace' "$WORK/e188d" && ok "and is refused by ADR-069's guard before the parser" || bad "padded: $(head -c 300 "$WORK/e188d")"
"$MRW" instructions > "$WORK/o188e" 2> /dev/null; want 0 $? "mrw instructions exits 0"
[ -s "$WORK/o188e" ] && ok "and prints the contract: the named command is the one that works" || bad "mrw instructions printed nothing"

# 189. ADR-098: an ast_grep index pages with `after`, as a grep index does. An
# ast_grep answer too large to serve was an INDEX that said "send the SAME grep
# again", while `after` without grep was refused: page two of a structural
# search could not be reached. A fake ast-grep on PATH reports one hit in each of
# 400 files, in REVERSE path order, and a 6,000-byte ceiling forces the index;
# every page is followed until next_index is empty and each file must appear
# exactly once. Each $MRW run is bounded by the subprocess timeout, which kills
# with SIGKILL (Go ignores SIGALRM, contract.md). The pair: `after` with plain
# specs is still refused, and names both finders.
fixture
d189=$(mktemp -d)
python3 - "$R" "$d189/hits.json" <<'PY'
import json, sys
root, out = sys.argv[1], sys.argv[2]
hits = []
for i in range(399, -1, -1):
    open("%s/document%05d.csv" % (root, i), "w").write("x\nthe NEEDLE is here\n")
    hits.append({"file": "document%05d.csv" % i, "range": {"start": {"line": 1}, "end": {"line": 1}}})
json.dump(hits, open(out, "w"))
PY
printf '%s\n' '#!/bin/sh' "cat '$d189/hits.json'" > "$d189/ast-grep"
chmod +x "$d189/ast-grep"
PATH="$d189:$PATH" MRW_BIN="$MRW" python3 - "$R" <<'PY'
import json, subprocess, sys, os
root, mrw = sys.argv[1], os.environ["MRW_BIN"]
def call(args):
    req = {"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "mrw_read", "arguments": args}}
    p = subprocess.run([mrw, "--root", root, "mcp", "--max-result-chars", "6000"],
                       input=json.dumps(req) + "\n", capture_output=True, text=True, timeout=30)
    return json.loads(p.stdout.splitlines()[0])["result"]
res = call({"ast_grep": "NEEDLE"})
assert not res.get("isError"), "the first ast_grep call was refused: %.300s" % res
text = res["content"][0]["text"]
assert "SAME ast_grep again" in text and "next_index is empty" in text, "the index does not name ast_grep: %.400s" % text
st = json.loads(res["content"][1]["text"])
seen, nxt, pages = set(st["index"]), st["next_index"], 1
assert nxt, "the first index was not cut short"
while nxt:
    pages += 1
    assert pages <= 50, "following next_index did not end"
    r = call({"ast_grep": "NEEDLE", "after": nxt})
    assert not r.get("isError"), "after with ast_grep was refused: %.300s" % r
    s = json.loads(r["content"][1]["text"])
    page = s["index"] if "index" in s else list(s["observed"].keys())
    assert page and not (seen & set(page)), "page %d is empty or repeats a file" % pages
    seen |= set(page)
    nxt = s.get("next_index") or ""
assert len(seen) == 400, "paging yielded %d distinct files, want 400" % len(seen)
r = call({"specs": ["document00000.csv"], "after": "x"})
t = r["content"][0]["text"]
assert r.get("isError") and "grep or ast_grep" in t, "after with plain specs: %.300s" % r
PY
[ $? -eq 0 ] && ok "an ast_grep index pages to the end over the built binary, and after with plain specs is refused naming both finders" \
             || bad "the ast_grep index does not page, repeats or loses a file, or after with plain specs is accepted"
rm -rf "$d189"

# 190. ADR-098: the handshake says what the code does. The server told a host
# to repeat "until next_index is absent" (it is empty on the last page), that
# exclude was "only meaningful" with a finder (it is refused without one), and
# that guards are "checked on every op" (create, unlink and rename refuse
# anchor= and lines=). The corrected sentences are read from the built binary's
# initialize and tools/list; the pair: none of the old claims is served, while
# the true `next_read is absent` stays.
fixture
MRW_BIN="$MRW" python3 - "$R" <<'PY'
import json, subprocess, sys, os
# Driven from here, not through `bounded`: a backgrounded command in a
# non-interactive shell reads /dev/null, not the pipe. The timeout kills with
# SIGKILL, which Go cannot ignore (contract.md).
reqs = [{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-06-18",
         "capabilities": {}, "clientInfo": {"name": "c", "version": "1"}}},
        {"jsonrpc": "2.0", "id": 2, "method": "tools/list"}]
p = subprocess.run([os.environ["MRW_BIN"], "--root", sys.argv[1], "mcp"], capture_output=True, text=True,
                   input="".join(json.dumps(r) + "\n" for r in reqs), timeout=30)
lines = [json.loads(l) for l in p.stdout.splitlines() if l.startswith("{")]
ins = next(m["result"]["instructions"] for m in lines if m.get("id") == 1)
tl = next(m["result"]["tools"] for m in lines if m.get("id") == 2)
text = ins + json.dumps(tl)
for s in ["`next_index` is empty", "Refused without `grep` or `ast_grep`", "refused on create, unlink and rename",
          "rename takes address `-` and one body line", "unless the range ends at the file's last line", "next_read is absent"]:
    assert s in text, "not served: %r" % s
for s in ["next_index` is absent", "Only meaningful with", "Guards, checked on every op"]:
    assert s not in text, "still served: %r" % s
PY
[ $? -eq 0 ] && ok "initialize and tools/list serve the corrected sentences and none of the old claims" \
             || bad "the handshake still serves a claim the code contradicts, or lost a correction"

# 191. ADR-099: `mrw check --full PATH` is refused, and runs nothing. It ran the
# whole-project check and dropped PATH in silence (--full set paths to nil after
# reading them). The declared check touches a marker, so a check that ran leaves
# it. The pair: --full alone runs the check.
fixture
printf '{"check":"touch marker191"}\n' > "$R/.quality-harness.json"
out=$(m check --full a.go 2> "$WORK/e191"); want 2 $? "check --full a.go exits 2"
{ [ -z "$out" ] && grep -q 'it takes no PATH' "$WORK/e191" && [ ! -e "$R/marker191" ]; } \
  && ok "and names the fix on stderr, with nothing on stdout and no check run" || bad "check --full a.go: stdout $out, stderr $(cat "$WORK/e191")"
m check --full > /dev/null 2>&1; want 0 $? "the pair: check --full alone exits 0"
[ -e "$R/marker191" ] && ok "and runs the whole-project check" || bad "check --full alone ran nothing"

# 192. ADR-099: `help` is a path to read, write and check. urfave's per-command
# `help` subcommand (alias `h`) won over a file of that name, even after `--`.
# The plan and the check are proved on disk: the plan file named help is applied
# (write resolves it from the working directory), and a scoped check receives
# help as {files}. The pair: the --help flag still prints read's help, and
# `mrw help` still lists the commands.
fixture
printf 'body of help192\n' > "$R/help"; printf 'body of h192\n' > "$R/h"
out=$(m read help 2>&1); want 0 $? "read help exits 0"
grep -q 'body of help192' <<<"$out" && ok "and serves the file named help" || bad "read help: $(head -c 300 <<<"$out")"
out=$(m read -- h 2>&1); want 0 $? "read -- h exits 0"
grep -q 'body of h192' <<<"$out" && ok "and serves the file named h" || bad "read -- h: $(head -c 300 <<<"$out")"
printf '@@ made192.txt 0 create\nmade\n' > "$R/help"
( cd "$R" && "$MRW" write --no-check help > /dev/null 2>&1 ); want 0 $? "write help, run in the checkout, exits 0"
[ -e "$R/made192.txt" ] && ok "and applies the plan in the file named help" || bad "write help applied nothing"
printf '{"check":"true","scoped_check":"echo FILES={files} > scoped192"}\n' > "$R/.quality-harness.json"
m check help > /dev/null 2>&1; want 0 $? "check help exits 0"
grep -qx 'FILES=help' "$R/scoped192" && ok "and scopes the check to the file named help" || bad "check help: $(cat "$R/scoped192" 2>&1)"
out=$(m read --help 2>&1); want 0 $? "the pair: read --help exits 0"
grep -q 'USAGE' <<<"$out" && ok "and still prints read's help" || bad "read --help: $(head -c 300 <<<"$out")"
out=$("$MRW" help 2>&1); want 0 $? "mrw help exits 0"
grep -q '^COMMANDS:' <<<"$out" && ok "and still lists the commands" || bad "mrw help: $(head -c 300 <<<"$out")"

# 193. ADR-100: every `mrw check --json` refusal is one document. A path outside
# the root, a path not there and --full with a PATH printed only a message on
# stderr under --json, so a consumer parsing stdout read nothing. Each is now
# {"error": ...}, exit 2, with no exit_code to read a verdict out of. The pair:
# the same refusal without --json prints nothing on stdout.
fixture
printf '{"check":"touch marker193"}\n' > "$R/.quality-harness.json"
for a in ../outside nosuchdir '--full a.go'; do
  # $a is split on purpose: '--full a.go' is two arguments.
  # shellcheck disable=SC2086
  out=$(m check --json $a 2>/dev/null); want 2 $? "check --json $a exits 2"
  jq -se 'length == 1 and (.[0] | type == "object" and keys == ["error"] and (.error | length > 0))' <<<"$out" > /dev/null \
    && ok "and prints exactly one document holding only the error, so no exit_code" || bad "check --json $a: $out"
  # shellcheck disable=SC2086
  out=$(m check $a 2>/dev/null); want 2 $? "the pair: check $a exits 2"
  [ -z "$out" ] && ok "and prints nothing on stdout" || bad "check $a wrote stdout: $out"
done
[ ! -e "$R/marker193" ] && ok "and no refusal ran the check" || bad "a refusal ran the check"
out=$("$MRW" -C "$R" check --json --then-sh='true ' 2>/dev/null); want 2 $? "the boundary: a padded attached flag value is refused in main, exit 2"
[ -z "$out" ] && ok "and, refused before any flag is parsed, prints nothing on stdout" || bad "pre-parse refusal wrote stdout: $out"

# 194. ADR-100: at MRW_STEP_DEPTH 8 every form of mrw check that reaches the
# Action hears ADR-095's depth refusal first. `check --full a.go` answered with
# the --full refusal because the depth check came after it. The pair: at 7 it
# names --full.
fixture
out=$(env MRW_STEP_DEPTH=8 "$MRW" -C "$R" check --full a.go 2>&1); want 2 $? "check --full a.go at depth 8 exits 2"
grep -q 'MRW_STEP_DEPTH' <<<"$out" && ok "and names the depth limit" || bad "at 8: $out"
out=$(env MRW_STEP_DEPTH=7 "$MRW" -C "$R" check --full a.go 2>&1); want 2 $? "the pair: at depth 7 it exits 2"
grep -q 'it takes no PATH' <<<"$out" && ok "and names --full" || bad "at 7: $out"

# 195. ADR-101: a check that did not run has no exit code of zero. In a tree with
# no harness and no go.mod, `check --json` and the check block of
# `write --check --json` said "exit_code": 0 beside "ran": false, so a consumer
# reading exit_code alone read a pass. Both now say -1. The pair: a declared
# passing check says ran true and exit_code 0.
fixture
rm -f "$R/go.mod"
m check --json --full > "$WORK/j195" 2> /dev/null; want 2 $? "check --json with no check exits 2"
jq -se 'length == 1 and .[0].ran == false and .[0].exit_code == -1' "$WORK/j195" > /dev/null \
  && ok "and its receipt says ran false, exit_code -1" || bad "check --json: $(head -c 300 "$WORK/j195")"
printf '@@ a.go 3 replace\nfunc A() int { return 195 }\n' > "$R/p195.mrw"
( cd "$R" && "$MRW" write --check --json p195.mrw ) > "$WORK/w195" 2> /dev/null; want 2 $? "write --check --json with no check exits 2"
jq -se 'length == 1 and .[0].applied == true and .[0].check.ran == false and .[0].check.exit_code == -1' "$WORK/w195" > /dev/null \
  && ok "and its check block says ran false, exit_code -1" || bad "write --check --json: $(head -c 400 "$WORK/w195")"
printf '{"check":"true"}\n' > "$R/.quality-harness.json"
m check --json --full > "$WORK/p195" 2> /dev/null; want 0 $? "the pair: a declared passing check exits 0"
jq -se 'length == 1 and .[0].ran == true and .[0].exit_code == 0' "$WORK/p195" > /dev/null \
  && ok "and says ran true, exit_code 0" || bad "the pair: $(head -c 300 "$WORK/p195")"

# 196. ADR-102: a commit that failed after a file landed is one partially_applied,
# counted in landed, and the file it wrote takes the next write without a re-read.
# No seam: content renames commit before path ops, and an unlink in a read-only
# directory fails at commit. It was one refused_apply, and the next write to the
# landed file was refused as changed since read. The pair: with the directory
# writable the unlink applies. uid 0 ignores the permission bits, so it skips there.
fixture
mkdir -p "$R/d"; printf 'x\n' > "$R/d/x.txt"; printf 'a\n' > "$R/a.txt"
m read a.txt d/x.txt >"$WORK/served.out"
chmod 555 "$R/d"
if [ -w "$R/d" ]; then
  chmod 755 "$R/d"
  skip "a partial commit is partially_applied (permission bits not enforced here — running as root?)"
else
  printf '@@ a.txt 1 replace\nA\n@@ d/x.txt - unlink\n' | m write --no-check - > "$WORK/o196" 2>&1; want 2 $? "a commit stopped by the read-only directory exits 2"
  { grep -q 'PARTIALLY APPLIED' "$WORK/o196" && grep -qx A "$R/a.txt" && [ -e "$R/d/x.txt" ]; } \
    && ok "and says PARTIALLY APPLIED, a.txt written and d/x.txt kept" || bad "partial: $(head -c 400 "$WORK/o196")"
  m stats --json > "$WORK/s196" 2>/dev/null
  jq -se 'length == 1 and (.[0] | .counts.partially_applied == 1 and .counts.refused_apply == 0 and .landed == 1)' "$WORK/s196" > /dev/null \
    && ok "stats counts it partially_applied, and landed" || bad "stats: $(head -c 400 "$WORK/s196")"
  printf '@@ a.txt 1 replace\nAA\n' | m write --no-check - > /dev/null 2>&1; want 0 $? "the file the partial commit wrote takes the next write without a re-read"
  chmod 755 "$R/d"
  printf '@@ d/x.txt - unlink\n' | m write --no-check - > /dev/null 2>&1; want 0 $? "the pair: with d writable the unlink applies"
fi

# 197. ADR-102: mrw_write answers a write that landed and could not save its
# ledger with its receipt — isError, applied true, and error — where it answered
# a bare JSON-RPC error a client could not tell from a write that did nothing.
# The pair: with the ledger writable the receipt carries no error.
fixture
m read a.go >"$WORK/served.out"
led197="$(m seen | head -1)/seen"
chmod 444 "$led197"
if [ -w "$led197" ]; then
  chmod 600 "$led197"
  skip "mrw_write sends the receipt on a ledger failure (permission bits not enforced here — running as root?)"
else
  req=$(printf '@@ a.go 3 replace\nfunc A() int { return 197 }\n' | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read()}}}))')
  printf '%s\n' "$req" | "$MRW" -C "$R" mcp > "$WORK/j197" 2> /dev/null
  { grep -q 'return 197' "$R/a.go" && jq -se 'length == 1 and (.[0] | .error == null and .result.isError == true and .result.structuredContent.applied == true and (.result.structuredContent.error | length > 0))' "$WORK/j197" > /dev/null; } \
    && ok "the write landed and the answer is its receipt, applied, naming the ledger error" || bad "ledger failure over MCP: $(head -c 400 "$WORK/j197")"
  chmod 600 "$led197"
  m read a.go >"$WORK/served.out"
  req=$(printf '@@ a.go 3 replace\nfunc A() int { return 1970 }\n' | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read()}}}))')
  printf '%s\n' "$req" | "$MRW" -C "$R" mcp > "$WORK/p197" 2> /dev/null
  jq -se 'length == 1 and (.[0] | .result.structuredContent.applied == true and (.result.structuredContent | has("error") | not))' "$WORK/p197" > /dev/null \
    && ok "the pair: with the ledger writable the receipt has no error" || bad "the pair: $(head -c 400 "$WORK/p197")"
fi

# 198. ADR-103: the filesystem root is a root. With `--root /` every path was
# refused as outside it — the containment test asked for the prefix `//`. A file
# beneath it, named root-relative, is served. The pair: a path that leaves a real
# root is still refused.
fixture
f198="$(cd "$R" && pwd -P)/a.go"
out=$("$MRW" --root / read "${f198#/}" 2>&1); want 0 $? "--root / read <root-relative path> exits 0"
grep -q 'func A' <<<"$out" && ok "and serves the file" || bad "--root /: $(head -c 300 <<<"$out")"
printf 'secret198\n' > "$WORK/outside198.txt"
out=$(m read ../outside198.txt 2>&1); want 1 $? "the pair: a path out of a real root is still refused, the file there notwithstanding"
{ grep -q 'outside the root' <<<"$out" && ! grep -q secret198 <<<"$out"; } && ok "and names the boundary without serving it" || bad "escape: $(head -c 300 <<<"$out")"

# 199. ADR-104: mrw mcp reads a request line up to 64 MiB. It read a line of any
# length whole; a longer one is answered -32600 (id null) naming the limit, the
# rest of the line is discarded, and the next line is served.
fixture
python3 -c 'import sys; sys.stdout.write("{\"pad\":\"" + "a" * (65 << 20) + "\"}\n" + "{\"jsonrpc\":\"2.0\",\"id\":9,\"method\":\"tools/list\"}\n")' \
  | "$MRW" -C "$R" mcp > "$WORK/j199" 2> /dev/null; want 0 $? "the server survives a 65 MiB line and ends at EOF"
jq -se 'length == 2 and (.[0] | .id == null and .error.code == -32600 and (.error.message | test("67108864"))) and .[1].id == 9 and (.[1].result.tools | length > 0)' "$WORK/j199" > /dev/null \
  && ok "the long line gets -32600 naming the limit, and the next request is answered" || bad "oversized request: $(head -c 400 "$WORK/j199")"

# 200. ADR-104: a check's tail keeps at most 4 KiB of a line. The tail read the
# whole log; a longer line now ends " … [N more bytes]" (the log file keeps it).
# The pair: a short last line is shown whole.
fixture
printf '%s\n' 'awk '"'"'BEGIN { s = sprintf("%10000s", ""); gsub(/ /, "y", s); print s; exit 1 }'"'" > "$R/long200.sh"
printf '{"check":"sh long200.sh"}\n' > "$R/.quality-harness.json"
m check --json --full > "$WORK/j200" 2> /dev/null; want 3 $? "a failing check with a 10,000-character last line exits 3"
jq -se 'length == 1 and (.[0].tail[-1] | (length < 4200) and test("more bytes\\]$"))' "$WORK/j200" > /dev/null \
  && ok "and its tail shows the line capped with the count of what was cut" || bad "long tail line: $(head -c 300 "$WORK/j200")"
printf '{"check":"echo short; exit 1"}\n' > "$R/.quality-harness.json"
m check --json --full > "$WORK/p200" 2> /dev/null; want 3 $? "the pair: a failing check with a short last line exits 3"
jq -se 'length == 1 and .[0].tail[-1] == "short"' "$WORK/p200" > /dev/null && ok "and its tail shows it whole" || bad "short tail: $(head -c 300 "$WORK/p200")"
printf '{"check":"sh long200.sh; exit 0"}\n' > "$R/.quality-harness.json"
m check --json --full > "$WORK/k200" 2> /dev/null; want 0 $? "a passing check with a 10,000-character line exits 0"
jq -se 'length == 1 and (.[0].output_file | length > 0)' "$WORK/k200" > /dev/null && [ -s "$(jq -r .output_file "$WORK/k200")" ] \
  && ok "and keeps its log, which the cut line's marker points at" || bad "passing long line: $(head -c 300 "$WORK/k200")"
printf '{"check":"exit 0"}\n' > "$R/.quality-harness.json"
out=$(m check --full --then-sh 'sh long200.sh; exit 0' 2>&1); want 0 $? "a passing step with a 10,000-character line exits 0"
log=$(grep -A3 '^then 1/1' <<<"$out" | sed -n 's/^full output: //p' | head -1)
{ [ -n "$log" ] && [ -s "$log" ]; } && ok "and its receipt names the log it kept" || bad "passing step, no log named: $(head -c 300 <<<"$out")"

# 201. ADR-105: a state file is replaced whole, never rewritten in place. The
# ledger was truncated and rewritten, so a reader or a killed run could see part
# of it — a lost licence. A write now replaces it by rename: its inode changes
# and no temp stays beside it. The pair: the replaced ledger still refuses an
# edit to a file nobody read, and licenses the next write to one that was.
fixture
printf 'b\n' > "$R/b201.txt"
m read a.go >"$WORK/served.out"
printf '@@ b201.txt 1 replace\nB\n' > "$R/u201.mrw"
m write --no-check "$R/u201.mrw" > /dev/null 2>&1; want 1 $? "an edit to a file nobody read is refused"
[ "$(cat "$R/b201.txt")" = b ] && ok "and leaves it unchanged" || bad "the unread file changed: $(cat "$R/b201.txt")"
sd201=$(m seen | head -1)
i201=$(ls -i "$sd201/seen" 2>/dev/null | awk '{print $1}')
printf '@@ a.go 1 replace\npackage a\n' > "$R/p201.mrw"
m write --no-check "$R/p201.mrw" > /dev/null 2>&1; want 0 $? "a write that saves the ledger exits 0"
j201=$(ls -i "$sd201/seen" 2>/dev/null | awk '{print $1}')
{ [ -n "$i201" ] && [ -n "$j201" ] && [ "$i201" != "$j201" ]; } && ok "and the ledger was replaced by rename (inode $i201 -> $j201)" || bad "the ledger was rewritten in place: inode '$i201' -> '$j201'"
ls -A "$sd201" | grep -q '\.tmp-' && bad "a temp state file was left: $(ls -A "$sd201" | tr '\n' ' ')" || ok "and no temp file stays beside it"
printf '@@ a.go 1 replace\npackage a // 201\n' > "$R/q201.mrw"
m write --no-check "$R/q201.mrw" > /dev/null 2>&1; want 0 $? "the pair: the replaced ledger licenses the next write"

# 202. ADR-108: an MCP checkpoint brackets consecutive served lines only. A
# pattern matching lines 1 and 100 of one file served them under one header as
# ONE span, "open lines 1-100", whose acknowledgement licensed the 98 lines
# between that nobody saw. Each run now gets its own checkpoint. The pair: a
# consecutive run still gets one.
fixture
seq 1 100 > "$R/g202.txt"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["g202.txt:/^(1|100)$/"]}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["g202.txt:7-9"]}}}' \
  | "$MRW" -C "$R" mcp > "$WORK/j202" 2> /dev/null; want 0 $? "the server answers two reads and ends at EOF"
jq -se '.[0].result.content[0].text | test("open lines 1-1 \\(1 lines? follows?\\)") and test("open lines 100-100 ") and (test("open lines 1-100") | not)' "$WORK/j202" > /dev/null \
  && ok "lines 1 and 100 get a checkpoint each, none spanning the gap" || bad "gap checkpoint: $(head -c 600 "$WORK/j202")"
jq -se '.[1].result.content[0].text | test("open lines 7-9 ") and ([scan("-- ck [^ ]+ open")] | length == 1)' "$WORK/j202" > /dev/null \
  && ok "the pair: lines 7-9 get one checkpoint" || bad "run checkpoint: $(head -c 600 "$WORK/j202")"

# 203. ADR-108: a body=@ file keeps the target's line endings. Its lines were
# split at \n alone, so a CRLF body kept its \r and a CRLF target came out
# \r\r\n. The body file is now split like every other text mrw reads. The pair:
# an LF body into an LF target is unchanged.
fixture
printf 'a\r\nb\r\n' > "$R/t203.txt"; printf 'X\r\n' > "$R/b203.txt"
printf 'a\nb\n' > "$R/l203.txt"; printf 'Y\n' > "$R/c203.txt"
m read t203.txt l203.txt >"$WORK/served.out"
printf '@@ t203.txt 1 replace body=@b203.txt\n@@ l203.txt 1 replace body=@c203.txt\n' > "$R/p203.mrw"
m write --no-check "$R/p203.mrw" > /dev/null 2>&1; want 0 $? "a write with two body=@ hunks exits 0"
[ "$(od -An -c "$R/t203.txt" | tr -s ' ')" = "$(printf 'X\r\nb\r\n' | od -An -c | tr -s ' ')" ] \
  && ok "the CRLF target holds X CRLF b CRLF, no doubled CR" || bad "CRLF body: $(od -An -c "$R/t203.txt")"
[ "$(od -An -c "$R/l203.txt" | tr -s ' ')" = "$(printf 'Y\nb\n' | od -An -c | tr -s ' ')" ] \
  && ok "the pair: the LF target holds Y LF b LF" || bad "LF body: $(od -An -c "$R/l203.txt")"

# 204. ADR-108: a first page counts only what read would serve. read refuses a
# FIFO, but under a ceiling smaller than that refusal the server tried a first
# page, whose line count opened the FIFO and waited for a writer: the server
# hung. It answers. The pair: under an ordinary ceiling the FIFO is refused.
fixture
mkfifo "$R/p204"
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["p204"]}}}' > "$WORK/q204"
bounded 10 "$WORK/o204" sh -c 'exec "$1" -C "$2" mcp --max-result-chars 16 < "$3"' _ "$MRW" "$R" "$WORK/q204"; want 0 $? "a FIFO read under a 16-character ceiling ends at EOF instead of hanging"
grep -q '"id":1' "$WORK/o204" && ok "and the request is answered" || bad "FIFO under a small ceiling: $(head -c 400 "$WORK/o204")"
bounded 10 "$WORK/k204" sh -c 'exec "$1" -C "$2" mcp < "$3"' _ "$MRW" "$R" "$WORK/q204"; want 0 $? "the pair: the same read under the default ceiling ends at EOF"
grep '"id":1' "$WORK/k204" | jq -e '.result.isError == true' > /dev/null && ok "and refuses the FIFO" || bad "FIFO refusal: $(head -c 400 "$WORK/k204")"

# 205. ADR-108: the working set reads back every line it saves. A 70,000-byte
# note was saved and then failed every load, `mrw iter clear` included. Save
# refuses it, naming the size; a line that long already in the file, from an
# older binary, is skipped and the rest loads. The pair: a short note saves.
fixture
long205=$(python3 -c 'print("n" * 70000)')
out=$(m iter note "$long205" 2>&1); want 2 $? "a 70,000-byte note is refused"
grep -q '70000 bytes' <<<"$out" && ok "and the refusal names its size" || bad "long note: $(head -c 300 <<<"$out")"
m iter note short > /dev/null 2>&1; want 0 $? "the pair: a short note saves"
sd205=$(m seen | head -1)
{ printf '# ok\na.go\n'; python3 -c 'print("x" * 70000)'; printf 'b.go\n'; } > "$sd205/iteration"
out=$(m iter 2>&1); want 0 $? "a working set holding a 70,000-byte line loads"
{ grep -q 'a\.go' <<<"$out" && grep -q 'b\.go' <<<"$out"; } && ok "and keeps the entries around it" || bad "legacy working set: $(head -c 300 <<<"$out")"
m iter clear > /dev/null 2>&1; want 0 $? "and clears"

# 206. ADR-108: a permission issued under the old checkpoint rules is not
# honoured. Up to v1.37.1 an MCP checkpoint spanned a sparse read's gaps, so a
# #mrw-seen v2 span may cover lines no read served; it licensed them after an
# upgrade. A v2 ledger is now discarded, and the notice says so. The pair: a
# read of the line licenses the same write.
fixture
seq 1 100 > "$R/a206.txt"
m read a206.txt:1 >"$WORK/served.out"
sd206=$(m seen | head -1)
sha206=$(awk '$3 == "a206.txt" { print $1 }' "$sd206/seen")
printf '#mrw-seen v2\n%s  1-100  a206.txt\n' "$sha206" > "$sd206/seen"
printf '@@ a206.txt 50 replace\nfifty\n' > "$R/p206.mrw"
out=$(m write --no-check "$R/p206.mrw" 2>&1); want 1 $? "a write to line 50, licensed only by a v2 span 1-100, is refused"
grep -q 'written by an older mrw' <<<"$out" && ok "and the notice says the ledger was discarded" || bad "v2 ledger: $(head -c 300 <<<"$out")"
m read a206.txt:50 >"$WORK/served.out"
m write --no-check "$R/p206.mrw" > /dev/null 2>&1; want 0 $? "the pair: after a read of line 50 the write applies"

# 207. ADR-109: a FIFO named .quality-harness.json is refused, not waited on.
# The config was read with os.ReadFile, so a FIFO there hung `mrw check` and
# every write whose check was due until something wrote to the pipe. It is
# refused as not a regular file, exit 2. The pair: a regular config runs.
fixture
rm -f "$R/.quality-harness.json"; mkfifo "$R/.quality-harness.json"
bounded 10 "$WORK/o207" "$MRW" -C "$R" check; want 2 $? "mrw check with a FIFO config exits 2 instead of hanging"
grep -q 'not a regular file' "$WORK/o207" && ok "and names why" || bad "FIFO config: $(head -c 300 "$WORK/o207")"
rm -f "$R/.quality-harness.json"; printf '{"check":"exit 0"}\n' > "$R/.quality-harness.json"
bounded 10 "$WORK/k207" "$MRW" -C "$R" check; want 0 $? "the pair: a regular config's check runs and passes"

# 208. ADR-109: legacy state in the checkout is not waited on. Before ADR-004 the
# ledger lived at .mrw/seen in the checkout, and mrw still migrates it at every
# start and loads it when the state directory has none, both with a blocking
# open: a FIFO there hung every command. The read runs, and the FIFO is left
# where it is rather than migrated. The pair: the read served the file.
fixture
mkdir -p "$R/.mrw"; mkfifo "$R/.mrw/seen"
( cd "$R" && bounded 10 "$WORK/o208" "$MRW" read a.go ); want 0 $? "a read with a FIFO at .mrw/seen exits 0 instead of hanging"
[ -p "$R/.mrw/seen" ] && ok "and the FIFO is left where it is, not migrated" || bad "the legacy FIFO was moved or replaced"
grep -q '^==> a.go ' "$WORK/o208" && ok "the pair: the read served the file" || bad "FIFO legacy ledger: $(head -c 300 "$WORK/o208")"
rm -f "$R/.mrw/seen"

# 209. ADR-110: a writer waits a bounded time for the write lock. A write that
# found it held waited for ever, in silence; it is refused after
# MRW_WRITE_LOCK_TIMEOUT seconds, exit 2, nothing applied, naming the lock. The
# pair: once the holder lets go, the same write applies.
fixture
m read a.go >"$WORK/served.out"
sd209=$(m seen | head -1)
python3 -c 'import fcntl, sys, time
f = open(sys.argv[1], "a"); fcntl.flock(f, fcntl.LOCK_EX); open(sys.argv[2], "w").close(); time.sleep(60)' "$sd209/seen.write.lock" "$WORK/ready209" &
holder209=$!
for _ in $(seq 1 50); do [ -e "$WORK/ready209" ] && break; sleep 0.1; done
cp "$R/a.go" "$WORK/a209.before"
printf '@@ a.go 1 replace\npackage a // 209\n' > "$R/p209.mrw"
bounded 15 "$WORK/o209" env MRW_WRITE_LOCK_TIMEOUT=1 "$MRW" -C "$R" write --no-check "$R/p209.mrw"; want 2 $? "a write that finds the write lock held is refused after MRW_WRITE_LOCK_TIMEOUT"
grep -q 'write lock' "$WORK/o209" && grep -q 'nothing was applied' "$WORK/o209" && ok "and says the lock was held and nothing was applied" || bad "held write lock: $(head -c 300 "$WORK/o209")"
cmp -s "$R/a.go" "$WORK/a209.before" && ok "and the file is unchanged" || bad "a.go changed under a held write lock"
kill "$holder209" 2>/dev/null; wait "$holder209" 2>/dev/null
m write --no-check "$R/p209.mrw" > /dev/null 2>&1; want 0 $? "the pair: once the holder lets go, the same write applies"

# 210. ADR-111: every key the built binary's `write --json` prints is one
# docs/receipts.txt lists, so a caller may rely on it and a removal cannot land
# unseen. The pair: the same lookup reports a key the file does not list.
fixture
m read a.go >"$WORK/served.out"
printf '@@ a.go 1 replace\npackage a // 210\n' > "$R/p210.mrw"
m write --no-check --json "$R/p210.mrw" > "$WORK/j210" 2>/dev/null; want 0 $? "a write --json exits 0"
keys210() { jq -r 'paths | select(.[-1] | type != "number") | map(if type == "number" then "[]" else "." + . end) | join("") | ltrimstr(".") | gsub("\\.\\[\\]"; "[]")' "$1" | sort -u; }
rcpt210="$SRC/docs/receipts.txt"
unlisted210() { while IFS= read -r k; do grep -qxF "write $k" "$rcpt210" || echo "$k"; done; }
miss210=$(keys210 "$WORK/j210" | unlisted210)
[ -z "$miss210" ] && [ -n "$(keys210 "$WORK/j210")" ] && ok "and every key it printed is listed in docs/receipts.txt" || bad "write --json keys not in docs/receipts.txt: $miss210"
printf '{"not_a_field":1}\n' > "$WORK/k210"
[ "$(keys210 "$WORK/k210" | unlisted210)" = "not_a_field" ] && ok "the pair: a key the file does not list is reported" || bad "the lookup did not report an unlisted key"

# 211. ADR-112: a write names a file that changed while its check ran. The check
# runs after the write lock is released, so the check itself — or another writer —
# can change a file the write landed, and the verdict was about a tree that had
# moved. Advisory: the exit code stays the check's. The pair: a check that
# changes nothing names nothing.
fixture
printf '{"check":"printf x >> a.go"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 1 replace\npackage a // 211\n' > "$R/p211.mrw"
out=$(m write --check "$R/p211.mrw" 2>&1); want 0 $? "a write whose check appends to the file it wrote exits 0"
grep -q '^drift: a.go changed while the check ran' <<<"$out" && ok "and names the file as drift" || bad "no drift line: $(head -c 300 <<<"$out")"
m read a.go >"$WORK/served.out"
m write --check --json "$R/p211.mrw" > "$WORK/j211" 2>/dev/null; want 0 $? "the same write under --json exits 0"
jq -e '.drift == ["a.go"]' "$WORK/j211" > /dev/null && ok "and its receipt carries drift [a.go]" || bad "drift receipt: $(head -c 300 "$WORK/j211")"
printf '{"check":"exit 0"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
m write --check --json "$R/p211.mrw" > "$WORK/k211" 2>/dev/null; want 0 $? "the pair: a write whose check changes nothing exits 0"
jq -e 'has("drift") | not' "$WORK/k211" > /dev/null && ok "and its receipt carries no drift" || bad "drift with an idle check: $(head -c 300 "$WORK/k211")"

# 212. B1 (BACKLOG "From ADR-108"): a foreign-format plan is held to the read it
# was written against. The read ledger records the file's whole sha, and apply
# refuses a file changed since, so --format=apply_patch and
# --format=search_replace both refuse a file edited after its read, exit 1,
# leaving it as it was. The pair: once the file is read again, the plan applies.
fixture
printf 'one\ntwo\nthree\n' > "$R/f212.txt"
m read f212.txt >"$WORK/served.out"
printf 'one\ntwo\nthree\nfour\n' > "$R/f212.txt"
printf '*** Begin Patch\n*** Update File: f212.txt\n@@\n one\n-two\n+TWO\n three\n*** End Patch\n' > "$R/p212.patch"
printf 'f212.txt\n<<<<<<< SEARCH\ntwo\n=======\nTWO\n>>>>>>> REPLACE\n' > "$R/p212.sr"
for fmt in apply_patch:p212.patch search_replace:p212.sr; do
  out=$(m write --no-check --format="${fmt%%:*}" "$R/${fmt#*:}" 2>&1); want 1 $? "${fmt%%:*} over a file changed after its read is refused"
  grep -q 'changed since mrw last saw it' <<<"$out" && ok "and names why" || bad "${fmt%%:*}: $(head -c 300 <<<"$out")"
done
[ "$(cat "$R/f212.txt")" = "$(printf 'one\ntwo\nthree\nfour')" ] && ok "and the file is as it was" || bad "f212.txt changed: $(cat "$R/f212.txt")"
m read f212.txt >"$WORK/served.out"
m write --no-check --format=apply_patch "$R/p212.patch" > /dev/null 2>&1; want 0 $? "the pair: after a fresh read the apply_patch plan applies"

# 213. ADR-113: mrw_write runs the project's check after a write that touches
# code and returns its verdict in the receipt, by the rule `mrw write` uses. A
# check that ran and failed leaves the write applied and the call not isError,
# and is counted failed_check. The pair: check: false runs none, and applies.
fixture
printf '{"check":"echo boom213; exit 3"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
req213() {
  python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":json.loads(sys.argv[1])}}))' "$1"
}
out=$(req213 '{"plan":"@@ a.go 3 replace\nfunc A() int { return 213 }\n"}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && ok "mrw_write runs the check: ran, exit 3, applied, not isError" || bad "mrw_write did not report its failed check: $(head -c 400 <<<"$out")"
import json, sys
r = json.loads(sys.argv[1])["result"]
sc = r["structuredContent"]
c = sc.get("check") or {}
sys.exit(0 if c.get("ran") is True and c.get("exit_code") == 3 and sc.get("applied") is True and not r.get("isError") else 1)
PY
m stats --json | grep -q '"failed_check": 1' && ok "and counts it failed_check" || bad "the failed check was not counted: $(m stats --json | head -c 300)"
out=$(req213 '{"plan":"@@ a.go 3 replace\nfunc A() int { return 2130 }\n","check":false}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && ok "the pair: check: false runs none, and the write applies" || bad "check: false: $(head -c 400 <<<"$out")"
import json, sys
sc = json.loads(sys.argv[1])["result"]["structuredContent"]
sys.exit(0 if "check" not in sc and sc.get("applied") is True else 1)
PY
# 214. ADR-114: a file is edited and renamed in one plan. apply_patch's Update
# File + Move to + a hunk lands the edit at the destination and removes the
# source, exit 0, each path named once in the receipt. The pair, which the old
# binary cannot pass: a move onto a destination that exists refuses the whole
# plan, exit 1, and the source is not edited — the edit and the move are one.
fixture
printf 'stay\n' > "$R/m214.txt"
m read m214.txt >"$WORK/served.out"
printf '%s\n' '*** Begin Patch' '*** Update File: m214.txt' '*** Move to: moved/n214.txt' '@@' '-stay' '+gone' '*** End Patch' > "$R/p214.patch"
out=$(m write --no-check --format=apply_patch "$R/p214.patch" 2>&1); want 0 $? "a move with a hunk applies in one plan"
{ [ "$(cat "$R/moved/n214.txt" 2>/dev/null)" = gone ] && [ ! -e "$R/m214.txt" ]; } && ok "and the edit is at the destination, the source gone" || bad "move+edit left: $(ls -R "$R" | head -20)"
[ "$(grep -cE '^(removed|wrote|created) m214\.txt' <<<"$out")" = 1 ] && grep -q '^removed m214.txt .*renamed to moved/n214.txt' <<<"$out" && ok "and the receipt names the source once, renamed" || bad "receipt: $out"
printf 'stay\n' > "$R/k214.txt"
printf 'taken\n' > "$R/l214.txt"
m read k214.txt >"$WORK/served.out"
printf '%s\n' '*** Begin Patch' '*** Update File: k214.txt' '*** Move to: l214.txt' '@@' '-stay' '+gone' '*** End Patch' > "$R/q214.patch"
m write --no-check --format=apply_patch "$R/q214.patch" > /dev/null 2>&1; want 1 $? "the pair: a move onto a destination that exists is refused"
{ [ "$(cat "$R/k214.txt")" = stay ] && [ "$(cat "$R/l214.txt")" = taken ]; } && ok "and the source is not edited, the destination untouched" || bad "a refused move changed the tree"

# 216. ADR-116: inside a checkout a --grep walk skips what .gitignore ignores,
# and any walk skips a binary file, and says how many with the flag that walks
# them. The pair: --no-ignore serves both, and a path named is served anyway.
fixture
mkdir -p "$R/.git" "$R/gen216"
printf 'gen216/\n' > "$R/.gitignore"
printf 'NEEDLE216\n' > "$R/gen216/x.txt"
printf 'NEEDLE216\n' > "$R/s216.txt"
printf 'NEEDLE216\0\n' > "$R/b216.dat"
out=$(m read --grep NEEDLE216 2>&1)
if grep -q 's216.txt' <<<"$out" && ! grep -q 'gen216/x.txt' <<<"$out" && ! grep -q 'b216.dat' <<<"$out" \
   && tail -n 1 <<<"$out" | grep -q -- '^-- skipped:.*--no-ignore'; then
  ok "--grep skips an ignored directory and a binary file, and says so as its last line"
else bad "--grep served what .gitignore ignores, or skipped it silently: $(head -c 400 <<<"$out")"; fi
out=$(m read --grep NEEDLE216 --no-ignore 2>&1)
grep -q 'gen216/x.txt' <<<"$out" && grep -q 'b216.dat' <<<"$out" && ! grep -q -- '-- skipped:' <<<"$out" \
  && ok "the pair: --no-ignore serves both" || bad "--no-ignore still skipped: $(head -c 400 <<<"$out")"
out=$(m read --grep NEEDLE216 gen216/x.txt 2>&1)
grep -q 'gen216/x.txt' <<<"$out" && ok "a path named is served though ignored" || bad "a named ignored path was skipped: $(head -c 400 <<<"$out")"
req216() {
  python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":json.loads(sys.argv[1])}}))' "$1"
}
out=$(req216 '{"grep":"NEEDLE216"}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && ok "mrw_read counts what its grep skipped" || bad "mrw_read skipped: $(head -c 400 <<<"$out")"
import json, sys
sc = json.loads(json.loads(sys.argv[1])["result"]["content"][-1]["text"])
sys.exit(0 if sc.get("skipped") == {"ignored": 0, "ignored_dirs": 1, "binary": 1, "nested": 0, "unkeepable": 0} else 1)
PY
out=$(req216 '{"grep":"NEEDLE216","no_ignore":true}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && ok "the pair: no_ignore walks all and counts nothing" || bad "no_ignore: $(head -c 400 <<<"$out")"
import json, sys
r = json.loads(sys.argv[1])["result"]
sc = json.loads(r["content"][-1]["text"])
sys.exit(0 if "skipped" not in sc and "gen216/x.txt" in sc.get("observed", {}) and "b216.dat" in sc.get("observed", {}) else 1)
PY

# 215. ADR-115: mrw_write takes then, names of steps declared in
# .quality-harness.json, run after a passing check, verdicts in the receipt's
# then. The pair: a name the project did not declare is refused before
# anything is written.
fixture
printf '{"check":"exit 0","steps":{"ok215":"echo fine215"}}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
req215() {
  python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":json.loads(sys.argv[1])}}))' "$1"
}
out=$(req215 '{"plan":"@@ a.go 3 replace\nfunc A() int { return 215 }\n","then":["ok215"]}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && ok "mrw_write runs a declared step after its check" || bad "the step did not run: $(head -c 400 <<<"$out")"
import json, sys
r = json.loads(sys.argv[1])["result"]
st = (r["structuredContent"].get("then") or {}).get("steps") or []
sys.exit(0 if len(st) == 1 and st[0].get("status") == "pass" and not r.get("isError") else 1)
PY
before=$(cat "$R/a.go")
out=$(req215 '{"plan":"@@ a.go 3 replace\nfunc A() int { return 2150 }\n","then":["nope215"]}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && [ "$(cat "$R/a.go")" = "$before" ] && ok "the pair: an undeclared step is refused and nothing is written" || bad "undeclared step: $(head -c 400 <<<"$out")"
import json, sys
r = json.loads(sys.argv[1])["result"]
sys.exit(0 if r.get("isError") and "nothing was written" in r["content"][0]["text"] else 1)
PY

# 217. ADR-117: mrw_read takes max_lines, stat and files_from, each as the CLI's
# flag does. max_lines 1 serves one line and names the cut as max_lines; stat
# serves the header and no line; files_from reads a list in the root. The pairs:
# a negative cap is refused, and files_from "-" and a path out of the root are.
fixture
printf 'one\ntwo\nthree\n' > "$R/m217.txt"
printf '# a note\nm217.txt:3\n' > "$R/list217"
req217() {
  python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":json.loads(sys.argv[1])}}))' "$1" | "$MRW" -C "$R" mcp 2>/dev/null
}
check217() {
  python3 - "$1" "$2" <<'PY'
import json, sys
r = json.loads(sys.argv[1])["result"]
t = r["content"][0]["text"]
want = sys.argv[2]
ok = {
  "cap": not r.get("isError") and "1| one" in t and "2| two" not in t and "max_lines reached" in t and "--max-lines" not in t,
  "stat": not r.get("isError") and "==> m217.txt" in t and "| one" not in t and "-- ck " not in t,
  "list": not r.get("isError") and "3| three" in t and "1| one" not in t,
  "refused": r.get("isError") is True,
}[want]
sys.exit(0 if ok else 1)
PY
}
out=$(req217 '{"specs":["m217.txt"],"max_lines":1}'); check217 "$out" cap && ok "max_lines 1 serves one line and names the cut max_lines" || bad "max_lines: $(head -c 300 <<<"$out")"
out=$(req217 '{"specs":["m217.txt"],"max_lines":-1}'); check217 "$out" refused && ok "the pair: a negative max_lines is refused" || bad "max_lines -1: $(head -c 300 <<<"$out")"
out=$(req217 '{"specs":["m217.txt"],"stat":true}'); check217 "$out" stat && ok "stat serves the header, no line and no checkpoint" || bad "stat: $(head -c 300 <<<"$out")"
out=$(req217 '{"files_from":"list217"}'); check217 "$out" list && ok "files_from reads a list in the root" || bad "files_from: $(head -c 300 <<<"$out")"
out=$(req217 '{"files_from":"-"}'); check217 "$out" refused && ok "the pair: files_from - is refused" || bad "files_from -: $(head -c 300 <<<"$out")"
out=$(req217 '{"files_from":"../list217"}'); check217 "$out" refused && ok "the pair: files_from out of the root is refused" || bad "files_from ..: $(head -c 300 <<<"$out")"

# 218. ADR-118: a start pattern that matches several lines picks one with
# occurrence=N, once every match before it has been served. Three functions
# named X: after a read of the file, occurrence=2 changes the second alone.
# The pairs: occurrence=4 of three is refused and the file unchanged; on a line
# address it is refused; and after a read of the second function alone, the
# first match unread, occurrence=2 is refused naming it.
fixture
printf 'package demo\n\nfunc X() int {\n\treturn 1\n}\n\nfunc X() int {\n\treturn 2\n}\n\nfunc X() int {\n\treturn 3\n}\n' > "$R/x218.go"
before218=$(cat "$R/x218.go")
m read x218.go >"$WORK/served.out"
printf '@@ x218.go /^\\treturn/ replace occurrence=2\n\treturn 20\n' > "$R/p218.mrw"
m write --no-check "$R/p218.mrw" > /dev/null 2>&1; want 0 $? "occurrence=2 applies after a read"
[ "$(grep -c 'return 20' "$R/x218.go")" = 1 ] && [ "$(sed -n 8p "$R/x218.go")" = "$(printf '\treturn 20')" ] \
  && ok "and only the second match changed" || bad "occurrence=2 changed: $(cat "$R/x218.go")"
printf '%s\n' "$before218" > "$R/x218.go"; m read x218.go >"$WORK/served.out"
printf '@@ x218.go /^\\treturn/ replace occurrence=4\n\treturn 40\n' > "$R/p218.mrw"
m write --no-check "$R/p218.mrw" > /dev/null 2>&1; want 1 $? "the pair: occurrence=4 of three is refused"
[ "$(cat "$R/x218.go")" = "$before218" ] && ok "and the file is unchanged" || bad "a refused occurrence changed the file"
printf '@@ x218.go 8 replace occurrence=2\n\treturn 20\n' > "$R/p218.mrw"
m write --no-check "$R/p218.mrw" > /dev/null 2>&1; want 2 $? "occurrence= on a line address is refused by the parser"
XDG_STATE_HOME="$WORK/st218" "$MRW" -C "$R" read 'x218.go:7-10' >"$WORK/served.out"
printf '@@ x218.go /^\\treturn/ replace occurrence=2\n\treturn 20\n' > "$R/p218.mrw"
out=$(XDG_STATE_HOME="$WORK/st218" "$MRW" -C "$R" write --no-check "$R/p218.mrw" 2>&1); rc=$?
want 1 "$rc" "occurrence=2 with the first match unread is refused"
grep -q 'lines 4 of x218.go' <<<"$out" && ok "and the refusal names the unread match" || bad "refusal: $out"
# And right after a write: the file is wholly licensed for edits, but its
# matches were never shown, so occurrence= still needs them read.
printf '%s\n' "$before218" > "$R/x218.go"
XDG_STATE_HOME="$WORK/st218b" "$MRW" -C "$R" read 'x218.go:1' >"$WORK/served.out"
printf '@@ x218.go 1 replace\npackage demo // edited\n' > "$R/p218.mrw"
XDG_STATE_HOME="$WORK/st218b" "$MRW" -C "$R" write --no-check "$R/p218.mrw" > /dev/null 2>&1; want 0 $? "a write of line 1 lands"
printf '@@ x218.go /^\\treturn/ replace occurrence=2\n\treturn 20\n' > "$R/p218.mrw"
XDG_STATE_HOME="$WORK/st218b" "$MRW" -C "$R" write --no-check "$R/p218.mrw" > /dev/null 2>&1; want 1 $? "right after the write, occurrence=2 with no match read is refused"
XDG_STATE_HOME="$WORK/st218b" "$MRW" -C "$R" read 'x218.go:/^\treturn/' >"$WORK/served.out"
XDG_STATE_HOME="$WORK/st218b" "$MRW" -C "$R" write --no-check "$R/p218.mrw" > /dev/null 2>&1; want 0 $? "the pair: after a read of the matches it applies"

# 219. ADR-119: an applied replace whose body ends in the closer the file
# already had carries a closer hint and stays ok. Blade, replacing line 7 as
# the field report did: the @endif left at 8 is named at line 10 and counted in
# hints. The pair: replacing 7-8, through it, names nothing. And a .md fence
# closed twice is named on the human receipt too: closer runs on prose.
fixture
printf '%s\n' '<div>' '  <h1>{{ $title }}</h1>' '' '  <ul>' '  @foreach ($xs as $x)' '  @endforeach' \
  '    @if ($items)' '    @endif' '  </ul>' '</div>' > "$R/v219.blade.php"
before219=$(cat "$R/v219.blade.php")
m read v219.blade.php >"$WORK/served.out"
printf '@@ v219.blade.php 7 replace\n    @if ($items->isNotEmpty())\n        <li>x</li>\n    @endif\n' > "$R/p219.mrw"
out=$(m write --no-check --json "$R/p219.mrw" 2>&1); want 0 $? "a replace that leaves the old @endif below it applies"
grep -q '"closer": "line 10 repeats the body'"'"'s last line: @endif"' <<<"$out" && grep -q '"hints": 1' <<<"$out" \
  && ok "and its receipt names the survivor at line 10 and counts one hint" || bad "no closer hint: $out"
printf '%s\n' "$before219" > "$R/v219.blade.php"; m read v219.blade.php >"$WORK/served.out"
printf '@@ v219.blade.php 7-8 replace anchor="@if ($items)"\n    @if ($items->isNotEmpty())\n        <li>x</li>\n    @endif\n' > "$R/p219.mrw"
out=$(m write --no-check --json "$R/p219.mrw" 2>&1); want 0 $? "the pair: a replace through the @endif applies"
! grep -q '"closer"\|"hints"' <<<"$out" && ok "and carries no hint" || bad "a correct replace carried a hint: $out"
# The review of #322: an inner block replaced through its own `}`, the outer
# `}` right below. The tokens match, and the range already ended in it.
printf 'package n\n\nfunc A() {\n\tif x {\n\t\tf()\n\t}\n}\n' > "$R/n219.go"
m read n219.go >"$WORK/served.out"
printf '@@ n219.go 4-6 replace anchor="if x {"\n\tif y {\n\t\tg()\n\t}\n' > "$R/p219.mrw"
out=$(m write --no-check --json "$R/p219.mrw" 2>&1); want 0 $? "a replace of an inner block through its own } applies"
! grep -q '"closer"\|"hints"' <<<"$out" && ok "and carries no hint for the outer }" || bad "a through-closer replace carried a hint: $out"
printf '%s\n' intro '```go' 'x := 1' '```' outro > "$R/n219.md"
m read n219.md >"$WORK/served.out"
printf '@@ n219.md 3 replace\nx := 2\n```\n' > "$R/p219.mrw"
out=$(m write "$R/p219.mrw" 2>&1); want 0 $? "a .md replace that closes its fence twice applies"
grep -q 'closer line 5 repeats' <<<"$out" && grep -q '0 advisories, 1 hint — applied' <<<"$out" \
  && ok "and the human receipt names it and counts the hint beside the advisories" || bad "prose closer: $out"

# 220. ADR-121: while an mrw_write's check runs, mrw mcp answers what arrives
# after it. A write whose check sleeps 2 s, then a ping: the ping's answer comes
# first. The pair: the same write with check: false is answered before the ping,
# in order, as every quick answer is.
fixture
printf 'package a\nfunc A() {}\n' > "$R/a220.go"
printf '{"check":"sleep 2"}\n' > "$R/.quality-harness.json"
m read a220.go >"$WORK/served.out"
w220='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ a220.go 2 replace\\nfunc A() { _ = 1 }\\n"%s}}}'
p220='{"jsonrpc":"2.0","id":2,"method":"ping"}'
# A plain pipe, not bounded: bounded gives its command /dev/null for stdin. The
# server ends at end of input once the 2 s check has answered.
{ printf "$w220\n" ''; sleep 0.5; printf '%s\n' "$p220"; } | "$MRW" -C "$R" mcp > "$WORK/out220" 2>/dev/null
order=$(grep -o '"id":[12]' "$WORK/out220" | tr -d '\n')
[ "$order" = '"id":2"id":1' ] && ok "a ping sent during a write's check is answered first" || bad "answer order $order: $(cat "$WORK/out220")"
printf 'package a\nfunc A() {}\n' > "$R/a220.go"; m read a220.go >"$WORK/served.out"
{ printf "$w220\n" ',"check":false'; sleep 0.5; printf '%s\n' "$p220"; } | "$MRW" -C "$R" mcp > "$WORK/out220b" 2>/dev/null
order=$(grep -o '"id":[12]' "$WORK/out220b" | tr -d '\n')
[ "$order" = '"id":1"id":2' ] && ok "the pair: with no check the answers keep their order" || bad "answer order $order: $(cat "$WORK/out220b")"
rm -f "$R/.quality-harness.json"

# 221. ADR-122: --ast-grep serves the files --grep would. Inside a checkout that
# ignores gen/, a fake ast-grep reports a hit in gen/a.go and one in b.go: b.go
# is served, gen/a.go is not, and the -- skipped: line counts gen/. The pair:
# with --no-ignore both are served and nothing is counted.
fixture
d221=$(mktemp -d)
mkdir -p "$R/.git" "$R/gen"; printf 'gen/\n' > "$R/.gitignore"
printf 'package gen\n' > "$R/gen/a.go"; printf 'package b\n' > "$R/b221.go"
printf '%s\n' '#!/bin/sh' 'printf "%s" "[{\"file\":\"gen/a.go\",\"range\":{\"start\":{\"line\":0},\"end\":{\"line\":0}}},{\"file\":\"b221.go\",\"range\":{\"start\":{\"line\":0},\"end\":{\"line\":0}}}]"' > "$d221/ast-grep"
chmod +x "$d221/ast-grep"
out=$(env PATH="$d221:$PATH" "$MRW" -C "$R" read --ast-grep 'package $A' 2>&1); want 0 $? "--ast-grep inside a checkout that ignores gen/ reads"
grep -q '==> b221.go' <<<"$out" && ! grep -q '==> gen/a.go' <<<"$out" && grep -q -- '-- skipped: 1 director(ies) .gitignore ignores' <<<"$out" \
  && ok "and serves b221.go, not the ignored gen/a.go, counting gen/" || bad "ast-grep served: $out"
out=$(env PATH="$d221:$PATH" "$MRW" -C "$R" read --ast-grep 'package $A' --no-ignore 2>&1); want 0 $? "the pair: --no-ignore reaches --ast-grep"
grep -q '==> gen/a.go' <<<"$out" && ! grep -q -- '-- skipped:' <<<"$out" && ok "and serves both, counting nothing" || bad "--no-ignore ast-grep: $out"
rm -rf "$d221" "$R/.git" "$R/.gitignore" "$R/gen"

# 222. ADR-124: {dirs} is the portable scope. mrw check on two files in one
# directory and one in another, with scoped_check "echo SCOPED {dirs}", runs
# SCOPED ./a ./b; the pair, a step command holding {dirs}, is refused, exit 2.
fixture
mkdir -p "$R/a222" "$R/b222"; printf 'x\n' > "$R/a222/x.rs"; printf 'y\n' > "$R/a222/y.rs"; printf 'z\n' > "$R/b222/z.py"
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {dirs}"}\n' > "$R/.quality-harness.json"
out=$("$MRW" -C "$R" check a222/x.rs a222/y.rs b222/z.py 2>&1); want 0 $? "mrw check with a {dirs} template runs"
grep -q 'SCOPED ./a222 ./b222' <<<"$out" && ok "and substitutes each directory once" || bad "{dirs}: $out"
out=$("$MRW" -C "$R" check a222/x.rs --then-sh 'echo {dirs}' 2>&1); want 2 $? "the pair: a step holding {dirs} is refused"
grep -q 'holds {dirs}' <<<"$out" && ok "and the refusal names {dirs}" || bad "step {dirs}: $out"
"$MRW" -C "$R" check a222/x.rs --then-sh 'echo ok' > /dev/null 2>&1; want 0 $? "while a step holding no placeholder runs"
rm -f "$R/.quality-harness.json"

# 229. The Codex review of v1.42.0..v1.47.0, finding 3: --no-ignore turns off
# the ignore rules for --ast-grep, never .git. A fake ast-grep reports a hit in
# .git/x229.go and one in b229.go; with --no-ignore b229.go is served and the
# .git hit is not, as the walk prunes .git. The pair: b229.go alone is served
# without the flag too, so the row is about .git and not the flag's absence.
fixture
d229=$(mktemp -d)
mkdir -p "$R/.git"; printf 'package g\n' > "$R/.git/x229.go"; printf 'package b\n' > "$R/b229.go"
printf '%s\n' '#!/bin/sh' 'printf "%s" "[{\"file\":\".git/x229.go\",\"range\":{\"start\":{\"line\":0},\"end\":{\"line\":0}}},{\"file\":\"b229.go\",\"range\":{\"start\":{\"line\":0},\"end\":{\"line\":0}}}]"' > "$d229/ast-grep"
chmod +x "$d229/ast-grep"
out=$(env PATH="$d229:$PATH" "$MRW" -C "$R" read --ast-grep 'package $A' --no-ignore 2>&1); want 0 $? "--ast-grep --no-ignore with a hit inside .git reads"
grep -q '==> b229.go' <<<"$out" && ! grep -q '==> .git/x229.go' <<<"$out" && ok "and serves b229.go, not the hit inside .git" || bad "--no-ignore .git: $out"
out=$(env PATH="$d229:$PATH" "$MRW" -C "$R" read --ast-grep 'package $A' 2>&1); want 0 $? "the pair: without --no-ignore it reads too"
grep -q '==> b229.go' <<<"$out" && ! grep -q '==> .git/x229.go' <<<"$out" && ok "and serves b229.go alone" || bad "ast-grep .git: $out"
out=$(env PATH="$d229:$PATH" "$MRW" -C "$R" read --ast-grep 'package $A' --no-ignore .git 2>&1); want 0 $? "--ast-grep --no-ignore over a named .git reads"
grep -q '==> .git/x229.go' <<<"$out" && ok "and serves the hit inside the .git it was given" || bad "named .git: $out"
rm -rf "$d229" "$R/.git"

# 225. ADR-127: the drift advisory names another writer's write. The first
# write's check waits on a gate; a second write lands on another file inside
# it; the first receipt carries "drift_writers": 1. ADR-112's per-file drift
# could not see it, the other file not being the first write's. The pair: the
# same write with no second writer carries no drift_writers.
fixture
g225=$(mktemp -d)
printf 'package x\n' > "$R/x225.go"; printf 'package y\n' > "$R/y225.go"
printf '{"check":"touch %s/started; i=0; while [ ! -e %s/go ] && [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done"}\n' "$g225" "$g225" > "$R/.quality-harness.json"
m read x225.go y225.go >"$WORK/served.out"
printf '@@ x225.go 1 replace\npackage x2\n' > "$WORK/p225a"; printf '@@ y225.go 1 replace\npackage y2\n' > "$WORK/p225b"
bounded 30 "$WORK/o225" "$MRW" -C "$R" write --json "$WORK/p225a" &
p225=$!
i=0; while [ ! -e "$g225/started" ] && [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done
[ -e "$g225/started" ] || bad "the first write's check never started"
bounded 30 "$WORK/o225b" "$MRW" -C "$R" write --no-check "$WORK/p225b"; want 0 $? "a second write lands while the first write's check runs"
touch "$g225/go"; wait "$p225"; want 0 $? "the first write's check passes"
grep -q '"drift_writers": 1' "$WORK/o225" && ok "and its receipt counts the other write" || bad "drift_writers: $(head -c 400 "$WORK/o225")"
rm -f "$g225/started" "$g225/go"; m read x225.go >"$WORK/served.out"
printf '@@ x225.go 1 replace\npackage x3\n' > "$WORK/p225c"
bounded 30 "$WORK/o225c" "$MRW" -C "$R" write --json "$WORK/p225c" &
p225=$!
i=0; while [ ! -e "$g225/started" ] && [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done
[ -e "$g225/started" ] || bad "the pair's check never started"
touch "$g225/go"; wait "$p225"; want 0 $? "the pair: a write with no other writer passes"
! grep -q 'drift_writers' "$WORK/o225c" && ok "and carries no drift_writers" || bad "pair: $(head -c 400 "$WORK/o225c")"
rm -rf "$g225" "$R/.quality-harness.json"

# 226. ADR-128: the MCP surface. An initialize line that starts with a UTF-8
# byte-order mark is answered, where it was a -32700 parse error; the pair: a
# line that is not JSON still is one. An mrw_write on a file never read is
# refused without the CLI's "pass --force", which mrw_write cannot take; the
# pair: the CLI's own refusal of the same plan keeps it.
fixture
out=$(printf '\357\273\277{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n' | m mcp 2>/dev/null)
grep -q '"result"' <<<"$out" && ! grep -q -- '-32700' <<<"$out" && ok "a BOM'd initialize is answered" || bad "BOM initialize: $out"
out=$(printf 'not json\n' | m mcp 2>/dev/null)
grep -q -- '-32700' <<<"$out" && ok "the pair: a line that is not JSON is still a parse error" || bad "garbage: $out"
printf 'never served\n' > "$R/unread226.md"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ unread226.md 1 replace\\nx\\n"}}}\n' | m mcp 2>/dev/null)
grep -q 'has not been read' <<<"$out" && ! grep -q -- '--force' <<<"$out" && ok "an mrw_write refusal does not advise --force" || bad "mcp force: $out"
printf '@@ unread226.md 1 replace\nx\n' > "$WORK/p226"
out=$(m write --no-check "$WORK/p226" 2>&1)
grep -q -- '--force' <<<"$out" && ok "the pair: the CLI's refusal keeps it" || bad "cli force: $out"
m read a.go >"$WORK/served.out"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go:1"],"ack":["nosuch226"]}}}\n' | m mcp 2>/dev/null)
grep -q 'matched no checkpoint.*nosuch226' <<<"$out" && ok "an ack id that matches nothing is named" || bad "unknown ack: $out"
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["a.go:1"]}}}\n' | m mcp 2>/dev/null)
! grep -q 'matched no checkpoint' <<<"$out" && ok "the pair: a read with no stale ack names none" || bad "no ack: $out"
printf '{"check":"sleep 30"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}\n{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":"@@ a.go 3 replace\\nfunc A() int { return 9 }\\n"}}}\n' > "$WORK/i226a"
printf '{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7}}\n' > "$WORK/i226b"
# Under `bounded`, not an alarm: Go ignores SIGALRM, so only a kill bounds mrw
# (contract.md; the in-process review of #343). sh execs mrw, so the pid bounded
# kills on a timeout is mrw's own, not a wrapper's (the Codex review of #343);
# the input is a process substitution, since an async command's stdin is
# /dev/null, and it ends on its own.
s226=$SECONDS
bounded 25 "$WORK/o226" sh -c 'exec "$1" -C "$2" mcp < "$3" 2>/dev/null' _ "$MRW" "$R" <(cat "$WORK/i226a"; sleep 1; cat "$WORK/i226b"; sleep 2)
t226=$((SECONDS - s226))
grep -q '"skipped":"interrupted"' "$WORK/o226" && [ "$t226" -lt 15 ] && ok "a cancel stops the write's running check, reported interrupted" || bad "cancel (${t226}s): $(head -c 400 "$WORK/o226")"
rm -f "$R/.quality-harness.json"

# 230. ADR-129: a rename may change only the case of a name. On a filesystem
# that folds case, A.txt finds a.txt — the source — and the rename was refused
# "already exists". It applies now and the directory spells it A.txt. Linux CI
# does not fold case, so the respelling is driven where it does (a local macOS
# run) and skipped, saying so, elsewhere. The pair runs everywhere: a rename
# onto a hard link of the source — the same file under a name that really
# exists, which POSIX rename(2) "does nothing" to — is still refused.
fixture
printf 'x\n' > "$R/a230.txt"
m read a230.txt >"$WORK/served.out"
if [ -e "$R/A230.TXT" ]; then
  printf '@@ a230.txt - rename\nA230.txt\n' > "$WORK/p230"
  out=$(m write --no-check "$WORK/p230" 2>&1); want 0 $? "a case-only rename applies on a folding filesystem"
  ls "$R" | grep -qx 'A230.txt' && ! ls "$R" | grep -qx 'a230.txt' && ok "and the directory spells it as the plan asked" || bad "case-only rename: $(ls "$R" | grep -i a230) :: $out"
else
  skip "a case-only rename (this filesystem does not fold case: a230.txt and A230.txt are two names)"
fi
printf 'y\n' > "$R/h230.txt"
if ln "$R/h230.txt" "$R/k230.txt" 2>/dev/null; then
  m read h230.txt >"$WORK/served.out"
  printf '@@ h230.txt - rename\nk230.txt\n' > "$WORK/q230"
  out=$(m write --no-check "$WORK/q230" 2>&1); want 1 $? "a rename onto a hard link of the source is refused"
  grep -q 'already exists' <<<"$out" && [ -e "$R/h230.txt" ] && [ -e "$R/k230.txt" ] && ok "naming the destination as existing, both names kept" || bad "hard link: $out"
else
  skip "a rename onto a hard link (no hard links here)"
fi

# 238. ADR-137: a read can cut a long line to a window, and a cut line licenses
# nothing. A 400-character line read with --max-cols 40 is served as a window
# round the match, marked with its columns; what was cut was not shown, so a plan
# that replaces that line is refused as unread and the file is unchanged. The
# pair: the same line read whole licenses the same plan.
fixture
printf 'short one\n%sNEEDLE%s\nshort three\n' "$(printf 'x%.0s' $(seq 1 300))" "$(printf 'y%.0s' $(seq 1 94))" > "$R/w238.txt"
out=$(m read --max-cols 40 --context 1 'w238.txt:/NEEDLE/' 2>&1); want 0 $? "a read with --max-cols exits 0"
{ grep -q 'NEEDLE' <<<"$out" && grep -q '\[cols [0-9]*-[0-9]* of 400\]' <<<"$out" && [ "$(awk '{ if (length($0) > m) m = length($0) } END { print m }' <<<"$out")" -lt 120 ]; } \
  && ok "and serves the long line as a window round the match, marked with its columns" || bad "cut read: $out"
out=$(printf '@@ w238.txt 2 replace\nREPLACED\n' | m write --no-check - 2>&1); want 1 $? "a plan to the cut line exits 1"
{ grep -q 'has not been read' <<<"$out" && grep -q NEEDLE "$R/w238.txt"; } && ok "refused as unread, the file unchanged" || bad "cut then write: $out"
m read w238.txt:2 >"$WORK/served238.out"; want 0 $? "the pair: the same line read whole exits 0"
out=$(printf '@@ w238.txt 2 replace\nREPLACED\n' | m write --no-check - 2>&1); want 0 $? "and licenses the same plan"
grep -qx REPLACED "$R/w238.txt" && ok "which lands" || bad "whole read then write: $out"
rm -f "$R/w238.txt" "$WORK/served238.out"

# 237. ADR-136: reads say less. A read of several multi-line ranges printed the
# "needs a served line after" note at each; it is printed once, at the first.
# The stale-ledger notice was ninety words of version history printed for every
# subcommand; it is one line, and `version`, `instructions` and `stats`, which
# never read the ledger, print none. The pairs: one range keeps its note, and
# `read` still announces a stale ledger.
fixture
printf 'l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\n' > "$R/q237.txt"
out=$(m read q237.txt:2-3 q237.txt:5-6 2>&1); want 0 $? "a read of two multi-line ranges exits 0"
[ "$(grep -c 'needs a served line after' <<<"$out")" = 1 ] && ok "and prints the neighbour note once" || bad "note count: $out"
out=$(m read q237.txt:2-3 2>&1)
grep -q 'needs a served line after 3' <<<"$out" && ok "the pair: one range still gets its note" || bad "single range: $out"
SD237=$(m seen | head -1 | sed 's/.*: //')
printf 'not a ledger\n' > "$SD237/seen"
for sub in version instructions stats; do
  err=$(m $sub 2>&1 >/dev/null)
  grep -q 'read ledger' <<<"$err" && bad "mrw $sub printed the stale-ledger notice: $err" || ok "mrw $sub prints no stale-ledger notice"
done
err=$(m read q237.txt:2 2>&1 >"$WORK/served237.out")
{ grep -q 'written by an older mrw' <<<"$err" && grep -q 'line endings' <<<"$err" && [ "$(wc -l <<<"$err")" -le 1 ]; } \
  && ok "the pair: read announces the stale ledger in one line naming both causes" || bad "stale notice: $err"
rm -f "$R/q237.txt"

# 236. ADR-134: a hard link to mrw's own state is not served. ADR-077 compared a
# path with the state base and never the file with the files inside it, so a
# hard link in the root to the ledger was read whole and matched by --grep (the
# Windows chaos round on v1.52.0, reproduced on macOS). It is refused naming
# mrw's own state, and a walk drops it. The pair: a hard link to an ordinary
# file is served.
fixture
st236() { XDG_STATE_HOME="$WORK/st236" "$MRW" -C "$R" "$@"; }
printf 'needle236\n' > "$R/ord236.txt"
st236 read ord236.txt >"$WORK/served236.out"
led=$(ls "$WORK"/st236/mrw/*/seen 2>/dev/null | head -1)
[ -n "$led" ] && ok "the ledger lives outside the root for this row" || bad "no ledger under $WORK/st236: $(ls -R "$WORK/st236" 2>&1 | head)"
if ln "$led" "$R/hl236.txt" 2>/dev/null && ln "$R/ord236.txt" "$R/ord236b.txt" 2>/dev/null; then
  out=$(st236 read hl236.txt 2>&1); rc=$?
  want 1 "$rc" "a read of a hard link to the ledger exits 1"
  { grep -q 'own state' <<<"$out" && ! grep -q 'mrw-seen' <<<"$out"; } && ok "naming mrw's own state, the ledger unserved" || bad "hard link read: $out"
  out=$(st236 read --grep 'ord236' 2>&1)
  { ! grep -q 'hl236' <<<"$out" && ! grep -q 'mrw-seen' <<<"$out"; } && ok "and a walk neither serves nor matches it" || bad "hard link grep: $out"
  out=$(st236 read --grep 'needle236' 2>&1); rc=$?
  { [ "$rc" = 0 ] && grep -q '^==> ord236b.txt' <<<"$out"; } && ok "the pair: a hard link to an ordinary file is served" || bad "ordinary hard link grep (exit $rc): $out"
else
  skip "a hard link to the ledger (this filesystem makes none between $WORK and $R)"
fi
rm -f "$R/hl236.txt" "$R/ord236.txt" "$R/ord236b.txt" "$WORK/served236.out"
rm -rf "$WORK/st236"

# 235. ADR-129 amendment (the Windows chaos round, 2026-10-09). A plan that
# spells the source otherwise than its directory does, and names as the
# destination the spelling the directory holds, was refused "dest already
# exists" — the dest is the source's own entry. It is refused naming the cause,
# the source's spelling. Where a filesystem does not fold case, as on Linux CI,
# the pair is skipped saying so: P235.TXT and p235.txt are two names there.
fixture
printf 'x\n' > "$R/p235.txt"
if [ -e "$R/P235.TXT" ]; then
  m read P235.TXT >"$WORK/served.out"
  printf '@@ P235.TXT - rename\np235.txt\n' > "$WORK/p235"
  out=$(m write --no-check "$WORK/p235" 2>&1); want 1 $? "a rename whose source is spelled otherwise than its directory exits 1"
  { grep -q 'not spelled that way' <<<"$out" && ls "$R" | grep -qx 'p235.txt'; } && ok "naming the source's spelling, the directory unchanged" || bad "other-spelling rename: $out"
else
  skip "a rename from a spelling the directory does not hold (this filesystem does not fold case)"
fi

# 234. ADR-133: a read sent to the null device licenses nothing. Its answer
# reached nobody, yet the ledger recorded every line, so a write to them went
# through as if they had been read. The read still exits 0 and says on stderr
# that nothing was recorded; a write to that line is then refused as unread.
# The pair: the same read sent to a file licenses the same write.
fixture
printf 'one\ntwo\n' > "$R/f234.txt"
err=$(m read f234.txt:2 2>&1 >/dev/null); want 0 $? "a read sent to the null device exits 0"
grep -q 'null device, so nothing was recorded' <<<"$err" && ok "and says nothing was recorded" || bad "null-device read stderr: $err"
out=$(printf '@@ f234.txt 2 replace\nTWO\n' | m write --no-check - 2>&1); want 1 $? "a write to a line read only to the null device exits 1"
{ grep -q 'has not been read' <<<"$out" && grep -qx 'two' "$R/f234.txt"; } && ok "naming it unread, the file unchanged" || bad "null-device read then write: $out"
m read f234.txt:2 >"$WORK/served234.out"; want 0 $? "the pair: the same read sent to a file exits 0"
out=$(printf '@@ f234.txt 2 replace\nTWO\n' | m write --no-check - 2>&1); want 0 $? "and licenses the same write"
grep -qx 'TWO' "$R/f234.txt" && ok "which lands" || bad "file-read then write: $out"
rm -f "$R/f234.txt" "$WORK/served234.out"

# 233. ADR-130: a walk does not enter a nested repository. Inside a checkout, a
# directory holding its own .git is another project's, and git does not descend
# into it; mrw walked it with its own rules. --grep serves the outer match only
# and says on the -- skipped: line that it did not enter one. The pairs: the
# directory named is walked, and --no-ignore walks it.
fixture
mkdir -p "$R/.git" "$R/nest233/.git"
printf 'needle233\n' > "$R/top233.txt"; printf 'needle233\n' > "$R/nest233/in.txt"
out=$(m read --grep needle233 2>&1); want 0 $? "a grep in a checkout holding a nested repository exits 0"
{ grep -q 'top233.txt' <<<"$out" && ! grep -q 'nest233/in.txt' <<<"$out" && grep -q '1 nested repositor(ies), not entered' <<<"$out"; } \
  && ok "and does not enter it, saying so" || bad "nested: $out"
out=$(m read --grep needle233 nest233 2>&1)
grep -q 'nest233/in.txt' <<<"$out" && ok "the pair: the nested repository named is walked" || bad "named nested: $out"
out=$(m read --grep needle233 --no-ignore 2>&1)
grep -q 'nest233/in.txt' <<<"$out" && ok "the pair: --no-ignore walks it" || bad "no-ignore nested: $out"
out=$(printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"grep":"needle233"}}}' | "$MRW" -C "$R" mcp 2>/dev/null)
python3 - "$out" <<'PY' && ok "mrw_read counts it as skipped.nested" || bad "mrw_read nested: $(head -c 400 <<<"$out")"
import json, sys
sc = json.loads(json.loads(sys.argv[1])["result"]["content"][-1]["text"])
sys.exit(0 if sc.get("skipped", {}).get("nested") == 1 else 1)
PY
rm -rf "$R/.git" "$R/nest233" "$R/top233.txt"

# 231. ADR-132: a refusal before the first rename whose cause is the target's
# exits 1, as one found at validation does. A create under a read-only
# directory fails staging with EACCES; it exited 2 and printed the whole error
# again under the receipt. The --then-sh step asked for is not run. The pair:
# the same create in a writable directory exits 0. Not as root, for whom the
# mode is not enforced.
fixture
mkdir -p "$R/ro231" "$R/w231"; chmod 555 "$R/ro231"
if ( : > "$R/ro231/.probe" ) 2>/dev/null; then
  rm -f "$R/ro231/.probe"; chmod 755 "$R/ro231"
  skip "a refusal for a read-only directory (the mode is not enforced here — running as root?)"
else
  out=$(printf '@@ ro231/n.txt 0 create\nx\n' | m write --no-check --then-sh 'echo ran231' - 2>&1); want 1 $? "a create a read-only directory refuses exits 1"
  { grep -q 'nothing was written' <<<"$out" && grep -q 'echo ran231 — NOT RUN' <<<"$out" && [ ! -e "$R/ro231/n.txt" ]; } \
    && ok "naming nothing written, its step not run" || bad "read-only create: $out"
  chmod 755 "$R/ro231"
fi
out=$(printf '@@ w231/n.txt 0 create\nx\n' | m write --no-check - 2>&1); want 0 $? "the pair: the same create in a writable directory exits 0"

# 232. ADR-132: a check stopped by its deadline is headed TIMED OUT, not FAIL —
# no process gave a verdict — and the exit stays 3. The pair: a check that ran
# and failed is still headed FAIL.
fixture
printf '{"check":"sleep 30","timeout_seconds":1}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' > "$WORK/p232"
bounded 30 "$WORK/o232" "$MRW" -C "$R" write "$WORK/p232"; want 3 $? "a write whose check times out exits 3 (ADR-132)"
grep -q '^check TIMED OUT' "$WORK/o232" && ! grep -q '^check FAIL' "$WORK/o232" && ok "and its check is headed TIMED OUT" || bad "timeout headline: $(head -c 400 "$WORK/o232")"
printf '{"check":"false"}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 10 }\n' > "$WORK/q232"
m write "$WORK/q232" > "$WORK/r232" 2>&1; want 3 $? "the pair: a write whose check fails exits 3"
grep -q '^check FAIL' "$WORK/r232" && ok "and is headed FAIL" || bad "failed headline: $(head -c 400 "$WORK/r232")"
rm -f "$R/.quality-harness.json"

# 223. ADR-125: a target mrw cannot open is refused on its hunk, naming why.
# A plan over a.txt and a write-only b.txt (mode 200: not read-only, so ADR-076
# does not take it first) printed one bare "mrw: …" line at exit
# 2 and no receipt. Now: exit 1, b's hunk fails naming "permission denied", a's
# is skipped, nothing is written. The pair: the same plan with b readable, exit 0.
fixture
printf 'a\n' > "$R/a223.txt"; printf 'b\n' > "$R/b223.txt"; chmod 200 "$R/b223.txt"
printf '@@ a223.txt 1 replace\nA\n@@ b223.txt 1 replace\nB\n' > "$WORK/p223"
if [ -r "$R/b223.txt" ]; then
  skip "an unreadable target gets a receipt (permission bits not enforced here — running as root?)"
else
  out=$(m write --force --no-check "$WORK/p223" 2>&1); want 1 $? "a plan over a write-only target exits 1"
  { grep -q 'FAIL.*b223.txt.*permission denied' <<<"$out" && grep -q 'skip.*a223.txt' <<<"$out" && [ "$(cat "$R/a223.txt")" = a ]; } \
    && ok "with a receipt naming permission denied, the sibling skipped and unchanged" || bad "unreadable: $out"
fi
chmod 644 "$R/b223.txt"
m write --force --no-check "$WORK/p223" > /dev/null 2>&1; want 0 $? "the pair: the same plan with b readable applies"

# 162. ADR-080: nothing mrw starts outlives the call. A check that passed and an
# ast-grep that answered and exited 0 each left a background grandchild running
# after mrw returned: the group was killed only on a timeout or an interrupt
# (M: reap always; the waiver on #232).
fixture
d162=$(mktemp -d)
printf '%s\n' '#!/bin/sh' "sleep 300 >/dev/null 2>&1 & echo \$! > '$d162/ag.pid'" \
  'printf "%s" "[{\"file\":\"a.go\",\"range\":{\"start\":{\"line\":2},\"end\":{\"line\":2}}}]"' 'exit 0' > "$d162/ast-grep"
chmod +x "$d162/ast-grep"
bounded 10 "$WORK/out162" env PATH="$d162:$PATH" "$MRW" -C "$R" read --ast-grep 'func A'; want 0 $? "an ast-grep that answers and exits 0 is served"
printf '{"check":"sleep 300 >/dev/null 2>&1 & echo $! > %s/ck.pid; exit 0"}\n' "$d162" > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 7 }\n' > "$R/p162.mrw"
bounded 30 "$WORK/out162b" "$MRW" -C "$R" write --check "$R/p162.mrw"; want 0 $? "a write whose check passes exits 0"
for f in ag ck; do
  pid=$(cat "$d162/$f.pid" 2>/dev/null)
  alive=1; for _ in $(seq 1 30); do kill -0 "${pid:-999999999}" 2>/dev/null || { alive=0; break; }; sleep 0.1; done
  { [ -n "$pid" ] && [ "$alive" = 0 ]; } && ok "the $f child's background process is gone when mrw returns" \
    || { kill -9 "${pid:-999999999}" 2>/dev/null; bad "the $f child's background process outlived mrw (pid '$pid')"; }
done

# 163. ADR-080: a kept check log is named, and old ones are pruned. A timed-out
# check kept its log and named nowhere to find it, and nothing bounded how many
# logs piled up in the temp directory. TMPDIR is this run's own (top of file).
fixture
touch -t 202001010000 "$TMPDIR/mrw-check-old163.log"; : > "$TMPDIR/mrw-check-young163.log"
printf '{"check":"echo started; sleep 30","timeout_seconds":1}\n' > "$R/.quality-harness.json"
m read a.go >"$WORK/served.out"
printf '@@ a.go 3 replace\nfunc A() int { return 8 }\n' > "$R/p163.mrw"
out=$(m write "$R/p163.mrw" 2>&1); rc=$?
want 3 "$rc" "a write whose check times out exits 3"
log=$(grep -o 'timed out after [^—]*— full output: [^ ]*' <<<"$out" | head -1 | sed 's/.*full output: //')
{ [ -n "$log" ] && [ -f "$log" ]; } && ok "and the timed-out check names its log, which is there" || bad "no log named: $out"
[ ! -e "$TMPDIR/mrw-check-old163.log" ] && ok "a check log older than a week is pruned" || bad "the old log is still there"
[ -e "$TMPDIR/mrw-check-young163.log" ] && ok "a young one is kept" || bad "the young log was removed"
grep -q 'removed 1 check log(s) older than 7 days' <<<"$out" && ok "and the receipt says what it removed" || bad "the prune was silent: $out"

# Nothing this run started may outlive it. Checked after the last row, so every
# row is covered; §60 above proves an orphan is visible to this group check. A
# killed process is a zombie until its adopter reaps it, and pgrep lists
# zombies: poll, bounded, rather than trust one sleep.
for _ in $(seq 1 30); do
  pgrep -g $$ > "$WORK/kids"; prc=$?   # 0 matched, 1 none; anything else is pgrep itself failing
  left=$(grep -vx "$$" "$WORK/kids" | tr '\n' ' ')
  [ -z "$left" ] && break; sleep 0.1
done
[ "$prc" -le 1 ] && [ -z "$left" ] && ok "no process of this run survives it" \
  || bad "survivors in the run's group after 3 s (pgrep exit $prc): $left$(ps -o pid,ppid,etime,comm -p "$(echo $left | tr ' ' ',')" 2>/dev/null | tail -n +2 | tr '\n' ';')"

if [ "$fails" -eq 0 ]; then
  echo "contract holds"
else
  echo "$fails assertion(s) FAILED"
fi
exit $(( fails > 0 ))
