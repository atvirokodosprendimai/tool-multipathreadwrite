# Task ADR-215-T2: the words and the contract row

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** one sentence in `mrw instructions` and the README; contract §381
**Consumes:** T1's behaviour; `internal/guide/guide.go`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the instructions say a plan path is relative to the root`, `contract 381 drives the built binary for the refusal and its pairs`, `every named test passed and none skipped`

## Goal

ADR-215's words, and the contract row `lifecycle.md` §5 asks for.

- **The guide**: the full instructions say that every path in a plan is relative to the root, that one beginning with a separator or naming a drive is refused naming the spelling to write, and that `mrw read` takes a real path; `mrw instructions --core` stays under 300 words (ADR-141).
- **Contract §381**: a plan `@@ /dd/zz.txt 1 replace` on an existing `dd/zz.txt`, and a `create` of `/sub/y.txt`, each exit 1 with the sentence and the spelling (`dd/zz.txt`, `sub/y.txt`) and the tree unchanged, with no `sub/` made; the absolute path of a file inside the root the same; a one-component header `@@ /zz.txt 1 replace` exit 1 with the same sentence (not exit 2). The pairs that must still pass: the same plans with the relative spelling apply; `mrw read` of the absolute path inside the root is served.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go`, `README.md` | edit | the sentence |
| `scripts/contract.sh` | edit | §381 |

## Ordered Steps

1. [S1] §381 first, and the guide test that holds the sentence. RED: the row fails on `main`. [proof: acceptance]
2. [S2] The words. Mutants, each killed: the guide sentence dropped; §381's pair drops the relative-spelling case. `bash -n scripts/contract.sh`. Then `adr-lint` on every record whose test lock names a file this task edits. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env go test -count=1 -v ./internal/guide -run '^TestTheInstructionsSayAPlanPathIsRelativeToTheRoot$' > "$d/t" \
  && grep -q '^--- PASS: TestTheInstructionsSayAPlanPathIsRelativeToTheRoot (' "$d/t" \
  && ! grep -q -- '--- SKIP' "$d/t" \
  && s381="$(awk '/^# [0-9]+\. /{on = /^# 381\. /} on' scripts/contract.sh)" \
  && grep -qF 'is absolute, and every path in a plan is relative to the root' <<<"$s381" \
  && grep -qF '@@ /zz.txt' <<<"$s381" \
  && bash -n scripts/contract.sh
```

What each clause holds. The guide test exists, passes and is not skipped. §381's text, read up to its next heading, names the refusal and the one-component header it asserts; that it runs is the contract gate's. The number is read from `scripts/contract.sh`, not from this file.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsSayAPlanPathIsRelativeToTheRoot` | `internal/guide/rooted215_test.go` | the sentence in the full text, the core under 300 words | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the guide test and §381 |
| 2 — something selects it | every `mrw instructions` |
| 3 — the caller can discover it | the instructions, the README, the refusal |
| 4 — it is used | telemetry is refused by ADR-009; the evidence is #648 |

## Mutation Log

## Verification Log

## Invariants

- `mrw instructions --core` stays under 300 words.

## Risks

- Editing `guide.go` can move other records' test locks; S2 lints and relocks them.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI with §381 passing; every gate in `AGENTS.md` passes; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- A release note (external: the release owner writes it from the record's Served-path change)
