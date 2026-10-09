# Task ADR-134-T1: a file that is the same file as a state file is refused at the boundary

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `identity` in `internal/rooted/identity_unix.go` and `internal/rooted/identity_windows.go`; `stateLinks` in `internal/rooted/hardlink.go`; `state.DirPath` in `internal/state/state.go`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served`

## Goal

Decisions 1–5 of the record, with a test that fails before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go` | edit | `resolveIn` and `Resolver.Resolve` ask `stateLinks` after the state-directory check |
| `internal/rooted/hardlink.go`, `internal/rooted/identity_unix.go`, `internal/rooted/identity_windows.go` | add | the comparison with this checkout's state directory; the identity and link count: `Stat_t` on unix, an attributes-only shared handle on Windows |
| `internal/state/state.go`, `internal/state/dirpath134_test.go` | edit, add | `DirPath`: the state directory without making it |
| `internal/rooted/hardlink134_test.go`, `hardlink134_unix_test.go`, `hardlink134_windows_test.go`, `hardlink134_other_test.go` | add | the tests: refused, symlink to the link, fail closed, held ledger, another checkout's ledger |
| `internal/read/hardlink134_test.go` | add | a `--grep` walk drops the linked file |
| `cmd/mrw/main.go`, `cmd/mrw/hardlink134_test.go` | edit, add | the plan file and the `--files-from` list, which never pass `Resolve`, are judged by `rooted.InStateOrLinked` |
| `scripts/contract.sh` | edit | §236 |
| `AGENTS.md` | edit | the ADR-077 paragraph names the hard link |

## Ordered Steps

1. [S1] Write `TestAHardLinkToMrwsStateIsRefused`: with the state base made and a hard link in the root to a file in it, `Resolve` and a `Resolver` both refuse the link, naming mrw's own state; a hard link to an ordinary file in the root is served by both. Write `TestAWalkDropsAHardLinkToMrwsState` in `internal/read`. Confirm RED.
2. [S2] `identity`, `stateLinks` and `state.DirPath`: a file with a link count of one is never compared; above one (or unreadable), this checkout's state directory is listed once for regular files with a link count above one and the candidate's volume and file index are looked up among theirs. A comparison that cannot be completed refuses. `resolveIn` asks it of the real path, `Resolver.Resolve` of the `Lstat` it already holds, listing once per `Resolver`. Mutants: the lookup answering false (the link is served); the `Resolver` fast path never asking; the pre-test skipping a file with two names; an incomplete listing treated as a complete one. [proof: mutation]
3. [S3] Contract §236 — a hard link to the ledger read exits 1 naming mrw's own state, and the pair, a hard link to an ordinary file, exits 0. AGENTS.md. [proof: acceptance]
4. [S4] Windows timing: push a draft and run the ADR-123/131 timing workflow on windows-latest before and after; record both walks in the ADR Risks row. [proof: human: the numbers come from a CI runner, not from a test]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/rooted/ -count=1 -timeout 300s -run 'TestAHardLinkToMrwsStateIsRefused' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAHardLinkToMrwsStateIsRefused \(' "$out" \
  && go test ./internal/read/ -count=1 -timeout 300s -run 'TestAWalkDropsAHardLinkToMrwsState' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWalkDropsAHardLinkToMrwsState \(' "$out" \
  && go test ./internal/rooted/ ./internal/read/ -count=1 -timeout 900s \
  && grep -q '^# 236\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAHardLinkToMrwsStateIsRefused` | `internal/rooted/hardlink134_test.go` | `Resolve` and a `Resolver` refuse a hard link to a state file and serve a hard link to an ordinary file | none | S1, S2 |
| `TestAWalkDropsAHardLinkToMrwsState` | `internal/read/hardlink134_test.go` | a `--grep` walk neither serves nor matches the linked file | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `hardLinkedToState` and `linkCount` |
| 2 — something selects it | `resolveIn` and `Resolver.Resolve`, on every served regular file |
| 3 — the caller can discover it | the refusal text; AGENTS.md |
| 4 — it is used | the 2026-10-09 Windows chaos round (three sessions); no telemetry (ADR-009) |

## Invariants

- A path that is not a hard link to a state file is judged exactly as before.
- A file with a link count of one costs no syscall beyond the `Lstat` the resolver already took, on unix.

## Risks

- The Windows open per served file; the record's Risks row and S4 measure it.

## Stop Condition

Stop and ask if the Windows walk is more than 1.3 times slower than at v1.52.0 on the timing workflow: the pre-test then needs a volume test the record does not yet specify.

## Out of Scope

- A copy of a state file, and a link whose original a later save replaced (permanent: boundary: the record's Out of Scope)

## Mutation Log
- 2026-10-09 · 10d8a40* · mutant inconclusive · exit 1 · `internal/rooted/hardlink.go` · S2: the comparison with the state base answers false — resolveIn serves a hard link to the ledger · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-09 · 8e66a3b · mutant inconclusive · exit 1 · `internal/rooted/hardlink.go` · S2: the comparison with the state base answers false — resolveIn serves a hard link to the ledger · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-09 · 8e66a3b* · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: the comparison with the state base answers false — resolveIn serves a hard link to the ledger · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 8e66a3b* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: the Resolver fast path never compares — a walk serves a hard link to the ledger · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 8e66a3b* · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: the link-count pre-test skips a file with two names — the comparison never runs for the ledger link · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 83cf91c · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: the lookup among the state files answers no — resolveIn serves a hard link to the ledger · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 83cf91c* · mutant killed · exit 1 · `internal/rooted/rooted.go` · S2: the Resolver fast path never refuses — a walk serves a hard link to the ledger · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 83cf91c* · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: the pre-test skips a file with two names — the lookup never runs for the ledger link · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 83cf91c* · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: an incomplete listing of the state directory is treated as complete — the alias is served · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 5d45593 · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: the state directory is listed as spelled — a moved and linked directory scans empty and complete · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served
- 2026-10-09 · 0dc41c9 · mutant killed · exit 1 · `internal/rooted/hardlink.go` · S2: a link in the state directory is skipped — the ledger moved out and linked back is not compared · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · covers:a hard link to a state file is refused by the boundary and a hard link to an ordinary file is served

## Verification Log
- 2026-10-09 · 10d8a40* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:14332
- 2026-10-09 · 8e66a3b · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:17114
- 2026-10-09 · 8e66a3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:15535
- 2026-10-09 · 8e66a3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:14703
- 2026-10-09 · 8e66a3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:16026
- 2026-10-09 · 8e66a3b* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:13716
- 2026-10-09 · human-observed · Zy's session read the windows-latest timing-123 run 37977126786 (v1.52.0 vs the ADR-134 branch): plain tree 243 to 298 ms, junction root 236 to 306 ms, BenchmarkWalkDeep131 1.25x; all within the 1.3x bar, recorded in the ADR Risks row
- 2026-10-09 · 83cf91c · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:14481
- 2026-10-09 · 83cf91c* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:15025
- 2026-10-09 · 83cf91c* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:13814
- 2026-10-09 · 83cf91c* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:13875
- 2026-10-09 · 83cf91c* · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:13822
- 2026-10-09 · 5d45593 · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:16923
- 2026-10-09 · 0dc41c9 · exit 0 · `set -o pipefail …` · acceptance-sha256:95e68c98f38b7c475fae0d5f735bc51c97122eb93776162969b560bbbea5d975 · ms:13973
