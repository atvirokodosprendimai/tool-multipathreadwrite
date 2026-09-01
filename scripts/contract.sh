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
#     testdata, one outside the root — must still fall back rather than scope
#     to nothing.
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
out=$(m check ../outside 2>&1)
grep -q 'echo FULL' <<<"$out" && ok "a path outside the root falls back, as read and write refuse it" || bad "scoped outside the root: $out"
out=$(m check link 2>&1)
# Not a second boundary row: WalkDir does not follow a symlink, so this one
# falls back whether or not rooted.Resolve refuses it. It is here because the
# symlink is the spelling a caller actually reaches for.
grep -q 'echo FULL' <<<"$out" && ok "a symlinked directory is not scoped either" || bad "scoped through a symlink: $out"
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

# 16. A plan aborts with the tree UNTOUCHED when a file cannot be written.
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
if [ "$fails" -eq 0 ]; then
  echo "contract holds"
else
  echo "$fails assertion(s) FAILED"
fi
exit $(( fails > 0 ))
