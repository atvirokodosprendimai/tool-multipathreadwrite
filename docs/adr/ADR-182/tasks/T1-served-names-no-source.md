# Task ADR-182-T1: `Observation.Served` names what was shown and what is licensed, and no source

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the wording of ADR-182 Decision 1; contract §316; `TestServedSaysWhatWasShownAndWhatIsLicensedForEveryShape`, `TestSeenNamesNoProvenanceTheLedgerDoesNotHold`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1

## Goal

Decision 1: for a `Written` observation `Served` returns `<shown>, and licensed <licence>` — `no lines` or the shown spans; `the whole file`, `none` or the licensed spans — so `mrw seen` and the three `mrw served …` refusals stop crediting a write or a `--sub` plan for a licence the ledger cannot attribute. The non-`Written` text, the ledger and every licence stay as they are.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/served182_test.go` | new | the five written shapes and the three read shapes, by `Served` |
| `cmd/mrw/seen182_test.go` | new | the built binary: `--quiet` ranged, `--quiet` whole, a `--sub` plan, a write, then `mrw seen` and a refused write |
| `internal/seen/seen.go` | edit | `Served` (`:115-136`): the `Written` branch; `quoteSpans` (`:139`) unchanged |
| `internal/seen/seen_test.go` | edit | `:308,311` (`TestAWrittenFileIsLicensedButNotShown`, ADR-118 T1) take `lines 2-3, and licensed the whole file` and `no lines, and licensed the whole file` |
| `cmd/mrw/sub161_test.go` | edit | `:212` (`TestASubReadRefusalsAndExits`, ADR-161 T1) takes `no lines, and licensed lines` |
| `docs/adr/ADR-075-one-writer-per-checkout/tasks/T1-one-writer-per-checkout.md`, `docs/adr/ADR-079-bookkeeping-survives-racing-processes/tasks/T1-one-lock-per-state-file.md`, `docs/adr/ADR-118-a-pattern-address-picks-the-nth-match/tasks/T1-occurrence.md`, `docs/adr/ADR-150-the-ledger-names-its-writer/tasks/T1-name-the-writer.md` | edit | each one's inline test lock (`test-lock-b64`) holds `TestAWrittenFileIsLicensedButNotShown`'s body, which S5 changes. Only the receipt is refreshed; the decision does not change |
| `docs/adr/ADR-161-instructions-lead-with-recipes/tasks/T1-sub.md`, `docs/adr/ADR-164-mrw-says-what-it-will-change-and-what-it-did-not-check/tasks/T2-dry-run-shows-each-change.md` | edit | each one's inline test lock holds `TestASubReadRefusalsAndExits`'s body. Only the receipt is refreshed |
| `scripts/contract.sh` | edit | §316, after §315 (`:10751`) and before the survivor check |

No other contract row changes: `scripts/contract.sh:4148` greps only `has not been read: mrw served`. No doc names the old text (`grep -rn 'since mrw wrote it\|a --sub plan licensed' --include='*.md' .` matches nothing outside this record).

## Ordered Steps

1. [S1] Write `TestServedSaysWhatWasShownAndWhatIsLicensedForEveryShape` (a table: `{Written, Shown: []}` → `no lines, and licensed the whole file`; `{Written, Shown: 2-3}` → `lines 2-3, and licensed the whole file`; `{Written, Shown: [], Spans: 1-3}` → `no lines, and licensed lines 1-3`; `{Written, Shown: 2-3, Spans: 1-3}` → `lines 2-3, and licensed lines 1-3`; `{Written, Shown: [], Spans: []}` → `no lines, and licensed none`; `{Written, Shown: 2-3, Spans: []}` → `lines 2-3, and licensed none`; the ledger line `w2-3|7-8` loaded through `Load` → `lines 2-3, and licensed lines 7-8`; and, unchanged, `{Spans: nil}` → `the whole file`, `{Spans: []}` → `no lines`, `{Spans: 1-3,7}` → `lines 1-3,7`; and no result containing `mrw wrote` or `plan`) and `TestSeenNamesNoProvenanceTheLedgerDoesNotHold` (a git checkout; `mrw read --quiet h.txt:1-2`, `mrw read --quiet g.txt`, `mrw read --quiet 'e.txt:/nomatch/'`, and `mrw read --grep Old --sub New s.txt` with stdout to a file; `mrw seen` prints `no lines, and licensed lines 1-3  h.txt` and `no lines, and licensed lines 1  s.txt` (the same form for a quiet read and a plan), `no lines, and licensed the whole file  g.txt` and `no lines, and licensed none  e.txt`, and prints neither `since mrw wrote it` nor `a --sub plan licensed` anywhere; a write to `h.txt` line 4 is refused exit 1 naming `mrw served no lines, and licensed lines 1-3` and writes nothing; a write to line 2 applies; after it `mrw seen` prints `no lines, and licensed the whole file  h.txt`). See both RED on the current `Served`. [proof: acceptance]
2. [S2] `Served`'s `Written` branch renders `<shown>, and licensed <licence>`. Mutant: restore `" since mrw wrote it"` for nil `Spans` — the whole-file quiet case must go red. [proof: mutation]
3. [S3] Mutant: restore `", and a --sub plan licensed "` for non-nil `Spans` — the ranged quiet case must go red. [proof: mutation]
4. [S4] The empty licence renders `none`. Mutant: drop that branch, so `quoteSpans` of no spans prints `lines ` — the `e.txt` case must go red. [proof: mutation]
5. [S5] `seen_test.go:308,311` and `sub161_test.go:212` take the new text. Both bodies are locked inline in six receipts. Refresh each with `adr-verify <task> --relock --replace-hashes` (ADR-075 T1, ADR-079 T1, ADR-118 T1 and ADR-150 T1 for `TestAWrittenFileIsLicensedButNotShown`; ADR-161 T1 and ADR-164 T2 for `TestASubReadRefusalsAndExits`; paths in Affected Files). Then run `adr-lint` over all six records, and over every record whose lock names another test in `seen_test.go` or `sub161_test.go`, since a `_test.go` edit can move it. [proof: acceptance]
6. [S6] Contract §316: in a git checkout, `--quiet h.txt:1-2` then `mrw seen` names `no lines, and licensed lines 1-3`; `--quiet g.txt` then `mrw seen` names `no lines, and licensed the whole file`; neither old phrase in `mrw seen`; the pair: a write to line 4 is refused exit 1 with the same licence named after `mrw served`, and a write to line 2 applies exit 0. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/seen/ ./cmd/mrw/ -count=1 -v -run '^(TestServedSaysWhatWasShownAndWhatIsLicensedForEveryShape|TestSeenNamesNoProvenanceTheLedgerDoesNotHold|TestAWrittenFileIsLicensedButNotShown|TestASubReadRefusalsAndExits|TestAFromCopyRefusesLinesLicensedButNeverShown)$' 2>&1) \
  && grep -qE '^--- PASS: TestServedSaysWhatWasShownAndWhatIsLicensedForEveryShape \(' <<<"$out" \
  && grep -qE '^--- PASS: TestSeenNamesNoProvenanceTheLedgerDoesNotHold \(' <<<"$out" \
  && ! grep -rqF -e 'since mrw wrote it' -e 'a --sub plan licensed' --include='*.go' --exclude='*_test.go' internal cmd \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 316\. ' scripts/contract.sh \
  && grep -qF 'PASS  mrw seen names a ranged --quiet licence by what the ledger holds' <<<"$c" \
  && grep -qF 'PASS  mrw seen names a whole-file --quiet licence without claiming a write' <<<"$c" \
  && grep -qF 'PASS  the pair: the refusal of an unlicensed line names the same licence' <<<"$c" \
  && grep -qF 'PASS  the pair: a licensed line is still writable' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/state \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/check internal/state)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestServedSaysWhatWasShownAndWhatIsLicensedForEveryShape` | `internal/seen/served182_test.go` | every shape `parseLine` can produce renders as Decision 1's table, a disjoint `w2-3\|7-8` included; the read shapes unchanged | none | S1, S2, S3, S4 |
