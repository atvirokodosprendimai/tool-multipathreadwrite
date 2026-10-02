# Task ADR-114-T2: apply_patch compiles Move to with hunks

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `--format=apply_patch` compiles `*** Move to:` with hunks
**Consumes:** the engine accepts edit + rename (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `apply_patch compiles Move to with hunks`

## Goal

`ingest.CompileApplyPatch` compiles an `*** Update File:` section that carries `*** Move to:` and hunks into the update hunks against the source plus one `rename`, and the CLI and MCP apply it in one plan.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/ingest/applypatch.go` | edit | the three refusals go; the rename is emitted when the section ends |
| `internal/ingest/move114_test.go` | add | `TestAMoveWithHunksCompiles` |
| `internal/ingest/applypatch_unlink_test.go` | edit | `TestCompileMoveToIsRename`'s with-hunks half now compiles and applies |
| `AGENTS.md`, `README.md` | edit | the ops paragraph |
| `scripts/contract.sh` | edit | §214; §99's second pair now asserts the move applies |

## Ordered Steps

1. [S1] Write `TestAMoveWithHunksCompiles`: a section with Move to before its hunks, and one with Move to after them, compile to the hunks plus a rename and apply through `plan.Parse` and `apply.Apply` to the edited content at the destination; a Move to with no Update File stays refused. Confirm RED. [proof: mutation]
2. [S2] The compiler change; AGENTS.md and README; contract §214 through `$MRW` — `write --format=apply_patch` moves and edits a file, exit 0, and the pair: a move whose hunk matches nothing is refused at compile, exit 2 (ADR-051's code), nothing moved. §99's second pair now has an unread source refused, exit 1. Mutant: the rename is not emitted. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/ingest/ -count=1 -timeout 300s -run 'TestAMoveWithHunksCompiles' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAMoveWithHunksCompiles \(' "$out" \
  && ! grep -q 'Move to with hunks is not compiled' internal/ingest/applypatch.go \
  && grep -q '^# 214\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l cmd internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAMoveWithHunksCompiles` | `internal/ingest/move114_test.go` | Move to before or after its hunks compiles to the hunks plus a rename that applies to the edited content at the destination; a Move to with no Update File is refused | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every apply_patch document with Move to and hunks |
| 3 — the caller can discover it | AGENTS.md and README; the receipt |
| 4 — it is used | Codex emits this shape; telemetry is refused (ADR-009) |

## Mutation Log
- 2026-10-02 · e1c7646* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · the rename is not emitted: the edit lands in place and nothing moves · acceptance-sha256:4993d731a69642b394fc5e459e0132353336d38a24e5c5d093986bfe962ef890
- 2026-10-02 · 8307f87* · mutant killed · exit 1 · `internal/ingest/applypatch.go` · a Move to between hunk lines joins them · acceptance-sha256:4993d731a69642b394fc5e459e0132353336d38a24e5c5d093986bfe962ef890

## Invariants

- A Move to with no hunks compiles as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The engine (permanent: boundary: T1)

## Verification Log
- 2026-10-02 · e1c7646* · exit 1 · `set -o pipefail …` · acceptance-sha256:4993d731a69642b394fc5e459e0132353336d38a24e5c5d093986bfe962ef890 · ms:729 · test-lock-sha256:5bf7d57b87132226c06fbedf26a4349514a78ce78d12869a8ee2f06ad80a15f7 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvaW5nZXN0L21vdmUxMTRfdGVzdC5nbwlUZXN0QU1vdmVXaXRoSHVua3NDb21waWxlcwlkM2MwNTVhZmQxNDMxZTY0ODEwNjBhOWIxZThkOTk5OTFjMDQ0MTI0YTYyYTMyNmNhM2NjNzljMDA1MTRlZDNm
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAMoveWithHunksCompiles
      move114_test.go:26: move first: compile refused a move with a hunk: apply_patch: Move to with hunks is not compiled this slice
  --- FAIL: TestAMoveWithHunksCompiles (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest	0.083s
  FAIL
  ```
- 2026-10-02 · e1c7646* · exit 0 · `set -o pipefail …` · acceptance-sha256:4993d731a69642b394fc5e459e0132353336d38a24e5c5d093986bfe962ef890 · ms:1105
- 2026-10-02 · 8307f87* · exit 0 · `set -o pipefail …` · acceptance-sha256:4993d731a69642b394fc5e459e0132353336d38a24e5c5d093986bfe962ef890 · ms:768
- 2026-10-02 · 8307f87* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:4993d731a69642b394fc5e459e0132353336d38a24e5c5d093986bfe962ef890 · ms:0 · test-lock-sha256:e700d7f43f52d0f11daded415acd91693005833b809c47c733912a07e999f843 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvaW5nZXN0L21vdmUxMTRfdGVzdC5nbwlUZXN0QU1vdmVXaXRoSHVua3NDb21waWxlcwliZTdkNmYzMTQyNTBiZjllYjNjNWY2N2M2M2E4MjQ3YzM0NjUzNDNmY2ExZWRkMzZmNTI5MmVlZDNhZThhMTIx · test-lock-kind:replace
