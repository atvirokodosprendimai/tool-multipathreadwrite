# Task ADR-215-T2: the words and the contract row

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** one sentence in `mrw instructions` and the README; contract §381; the fence's own probe
**Consumes:** T1's behaviour; `internal/guide/guide.go`
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `the instructions say a plan path is relative to the root`, `the built binary refuses a rooted plan path and applies the relative one`, `every named test passed and none skipped`

## Goal

ADR-215's words, and the row `lifecycle.md` §5 asks for, run and not only read.

- **The guide**: the full instructions say every path in a plan is relative to the root, that one beginning with a separator or naming a drive is refused (naming the spelling only when it is clear), and that `mrw read` takes a real path; `mrw instructions --core` stays under 300 words (ADR-141).
- **Contract §381**: a plan `@@ /dd/zz.txt 1 replace` on an existing `dd/zz.txt` and a `create` of `/sub/y.txt` each exit 1 with the sentence and the tree unchanged, with no `sub/` made; the absolute path of a file inside the root the same; a rename to a rooted destination of a licensed source the same. The pairs that must still pass: the same plans with the relative spelling apply (bytes compared); `mrw read` of the absolute path inside the root is served.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go`, `README.md` | edit | the sentence |
| `scripts/contract.sh` | edit | §381 |
| `internal/guide/rooted215_test.go` | new | the guide test |

## Ordered Steps

1. [S1] §381 and the guide test first. RED: the row fails on `main`. [proof: acceptance]
2. [S2] The words. Mutants, each killed by an EXECUTED result: the guide sentence dropped; the relative-success input of the fence's probe changed to a rooted one (its `cmp` fails); the expected exit of the negative case changed. `bash -n scripts/contract.sh`. Then `adr-lint` on every record whose test lock names a file this task edits. [proof: mutation]

## Acceptance

```bash
d="$(mktemp -d)" \
  && env go test -count=1 -v ./internal/guide -run '^TestTheInstructionsSayAPlanPathIsRelativeToTheRoot$' > "$d/t" \
  && grep -q '^--- PASS: TestTheInstructionsSayAPlanPathIsRelativeToTheRoot (' "$d/t" \
  && ! grep -q -- '--- SKIP' "$d/t" \
  && go build -o "$d/mrw" ./cmd/mrw \
  && mkdir -p "$d/r/dd" \
  && printf 'one\n' > "$d/r/dd/zz.txt" \
  && XDG_STATE_HOME="$d/st" "$d/mrw" --root "$d/r" read dd/zz.txt > "$d/read.out" 2>&1 \
  && { printf '@@ /dd/zz.txt 1 replace anchor="one"\nONE\n@@ /sub/y.txt 0 create\ny\n' | XDG_STATE_HOME="$d/st" "$d/mrw" --root "$d/r" write --no-check - > "$d/neg.out" 2>&1; [ "$?" = 1 ]; } \
  && grep -q 'is absolute, and every path in a plan is relative to the root' "$d/neg.out" \
  && printf 'one\n' | cmp - "$d/r/dd/zz.txt" \
  && [ ! -e "$d/r/sub" ] \
  && printf '@@ dd/zz.txt 1 replace anchor="one"\nONE\n@@ sub/y.txt 0 create\ny\n' | XDG_STATE_HOME="$d/st" "$d/mrw" --root "$d/r" write --no-check - > "$d/pos.out" 2>&1 \
  && printf 'ONE\n' | cmp - "$d/r/dd/zz.txt" \
  && printf 'y\n' | cmp - "$d/r/sub/y.txt" \
  && bash -n scripts/contract.sh
```

What each clause holds. The guide test passes. The probe runs the binary this tree builds: the rooted plan exits 1 with the sentence and changes nothing (the file is still `one`, no `sub/`), and the relative plan, the positive pair, applies and its two files compare equal to the expected bytes with `cmp`. The pair is executed, not read: deleting it, or changing its input to a rooted one, makes a `cmp` or the exit test fail. §381 itself is run by the contract gate.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheInstructionsSayAPlanPathIsRelativeToTheRoot` | `internal/guide/rooted215_test.go` | the sentence in the full text, the core under 300 words | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the guide test, the probe and §381 |
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

- A release note (external: the release owner writes it from the record's Decision 9)
