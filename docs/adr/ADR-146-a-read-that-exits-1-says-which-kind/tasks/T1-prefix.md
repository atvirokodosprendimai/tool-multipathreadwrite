# Task ADR-146-T1: The exit-1 line of a read starts with its kind

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the `no match:` and `incomplete:` prefixes
**Consumes:** `refusals`, `problems` in `readCmd` (existing)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a read that exits 1 begins its last line with no match when it found nothing and was refused nothing, and with incomplete otherwise`

## Goal

Decisions 1 to 4 of the record, with a test that fails before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the three exit-1 returns of `readCmd` |
| `cmd/mrw/readexit146_test.go` | add | the four cases |
| `README.md`, `AGENTS.md` | edit | the prefixes stated |
| `scripts/contract.sh` | edit | §250 |

## Ordered Steps

1. [S1] Write `TestAReadThatExitsOneSaysWhichKind`. Confirm RED.
2. [S2] The prefixes, and the refused-while-no-match line. Mutant: the no-match return never says `incomplete` when paths were refused. [proof: mutation]
3. [S3] README, AGENTS.md and contract §250. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 600s -run 'TestAReadThatExitsOneSaysWhichKind' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAReadThatExitsOneSaysWhichKind \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && grep -q '^# 250\. ' scripts/contract.sh \
  && grep -q 'mrw: incomplete: ' README.md \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadThatExitsOneSaysWhichKind` | `cmd/mrw/readexit146_test.go` | a grep that matches nothing says `no match:`; a missing path says `incomplete:`; a grep that matches nothing while another path is refused says `incomplete:` and names the refusal; a read that serves everything exits 0 | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the prefixes in `readCmd` |
| 2 — something selects it | `mrw read`, every call that exits 1 |
| 3 — the caller can discover it | README and AGENTS.md state them; the line itself carries them |
| 4 — it is used | contract §250 drives the built binary; no telemetry (ADR-009) |

## Invariants

- Exit codes and stdout are unchanged.
- Every phrase printed before survives in the line.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a read exits 1 on a path with none of the three returns.

## Out of Scope

- A distinct exit code (permanent: boundary: Zy chose the prefix)

## Mutation Log
- 2026-10-11 · 57da6cd · mutant killed · exit 1 · `cmd/mrw/readline.go` · S2: a no-match beside a refused path never says incomplete · acceptance-sha256:b681eca6998ba2de53d4e1697dd6a97acac1eca0ef914ab4f1c9b8fd392046e9 · covers:a read that exits 1 begins its last line with no match when it found nothing and was refused nothing, and with incomplete otherwise
- 2026-10-11 · 57da6cd* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: a read with unserved ranges has no kind · acceptance-sha256:b681eca6998ba2de53d4e1697dd6a97acac1eca0ef914ab4f1c9b8fd392046e9 · covers:a read that exits 1 begins its last line with no match when it found nothing and was refused nothing, and with incomplete otherwise

## Verification Log
- 2026-10-11 · eaa26e2* · exit 1 · `set -o pipefail …` · acceptance-sha256:b681eca6998ba2de53d4e1697dd6a97acac1eca0ef914ab4f1c9b8fd392046e9 · ms:1608 · test-lock-sha256:40b8d9ca5e1f09167a185c8a7f5d6955e835bd98e52e38ea9a97ca169fa72a6a · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9yZWFkZXhpdDE0Nl90ZXN0LmdvCVRlc3RBUmVhZFRoYXRFeGl0c09uZVNheXNXaGljaEtpbmQJZDE5ZmJhOGQyNzJmNmI0MmY2ODhjOThhZDExNTlhOWY5OTc3OWFlOTA4NjljZWY1ODViMzc5MjQwOTY5MmUwNg
  ```
  --- last 9 line(s) of stdout
  === RUN   TestAReadThatExitsOneSaysWhichKind
      readexit146_test.go:21: a grep that matched nothing: "no file matched /ABSENT/", want it to begin `no match: no file matched /ABSENT/`
      readexit146_test.go:26: a path that does not exist: "1 range(s) could not be served", want it to begin `incomplete: 1 range(s) could not be served`
      readexit146_test.go:31: one file served, one missing: "1 range(s) could not be served", want it to begin `incomplete: `
      readexit146_test.go:38: a grep that matched nothing beside a refused link: "no file matched /ABSENT/", want `incomplete: no file matched /ABSENT/, and 1 path(s) were refused`
  --- FAIL: TestAReadThatExitsOneSaysWhichKind (0.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.561s
  FAIL
  ```
- 2026-10-11 · 57da6cd · exit 0 · `set -o pipefail …` · acceptance-sha256:b681eca6998ba2de53d4e1697dd6a97acac1eca0ef914ab4f1c9b8fd392046e9 · ms:41529
- 2026-10-11 · 57da6cd* · exit 0 · `set -o pipefail …` · acceptance-sha256:b681eca6998ba2de53d4e1697dd6a97acac1eca0ef914ab4f1c9b8fd392046e9 · ms:39680
- 2026-10-11 · 57da6cd* · exit 0 · `set -o pipefail …` · acceptance-sha256:b681eca6998ba2de53d4e1697dd6a97acac1eca0ef914ab4f1c9b8fd392046e9 · ms:39020
