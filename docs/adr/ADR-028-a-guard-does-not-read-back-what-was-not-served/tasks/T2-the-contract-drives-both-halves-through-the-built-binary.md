# Task ADR-028-T2: The contract drives both halves through the built binary

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (one contract section)
**Owner:** Zy
**Produces:** contract §66
**Consumes:** the anchor check evaluated only for lines the ledger licensed (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the built binary printing no unserved line on a failed anchor`, `the built binary still quoting a served line`

## Goal

Prove in the BUILT binary that a failed anchor on an unserved line prints none of it, and that a
failed anchor on a served line still quotes it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `scripts/contract.sh` | edit | New `# 66.` section. A unit test proves the ordering inside the package; it cannot prove the shipped binary has it, which is why §53 exists. This is also the only place the two halves are asserted against one binary in one run. ⚠ It was authored as `# 65.`, which ADR-027 already owned; the number moved to 66 in review, and the two `bd73ee0` Mutation Log entries below still name §65 because that is what they were written against. |

## Ordered Steps

1. [S1] Confirm §66 does not exist and that the row is RED against a binary built before T1: the leaked line appears in the refusal on the pre-ADR-028 tree. A row that cannot fail asserts nothing. [proof: acceptance]
2. [S2] Write §66 with both halves and a distinctive sentinel: serve line 1, anchor line 2 with a wrong value, and assert the sentinel text of line 2 is ABSENT from the output while the ledger's own refusal is present. [proof: acceptance]
3. [S3] Assert the other half in the same section: with the line served, a failed anchor still quotes it. Without this the row would pass against a binary that had stopped checking anchors. [proof: acceptance]
4. [S4] Run `./scripts/contract.sh` whole, and every other gate. [proof: acceptance]
5. [S5] Drive ALL FOUR anchored ops, not the ones that came to mind. `replace` and `delete` share the moved inline check; the two insertions reach the anchor through a guard closure. The first version of this section carried three of the four while the record above it claimed all four — a false coverage claim, caught by the third Codex review of PR #128 and by no gate, because a section that runs three ops passes exactly like one that runs four. The added `delete` case was confirmed RED against the v1.4.0 binary built from `bd73ee0`, where it prints `UNSERVED-SENTINEL-42`. [proof: acceptance]
6. [S6] ⚠ **A served-line control must assert the REFUSAL, not only that the line appears.** The `delete` control added in S5 grepped the output for `public line` and checked no exit code. A `delete` that has stopped checking anchors SUCCEEDS, and ADR-008 made its receipt say what it removed — `ok s.txt 1 delete -1 +0 from "public line" to "public line"` — so the grep is satisfied by exactly the binary the control exists to catch. That receipt was confirmed by a direct run against an isolated fixture, NOT by the mutant: the mutant run of the day shared one fixture and reached `delete` after line 1 had been rewritten, so it failed for a different reason (see the note above the Mutation Log). The fourth Codex review of PR #128 found the shape, and `insert-after`'s control had it too — surviving only because its receipt prints no content, which is an accident of receipt shape and not a property of the check. Both now assert exit 1 and match the anchor message. [proof: mutation]
7. [S7] ⚠ **One fresh fixture per case, or the section tests whichever case ran first.** Every case shared one file and one ledger. A case that wrongly SUCCEEDS writes the file AND records the ledger WHOLE, so every later case becomes a different test than its name says — an unserved line that is no longer unserved, a sentinel that is no longer there. Not hypothetical: the S6 mutant run printed `from "X" to "X"` where `public line` was meant. Found by the fifth Codex review of PR #128. The section now calls `anchored_fixture` before each case and loops all four ops, taking it from seven plan cases carrying 17 checks to eight carrying 24 — including the ledger's own message asserted PRESENT beside each sentinel-ABSENT check, since an absence assertion alone is satisfied by empty output. The isolation is what makes the mutant discriminate: with the anchor disabled on the `replace`/`delete` site only, those two controls go red and both insertion controls stay green. [proof: mutation]

## Acceptance

```bash
set -o pipefail
grep -q '^# 66\. ADR-028: a guard does not read back what was not served\.$' scripts/contract.sh \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr028-t2.out \
  && grep -q '^contract holds$' /tmp/adr028-t2.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr028-t2.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `§66` | `scripts/contract.sh` | For all four anchored ops — `replace`, `delete`, `insert-after`, `insert-before` — the built binary refuses an anchored hunk on an unserved line without printing any of that line, and still quotes a served line whose anchor failed | — | S1, S2, S3, S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `§66` in `scripts/contract.sh` |
| 2 — something selects it | `./scripts/contract.sh` runs every section against `$MRW`, the binary built at the top of the script, so the row fails if the shipped ordering is wrong even when the unit test passes |
| 3 — the caller can discover it | The refusal itself; no documented surface changes |
| 4 — it is used | The contract runs in CI on every push; no telemetry, per ADR-009 |

## Mutation Log

⚠ **The `2639749` entry below states a mechanism its own run did not exercise, and is
superseded by the `7d75c3a` one.** Its reasoning — that a line-only control passes because
`delete`'s SUCCESS receipt prints the removed line — is true of the code and is why the control was
changed. But that run shared one fixture across the whole section, so by the time it reached
`delete` an earlier hunk had rewritten line 1 to `X`: the grep failed, for a different reason than
the entry gives. Kept rather than removed, because "the run that motivated the change did not
isolate the mechanism" is itself the finding, and it is what S7 exists for. The `7d75c3a` entry runs
against one fresh fixture per case and discriminates cleanly. Marked on the sixth Codex review of
PR #128.
⚠ **The two `bd73ee0` entries say §65, which is this section's authoring number.** It collided with
ADR-027's and became §66 in review; the entries are tool-written and are left as they ran. Read §65
in them as §66. Found by the seventh Codex review of PR #128, alongside the same stale number in the
Affected Files table above, which is prose and IS corrected.
- 2026-09-07 · bd73ee0* · mutant killed · exit 1 · `internal/apply/apply.go` · the built binary reads back the unserved line again, so §65 sees the sentinel in the refusal — the row drives the shipped binary, which the unit test cannot · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · covers:the built binary printing no unserved line on a failed anchor
- 2026-09-07 · bd73ee0* · mutant killed · exit 1 · `internal/apply/apply.go` · the built binary stops checking anchors, so the served-line half of §65 no longer sees the line quoted and the row goes red rather than crediting a removal as a fix · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · covers:the built binary still quoting a served line
- 2026-09-07 · eed0cd4* · mutant killed · exit 1 · `internal/apply/apply.go` · insert-before reverts to the old ordering in the built binary, so §66 sees the sentinel in the refusal for that op while replace and delete stay clean · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · covers:the built binary printing no unserved line on a failed anchor
- 2026-09-07 · eed0cd4* · mutant killed · exit 1 · `internal/apply/apply.go` · the built binary stops checking anchors on replace, so the served-line half of §66 no longer sees the line quoted and a removal cannot pass as a fix · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · covers:the built binary still quoting a served line
- 2026-09-07 · 2639749* · mutant killed · exit 1 · `internal/apply/apply.go` · the anchor stops being checked for replace and delete — the case a served-line control that only greps for the line text does NOT catch, because deletes print the removed line in their SUCCESS receipt (ADR-008), so the grep passes against a binary that has stopped checking · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · covers:the built binary still quoting a served line
- 2026-09-07 · 7d75c3a* · mutant killed · exit 1 · `internal/apply/apply.go` · the anchor stops being checked on the replace/delete site only. With one fresh fixture per case the section now DISCRIMINATES: the replace and delete served-line controls go red and the two insertion controls stay green, because they reach the anchor through the other site — evidence about each op rather than about whichever ran first · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · covers:the built binary still quoting a served line

## Invariants

- Every existing contract section still passes; §66 uses its own fixture.
- The row drives `$MRW`, never a Go test.
- Both halves live in one section, because a row asserting only the absence would pass against a binary that printed nothing at all.

## Risks

- A sentinel that appears elsewhere in the output would make the absence assertion vacuous. Mitigated by using a string that appears nowhere else in the fixture or in mrw's own vocabulary.

## Stop Condition

Stop and ask if §66 cannot be made red against the pre-ADR-028 tree: a row green before the fix
exists is asserting nothing, and finding that out here is the point of S1.

## Out of Scope

- The ordering itself — that is T1's job

## Verification Log
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · ms:32344
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · ms:29195
- 2026-09-07 · bd73ee0* · exit 0 · `set -o pipefail …` · acceptance-sha256:7d34cb49b2d0d244d0de5c079bb7bf76eec57211b01fd9580e8adfe9eeae1a71 · ms:31276
- 2026-09-07 · eed0cd4* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · ms:32433
- 2026-09-07 · eed0cd4* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · ms:29918
- 2026-09-07 · eed0cd4* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · ms:29943
- 2026-09-07 · 29efa5d* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · ms:34952
- 2026-09-07 · 2639749* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · ms:33624
- 2026-09-07 · 7d75c3a* · exit 0 · `set -o pipefail …` · acceptance-sha256:7cc5f992045a9f073f8bb609eca768382330e3717a607bbc77cf7f2a0c15e731 · ms:34380
