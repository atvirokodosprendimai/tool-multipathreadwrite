# Task ADR-077-T1: mrw's state is refused at the boundary

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `state.Base`, `rooted.InState`; the refusal in `rooted.Resolve`; the ast-grep drop
**Consumes:** `state.stateHome`, `rooted.Contains`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a path inside the state base is refused`, `naming the base does not make it`, `a discovered state file is dropped`, `the ack store and the ledger are not served over MCP`, `a contract row drives the binary`, `ADR-007 records the amendment`, `the packages vet for Windows`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

With the state base under the root, a read served mrw's ledger and ack store and a plan could edit
them. Refuse every path inside the base, at the boundary.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/state.go`, `internal/state/prune.go` | edit | `Base`, used by `Dir` and `openBase` |
| `internal/rooted/rooted.go` | edit | `InState`; the refusal in `Resolve` |
| `internal/read/astgrep.go` | edit | a discovered hit inside the base is dropped |
| `*/state077_test.go`, `internal/state/base077_test.go` | new | the tests below |
| `scripts/contract.sh` | edit | §154 |
| `AGENTS.md`, `docs/adr/ADR-007-mrw-finds-the-files-it-serves.md`, `docs/adr/BACKLOG.md` | edit | the rule and the amendment |

## Ordered Steps

1. [S1] Write the tests; confirm RED. [proof: mutation]
2. [S2] Implement; GREEN. [proof: mutation]
   Mutants: `InState` returns false; `Resolve` skips the `InState` refusal; the ast-grep drop removed; `RealAsFarAsItExists` returns its argument unresolved.
