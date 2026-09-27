# Task ADR-086-T4: a --json plan refuses a name its receipt cannot represent

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `--json` refusal in the CLI write action
**Consumes:** the byte-preserving `splitHeader` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a --json create with an invalid name writes nothing`, `a --json rename to an invalid name writes nothing`, `a valid name still applies`, `a contract row drives the binary`, `the other engine packages are unchanged`, `go.mod declares one requirement`

## Goal

`encoding/json` turns a byte that is not valid UTF-8 into U+FFFD, so once T1 kept the byte, a `--json` receipt on a filesystem that holds it named a file other than the one written (Codex review of #254).

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/json086_test.go` | new | a create and a rename to an invalid name under `--json` write nothing; a valid create applies |
| `cmd/mrw/main.go` | edit | the refusal after the plan parsed |
| `scripts/contract.sh` | edit | §169 |

## Ordered Steps

1. [S1] Write `TestAJSONPlanWithANameItCannotRepresentWritesNothing`; it fails without the refusal. [proof: mutation]
2. [S2] Refuse such a plan under `--json` before anything lands. [proof: mutation]
3. [S3] Contract §169 pairs the refusal with a valid `--json` create. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -count=1 -timeout 180s -run 'TestAJSONPlanWithANameItCannotRepresentWritesNothing' -v 2>&1 | tee /tmp/adr086-T4.out \
  && missing=$(for t in TestAJSONPlanWithANameItCannotRepresentWritesNothing; do grep -qE "^--- PASS: $t \(" /tmp/adr086-T4.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 169\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/seen internal/check internal/state internal/lines internal/iter internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAJSONPlanWithANameItCannotRepresentWritesNothing` | `cmd/mrw/json086_test.go` | exit 2 naming the byte, nothing written; a valid name applies | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the check after `parsed = true` |
| 2 — something selects it | every CLI `write --json` |
| 3 — the caller can discover it | the refusal says why and names `--json` as the thing to drop |
| 4 — it is used | the Codex review of #254 traced it; ADR-009 refuses telemetry |

## Verification Log
(empty until execute)
- 2026-09-27 · 318a8c4* · exit 1 · `set -o pipefail …` · acceptance-sha256:3d0dfbeed2a71c1ba54318271f9ef607d92f10ef6a6fe51d23e2a783611c3cb7 · ms:2393 · test-lock-sha256:7ef9d92c7667e0c9f864c5331680be842766b899fb070e530fddd1a1dd3dcaed · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWNtZC9tcncvanNvbjA4Nl90ZXN0LmdvCVRlc3RBSlNPTlBsYW5XaXRoQU5hbWVJdENhbm5vdFJlcHJlc2VudFdyaXRlc05vdGhpbmcJOWVjOWM1NDBiYWIwZDA4YWRmMGM2Nzc4MjA0MTdjZjFiODFlNzUwMjlhNjM2NmYzNjc5NjMwNTQwMGRkMmI1ZQ
  ```
  --- last 10 line(s) of stdout (of 77 after folding 77 raw)
              "advisory_writes": 0,
              "window": 0,
              "fires": false
            },
            "error": "b.txt: the filesystem will not create this name: open /private/var/folders/cp/56m_2hr965zcc37hrln0_fz80000gn/T/TestAJSONPlanWithANameItCannotRepresentWritesNothing3052405085/002/bad�name.txt: illegal byte sequence"
          }
  --- FAIL: TestAJSONPlanWithANameItCannotRepresentWritesNothing (0.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.246s
  FAIL
  ```
- 2026-09-27 · 318a8c4* · exit 0 · `set -o pipefail …` · acceptance-sha256:3d0dfbeed2a71c1ba54318271f9ef607d92f10ef6a6fe51d23e2a783611c3cb7 · ms:526
- 2026-09-27 · 318a8c4* · exit 0 · `set -o pipefail …` · acceptance-sha256:3d0dfbeed2a71c1ba54318271f9ef607d92f10ef6a6fe51d23e2a783611c3cb7 · ms:1429

## Mutation Log
(empty until execute)
- 2026-09-27 · 318a8c4* · mutant killed · exit 1 · `cmd/mrw/main.go` · a --json plan with an invalid name is not refused · acceptance-sha256:3d0dfbeed2a71c1ba54318271f9ef607d92f10ef6a6fe51d23e2a783611c3cb7 · covers:a --json create with an invalid name writes nothing

## Invariants

- A plan without `--json` is unaffected.

## Risks

- None beyond the record's.

## Out of Scope

- `mrw_write` (permanent: boundary: the MCP surface already refuses an argument that is not valid UTF-8, ADR-078)

## Stop Condition

The fence exits 0 and contract §169 passes.
