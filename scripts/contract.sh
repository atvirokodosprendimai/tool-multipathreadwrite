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

cd "$(dirname "$0")/.."
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# Build our OWN binary inside WORK rather than sharing bin/mrw. Two fences ran
# concurrently under `adr-verify --sweep` on 2026-08-31 — one starting with
# `go build -o bin/mrw`, two others executing it — and the binary was rewritten
# under a running process. Both contract.sh-invoking fences failed, and passed
# on a re-run: a flake with a cause. A shared mutable artifact is the cause, so
# it is removed rather than retried. The build is cached, so this is cheap.
MRW=${MRW:+$(cd "$(dirname "$MRW")" && pwd)/$(basename "$MRW")}
MRW=${MRW:-$WORK/mrw}
go build -o "$MRW" ./cmd/mrw
trap 'rm -rf "$WORK"' EXIT
fails=0

ok()   { printf '  PASS  %s\n' "$1"; }
# A skip is honest; a silent pass is not. Used where the trigger is a
# permission bit, which uid 0 ignores — under root the row would go green
# without exercising anything, which is the defect class this file exists for.
skip() { printf '  SKIP  %s\n' "$1"; }
bad()  { printf '  FAIL  %s\n' "$1"; fails=$((fails + 1)); }
# want <expected-exit> <actual-exit> <description>
want() { [ "$1" = "$2" ] && ok "$3" || bad "$3 (exit $2, want $1)"; }

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
  "$MRW" -C "$R" read a.go b.go a_test.go >/dev/null
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
m read @1:1-2 >/dev/null; want 0 "$?" "@1:1-2 resolves"
m read @3     >/dev/null; want 0 "$?" "@3 resolves"
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
m read a.go >/dev/null
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
m read --stat a.go >/dev/null
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
m read q.go >/dev/null
out=$(printf '@@ q.go 3 replace anchor="= \\"muted\\""\nvar S = "MUTED"\n' | m write - 2>&1); rc=$?
want 0 "$rc" "an anchor may contain an escaped quote"

# 9. The ledger records what the caller was SHOWN, and the root is a boundary.
#    Both were silent before: a --stat read licensed an edit to a file whose
#    content had never been printed, and a ../ path walked straight out of the
#    directory mrw was pointed at.
R=$(mktemp -d "$WORK/shown-XXXXXX")
printf 'package demo\n\nfunc A() int { return 1 }\n' > "$R/a.go"
m read --stat a.go >/dev/null
out=$(printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - 2>&1); rc=$?
want 1 "$rc" "--stat prints no content, so it licenses no edit"
grep -q 'has not been read' <<<"$out" && ok "the reason names what was not shown" || bad "unclear reason"
m read a.go:1-1 >/dev/null
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
m read a.go:3 >/dev/null
printf '@@ a.go 3 replace anchor="func A"\nx\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "but re-reading line 3 licenses line 3 — one ranged call is the remedy"
R=$PREV_R
m read a.go >/dev/null
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
m read --max-lines 5 big.txt >/dev/null 2>&1
out=$(printf '@@ big.txt 40 replace\nREWRITTEN\n' | m write - 2>&1); rc=$?
want 1 "$rc" "a truncated read does not license an edit to a withheld line"
grep -q 'has not been read' <<<"$out" && ok "the reason names the unseen lines" || bad "unclear reason"
m read big.txt >/dev/null
out=$(printf '@@ big.txt 40 replace\nREWRITTEN\n' | m write - 2>&1); rc=$?
want 0 "$rc" "reading it whole afterwards licenses the edit"

# 11. body= may contain a real header when the plan says raw=true, and may not
#     when it does not.
fixture
out=$(printf '@@ a.go 3 replace body=1\n@@ b.go 3 replace\n' | m write - 2>&1); rc=$?
want 2 "$rc" "a valid header inside a counted body -> parse error"
out=$(printf '@@ a.go 3 replace body=1 raw=true\n@@ b.go 3 replace\n' | m write - 2>&1); rc=$?
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
#     mrw cannot place as a package — a typo, a directory of prose, one named
#     testdata — must still fall back rather than scope to nothing. A path
#     OUTSIDE the root is the exception and is refused: the fallback runs the
#     whole project, which covers a typo but covers nothing the caller named
#     when the name pointed elsewhere, so the PASS it printed was about a
#     different tree. These rows used to assert that fallback, under the name
#     "as read and write refuse it".
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
out=$(m check internal/aply 2>&1)
grep -q 'echo FULL' <<<"$out" && ok "a mistyped path falls back, not scopes to nothing" || bad "scoped to nothing: $out"
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

