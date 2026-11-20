# Task ADR-212-T1: the schema says a moved link has no content

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the two descriptions in `internal/mcp/schema.go`; the regenerated legacy golden and its line in `era_test.go`'s list of regenerations; the MCP receipt test; contract §364 extended to the wire shapes
**Consumes:** `mrw_write`'s output schema and receipt; `TestALegacyResultIsUnchangedByTheModernPath`; contract §364
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the schema states the rule for both keys`, `the legacy golden changed by those two strings only`, `the MCP receipt gives a moved link no content`, `the CLI receipt and a dry run have the shapes the record states`, `the behaviour is still #725's`, `every named test passed and none skipped`, `the engine and writer packages are untouched`

## Goal

ADR-212 Decisions 1 and 3.

- **`TestTheWriteSchemaSaysAMovedLinkCarriesNoContent`** (`internal/mcp`) reads `mrw_write`'s output schema from `tools/list` and asserts that `files.sha_after`'s description holds "Absent for a removed file and for a moved symbolic link, whose target mrw does not read." and `files.lines_after`'s holds "0 for a removed file and for a moved symbolic link, whose target mrw does not read: on a written row it is a count only when sha_after is present." Each assertion fails with a message naming its key (`files.sha_after's description lacks …`, `files.lines_after's description lacks …`).
- **`TestTheMCPReceiptOfAMovedLinkGivesNoContent`** (`internal/mcp`): a relative symlink `a/link.txt` read, then moved to `b/link.txt` by `mrw_write`; the structured receipt's row for `b/link.txt` has `written` true, `lines_after` 0 and no `sha_after`; the same plan with `dry_run` gives one row, `a/link.txt`, with `renamed_to` `b/link.txt` and `written` false, and no row for `b/link.txt`. Skipped only where the OS cannot make a symlink, saying so.
- **`TestALegacyResultIsUnchangedByTheModernPath`** passes against the regenerated golden; its comment gains "and once by ADR-212 … those two descriptions, nothing else".
- **Contract §364**, extended: the applied `--json` row for `b/link.txt` has `lines_after` 0 and `written` true besides no `sha_after`; the human receipt's line for it reads `sha -`; a `--dry-run --json` of the same plan gives the source row with `renamed_to` and `written` false and no destination row.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/schema.go` | edit | the two descriptions |
| `internal/mcp/schema212_test.go` | new | the schema test and the MCP receipt test |
| `internal/mcp/testdata/legacy_golden.jsonl` | regenerate (`MRW_UPDATE_LEGACY_GOLDEN=1`, once) | the two strings |
| `internal/mcp/era_test.go` | edit | its list of regenerations names ADR-212 |
| `scripts/contract.sh` | edit | §364's wire-shape rows |

## Ordered Steps

1. [S1] The two tests. RED: the schema test, on both keys. The MCP receipt test holds today, since its shapes are #725's, and stays. [proof: acceptance]
2. [S2] The descriptions; regenerate the golden; the comment; §364. Mutants, each killed, each by a named failure in the captured output: the `sha_after` sentence dropped (`files.sha_after's description lacks`); the `lines_after` sentence dropped (`files.lines_after's description lacks`); the golden left stale (`a legacy answer changed`); a third description changed in the golden (the fence's golden-delta clause). Then `adr-lint` on every record whose test lock names `era_test.go`, relocking what it reports. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env -u MRW_UPDATE_LEGACY_GOLDEN go test -count=1 -v ./internal/mcp -run '^(TestTheWriteSchemaSaysAMovedLinkCarriesNoContent|TestTheMCPReceiptOfAMovedLinkGivesNoContent|TestALegacyResultIsUnchangedByTheModernPath)$' > "$d/mcp" \
  && env -u MRW_UPDATE_LEGACY_GOLDEN go test -count=1 -v ./internal/writer -run '^TestAMovedSymlinkCarriesNoContentAndLicensesNothing$' > "$d/writer" \
  && grep -q '^--- PASS: TestTheWriteSchemaSaysAMovedLinkCarriesNoContent (' "$d/mcp" \
  && grep -q '^--- PASS: TestTheMCPReceiptOfAMovedLinkGivesNoContent (' "$d/mcp" \
  && grep -q '^--- PASS: TestALegacyResultIsUnchangedByTheModernPath (' "$d/mcp" \
  && grep -q '^--- PASS: TestAMovedSymlinkCarriesNoContentAndLicensesNothing (' "$d/writer" \
  && ! grep -q -- '--- SKIP' "$d/mcp" "$d/writer" \
  && base="$(git merge-base HEAD origin/main)" && [ -n "$base" ] \
  && g=internal/mcp/testdata/legacy_golden.jsonl \
  && git show "$base:$g" > "$d/golden.base" \
  && ! cmp -s "$d/golden.base" "$g" \
  && sed -e 's/ Absent for a removed file and for a moved symbolic link, whose target mrw does not read\.//' \
         -e 's/ 0 for a removed file and for a moved symbolic link, whose target mrw does not read: on a written row it is a count only when sha_after is present\.//' "$g" > "$d/golden.stripped" \
  && cmp -s "$d/golden.base" "$d/golden.stripped" \
  && grep -qF 'once by ADR-212' internal/mcp/era_test.go \
  && s364="$(awk '/^# [0-9]+\. /{on = /^# 364\. /} on' scripts/contract.sh)" \
  && grep -qF 'lines_after' <<<"$s364" \
  && grep -qF 'sha -' <<<"$s364" \
  && grep -qF -- '--dry-run' <<<"$s364" \
  && git diff --quiet "$base" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer ':(exclude)*_test.go' \
  && st="$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer)" \
  && [ -z "$st" ]
```

What each clause holds. The runs are made with `MRW_UPDATE_LEGACY_GOLDEN` unset, so the legacy test compares and never rewrites. The golden differs from the merge base's, and removing exactly the two new sentences gives the merge base's back, so nothing else in it changed. §364's text, read up to its next heading, names the shapes it now asserts; that it runs is the contract gate's. The engine packages and `internal/writer` have no committed non-test change since the merge base and no uncommitted or untracked file (ignored files excepted), and a failing `git status` fails the chain. `TestAMovedSymlinkCarriesNoContentAndLicensesNothing` and the MCP receipt test skip where symlinks cannot be made (Windows without the privilege); the fence runs on unix.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheWriteSchemaSaysAMovedLinkCarriesNoContent` | `internal/mcp/schema212_test.go` | the two descriptions, read from `tools/list`, each failing by its key's name | none | S1, S2 |
| `TestTheMCPReceiptOfAMovedLinkGivesNoContent` | `internal/mcp/schema212_test.go` | the applied and dry-run rows over MCP | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests and §364 |
| 2 — something selects it | every `tools/list` and every move of a symbolic link |
| 3 — the caller can discover it | the output schema; `docs/receipts.txt`'s header; README's rename paragraph |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #712 |

## Mutation Log

## Verification Log

## Invariants

- No receipt key is added, removed or renamed; the behaviour is #725's.

## Risks

- Editing `era_test.go` can move other records' test locks; S2 lints and relocks them.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI with §364's rows passing; every gate in `AGENTS.md` passes; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- The CLI's `--json` key text (permanent: boundary: the CLI has no per-key description; `docs/receipts.txt`'s header carries the rule)
