# Task ADR-145-T1: The install check migrates nothing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `migrateLegacyState` called from the root command's `Before`
**Consumes:** `state.Migrate` (existing)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a start the parser answers on its own, and version and instructions, migrate no legacy state, and every other command still does`

## Goal

Decisions 1 to 3 of the record, with a test that fails before them. The first design (a predicate on `os.Args` in `main`) was found incomplete twice by the Codex review of #392 and is replaced by moving the migration into `Before`; its task rows were removed with it and are in the git history of the branch.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/startup.go` | add | `migrateLegacyState` |
| `cmd/mrw/main.go` | edit | `Before` calls it for every verb but `version` and `instructions`; `main` no longer migrates |
| `cmd/mrw/startup145_test.go` | add | the table through the CLI, and the pair |
| `scripts/contract.sh` | edit | §249 |

## Ordered Steps

1. [S1] Write `TestAStartThatTouchesNoStateMigratesNothing`. Confirm RED.
2. [S2] Move the migration into `Before`, skipped for `version` and `instructions`. Mutant: the verb guard removed. [proof: mutation]
3. [S3] Contract §249 drives the built binary on a legacy `.mrw/seen`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAStartThatTouchesNoStateMigratesNothing' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAStartThatTouchesNoStateMigratesNothing \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && grep -q '^# 249\. ' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAStartThatTouchesNoStateMigratesNothing` | `cmd/mrw/startup145_test.go` | `version`, `instructions`, every version-flag spelling tried, `-h`, `--help`, `read -h` and `-C . version` migrate nothing, and a `read` after each does | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `migrateLegacyState` |
| 2 — something selects it | the root command's `Before`, which every command that is not answered by the parser passes |
| 3 — the caller can discover it | `mrw version` prints no "moved" line and writes nothing |
| 4 — it is used | contract §249 drives the built binary; no telemetry (ADR-009) |

## Invariants

- Every other command migrates exactly as before, once.
- Exit codes and output of `version` and `instructions` are unchanged.

## Risks

- A start that fails to parse no longer migrates. It never needed state.

## Stop Condition

Stop and ask if a command the parser answers on its own needs the migrated state.

## Out of Scope

- Where the migration looks (permanent: boundary: the working directory, as ADR-004 chose)

## Mutation Log
- 2026-10-10 · 7953ae1 · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the verb guard is removed, version and instructions migrate · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · covers:a start the parser answers on its own, and version and instructions, migrate no legacy state, and every other command still does
- 2026-10-10 · 7953ae1* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: no command migrates · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · covers:a start the parser answers on its own, and version and instructions, migrate no legacy state, and every other command still does

## Verification Log
- 2026-10-10 · ccef5af* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · ms:39107
- 2026-10-10 · ccef5af* · exit 1 · `set -o pipefail …` · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · ms:1257 · test-lock-sha256:76fd035e18a267cbb58f1b1ba7b92cff3e498cb6999c300df7ecfede71942d6d · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdGFydHVwMTQ1X3Rlc3QuZ28JVGVzdEFTdGFydFRoYXRUb3VjaGVzTm9TdGF0ZU1pZ3JhdGVzTm90aGluZwliNGI2M2Y5OTY2MDY0ZmIzOTNjMGQ1Y2QzOTBiOGI2MzU1NTcwYzhlMzEyZjRhYTY5M2IxNjZmMDMyMGMxMTFi
  ```
  --- last 10 line(s) of stdout (of 53 after folding 53 raw)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/--version= (0.01s)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/-v=0 (0.01s)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/-v=T (0.01s)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/-h (0.01s)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/--help (0.01s)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/read_-h (0.00s)
      --- FAIL: TestAStartThatTouchesNoStateMigratesNothing/-C_._version (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.337s
  FAIL
  ```
- 2026-10-10 · 7953ae1 · exit 0 · `set -o pipefail …` · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · ms:39313
- 2026-10-10 · 7953ae1* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · ms:43067
- 2026-10-10 · 7953ae1* · exit 0 · `set -o pipefail …` · acceptance-sha256:68c8ffa2a2e451531c3169f38ef9d761e2ca08a043cbb0da9a9026004671aa6a · ms:48308
