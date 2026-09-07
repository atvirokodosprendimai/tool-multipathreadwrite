# Task ADR-034-T4: The gate stops producing what the prune removes

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** S (one export and one comment in `scripts/contract.sh`)
**Owner:** Zy
**Produces:** a `contract.sh` run that leaves the state base as it found it
**Consumes:** contract §71 (T3) — the row that must keep passing under the pin
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a contract run adds no entry to the real state base`, `every row still passes under the pin`

## Goal

Stop this repository's own gate from converting one temporary directory into one permanent one on
every fixture, so a prune is a one-off cleanup rather than a chore.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | `export XDG_STATE_HOME="$WORK/state"` beside the existing `WORK=$(mktemp -d)` at `:67`, so per-fixture state lands under the directory the `EXIT` trap at `:73` already removes. The comment at `:1588` is updated, because its clause *"contract.sh does not pin `XDG_STATE_HOME`"* becomes false. |

## Ordered Steps

1. [S1] Write the RED measurement first: count the entries in the real base, run
   `./scripts/contract.sh`, count again, and confirm the count GREW. Record the delta. Without this
   number the fix is unfalsifiable — a run that adds nothing for an unrelated reason would look
   identical to a fix that works. [proof: acceptance]
2. [S2] Export `XDG_STATE_HOME` into `$WORK`. One line, beside `WORK`, before `MRW` is built and
   before the first `fixture`. ⚠ It must be `export`ed, not assigned: `$MRW` is a child process and
   an unexported variable never reaches it. [proof: mutation]
3. [S3] Update the comment at `:1588`. Isolation by fresh root is still the mechanism and still
   correct; the pin is a second layer. A comment that describes the opposite of what the line above
   it does is worse than none. [proof: acceptance]
4. [S4] Re-run the measurement from S1: the real base's count must be UNCHANGED across a full
   contract run, and the run must still print `contract holds`. [proof: acceptance]
5. [S5] Confirm no row depended on the unpinned base: §~54 reads the directory from
   `m seen | head -1` rather than constructing it, so it follows the pin. Verify by reading the row
   and by the whole-script pass. [proof: acceptance]
6. [S6] Run the full gate list stand-alone. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
B="${XDG_STATE_HOME:-$HOME/.local/state}/mrw"
before=$(ls "$B" 2>/dev/null | wc -l | tr -d ' ')
./scripts/contract.sh 2>&1 | tee /tmp/adr034-t4c.out \
  && grep -q '^contract holds$' /tmp/adr034-t4c.out \
  && after=$(ls "$B" 2>/dev/null | wc -l | tr -d ' ') \
  && [ "$before" = "$after" ] \
  && ! grep -q 'contract.sh does not pin' scripts/contract.sh \
  && grep -q 'export XDG_STATE_HOME=' scripts/contract.sh \
  && go test ./... -count=1 \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./... \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check \
  && [ "$(grep -c '^require ' go.mod)" = 1 ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `none — the fence IS the test` | `scripts/contract.sh` | The fence counts the real state base before and after a full contract run and requires the two to be equal, which no unit test can assert because the leak is a property of the script's own process tree | — | S1, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | The `export` at the top of `scripts/contract.sh` |
| 2 — something selects it | Every child process the script starts inherits it, which is why it is `export` and not an assignment |
| 3 — the caller can discover it | The updated comment at `:1588` says where a run's state goes and why |
| 4 — it is used | The Acceptance fence measures the real base across a real run; the number, not the code, is the evidence |

## Mutation Log

- 2026-09-07 · 882fdea* · mutant killed · exit 1 · `scripts/contract.sh` · the pin is assigned but not EXPORTED, so no child process sees it and every fixture writes to the real state base again — the whole defect, and the one-word spelling of it that looks correct · acceptance-sha256:2f97e180302c678f203201b70af2cbbf047b70abcbe863f7b567e7c115038097 · covers:a contract run adds no entry to the real state base

## Verification Log

- 2026-09-07 · 882fdea* · exit 0 · `set -o pipefail …` · acceptance-sha256:2f97e180302c678f203201b70af2cbbf047b70abcbe863f7b567e7c115038097 · ms:24248

## Invariants

- Fresh-root isolation is unchanged: every `fixture` still gets its own `mktemp -d` under `$WORK`,
  and no row shares a root with another. The pin adds a layer, it does not replace one.
- The `EXIT` trap at `:73` is the only cleanup. This task adds no second trap and no `rm -rf` of its
  own — one owner for one directory.
- §71 keeps passing. It pins `XDG_STATE_HOME` to a base of its own under `$WORK`, so it is
  unaffected by an outer pin to a sibling directory.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/check` stay
  byte-identical against the merge base, and `go.mod` declares exactly one requirement. This task
  touches no Go file at all.

## Risks

- A row could assert the state base's location and break under the pin. Grepped: `XDG_STATE_HOME`
  appears in `scripts/contract.sh` exactly once, in the comment this task rewrites. The whole-script
  pass is the check.
- The before/after count is taken on the developer's real base, so a concurrent mrw run in another
  terminal would make it disagree. Judged acceptable and named here: the failure is a false RED, not
  a false green, and re-running settles it.
- One Go test still leaks its state. Out of scope and filed; it is 1 entry against 22,590, and
  bundling 21 test files into this record would bury the one line that matters.

## Stop Condition

Stop and ask if pinning changes any row's verdict — that would mean a row depends on the real base,
which is a coupling neither this task nor `contract.sh`'s own comment knows about, and it needs
reading before either is changed.

## Out of Scope

- Pinning `XDG_STATE_HOME` in the 21 test files that do not (deferred: `docs/adr/BACKLOG.md`)
- `scripts/measure.sh`, which already pins it into its own `$SCRATCH` (permanent: fact: done on the `measure-ratio-rounds` branch; citation: file `scripts/measure.sh:1`)
- Making the pin a repository-wide default via `go.mod` or a `TestMain` (permanent: boundary: a Go test that wants isolation says so with `t.Setenv`; a hidden global would make an unpinned test look pinned)
