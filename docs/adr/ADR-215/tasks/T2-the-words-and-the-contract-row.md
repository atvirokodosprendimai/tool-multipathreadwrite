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
- 2026-11-27 · 5af1c752* · mutant killed · exit 1 · `internal/guide/guide.go` · guide sentence dropped · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · covers:the instructions say a plan path is relative to the root
- 2026-11-27 · 5af1c752* · mutant survived · exit 0 · `docs/adr/ADR-215/tasks/T2-the-words-and-the-contract-row.md` · probe relative-success input made rooted · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · covers:the built binary refuses a rooted plan path and applies the relative one
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-27 · 5af1c752* · mutant survived · exit 0 · `docs/adr/ADR-215/tasks/T2-the-words-and-the-contract-row.md` · negative case expected exit changed · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · covers:the built binary refuses a rooted plan path and applies the relative one
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-11-27 · 5af1c752* · mutant killed · exit 1 · `internal/apply/pathop.go` · refusal sentence changed · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · covers:the built binary refuses a rooted plan path and applies the relative one
- 2026-11-27 · 5af1c752* · mutant killed · exit 1 · `internal/guide/rooted215_test.go` · the named test skips · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · covers:every named test passed and none skipped

## Verification Log
- 2026-11-27 · 5af1c752* · exit 1 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:273 · test-lock-sha256:7001df5066e4392df92ff35561a14f5a77724f9accf521b30c2c766af2be1dbb · test-lock-file:docs/adr/locks/7001df5066e4392df92ff35561a14f5a77724f9accf521b30c2c766af2be1dbb.lock
  ```
  ```
- 2026-11-27 · 5af1c752* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:1205
- 2026-11-27 · 5af1c752* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:2230
- 2026-11-27 · 5af1c752* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:2370
- 2026-11-27 · 5af1c752* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:2418
- 2026-11-27 · 5af1c752* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:2219
- 2026-11-27 · 5af1c752* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:2335
- 2026-11-27 · b7243fb5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:5 · test-lock-sha256:1c115e47cc35a97bc10e424661d2993a2b8b0ae0c2fe71a53ebe5bd781989f8b · test-lock-file:docs/adr/locks/1c115e47cc35a97bc10e424661d2993a2b8b0ae0c2fe71a53ebe5bd781989f8b.lock · test-lock-kind:replace
- 2026-11-27 · b7243fb5* · exit 0 · `d="$(mktemp -d)" \ …` · acceptance-sha256:18b0cfa40de4156bf8a1e747666fa428992492346a9b8f93402434a5f6ba1395 · ms:873

## Invariants

- `mrw instructions --core` stays under 300 words.

## Risks

- Editing `guide.go` can move other records' test locks; S2 lints and relocks them.

## Stop Condition

The fence exits 0; `./scripts/contract.sh` exits 0 on Linux CI with §381 passing; every gate in `AGENTS.md` passes; `adr-debt` reports nothing broken or unreceipted.

## Out of Scope

- A release note (external: release owner: ADR-215 Decision 9)