| `TestSeenNamesNoProvenanceTheLedgerDoesNotHold` | `cmd/mrw/seen182_test.go` | the binary: `mrw seen` and a refusal name the licence, not a source, for a quiet read, a plan and a write | none | S1, S2, S3, S4 |
| `TestAWrittenFileIsLicensedButNotShown` | `internal/seen/seen_test.go` | ADR-118's written record round-trips; its `Served` text is the new one | none | S5 |
| `TestASubReadRefusalsAndExits` | `cmd/mrw/sub161_test.go` | ADR-161's `occurrence=2` refusal names the licence in the new text | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | every `mrw seen`, and every refusal at `internal/apply/apply.go:1357,1362,1573` over a written-shaped entry |
| 3 — the caller can discover it | `mrw seen`, which AGENTS.md §5 names as the file that settles a refusal |
| 4 — it is used | issue #555, met while reproducing robustness-plan item 1(c) on 2026-10-28, is the evidence for building it |

## Mutation Log
- 2026-10-29 · 41a53637* · mutant killed · exit 1 · `internal/seen/seen.go` · S2: restore the since-mrw-wrote-it text for a nil licence · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59
- 2026-10-29 · 41a53637* · mutant killed · exit 1 · `internal/seen/seen.go` · S3: restore the --sub plan text for a non-nil licence · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59
- 2026-10-29 · 41a53637* · mutant killed · exit 1 · `internal/seen/seen.go` · S4: drop the empty-licence branch so no spans print a dangling lines · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59

