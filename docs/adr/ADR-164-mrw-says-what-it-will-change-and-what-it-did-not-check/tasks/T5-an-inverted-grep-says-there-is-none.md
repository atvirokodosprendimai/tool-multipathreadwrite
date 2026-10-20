# Task ADR-164-T5: -v and -o on read get a pointer

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `foreignFlags` entries `v` and `o` in `cmd/mrw/foreignflag.go`; the pointer on the padded-value refusal
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `-v names it`, `-o names it`, `the padded shape names it`, `-z unchanged`, `the binary prints it`

## Goal

Decision 5: `mrw read --grep -v P …` (both the no-match and the padded-value shape), `mrw read -v …` and `mrw read -o …` end with a pointer.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/foreignflag.go` | edit | two entries; the comment that says `-v, -o` have no pointer |
| `cmd/mrw/main.go` | edit | the padded-argument refusal of the value after `--grep` appends the table's pointer when the flag taken as the pattern is in the table |
| `cmd/mrw/foreignflag140_test.go` | edit | line 45's negative case (`--grep -v` gets no hint) moves to `-z`; `TestEveryForeignFlagHintIsATrueSpelling` gains `-v`/`-o` cases (`--max-cols`) |
| `cmd/mrw/foreignflag164_test.go` | add | the new test |
| `scripts/contract.sh` | edit | §277 |

## Ordered Steps

1. [S1] Write the failing test and see it red: TestAnInvertedGrepSaysThereIsNone (`mrw read --grep -v a x` ends with `there is no inverted match` through `patternFlagHint`; `mrw read --grep -v ' a' x` and `--grep -i ' a' x` keep their edge-whitespace refusal and end with their pointers; `mrw read -v x` reaches `usageError` with the pointer; `mrw read -o x` names `--max-cols`; `--grep -z a x` gets no hint). [proof: acceptance]
2. [S2] The two entries, each run against the built binary first as ADR-140's were. [proof: mutation]
3. [S3] The pointer on the padded-value refusal. [proof: mutation]
4. [S4] Rewrite `foreignflag140_test.go`'s `-v` negative case to `-z`; add the `-v`/`-o` spelling cases. [proof: acceptance]
5. [S5] Contract §277: `--grep -v` exits as today with the pointer; the pair: `--grep -i` still prints ADR-140's pointer. [proof: human: the executor runs ./scripts/contract.sh once and reads the §277 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAnInvertedGrepSaysThereIsNone$' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAnInvertedGrepSaysThereIsNone \(' "$out" \
  && go test ./cmd/mrw/ -count=1 -timeout 300s -run 'TestAForeignReadFlagIsAnsweredWithMrwsSpelling|TestEveryForeignFlagHintIsATrueSpelling' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAForeignReadFlagIsAnsweredWithMrwsSpelling \(' "$out" \
  && grep -qE '^--- PASS: TestEveryForeignFlagHintIsATrueSpelling \(' "$out" \
  && grep '^# 277\. ' scripts/contract.sh > /tmp/adr164-t5-row.txt
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnInvertedGrepSaysThereIsNone` | `cmd/mrw/foreignflag164_test.go` | both `--grep -v` shapes, `-v`, `-o`; `-z` unchanged | none | S1, S2, S3 |
| `TestAForeignReadFlagIsAnsweredWithMrwsSpelling` | `cmd/mrw/foreignflag140_test.go` | ADR-140's entries; the negative case now on `-z` | none | S4 |
| `TestEveryForeignFlagHintIsATrueSpelling` | `cmd/mrw/foreignflag140_test.go` | the spellings, `-v`/`-o` cases added | none | S4 |


## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the two entries; the padded-value pointer |
| 2 — something selects it | `patternFlagHint` and `usageError` read `flagTables` (ADR-140); the padded-argument check now does too; contract §277 drives the binary |
| 3 — the caller can discover it | the message itself |
| 4 — it is used | nothing measures this; met once by this session (2026-10-20) |

## Mutation Log
(empty until execute)
- 2026-10-20 · abd7d4fe* · mutant killed · exit 1 · `cmd/mrw/foreignflag.go` · the table has no -v entry · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f
- 2026-10-20 · abd7d4fe* · mutant killed · exit 1 · `cmd/mrw/main.go` · the padded-value refusal never names the pointer · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f