3. [S3] Contract §154, AGENTS.md, ADR-007 and BACKLOG. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/rooted/ ./internal/state/ ./internal/read/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 240s -run 'TestACaseSpellingOfTheStateBaseIsRefused|TestTheStateCannotBeReadAsAListOrAPlan|TestAPathInsideMrwsStateIsRefused|TestBaseDoesNotCreateTheStateDirectory|TestTheWalkServesNothingFromMrwsState|TestTheAckStoreCannotBeReadOverMCP|TestAPlanCannotEditTheLedger|TestAstGrepDropsAHitInsideMrwsState' -v 2>&1 | tee /tmp/adr077-T1.out \
  && missing=$(for t in TestACaseSpellingOfTheStateBaseIsRefused TestTheStateCannotBeReadAsAListOrAPlan TestAPathInsideMrwsStateIsRefused TestBaseDoesNotCreateTheStateDirectory TestTheWalkServesNothingFromMrwsState TestTheAckStoreCannotBeReadOverMCP TestAPlanCannotEditTheLedger TestAstGrepDropsAHitInsideMrwsState; do grep -qE "^--- PASS: $t \(" /tmp/adr077-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 154\. ' scripts/contract.sh \
  && grep -q 'Amended by ADR-077' docs/adr/ADR-007-mrw-finds-the-files-it-serves.md \
  && GOOS=windows go vet ./internal/rooted/ ./internal/state/ ./internal/read/ \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/check internal/lines internal/iter internal/seen internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/apply internal/plan internal/check internal/lines internal/iter internal/seen internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPathInsideMrwsStateIsRefused` | `internal/rooted/state077_test.go` | refused before and after the base exists; a sibling of mrw/ is served | — | S1, S2 |
| `TestBaseDoesNotCreateTheStateDirectory` | `internal/state/base077_test.go` | `Base` names without making; `Dir` is under it | — | S1, S2 |
| `TestTheWalkServesNothingFromMrwsState` | `internal/read/state077_test.go` | a grep finds the caller's file and nothing of mrw's | — | S1, S2 |
| `TestTheAckStoreCannotBeReadOverMCP` | `internal/mcp/state077_test.go` | `pending.json` is refused over MCP | — | S1, S2 |
| `TestAPlanCannotEditTheLedger` | `internal/mcp/state077_test.go` | a plan on `seen` is refused, bytes unchanged | — | S1, S2 |
| `TestAstGrepDropsAHitInsideMrwsState` | `cmd/mrw/state077_test.go` | an ast-grep hit inside the base is dropped, the caller's served, exit 0 | — | S1, S2 |
| `TestACaseSpellingOfTheStateBaseIsRefused` | `internal/rooted/state077_test.go` | `.st/MRW` is refused where the filesystem folds case, served where it keeps it (reviews of #238) | — | S2 |
| `TestTheStateCannotBeReadAsAListOrAPlan` | `cmd/mrw/state077_test.go` | `--files-from` and a plan file naming the ack store are refused, and no checkpoint id is quoted (Codex review of #238) | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.InState`, `state.Base` |
| 2 — something selects it | every `Resolve`, so every surface |
| 3 — the caller can discover it | the refusal names mrw's own state; AGENTS.md says so |
| 4 — it is used | `--root "$HOME"` holds the state base by default |

## Verification Log
(empty until execute)
- 2026-09-26 · ce2318c* · exit 1 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:1533 · test-lock-sha256:242fe4ee1057b0b66448c5dbda826f4b6866800e00f98fc90c31b5cff5a3d030 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvc3RhdGUwNzdfdGVzdC5nbwlUZXN0QXN0R3JlcERyb3BzQUhpdEluc2lkZU1yd3NTdGF0ZQlkZjIwMmRjYmRkMDY1NTUyZDY0YmU2ZWIxODJhNzY5MDhhNDA3M2NlZWU0MjMzMjgyYjdlN2ZkNzg4ZjZiNTA3CmJvZHkJaW50ZXJuYWwvbWNwL3N0YXRlMDc3X3Rlc3QuZ28JVGVzdEFQbGFuQ2Fubm90RWRpdFRoZUxlZGdlcgk0ZTdkYTM0ZGNjNDQ4ZjQzMGU3M2EzNGIyYWMzYjE4NWEzNzNkYjNjZDYxMTNmOGMxY2Q3NjllYzA2MjRjNDdiCmJvZHkJaW50ZXJuYWwvbWNwL3N0YXRlMDc3X3Rlc3QuZ28JVGVzdFRoZUFja1N0b3JlQ2Fubm90QmVSZWFkT3Zlck1DUAkyOWM4OGFkMjA2NGFmNzljYTAwMDA2MGFjOTdmODhlYzM4Y2I5NmNiZmExYjBiMmIzMTNkZWM1NTljNzNhNzhhCmJvZHkJaW50ZXJuYWwvcmVhZC9zdGF0ZTA3N190ZXN0LmdvCVRlc3RUaGVXYWxrU2VydmVzTm90aGluZ0Zyb21NcndzU3RhdGUJM2FiZWM0Y2RiNzM0Y2M0MDExNDQzYTU1MmU0MDI3YjNjYTU4Mjc0NDI0ZTYxZjc4Mzc5ZDFhZTAwNjYzODg1MQpib2R5CWludGVybmFsL3Jvb3RlZC9zdGF0ZTA3N190ZXN0LmdvCVRlc3RBUGF0aEluc2lkZU1yd3NTdGF0ZUlzUmVmdXNlZAliNjUyMzgyZTI3YzZiMDZhYmIwMzc2ZGNkMTVmN2M0NDljOGVjYmY1NGYyZmNjODY1ZTE5NDljODJmMTQwNTg4CmJvZHkJaW50ZXJuYWwvc3RhdGUvYmFzZTA3N190ZXN0LmdvCVRlc3RCYXNlRG9lc05vdENyZWF0ZVRoZVN0YXRlRGlyZWN0b3J5CTI1NWIwMDIwMDc5YmQyY2E4M2M2ZWQ1NjFkMjRmOWI1ZTlhNTI1YTQ5MzM5NzBiNTMyZjZmOWM5YzVmNTk0ODg
  ```
  --- last 10 line(s) of stdout (of 45 after folding 45 raw)
          ==> .st/mrw/k/seen  1L  17B  sha 06ea3640
          @@ 1-1
              1| func Target() {}
          ==> hit.go  2L  29B  sha 6050a0bf
          @@ 2-2
              2| func Target() {}
  --- FAIL: TestAstGrepDropsAHitInsideMrwsState (0.26s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.498s
  FAIL
  ```
- 2026-09-26 · ce2318c* · exit 0 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:795
- 2026-09-26 · ce2318c* · exit 0 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:731
- 2026-09-26 · ce2318c* · exit 0 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:713
- 2026-09-26 · ce2318c* · exit 0 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:732
- 2026-09-26 · ce2318c* · exit 0 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:707
- 2026-09-26 · ce2318c* · exit 0 · `set -o pipefail …` · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · ms:656
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:1217
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:814
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:705
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:806
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:725
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:767
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:787
- 2026-09-26 · 2628cac* · exit 0 · `set -o pipefail …` · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · ms:685

## Mutation Log
(empty until execute)
- 2026-09-26 · ce2318c* · mutant killed · exit 1 · `internal/rooted/rooted.go` · InState never reports a path inside the base · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · covers:a path inside the state base is refused
- 2026-09-26 · ce2318c* · mutant killed · exit 1 · `internal/rooted/rooted.go` · Resolve skips the state refusal · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · covers:a path inside the state base is refused
- 2026-09-26 · ce2318c* · mutant killed · exit 1 · `internal/read/astgrep.go` · an ast-grep hit inside the state is served · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · covers:a discovered state file is dropped
- 2026-09-26 · ce2318c* · mutant killed · exit 1 · `internal/rooted/rooted.go` · the base and the path are compared by spelling · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · covers:a path inside the state base is refused
- 2026-09-26 · ce2318c* · mutant killed · exit 1 · `internal/state/state.go` · Base names a directory the state is not in · acceptance-sha256:21db7f7f209f96550480349104a5d593528adb6d9c9396b8f55d40a15d0bbee6 · covers:naming the base does not make it
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `internal/rooted/rooted.go` · InState never matches by spelling · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:a path inside the state base is refused
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `internal/rooted/rooted.go` · Resolve skips the state refusal · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:a path inside the state base is refused
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `internal/read/astgrep.go` · an ast-grep hit inside the state is served · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:a discovered state file is dropped
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `internal/state/state.go` · Base names a directory the state is not in · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:naming the base does not make it
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `internal/rooted/rooted.go` · a case spelling of the base is compared by string · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:a path inside the state base is refused
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `cmd/mrw/main.go` · a --files-from list inside the state is read · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:the ack store and the ledger are not served over MCP
- 2026-09-26 · 2628cac* · mutant killed · exit 1 · `cmd/mrw/main.go` · a plan file inside the state is read · acceptance-sha256:cdf2aa58e9cc9577c9917beb688e08c1a239452491b36f54a77806b9504e2290 · covers:the ack store and the ledger are not served over MCP

## Invariants

- Nothing inside mrw's state base is served or edited through mrw, on any surface.

## Risks

- A walk under a root that holds a large state directory walks into it and refuses each file (Out of Scope in the record: no second rule for the same fact).

## Out of Scope

- Skipping the directory before the walk enters it (permanent: fact: the boundary already refuses every file inside)

## Stop Condition

The fence exits 0 and every Mutation Log row reads `killed`.