## Invariants

- The ledger format, every licence, `ServedLine`, `Covers` and every exit code are unchanged.
- The non-`Written` text (`the whole file`, `no lines`, `lines …`) is unchanged.
- `internal/apply` and `cmd/mrw` production code do not change; only `internal/seen` does.

## Risks

- None beyond ADR-182's.

## Stop Condition

The fence exits 0; or a consumer that parses the old text is found, or a ledger shape outside Decision 1's table turns up: stop and bring it to Zy.

## Out of Scope

- Any change to what is licensed or recorded (permanent: boundary: ADR-118, ADR-161 and ADR-167 own it)

## Verification Log
- 2026-10-29 · 41a53637* · exit 0 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:143044
- 2026-10-29 · 41a53637* · exit 0 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:129056
- 2026-10-29 · 41a53637* · exit 0 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:95941
- 2026-10-29 · 41a53637* · exit 0 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:87818
- 2026-10-29 · 41a53637* · exit 1 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:3129 · test-lock-sha256:b3c26cbf77afdf5da8c3bd435ee6e4b95616f2e690d8579c6d4a261df9f9633d · test-lock-file:docs/adr/locks/b3c26cbf77afdf5da8c3bd435ee6e4b95616f2e690d8579c6d4a261df9f9633d.lock
  ```
  ```
- 2026-10-29 · 41a53637* · exit 0 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:95846
- 2026-10-29 · 7d04e687* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:0 · test-lock-sha256:8617fe8bdd1dbc268b516230fd2b986cb1cc0b83a843bd77b39b9a6b21ca768d · test-lock-file:docs/adr/locks/8617fe8bdd1dbc268b516230fd2b986cb1cc0b83a843bd77b39b9a6b21ca768d.lock · test-lock-kind:replace
- 2026-10-29 · 7d04e687* · exit 0 · `set -o pipefail …` · acceptance-sha256:96f6c2cee9de7aba6c9e6441bd5180daa9fdaed04bfa394970fac9688e143f59 · ms:101306
