# Task ADR-196-T2: the text stops teaching the `--` escape for a padded path

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** contract §140 retargeted; `internal/guide` text
**Consumes:** the v3.14.0 behaviour (T1)
**Data dependency:** hermetic
**Proof map:** v1

## Goal

`mrw instructions`, `--core`, AGENTS.md and README say a path reaches mrw as typed, and none of them says a padded path must go after `--` or that an attached padded value is refused.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | lines 118, 135, 156: the `--` rule and the lone-`-` lift |
| `internal/guide/*_test.go` | edit | any test pinning those phrases |
| `AGENTS.md` | edit | the ⚠ padded-path paragraph (line 393) |
| `README.md` | edit | the same rule, where it is stated |
| `scripts/contract.sh` | edit | §140 retargeted: the text says a path reaches mrw as typed; the pair: it does not say a padded path must go after `--` |
| `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly.md`, `docs/adr/ADR-161-*.md` | edit | a `**Superseded-in-part:**` note naming ADR-196 and the items; the Decision text left standing |

## Ordered Steps

1. [S1] Write `TestTheGuideSaysAPathReachesMrwAsTyped` (`internal/guide/guide196_test.go`): the plain text and `--core` say a path reaches mrw as typed and say neither `goes after --` nor that a padded attached value is refused. Retarget contract §140 to the same through the built binary. See both RED against T1's tree. [proof: acceptance]
2. [S2] Rewrite the guide's three sites and AGENTS.md's and README's paragraphs; keep ADR-141's `--core` under 300 words and eight rules. [proof: acceptance]
3. [S3] Add the superseded-in-part notes to ADR-069 and ADR-161. [proof: acceptance]
4. [S4] Mutant: put the old `goes after --` sentence back into `guide.go`; §140 goes red. [proof: mutation]

## Acceptance

```bash
set -o pipefail
c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 140\. ' scripts/contract.sh \
  && grep -qF 'PASS  mrw instructions say a path reaches mrw as typed' <<<"$c" \
  && grep -qF 'PASS  and they do not teach the -- escape for a padded path' <<<"$c" \
  && ! grep -qF 'goes after `--`' AGENTS.md \
  && ! grep -qF 'goes after --' internal/guide/guide.go \
  && grep -qF 'ADR-196' docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly.md \
  && out=$(go test ./internal/guide/ -count=1 -v -run '^TestTheGuideSaysAPathReachesMrwAsTyped$' 2>&1) \
  && grep -qE '^--- PASS: TestTheGuideSaysAPathReachesMrwAsTyped \(' <<<"$out"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheGuideSaysAPathReachesMrwAsTyped` | `internal/guide/guide196_test.go` | the text says a path reaches mrw as typed, and not the escape | none | S1, S2 |


## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the guide text |
| 2 — something selects it | `mrw instructions`, `--core` and `mrw skill` print it |
| 3 — the caller can discover it | it is the binary's own guide |
| 4 — it is used | every agent reads `--core` at session start (AGENTS.md) |
## Mutation Log
- 2026-11-10 · f3aabb9d* · mutant killed · exit 1 · `internal/guide/guide.go` · the core rule teaches the -- escape again, so the guide test and contract 140 must go red · acceptance-sha256:3f4e25db85e70ce66d3c0621f03c938a92a85bb4457f657603bf2b5fbced5e95

## Verification Log
- 2026-11-10 · f3aabb9d* · exit 1 · `set -o pipefail …` · acceptance-sha256:3f4e25db85e70ce66d3c0621f03c938a92a85bb4457f657603bf2b5fbced5e95 · ms:102754 · test-lock-sha256:11c3bea71565242fb45c7fe8d434a8263df70d9a129185c9c705690dc2bbf641 · test-lock-file:docs/adr/locks/11c3bea71565242fb45c7fe8d434a8263df70d9a129185c9c705690dc2bbf641.lock
  ```
  ```
- 2026-11-10 · f3aabb9d* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f4e25db85e70ce66d3c0621f03c938a92a85bb4457f657603bf2b5fbced5e95 · ms:84160
- 2026-11-10 · f3aabb9d* · exit 0 · `set -o pipefail …` · acceptance-sha256:3f4e25db85e70ce66d3c0621f03c938a92a85bb4457f657603bf2b5fbced5e95 · ms:82899

## Invariants

- ADR-037's shared sentences stay verbatim; ADR-141's `--core` stays eight rules under 300 words.

## Risks

- None beyond the record's.

## Stop Condition

The fence exits 0.

## Out of Scope

- The memoryd `craft` copy of the `mrw` skill (permanent: boundary: regenerated from the tagged binary at release, never by hand)