# 15c. A .go path that is not there falls back, like the directory typo it is.
#      Placing `filepath.Dir` without asking whether the FILE exists let a
#      mistyped name at a module root scope to `.`, run the root package and
#      report PASS — the silent omission section 15 exists to prevent, wearing
#      the one hat that still fitted.
fixture
mkdir -p "$R/pkg"
printf 'package pkg\n' > "$R/pkg/p.go"
# The fixture root already holds a package, which is what makes these rows able
# to fail: without the existence check a phantom .go name places `.` and scopes
# to the root package, so the run is green and covers nothing the caller named.
printf '{"check":"echo FULL","scoped_check":"echo SCOPED {packages}"}\n' > "$R/.quality-harness.json"
out=$(m check chek.go 2>&1)
grep -qF 'FULL' <<<"$out" && ! grep -qF 'SCOPED' <<<"$out" && ok "a mistyped .go file at the root is not scoped" || bad "scoped a phantom: $out"
out=$(m check nosuchdir/nope.go 2>&1)
grep -qF 'FULL' <<<"$out" && ! grep -qF 'SCOPED' <<<"$out" && ok "a .go file in a missing directory is not scoped" || bad "scoped a phantom dir: $out"
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
m read c.go >/dev/null
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
m read c.go >/dev/null
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
m read w.txt >/dev/null
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
m read a.go locked/f.go >/dev/null
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
  want 2 "$rc" "a plan naming an unwritable file fails"
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
  grep -qE '2 hunk\(s\), 2 file\(s\), 1 failed — NOTHING WRITTEN' <<<"$out" \
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
  printf '@@ a.go 3 replace\nfunc A() int { return 5 }\n' | m write - >/dev/null 2>&1; rc=$?
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
  want 2 "$rc" "a create into a new directory aborts with the rest of the plan"
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
m read 'led.txt:/nomatch/' >/dev/null 2>&1; rc=$?
want 1 "$rc" "a range that matches nothing is reported, not served"
printf '@@ led.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 1 "$rc" "and licenses NO edit — the line was never shown"
grep -q '^c$' "$R/led.txt" && ok "and the file is untouched" || bad "the write landed: $(sed -n 3p "$R/led.txt")"

fixture
printf 'a\nb\nc\nd\ne\n' > "$R/oob.txt"
m read 'oob.txt:99' >/dev/null 2>&1; rc=$?
want 1 "$rc" "a line past the end is reported the same way"
printf '@@ oob.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 1 "$rc" "and licenses no edit either"

# CONTROL A: a whole-file read still licenses the whole file. Breaking this
# trades a permissive bug for a restrictive one, and a guard that refuses
# ordinary work is a guard people turn off with --force.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/whole.txt"
m read whole.txt >/dev/null
printf '@@ whole.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "while a whole-file read still licenses the whole file"

# CONTROL B: a partial read still licenses exactly its own lines — the line it
# showed, and not the one it did not.
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/part.txt"
m read 'part.txt:2' >/dev/null
printf '@@ part.txt 2 replace\nBBB\n' | m write - >/dev/null 2>&1; rc=$?
want 0 "$rc" "and a partial read still licenses the line it showed"
fixture
printf 'a\nb\nc\nd\ne\n' > "$R/part2.txt"
m read 'part2.txt:2' >/dev/null
printf '@@ part2.txt 3 replace\nCCC\n' | m write - >/dev/null 2>&1; rc=$?
want 1 "$rc" "and still refuses the line it did not"

