# Task ADR-165-T1: a global flag typed after the subcommand is pointed home

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** the pointer in `usageError` (`cmd/mrw/main.go`) for a refused flag that is one of `mrw`'s own
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the pointer names the flag's home`, `every global flag is covered`, `the exit code is unchanged`, `foreignFlags still answer after it`

## Goal

Decision 3: `mrw read --root DIR f` (and any flag the application defines on the root command, by its long name, typed after a subcommand that refuses it) is refused, exit 2, ending `; --root goes before the subcommand: mrw --root DIR read …`. Measured on the built binary, 2026-10-21: urfave makes a root flag persistent, so `write`, `check`, `stats`, `iter`, `seen`, `mcp`, `instructions` and `version` already take `--root` after the verb; only `read` refuses it, because its own `-C` (context) clashes with `--root`'s alias and urfave then skips the root flag whole (`flagKinds`, `cmd/mrw/main.go`). The test asks every subcommand, so a future clash is covered.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | `usageError` looks the refused flag up in the root command's flag set before `foreignFlagHint`; the sentence with the flag's own value placeholder and the subcommand used |
| `cmd/mrw/foreignflag.go` | edit | the comment that lists which refusals get a pointer names this one |
| `cmd/mrw/rootflag165_test.go` | add | the new test |
| `scripts/contract.sh` | edit | §281 |
| `README.md`, `AGENTS.md` | edit | one sentence each beside ADR-151's `-C` note |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestAGlobalFlagTypedAfterTheSubcommandIsPointedHome (each flag the application defines on the root command — enumerated at run time from `cmd.Root().Flags`, leaving out urfave's `help` and `version` — typed by its long name after `read` and after `write` is refused exit 2 with a message ending in its home spelling; `mrw --root DIR read f` is served as before; `mrw read -C 1 --grep x f` keeps `-C` as read's context; `mrw read --grep -v a f` and `mrw read -v f` keep ADR-164's `-v` pointer; `mrw read --zz f` gets neither pointer). [proof: acceptance]
2. [S2] The lookup in `usageError`, after `subcommandForFlag` and before `foreignFlagHint`, over `cmd.Root().Flags` by long name, skipping urfave's help and version flags; a root alias is matched only when the subcommand does not define it; the sentence is built from the flag's definition, never a table. [proof: mutation]
3. [S3] The sentences in README.md and AGENTS.md; the comment in `foreignflag.go`. [proof: acceptance]
4. [S4] Contract §281: `mrw read --root "$R" f.txt` exits 2 with the pointer; the pair: `mrw --root "$R" read f.txt` serves the line. [proof: human: the executor runs ./scripts/contract.sh once and reads the §281 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAGlobalFlagTypedAfterTheSubcommandIsPointedHome$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAGlobalFlagTypedAfterTheSubcommandIsPointedHome \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAForeignReadFlagIsAnsweredWithMrwsSpelling$|TestAFlagNamedForASubcommandNamesTheSubcommand$|TestAnInvertedGrepSaysThereIsNone$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAForeignReadFlagIsAnsweredWithMrwsSpelling \(' "$out" \
  && grep -qE '^--- PASS: TestAFlagNamedForASubcommandNamesTheSubcommand \(' "$out" \
  && grep -qE '^--- PASS: TestAnInvertedGrepSaysThereIsNone \(' "$out" \
  && grep -q '^# 281\. ' scripts/contract.sh \
  && grep -q 'goes before the subcommand' README.md \
  && grep -q 'goes before the subcommand' AGENTS.md
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAGlobalFlagTypedAfterTheSubcommandIsPointedHome` | `cmd/mrw/rootflag165_test.go` | every root flag after `read` and `write` gets the pointer, exit 2; the home spelling still works; an unknown flag gets no pointer | none | S1, S2 |
| `TestAForeignReadFlagIsAnsweredWithMrwsSpelling` | `cmd/mrw/foreignflag140_test.go` | existing: ADR-140's pointers unchanged after the new lookup | none | — |
| `TestAFlagNamedForASubcommandNamesTheSubcommand` | `cmd/mrw/usage097_test.go` | existing: ADR-097's pointer keeps precedence | none | — |
| `TestAnInvertedGrepSaysThereIsNone` | `cmd/mrw/foreignflag164_test.go` | existing: ADR-164's `-v` pointer unchanged | none | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the lookup in `usageError` |
| 2 — something selects it | every parser refusal passes through `usageError` (ADR-097); contract §281 drives the binary |
| 3 — the caller can discover it | the message itself |
| 4 — it is used | the Codex report of 2026-10-21 lost a turn to it; nothing measures it after |

## Invariants

- Exit 2 and the `(see: mrw read --help)` tail are unchanged.
- `mrw --root DIR read …` is not touched.
- ADR-097's, ADR-140's and ADR-164's pointers still fire where they did: ADR-097's subcommand-named flag first, then this lookup, then `foreignFlags`. A root flag's long name is never a subcommand name and never in `foreignFlags`; a root alias a subcommand defines for itself (`-C` on `read`) never reaches `usageError`. The test asserts each.

## Risks

- urfave/cli's refusal text changes on upgrade: `refusedFlag` already handles that by falling through to the old wording, and this lookup uses the same handle.

## Out of Scope

- A pointer for a global flag typed with a wrong value (`--root` with no DIR): urfave's own message, unchanged (permanent: boundary: this task is about position, not value)

## Stop Condition

Stop if the root command's flag set cannot be read from `usageError` without a global: then the pointer needs a table after all, and the decision's "no table" is wrong — report, do not add the table.

## Mutation Log
(empty until execute)
- 2026-10-21 · 4756f972* · mutant killed · exit 1 · `cmd/mrw/main.go` · the root-flag pointer is never consulted · acceptance-sha256:10748d93cd06f00b74247c01887e68c087ee1923929893fd2e70b910cbdfa6d8

## Verification Log
(empty until execute)
- 2026-10-21 · 4756f972* · exit 0 · `set -o pipefail …` · acceptance-sha256:10748d93cd06f00b74247c01887e68c087ee1923929893fd2e70b910cbdfa6d8 · ms:3953
