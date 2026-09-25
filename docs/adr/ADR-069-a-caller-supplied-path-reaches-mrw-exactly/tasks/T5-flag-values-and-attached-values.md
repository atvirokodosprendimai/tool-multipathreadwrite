# Task ADR-069-T5: A flag value does not end the guard; an attached padded value is refused; contract §134

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `refusePaddedArgs` that skips flag values, `padAttached`, `refusePaddedFlagValues` in `main`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a flag value does not end the guard`, `an attached padded value is refused`, `the iter refusal keeps the verb`, `the binary refuses them`, `no engine file changes`

## Goal

The Codex review of v1.25.0 found three gaps in T1. First, `read --grep -- 'x '` got past the guard, because the helper stopped at a `--` that `--grep` consumed as its value. Second, `--files-from='list '` and `--root='dir '` were silently trimmed, because urfave trims an attached value along with its token. Third, the iter refusal suggested `mrw iter -- 'x '`, which makes the path the verb. The fix: the helper skips a value-taking flag's own value, so only a `--` in argument position ends it. An attached value that ends in whitespace is refused, naming the separate spelling. `main` checks the whole argv, because root flags never reach a subcommand's tail. The iter suggestion keeps its verb.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `refusePaddedArgs`, `padAttached`, `refusePaddedFlagValues`, `main` |
| `cmd/mrw/paddedflag_test.go` | new | the three tests |
| `scripts/contract.sh` | edit | §134 |

## Ordered Steps

1. [S1] Write the three tests and a no-op `refusePaddedFlagValues` stub; confirm RED on assertions. [proof: mutation]
   - `read --grep -- 'x '` exits 2 naming `'x '`. `read --grep plain --exclude ' x' x` exits 0, which v1.25.0 refused. `read --grep plain -- 'x '` exits 1 and does not serve `x`.
   - `--files-from='list '` exits 2 and says to pass the value as its own argument. `--files-from 'list '` serves `x `. `refusePaddedFlagValues` refuses `--root=/tmp/d ` and accepts a path after `--`.
   - `iter add 'x '` suggests `mrw iter add -- 'x '`.
2. [S2] Implement; GREEN; every `cmd/mrw` test stays green. [proof: mutation]
   Mutants: the value skip removed; `padAttached` returns nil; the iter prefix dropped.
3. [S3] §134 through the binary, including the root flag. [proof: mutation]
4. [S4] `gofmt`, `go vet`, unpiped. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 134\. ' scripts/contract.sh \
  && go test ./cmd/mrw/ -count=1 -v -run 'TestAFlagValueDoesNotEndThePaddedArgGuard|TestAnAttachedFlagValueWithTrailingSpaceIsRefused|TestTheIterRefusalKeepsTheVerb' 2>&1 | tee /tmp/adr069-T5.out \
  && grep -q '^--- PASS: TestAFlagValueDoesNotEndThePaddedArgGuard ' /tmp/adr069-T5.out \
  && grep -q '^--- PASS: TestAnAttachedFlagValueWithTrailingSpaceIsRefused ' /tmp/adr069-T5.out \
  && grep -q '^--- PASS: TestTheIterRefusalKeepsTheVerb ' /tmp/adr069-T5.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr069-T5.out \
  && ./scripts/contract.sh > /tmp/adr069-T5-contract.out 2>&1 \
  && grep -q '^  PASS  a -- consumed as a flag value does not end the guard' /tmp/adr069-T5-contract.out \
  && grep -q '^  PASS  and so is a padded attached root flag' /tmp/adr069-T5-contract.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/plan internal/seen internal/check internal/state internal/lines)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(gofmt -l cmd/mrw)" ] \
  && go vet ./cmd/mrw/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFlagValueDoesNotEndThePaddedArgGuard` | `cmd/mrw/paddedflag_test.go` | a `--` flag value does not end the guard; a padded flag value is no longer refused | — | S1, S2 |
| `TestAnAttachedFlagValueWithTrailingSpaceIsRefused` | `cmd/mrw/paddedflag_test.go` | attached padded values are refused, subcommand and root; the separate spelling works | — | S1, S2 |
| `TestTheIterRefusalKeepsTheVerb` | `cmd/mrw/paddedflag_test.go` | the iter suggestion is runnable | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests and §134 |
| 2 — something selects it | every CLI call passes `main` and its subcommand's Action |
| 3 — the caller can discover it | the refusal names the spelling that works |
| 4 — it is used | §134 drives the built binary |