# An empty file cannot satisfy a range either, and used to say so by staying
# silent at exit 0.
fixture
: > "$R/none.txt"
m read 'none.txt:1' >/dev/null 2>&1; rc=$?
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
m read stale.txt:2 >/dev/null 2>&1
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
m read five.txt ten.txt >/dev/null
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
m read 'led.txt:$' >/dev/null
out=$(m seen 2>&1)
grep -E 'led\.txt' <<<"$out" | grep -qE 'lines 5( |$)' \
  && ok "and the ledger records only the line that was served" \
  || bad "ledger over-records: $(grep led.txt <<<"$out")"

# 24. CONCURRENT invocations lose ledger entries, and the loss must stay SAFE.
#     Every `mrw read` rewrites the whole ledger (load, merge, save) with no
#     lock — ADR-002 puts locking permanently out of scope — so parallel reads
#     clobber each other: 40 racing reads kept 1 entry here, while the same 40
#     sequentially, or named in ONE call, keep all 40.
#
#     The lost entries are ACCEPTED. What must hold is the failure DIRECTION
#     ADR-004 leans on: a missing entry costs a re-read and never licenses a
#     wrong write. So the assertion is that the files still IN the ledger are
#     exactly the ones that can be written — not merely that some writes were
#     refused, which would pass on a build that refused everything, and not
#     that changed == applied, which would pass on a build that failed OPEN.
fixture
N=40
for i in $(seq 1 $N); do printf 'a\nb\nc\n' > "$R/r$i.txt"; done
for i in $(seq 1 $N); do m read "r$i.txt" >/dev/null 2>&1 & done
wait
kept=$(m seen 2>/dev/null | grep -cE '(^| )r[0-9]+\.txt$')
if [ "$kept" -ge "$N" ]; then
  # The race did not reproduce — a fast or serialising machine. Asserting the
  # safety property here would assert nothing, so it is skipped and said out
  # loud rather than reported as a pass.
  skip "concurrent reads lose ledger entries (no race observed here: $kept/$N kept)"
else
  ok "concurrent reads lose ledger entries ($kept/$N kept), as ADR-002 accepts"
  applied=0
  for i in $(seq 1 $N); do
    printf '@@ r%s.txt 2 replace\nB\n' "$i" | m write - >/dev/null 2>&1 && applied=$((applied + 1))
  done
  # THE ASSERTION: writability follows the ledger exactly. Fail-open makes this
  # 40; refuse-everything makes it 0; only honouring the surviving entries
  # makes it equal to what survived.
  want "$kept" "$applied" "and exactly the files still in the ledger are writable"
  changed=$(grep -lx 'B' "$R"/r*.txt 2>/dev/null | wc -l | tr -d ' ')
  want "$applied" "$changed" "and no file changed that was not applied"
fi

# 25. A directory that EXISTS and cannot be READ is refused, not read as one
#     holding no package. holdsPackage walks the directory and discarded the
#     walk's error, so a permission failure came back indistinguishable from a
#     directory of prose — and the scope then fell back to the whole-project
#     command. That is sound for a typo, where the full run covers the root,
#     and vacuous here: the caller named a directory, mrw could not open it,
#     and the answer was PASS at exit 0.
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
# CONTROLS: a readable directory still scopes, and a path that is not there
# still falls back — the decided behaviour for a typo, which this must not trade
# away for a refusal.
out=$(m check seen-dir 2>&1)
grep -qF 'SCOPED ./seen-dir/...' <<<"$out" && ok "while a readable directory still scopes" || bad "readable dir broke: $out"
out=$(m check nosuchdir 2>&1)
grep -q 'echo FULL' <<<"$out" && ok "and a mistyped path still falls back" || bad "a typo no longer falls back: $out"

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
m read ord-a.txt ord-b.txt >/dev/null 2>&1
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
m read crlf-plan.txt >/dev/null 2>&1
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
m read race.txt >/dev/null 2>&1
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
m read sub/real.go >/dev/null 2>&1
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
m read --exclude '*.go'                    >/dev/null 2>&1; want 2 $? "--exclude without --grep is a usage error"
m read --grep X --files-from -             >/dev/null 2>&1; want 2 $? "--grep with --files-from is a usage error"
m read --files-from - a.go                 >/dev/null 2>&1; want 2 $? "--files-from with positional paths is a usage error"
m read --grep X a.go:1-2                   >/dev/null 2>&1; want 2 $? "--grep with a positional range is a usage error"
m read --grep X --exclude '['              >/dev/null 2>&1; want 2 $? "a glob path.Match rejects is a usage error"
m read --grep '('                          >/dev/null 2>&1; want 2 $? "a pattern regexp rejects is a usage error"

