# Task ADR-103-T1: `Contains` holds under a filesystem root

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the separator-aware `Contains`; contract §198
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a root contains its children`, `a sibling is not contained`, `a contract row drives the binary`, `only the owned engine packages change`

## Goal

`rooted.Contains` admits every path beneath a root that ends in a separator (`/`, `C:\\`), and still refuses
`/repo-backup` under `/repo`; `mrw --root / read <path>` serves the file.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | `Contains` (`:176-178`) |
| `internal/rooted/contains103_test.go`, `cmd/mrw/rootfs103_test.go` | add | the tests below |
| `scripts/contract.sh` | edit | §198 |

## Ordered Steps

1. [S1] Write the failing tests `TestContainsHoldsUnderAFilesystemRoot` and `TestAReadUnderTheFilesystemRootIsServed`; confirm RED on `f1d5996`. [proof: mutation]
2. [S2] Fix `Contains`. [proof: mutation] Mutant: the separator appended unconditionally again.
3. [S3] Contract §198, driving `$MRW`: `--root / read <a file under $WORK, root-relative>` exits 0 and serves it; the pair, `--root "$R" read ../x` is still refused as outside the root. [proof: human: ./scripts/contract.sh run unpiped before the commit, exit 0 with §198's rows printed]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/rooted/ ./cmd/mrw/ -count=1 -timeout 300s -run 'TestContainsHoldsUnderAFilesystemRoot|TestAReadUnderTheFilesystemRootIsServed' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestContainsHoldsUnderAFilesystemRoot \(' "$out" \
  && grep -qE '^--- PASS: TestAReadUnderTheFilesystemRootIsServed \(' "$out" \
  && grep -q '^# 198\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/lines internal/iter \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/seen internal/lines internal/iter)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestContainsHoldsUnderAFilesystemRoot` | `internal/rooted/contains103_test.go` | under this platform's filesystem root (`/`, or the temp directory's volume root on Windows) a child and a grandchild are contained and the root contains itself; `/repo` contains `/repo/x` and not `/repo-backup` | — | S1, S2 |
| `TestAReadUnderTheFilesystemRootIsServed` | `cmd/mrw/rootfs103_test.go` | `mrw --root <filesystem root> read <temp file, root-relative>` exits 0 and serves its content | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every read, write and check resolves through `rooted`; every state access keys through `state.absReal` |
| 3 — the caller can discover it | the refusal no longer appears; one state directory per checkout (`mrw seen`) |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the 2026-09-30 Codex design review |

## Mutation Log
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/rooted/rooted.go` · the separator appended unconditionally again · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · covers:a root contains its children
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/rooted/rooted.go` · the prefix loses the root, so a sibling is contained · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · covers:a sibling is not contained
- 2026-09-30 · f1d5996* · mutant killed · exit 1 · `internal/lines/lines.go` · an engine package this record does not own changed · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · covers:only the owned engine packages change

## Invariants

- Off Windows, the state key for every root is unchanged.
- `/repo` still does not contain `/repo-backup`.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test's assertion must change to pass (a moved test relocks with a note; its assertions do not change).

## Out of Scope

- The other ADR-103 task.

## Verification Log
- 2026-09-30 · f1d5996* · exit 1 · `set -o pipefail …` · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · ms:405 · test-lock-sha256:c77838ea2606ddc09d720dacf60321927d72922f9ba1e6cc7853f1acd96033f3 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvcm9vdGZzMTAzX3Rlc3QuZ28JVGVzdEFSZWFkVW5kZXJUaGVGaWxlc3lzdGVtUm9vdElzU2VydmVkCTc0NjkzMmI5MzE0MDQ2NzY0ZGQxMmQwYTU2YjMyNWE4ZTcyNzdiZGM5ZjA1N2IzM2Y2NGMwYzk0MWFmNjc4MmIKYm9keQlpbnRlcm5hbC9yb290ZWQvY29udGFpbnMxMDNfdGVzdC5nbwlUZXN0Q29udGFpbnNIb2xkc1VuZGVyQUZpbGVzeXN0ZW1Sb290CWQyZDdlNzE3N2ExZGQ0YjcxNmM5YWFmOGNjZDE0ODM5MDZiOWU2NWUyZmVmNDFjOTI4MjI1ZDg3ZWU1NGVkYjU
  ```
  --- last 10 line(s) of stdout (of 14 after folding 14 raw)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted	0.126s
  === RUN   TestAReadUnderTheFilesystemRootIsServed
      rootfs103_test.go:22: read under /: exit 1:
          ==> private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAReadUnderTheFilesystemRootIsServed2765653125/002/f.txt  REFUSED  private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAReadUnderTheFilesystemRootIsServed2765653125/002/f.txt resolves to /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAReadUnderTheFilesystemRootIsServed2765653125/002/f.txt, which is outside the root /: read it with --root pointed where you mean
          1 range(s) could not be served
  --- FAIL: TestAReadUnderTheFilesystemRootIsServed (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.136s
  FAIL
  ```
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · ms:373
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · ms:412
- 2026-09-30 · f1d5996* · exit 0 · `set -o pipefail …` · acceptance-sha256:72fc600dd83b37db914496cd09fbe925c37724a08c8c23600a34836490878634 · ms:369
- 2026-09-30 · human-observed · S3 observed 2026-09-30: ./scripts/contract.sh run unpiped in the ADR-103 worktree (base f1d5996), exit 0 'contract holds', with §198 printed: --root / read of a.go by its root-relative path exits 0 and serves it; the pair, a path out of a real root, is still refused (exit 1)
