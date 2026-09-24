# Task ADR-064-T2: the taught pipeline runs under an agent-shaped stdin; contract §117

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** taught pipeline executed under open stdin; §117
**Consumes:** none
**Data dependency:** hermetic, needs rg on PATH (a visible SKIP where absent)
**Proof map:** v1
**Rests-on:** `the binary's taught pipeline completes under an open stdin`, `AGENTS.md's pipeline completes under an open stdin`, `a path-less pipeline is killed by the bound`, `absent rg is a visible skip`

## Goal

Contract §117 takes two copies of the `rg -l X . | sed … | mrw read --files-from -` line: the one the built binary prints, and the literal line from AGENTS.md.
- It runs each from a fixture with stdin an open pipe and a 5 s alarm; each must exit 0 and serve at least one file.
- The binary's line with its ` .` removed must be killed by the alarm.
- Where rg is absent, the row prints SKIP.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | §117, next free after §116. |

## Ordered Steps

1. [S1] Write §117 and confirm it is RED against a binary taught without the ` .`, then GREEN on the real build:
   - Extract the line with `"$MRW" instructions | grep -oE "rg -l X \. \| sed '[^']*' \| mrw read --files-from -"`, and `bad "the binary teaches no rg pipeline that names its path"` when it is empty.
   - Substitute `X` with a needle present in the fixture.
   - Run `cd "$R"` with `PATH="$(dirname "$MRW"):$PATH"` and `perl -e 'alarm shift; exec @ARGV' 5 bash -c "$cmd" < <(sleep 20) > "$WORK/117.out" 2>&1`, writing to a file and never through `$( )`. Want 0 and `^==> `.
   - Run the AGENTS.md literal (`grep -m1 -E "^rg -l .* \| mrw read .*--files-from -$" AGENTS.md`) the same way, after appending `func Handle() {}` to a fixture file.
   - Confirm RED in a mini-harness (the prologue helpers and §117 alone) against a binary built with `go build -overlay` over `internal/guide/guide.go` with the ` .` removed. The tree stays untouched, and the runner never rebuilds that binary. There the extraction is empty and the row reports `bad`.
   - Then confirm GREEN in the full `./scripts/contract.sh`.
   [proof: mutation]
2. [S2] Must-fail twin: the binary's line with ` . ` removed, run in the same harness. Want `rc -gt 128` and a duration ≥ 4 s (§111's `t0`/`t1`). [proof: acceptance]
3. [S3] `command -v rg || skip "rg absent — taught pipeline not executed"` wraps the row. Run `./scripts/contract.sh` unpiped before commit. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
grep -q '^# 117\. ' scripts/contract.sh \
  && command -v rg >/dev/null \
  && ./scripts/contract.sh > /tmp/adr064-t2.out 2>&1 \
  && grep -q '^  PASS  the taught rg pipeline completes under an open stdin' /tmp/adr064-t2.out \
  && grep -q "^  PASS  AGENTS.md's rg pipeline completes under an open stdin" /tmp/adr064-t2.out \
  && grep -q '^  PASS  without its path, rg reads the pipe and the bound kills it' /tmp/adr064-t2.out \
  && ! grep -q '^  SKIP  rg absent' /tmp/adr064-t2.out \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/apply internal/plan internal/seen internal/check internal/state internal/mcp internal/guide internal/read/walk.go
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | §117 |
| 2 — something selects it | CI runs `./scripts/contract.sh`; the row extracts from the built binary, so the `guide.go` line is what it runs |
| 3 — the caller can discover it | `mrw instructions` prints the pipeline; AGENTS.md carries it |
| 4 — it is used | ADR-009 refuses telemetry; the hang was found by the 2026-09-24 chaos pass |

## Mutation Log
(empty until execute)
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `internal/guide/guide.go` · the binary teaches the pipeline without its path: §117 must report it · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · covers:the binary's taught pipeline completes under an open stdin
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `scripts/contract.sh` · the must-fail twin keeps its path, so it completes: §117 must report that it cannot see the hang · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · covers:a path-less pipeline is killed by the bound
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md teaches the pipeline without its path: §117 must report the hang · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · covers:AGENTS.md's pipeline completes under an open stdin
- 2026-09-24 · 9874f44* · mutant killed · exit 1 · `scripts/contract.sh` · rg reads as absent: the row SKIPs and the fence must refuse a SKIP where rg exists · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · covers:absent rg is a visible skip

## Invariants

- No Go code changes.
- The row never uses `$( )` around a pipeline it may kill.
- A SKIP is printed, never silent.

## Risks

- The fence costs a full `contract.sh` run (about 25 s here) and needs rg, so it is a local fence; CI's run of the row may be a SKIP.

## Stop Condition

If the only way to go green is to run README.md's pipeline, or to widen §30's awk to ```` ```sh ````, stop.

## Out of Scope

- README.md's copy of the pipeline (permanent: boundary: the ADR's scope choice; agents are taught from the binary and AGENTS.md)

## Verification Log
(empty until execute)
- 2026-09-24 · 9874f44* · exit 1 · `set -o pipefail …` · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · ms:30910 · test-lock-sha256:d0a88974807379b1a850b3ee812a98f67f902bffe32b9e36b9a17cac84f8be11 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMw
  ```
  ```
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · ms:30618
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · ms:37561
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · ms:35857
- 2026-09-24 · 9874f44* · exit 0 · `set -o pipefail …` · acceptance-sha256:94ecfe57aed373ab3b5480f37dad9555bf271ef92c2992ea1d63198e9b369cf9 · ms:51285
