# Task ADR-024-T2: The contract drives a paged read through the built server and fails if it is flagged

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** Zy
**Produces:** `scripts/contract.sh` §62
**Consumes:** `pagedResult()` and `indexResult()` return `isError` absent (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the absence of isError in the server's JSON-RPC reply`, `the -- PARTIAL: notice in the reply's content[0]`, `errorResult still flagging a refusal`, `the built binary rather than a package-level call`

## Goal

Prove the promise against `$MRW` — the built binary, driven over a real pipe — so a unit test that
passes cannot stand in for a server that regressed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | Adds §62. A unit test proves the function; it cannot prove the shipped server returns it. §38 and §51 already drive `mrw mcp` over a real pipe, so the mechanism exists and is reused. |

## Ordered Steps

1. [S1] Confirm the fence is RED before the row exists: `grep -q '^# 62\. ' scripts/contract.sh` returns non-zero on the current tree, so the Acceptance command fails before any work. [proof: acceptance]
2. [S2] Add §62 to `scripts/contract.sh`, following the shape of §38: build a checkout whose file is large enough to page, send one `tools/call` for `mrw_read` with a bare path through `"$MRW" -C "$R" mcp`, and parse the reply. [proof: acceptance]
3. [S3] Assert the GOOD case: the reply carries no `isError`, its `content[0]` contains `-- PARTIAL:`, and its `content[1]` names a `next_read`. [proof: mutation]
4. [S4] Pair it with the case that MUST fail, per the repository's rule that a promise is paired with its violation: assert that a genuine refusal through the same server — `exclude` without `grep` — still comes back `isError: true`. A row that only checks the absence of a flag would pass against a server that never sets one. [proof: mutation]
5. [S5] Run the whole contract script stand-alone and record its exit code, never through a pipe. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 62\. ' scripts/contract.sh \
  && ./scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `contract.sh §62 (good case)` | `scripts/contract.sh` | The built server's paged reply carries no `isError`, carries `-- PARTIAL:` in `content[0]`, and names `next_read` in `content[1]` | — | S2, S3 |
| `contract.sh §62 (paired failure)` | `scripts/contract.sh` | A genuine refusal through the same server is still `isError: true`, so the row cannot pass against a server that never flags anything | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §62 in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every numbered section; the mutation at S3 restores `IsError: true` in `pagedResult` and §62 must go red |
| 3 — the caller can discover it | n/a: no declared interface — a contract row is a check, not a surface |
| 4 — it is used | Every contract run, and `CONTRIBUTING.md` requires the script before a PR opens |

## Mutation Log

- 2026-09-06 · 69b5f07* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on a page in the BUILT binary; §62 must go red, which is what distinguishes this row from the package-level test · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · covers:the absence of isError in the server's JSON-RPC reply
- 2026-09-06 · 69b5f07* · mutant killed · exit 1 · `internal/mcp/tools.go` · drops the flag from a refusal in the BUILT binary; §62 must go red, which is what stops the row passing against a server that flags nothing at all · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · covers:errorResult still flagging a refusal
- 2026-09-06 · 69b5f07* · mutant killed · exit 1 · `internal/mcp/tools.go` · lowercases the notice the page carries in content[0]; §62 must go red because that text is the only place a paged reply now says it is partial · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · covers:the -- PARTIAL: notice in the reply's content[0]

## Invariants

- §62 drives `$MRW`, the built binary, never a Go test helper. That is the whole reason the row exists.
- The exit code is read directly, never through a pipe — a pipeline reports the last command's status, and this repository has recorded a red run reading as green that way.
- §62's number stays 62; the next free section is computed with `grep -oE '^# [0-9]+\. ' scripts/contract.sh | sort -k2,2n | tail -1`, keyed on the second field and with the trailing space, per `.claude/rules/lifecycle.md`.

## Risks

- The fixture must be large enough to page on any machine that runs the contract. Mitigated: paging is triggered by `MaxResultChars`, a compile-time constant, not by anything host- or machine-dependent, so a fixture sized against it pages deterministically.
- The row could pass against a server that never sets `isError` at all. Mitigated: S4 pairs it with a refusal that must still be flagged.
- `the built binary rather than a package-level call` is declared in **Rests-on:** and carries NO bound mutant, deliberately. It is a property of how this row is written — it drives `$MRW` through `m mcp` — not a mechanism in the source, so every mutation that would break it also breaks the package-level test and proves nothing about the distinction. Declared anyway, so the gate reports one mechanism nothing has shown can fail rather than the record claiming coverage it does not have.

## Stop Condition

Stop and ask if the paged reply from the built binary disagrees with the package-level test from T1 —
that would mean the server and the tested function have diverged, which is a bigger finding than this
task and is ADR-010's territory.

## Out of Scope

- Any assertion about what a HOST does with the reply; §62 can only see what the server sends. The host half is the ADR's Context measurement and is not mechanically checkable from here.
- The ledger premise (deferred: `docs/adr/BACKLOG.md` under ADR-023)

## Verification Log
- 2026-09-06 · 69b5f07 · exit 1 · `set -o pipefail …` · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · ms:28
  ```
  ```
- 2026-09-06 · 69b5f07* · exit 0 · `set -o pipefail …` · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · ms:21659
- 2026-09-06 · 69b5f07* · exit 0 · `set -o pipefail …` · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · ms:21974
- 2026-09-06 · 69b5f07* · exit 0 · `set -o pipefail …` · acceptance-sha256:7bebd8687eb29d49a0302917cb8e65d252cdb3c95979f06a088587e8e7897cab · ms:22216
