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

echo
if [ "$fails" -eq 0 ]; then
  echo "contract holds"
else
  echo "$fails assertion(s) FAILED"
fi
exit $(( fails > 0 ))