## Verification Log
(empty until execute)
- 2026-10-20 · abd7d4fe* · exit 1 · `set -o pipefail …` · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f · ms:628 · test-lock-sha256:db3f46037cb53b635e38126d6d2143677e0586cb2dd80d3aed6f1227d9c8db06 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mb3JlaWduZmxhZzE0MF90ZXN0LmdvCVRlc3RBRm9yZWlnblJlYWRGbGFnSXNBbnN3ZXJlZFdpdGhNcndzU3BlbGxpbmcJYjdjNDU3M2EwNDNiYzMzY2E0ZmU1MjIzMGVlZDI2OGYzODE3M2M0ODU1OWZhNjE2NGE5ZTY1MWI4M2Y4NWEwNgpib2R5CWNtZC9tcncvZm9yZWlnbmZsYWcxNDBfdGVzdC5nbwlUZXN0RXZlcnlGb3JlaWduRmxhZ0hpbnRJc0FUcnVlU3BlbGxpbmcJMTZjNThjYWRhNmU3Y2I2NzYxNzdiOWIxMTdkNzgwM2U1MjQ4MWMyM2RmY2Y3MmUzZjM3M2UxNjZlOTdiOWM2OA
  ```
  --- last 10 line(s) of stdout (of 16 after folding 16 raw)
      foreignflag164_test.go:38: mrw ["read" "--grep" "-i" " a" "f.txt"]: lacks "(?i)":
          ' a' has edge whitespace the argument parser strips; put -- before the path: mrw read -- ' a'
      foreignflag164_test.go:38: mrw ["read" "-v" "f.txt"]: lacks "there is no inverted match: --grep serves the lines that match":
          flag provided but not defined: -v (see: mrw read --help)
      foreignflag164_test.go:38: mrw ["read" "-o" "f.txt"]: lacks "there is no match-only output: --max-cols N cuts each line to a window round its match":
          flag provided but not defined: -o (see: mrw read --help)
  --- FAIL: TestAnInvertedGrepSaysThereIsNone (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.262s
  FAIL
  ```
- 2026-10-20 · abd7d4fe* · exit 0 · `set -o pipefail …` · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f · ms:2117
- 2026-10-20 · abd7d4fe* · exit 0 · `set -o pipefail …` · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f · ms:1809
- 2026-10-20 · human-observed · observed by the executor 2026-10-20: S5 ./scripts/contract.sh exit 0 ('contract holds'); the §277 rows PASS — --grep -v a exits 1 saying there is no inverted match, --grep -v ' a' exits 2 naming it beside the padded-value refusal, and --grep -i keeps ADR-140's pointer; S4 the -v negative case of foreignflag140_test.go now runs on -z · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f
- 2026-10-20 · 5bf52ac6 · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:41ee32fed105a1f457221c9b8af6ac1dae6bf6e9430911584776d9dc5ae9db4f · ms:0 · test-lock-sha256:56e2536b151cf171b16b6207c07571660ffb259ef56ba09017540cc5593bce62 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9mb3JlaWduZmxhZzE0MF90ZXN0LmdvCVRlc3RBRm9yZWlnblJlYWRGbGFnSXNBbnN3ZXJlZFdpdGhNcndzU3BlbGxpbmcJM2RhZDBmZDMwZDg5NjY1YTY0ZDllODhmZTYzMGE2OGJiNjNhMDY3YzhmM2QxYjcyZjRjNjhhZGIwMGRiNDczZApib2R5CWNtZC9tcncvZm9yZWlnbmZsYWcxNDBfdGVzdC5nbwlUZXN0RXZlcnlGb3JlaWduRmxhZ0hpbnRJc0FUcnVlU3BlbGxpbmcJZmZkMWRkMDZiNDUzYzhiMzdjYTgwOTVlN2U3YzhlM2NmNWIwMzljMjdjMGQ4YWQ5NmM1MjNkOGEwODkxMWRmYQpib2R5CWNtZC9tcncvZm9yZWlnbmZsYWcxNjRfdGVzdC5nbwlUZXN0QW5JbnZlcnRlZEdyZXBTYXlzVGhlcmVJc05vbmUJZjE4NmVhMzEzMmZhZjdkMjk2MWI3ZDJjNWNiZTgwZTI3N2M1OTE2YzhmYTljY2RmNTdjNTAxZDU2ZDk2OWJjOQ · test-lock-kind:replace

## Invariants

- Exit codes unchanged; every existing `foreignFlags` entry unchanged; the padded refusal's existing words unchanged.

## Risks

- None beyond the record's.

## Out of Scope

- `-z` (permanent: boundary: no honest pointer; ADR-140)

## Stop Condition

Stop and bring it to Zy if the padded-argument check (it runs before the read, `cmd/mrw/main.go`) cannot tell which flag became the pattern without re-parsing argv.