## Verification Log
(empty until execute)
- 2026-09-25 · 5165f0a* · exit 1 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:936 · test-lock-sha256:0036500d792c04f9a96859ef39e7f6641820cfe1c4f501288258693a60d98e4f · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcGFkZGVkZmxhZ190ZXN0LmdvCVRlc3RBRmxhZ1ZhbHVlRG9lc05vdEVuZFRoZVBhZGRlZEFyZ0d1YXJkCTdkMWYyODA4OGU1YjZkZDAzOWJjMGIzYWNhMDljYjE1NzhmMzE0OWVhNDBmYmViMzA4YTZhMjgzOWMxNjhlMWYKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0QW5BdHRhY2hlZEZsYWdWYWx1ZVdpdGhUcmFpbGluZ1NwYWNlSXNSZWZ1c2VkCWZlOTJiZWVkN2QzMzAzNDRiYWQ5OWIyZTljNDc1N2ZjZjFjMDc4MzRiYzhmNTVmODdkZWU0ZmRkMzczZGUxN2IKYm9keQljbWQvbXJ3L3BhZGRlZGZsYWdfdGVzdC5nbwlUZXN0VGhlSXRlclJlZnVzYWxLZWVwc1RoZVZlcmIJODNiYTkyNmVmYTlkYzU5ZTVjODMxOTZjMmE3ZTk3NTk5YWIyMmZjMTZkMjg0NDUwNWE2NzM3OTM3MWUxMDg5OA
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
              1| plain
      paddedflag_test.go:65: a padded attached --root value was not refused
  --- FAIL: TestAnAttachedFlagValueWithTrailingSpaceIsRefused (0.00s)
  === RUN   TestTheIterRefusalKeepsTheVerb
      paddedflag_test.go:78: iter add 'x ' exited 2, want 2 suggesting mrw iter add -- 'x ':
          'x ' has edge whitespace the argument parser strips; put -- before the path: mrw iter -- 'x '
  --- FAIL: TestTheIterRefusalKeepsTheVerb (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.014s
  FAIL
  ```
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:52908
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:77228
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:60927
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:41158
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:35492
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:36537
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:36979
- 2026-09-25 · 5165f0a* · exit 0 · `set -o pipefail …` · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · ms:33758

## Mutation Log
(empty until execute)
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `cmd/mrw/main.go` · the value skip is dropped: read --grep -- x-space ends the guard again and the tests go red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:a flag value does not end the guard
- 2026-09-25 · 5165f0a* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · padAttached accepts everything: --files-from=list-space and --root=dir-space pass, the tests and §134 go red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:an attached padded value is refused
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-25 · 5165f0a* · mutant inconclusive · exit 1 · `cmd/mrw/main.go` · padAttached accepts everything: --files-from=list-space and --root=dir-space pass, the tests and §134 go red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:the binary refuses them
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `cmd/mrw/main.go` · the iter prefix loses its verb: the iter test goes red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:the iter refusal keeps the verb
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `internal/read/read.go` · an engine file ADR-069 does not own changes: the go/no-go guard must go red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:no engine file changes
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `cmd/mrw/main.go` · padAttached accepts everything: --files-from=list-space and --root=dir-space pass, the tests and §134 go red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:an attached padded value is refused
- 2026-09-25 · 5165f0a* · mutant killed · exit 1 · `cmd/mrw/main.go` · padAttached accepts everything: --files-from=list-space and --root=dir-space pass, the tests and §134 go red · acceptance-sha256:952c7f98a81a29366f41769ec0a63d9fde0e327265ebeb4ca0d92442431f015d · covers:the binary refuses them

## Invariants

- `read -- 'x '` and every unpadded call behave as before.
- A separate flag value is never refused.

## Risks

- A flag unknown to `cmd.Flags` is treated as value-taking, so the token after it is skipped. The only such flag is urfave's own `--help`, which exits before any Action.

## Out of Scope

- Recovering an attached value instead of refusing it (permanent: boundary: the parser has already trimmed it; ADR-069 refuses rather than guesses)

## Stop Condition

Stop if the fix needs a format change or an exit-code change.
