# Task ADR-105-T4: the failure matrix; the BACKLOG entries closed

**Depends-on:** T1, T3
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** README "What a failure leaves on disk"; AGENTS `left_behind`; three BACKLOG entries closed
**Consumes:** `Result.LeftBehind` (T1), the fsync outcome (T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every stage has a row`, `the BACKLOG entries are closed`

## Goal

The README says, for each stage a write can fail at, what is on disk, what the receipt says, the exit code and
how to recover; AGENTS names `left_behind` beside `dirs_created`; the three BACKLOG entries this record answers
are closed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the matrix; `left_behind` beside `dirs_created` (`:199`) |
| `AGENTS.md` | edit | `left_behind` beside `dirs_created` (`:438`) |
| `docs/adr/BACKLOG.md` | edit | "A `.mrw-aside-*` left behind", "A probe left behind", "The cleanup errors mrw ignores" closed |

## Ordered Steps

1. [S1] Confirm the fence RED: none of the clauses holds before the edit. [proof: mutation]
2. [S2] The matrix, with a row each for validation, staging, commit, undo, ledger, check, process death and power loss, and every one of its five columns filled; the power-loss row says staged tree files are not synced and a synced state file's name can still be lost until its directory is synced; the process-death row says a killed run may leave neither its undo nor a receipt; `left_behind` in README and AGENTS; the three named entries closed on their own first lines. [proof: mutation] Mutants: the power-loss row's recovery cell emptied; one named entry left open.

## Acceptance

```bash
set -o pipefail
m=$(sed -n '/^## What a failure leaves on disk/,/^## [^W]/p' README.md) \
  && [ -n "$m" ] \
  && for s in validation staging commit undo ledger check 'process death' 'power loss'; do r=$(grep -i "^| $s |" <<<"$m") && [ "$(awk -F'|' '{n=0; for (i=2;i<=6;i++) if ($i ~ /[^ ]/) n++; print n}' <<<"$r")" = 5 ] || exit 1; done \
  && grep -qi 'directory' <<<"$(grep -i '^| power loss |' <<<"$m")" \
  && grep -q 'left_behind' <<<"$m" \
  && grep -q 'left_behind' AGENTS.md \
  && grep -q 'A `.mrw-aside-\*` left behind.*Closed\*\* by ADR-105' docs/adr/BACKLOG.md \
  && grep -q 'A probe left behind.*Closed\*\* by ADR-105' docs/adr/BACKLOG.md \
  && grep -q 'The cleanup errors mrw ignores.*Closed\*\* by ADR-105' docs/adr/BACKLOG.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | no unit test: the deliverable is documentation, proved by the fence's greps and its two killed mutants | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the README section |
| 2 — something selects it | a `## ` section of the README, beside Safety, and pointed at from AGENTS |
| 3 — the caller can discover it | README and AGENTS |
| 4 — it is used | telemetry is refused (ADR-009) |

## Mutation Log
- 2026-09-30 · 1e77123* · mutant killed · exit 1 · `README.md` · the power-loss row's recovery cell emptied · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · covers:every stage has a row
- 2026-09-30 · 1e77123* · mutant killed · exit 1 · `docs/adr/BACKLOG.md` · one named entry left open · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · covers:the BACKLOG entries are closed

## Invariants

- No code changes.

## Risks

- A row drifts from the code: each names the receipt field or message it describes, so a reader can check it.

## Stop Condition

Stop and ask if a stage's behaviour is not what the code does.

## Out of Scope

- The other ADR-105 tasks, each in its own file.

## Verification Log
- 2026-09-30 · 1e77123* · exit 1 · `set -o pipefail …` · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · ms:39 · test-lock-sha256:23b064b03ff72496693b9bc81b662c58ebe66d024bb257334b6dfc8390fbb29d · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwp1bnByb3ZlbgnigJQJbm9uZQ
  ```
  ```
- 2026-09-30 · 1e77123* · exit 0 · `set -o pipefail …` · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · ms:122
- 2026-09-30 · 1e77123* · exit 0 · `set -o pipefail …` · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · ms:61
- 2026-09-30 · 1e77123* · exit 0 · `set -o pipefail …` · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · ms:98
- 2026-09-30 · 1e77123* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · ms:0 · test-lock-sha256:d0a88974807379b1a850b3ee812a98f67f902bffe32b9e36b9a17cac84f8be11 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMw · test-lock-kind:replace
- 2026-09-30 · human-observed · relock 2026-09-30 reviewed: the Tests table's placeholder row was respelled in the corpus's form for a task with no unit test (a dash in both name columns); T4 has no test to lock, and its fence reads the documents; approved
- 2026-09-30 · 657c86b* · exit 0 · `set -o pipefail …` · acceptance-sha256:47fdee897ab8ecd401abd4668f16875c6c8ed947e498bd01b3cb77889d5ede3f · ms:80