# The no-argument behaviour is UNCHANGED: without --grep it is still the
# working set, and an empty one is still the same usage error it always was.
m read >/dev/null 2>&1; want 2 $? "no arguments and no --grep still means the working set"

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

m read a.go >/dev/null 2>&1
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
m read a.go >/dev/null 2>&1
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
SRC=$(cd "$(dirname "$0")/.." && pwd)
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
m read a.go >/dev/null 2>&1

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
out=$(printf '@@ a.go 3 replace body=1 raw=true\n@@ still just a body line\n' | m write - 2>&1); rc=$?
want 0 "$rc" "body= with raw=true still applies — the escape hatch is intact"

# And a single guard of each kind is untouched. FRESH fixture: the hunk above
# rewrote line 3, so reusing the file here would fail on the anchor for a
# reason that has nothing to do with guards.
fixture
m read a.go >/dev/null 2>&1
out=$(printf '@@ a.go 3 replace anchor="func A"\nfunc A() int { return 9 }\n' | m write - 2>&1); rc=$?
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
m read a.go >/dev/null 2>&1
printf '\357\273\277@@ a.go 3 replace\nfunc A() int { return 7 }\n' > "$WORK/bom.mrw"
out=$(m write "$WORK/bom.mrw" 2>&1); rc=$?
want 0 "$rc" "a plan with a UTF-8 BOM applies"
grep -q 'return 7' "$R/a.go" \
  && ok "and the edit actually landed" \
  || bad "exit 0 but the file is unchanged: $(sed -n 3p "$R/a.go")"

# The same bytes without the BOM must still apply — the control that proves the
# BOM was the whole difference.
fixture
m read a.go >/dev/null 2>&1
printf '@@ a.go 3 replace\nfunc A() int { return 7 }\n' > "$WORK/nobom.mrw"
m write "$WORK/nobom.mrw" >/dev/null 2>&1
want 0 $? "and the same plan without a BOM still applies"

# A BOM in the BODY is content, not syntax.
fixture
m read a.go >/dev/null 2>&1
printf '@@ a.go 3 replace\n\357\273\277KEEP\n' > "$WORK/inner.mrw"
m write "$WORK/inner.mrw" >/dev/null 2>&1
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
m read a.go >/dev/null 2>&1
printf '\357\273\277@@ a.go 3 replace\nfunc A() int { return 7 }\n' >  "$WORK/f1.mrw"
printf '\357\273\277@@ a.go 4 replace\nfunc B() int { return 8 }\n' >  "$WORK/f2.mrw"
cat "$WORK/f1.mrw" "$WORK/f2.mrw" > "$WORK/both.mrw"
out=$(m write "$WORK/both.mrw" 2>&1); rc=$?
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
m read a.go >/dev/null 2>&1

printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' | m write - >/dev/null 2>&1
want 0 $? "a plan that applies exits 0"
printf '@@ a.go 3 frobnicate\nx\n' | m write - >/dev/null 2>&1
want 2 $? "a plan that does not parse is a usage error"
printf '@@ a.go 3 replace anchor="NOPE"\nx\n' | m write - >/dev/null 2>&1
want 1 $? "a plan that parses but does not apply exits 1"

# mrw's own answer for where this checkout's state lives, rather than guessing
# at XDG layout — contract.sh does not pin XDG_STATE_HOME, it isolates by
# giving every fixture a fresh root.
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

m read a.go >/dev/null 2>&1
printf '@@ a.go 3 replace\nfunc A() int { return 9 }\n' | m write - >/dev/null 2>&1
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
cli=$(printf '%s' "$PLAN" | "$MRW" -C "$R_CLI" write --json - 2>/dev/null); rc=$?
want 0 "$rc" "the CLI applies the plan and emits a receipt"

fixture
R_MCP=$R
req=$(printf '%s' "$PLAN" | python3 -c 'import json,sys; print(json.dumps({"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_write","arguments":{"plan":sys.stdin.read()}}}))')
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
out=$(printf '@@ a.go 4 replace\nfunc B() int { return 22 }\n' | "$MRW" -C "$R_KILL" write - 2>&1); rc=$?
want 0 "$rc" "and a CLI write after the killed server is licensed too"
# 39. ADR-010 T3: the documented host config names a subcommand this binary has.
#
# A config example naming a command the binary does not have is the
# documentation equivalent of a dangling pointer, and this repository shipped
# one on 2026-09-03. The README block is the install path for every MCP user, so
# it is checked against the binary rather than against a reader's patience.
#
# This drives `--help` and greps the README; it does not restate what the block
# should say, because a copy of the answer beside the answer is not a check.
args=$(python3 - "$(cat README.md)" <<'PY'
import json,re,sys
m=re.search(r'### Use it from an MCP host.*?```json\n(.*?)```', sys.argv[1], re.S)
if not m: print("NO-BLOCK"); sys.exit(0)
cfg=json.loads(m.group(1))
srv=cfg["mcpServers"]["mrw"]
print(srv["command"], " ".join(srv["args"]))
PY
)
[ "$args" != "NO-BLOCK" ] \
  && ok "the README carries a parseable MCP host config block" \
  || bad "the documented config block is missing or is not valid JSON"

cmdname=${args%% *}
subcmd=$(printf '%s' "${args#* }" | awk '{print $NF}')
[ "$cmdname" = "mrw" ] \
  && ok "and it invokes the binary by name" \
  || bad "the block invokes '$cmdname', not mrw"

# Redirect before grepping. Under this file's `set -o pipefail`, `grep -q` exits
# on its first match, `--help` takes SIGPIPE writing the rest, and the pipeline
# reports 141 — a row that fails for a reason having nothing to do with what it
# asks. The same trap ate ADR-010-T1's fence earlier the same day.
"$MRW" --help > "$WORK/help.out" 2>&1
grep -qE "^[[:space:]]+$subcmd[[:space:]]" "$WORK/help.out" \
  && ok "and names a subcommand mrw --help lists ($subcmd)" \
  || bad "the documented block names '$subcmd', which this binary does not have"

# And it must actually start: a subcommand that exists but rejects the args the
# README prints is the same dangling pointer one step later.
out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}\n' | "$MRW" "$subcmd" 2>&1); rc=$?
want 0 "$rc" "and the documented invocation actually starts a server"
grep -q 'mrw_write' <<<"$out" \
  && ok "which advertises the tools the README says it does" \
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
    for field in ("title","annotations","outputSchema","_meta"):
        assert field in t, "%s has no %s" % (name, field)
    props=t["outputSchema"].get("properties") or {}
    assert props, "%s declares a property-less schema, which validates anything" % name
    assert t["_meta"]["anthropic/maxResultSizeChars"] > 0, "%s declares no size limit" % name
assert tools["mrw_read"]["annotations"]["readOnlyHint"] is True
assert tools["mrw_write"]["annotations"]["readOnlyHint"] is False
assert tools["mrw_write"]["annotations"]["destructiveHint"] is True
PY
[ $? -eq 0 ] && ok "both tools declare title, annotations, outputSchema and _meta" \
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
m read a.go >/dev/null 2>&1
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

out=$(printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"mrw_read","arguments":{"specs":["wide.go"]}}}\n' | m mcp 2>/dev/null)
want 0 $? "the server answers a read that is too large"
python3 - "$out" <<'PY'
import json,re,sys
r=json.loads(sys.argv[1])["result"]
assert r.get("isError") is True, "an oversized read was not refused"
t=r["content"][0]["text"]
assert "200000" in t, "the refusal does not name the limit: %s" % t[:120]
assert "wide.go" in t, "the refusal does not name the file"
m=re.search(r"wide\.go:1-(\d+)", t)
assert m, "the refusal shows no range to retry with: %s" % t[:160]
assert int(m.group(1)) >= 100, "the suggested range %s is too small to be a useful retry" % m.group(1)
assert "padding padding" not in t, "the refusal carries file content; it truncated rather than refused"
PY
[ $? -eq 0 ] && ok "and refuses it, naming the limit and a range to retry with" \
             || bad "the oversized read was not refused legibly"

# A refused read must license NOTHING: no ledger entry claiming the caller saw it.
out=$(printf '@@ wide.go 2 replace\n// changed\n' | m write - 2>&1); rc=$?
want 1 "$rc" "and a write to the refused file is still refused as unread"

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

rule='3 or more edits, 2 or more files, or several ranges you need to read'
# AGENTS.md wraps the sentence across two lines, so compare on a whitespace-
# folded copy: the rule is the words, not where the paragraph happened to break.
tr -s '[:space:]' ' ' < "$SRC/AGENTS.md" | grep -qF "$rule" \
  && ok "AGENTS.md still states the trigger threshold this row holds the wire against" \
  || bad "the threshold sentence moved in AGENTS.md; the wire and the repository now teach different rules"

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
assert len(i) <= 4096, "the instructions are %d bytes; they are paid once per session" % len(i)
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
PY
[ $? -eq 0 ] && ok "and both tools say when to reach for them, and publish a worked example" \
             || bad "the tool descriptions still say only what the tools do"

# THE ROW: the published plan is one the SHIPPED BINARY accepts. An example
# asserted to be present stays green long after it stops being valid, and it is
# the one thing a caller copies verbatim.
python3 - "$out" > "$WORK/published.plan" <<'PY'
import json,sys
tools={t["name"]: t for t in json.loads(sys.argv[1])["result"]["tools"]}
sys.stdout.write(tools["mrw_write"]["inputSchema"]["properties"]["plan"]["examples"][0])
PY
python3 - "$WORK/published.plan" "$R" <<'PY'
import os,re,sys
# Build the tree the published plan addresses, planting each anchor on the line
# it guards, so the dry run judges the plan and not the fixture.
plan=open(sys.argv[1]).read(); root=sys.argv[2]
last={}; anchors={}
for h in re.finditer(r'^@@ (\S+) (\d+)(?:-(\d+))? (\S+)(.*)$', plan, re.M):
    p,a,b,op,rest=h.group(1),int(h.group(2)),h.group(3),h.group(4),h.group(5)
    hi=int(b) if b else a
    last[p]=max(last.get(p,0),hi)
    m=re.search(r'anchor="([^"]*)"',rest)
    if m: anchors.setdefault(p,{})[a]=m.group(1)
for p,n in last.items():
    full=os.path.join(root,p); os.makedirs(os.path.dirname(full),exist_ok=True)
    with open(full,'w') as f:
        for i in range(1,n+1): f.write(anchors.get(p,{}).get(i,"line %d"%i)+"\n")
PY
paths=$(python3 -c "
import re,sys
print(' '.join(sorted({m.group(1) for m in re.finditer(r'^@@ (\S+) ', open('$WORK/published.plan').read(), re.M)})))
")
# shellcheck disable=SC2086
m read $paths >/dev/null 2>&1
want 0 $? "the files the published plan names can be read"
out=$(m write --dry-run "$WORK/published.plan" 2>&1); rc=$?
want 0 "$rc" "and the plan mrw publishes to a host is one mrw itself accepts"
# "0 failed" contains the word, so match a VERDICT line: `fail` or `skip` in the
# first column. A substring check here would have been green on every run.
grep -qE '^(fail|skip) ' <<<"$out" \
  && bad "a hunk of the published example failed: $out" \
  || ok "and every hunk of the published example passes"

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
for t in tools:
    walk(t["outputSchema"], t["name"])
assert not missing, "undescribed propert(ies): %s" % ", ".join(missing)
# A walk that never descended would find only the top level and report success.
assert len(seen) >= 20, "the walk found only %d properties, so it is not descending: %s" % (len(seen), seen)
assert any(s.startswith("mrw_write:hunks.status") for s in seen), "the walk never reached a hunk verdict"
PY
[ $? -eq 0 ] && ok "every property of every declared outputSchema says what it means, at every depth" \
             || bad "the declared output schemas still describe only their types"

if [ "$fails" -eq 0 ]; then
  echo "contract holds"
else
  echo "$fails assertion(s) FAILED"
fi
exit $(( fails > 0 ))
