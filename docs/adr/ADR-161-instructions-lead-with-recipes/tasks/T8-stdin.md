# Task ADR-161-T8: mrw read - reads standard input

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (one source of lines, its refusals, one row)
**Owner:** Zy
**Produces:** `mrw read -`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `standard input is read like a file`, `it licenses nothing`

## Goal

A command's output is narrowed with mrw's own addresses and caps, not with `grep`, `sed -n` or `head`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/read/read.go` | edit | `Spec.Data` served in place of a file, the file loading moved to `load`, and `ReadStdin` bounded as a file is (ADR-104); served as `==> -` |
| `cmd/mrw/stdin.go` | add | `stdinRead` and its refusals; `liftLoneDash` |
| `cmd/mrw/main.go` | edit | `-` routed to `stdinRead`; `main` lifts a lone `-` before parsing (what selects both) |
| `cmd/mrw/stdin161_test.go` | add | `TestReadingStandardInputLicensesNothing`, `TestALoneDashMovesLast` |
| `internal/guide/guide.go` | edit | one sentence naming `-` |
| `scripts/contract.sh` | edit | §268; §137 (ADR-069 T8) now judges a padded token after a lone `-` |

## Ordered Steps

1. [S1] Write the failing test and the contract row and see them red. [proof: acceptance]
2. [S2] `mrw read [ranges and flags] -` serves standard input with every address, `--grep`, `-C`, `--max-lines` and `--max-cols`; header `==> -`; the 1 GiB bound applies; nothing recorded. [proof: mutation]
3. [S3] Usage error beside `--sub`, `--quiet`, `--stat`, `--count`, `--files-from` or another path; `./-` names a file called `-`. [proof: mutation]
4. [S4] Contract §268: `printf 'a\nb\nc\n' | $MRW read --grep b -` serves line 2, exit 0; then a plan for a file the same command never read is refused; the pair, `... | $MRW read --grep z -`, exits 1 `no match:`. [proof: mutation]
5. [S5] A lone `-` that is a positional moves last before parsing, on every subcommand, so the flags after it are read: urfave/cli stops at a lone `-` and drops the rest, and `write - --dry-run` wrote (Zy, 2026-10-16: "Honor them"). It moves only when every token after it is a flag or a flag's value; anything else after a lone `-` (a path, a second `-`, a `--`) would be dropped, so it is refused by name, as is a flag left without its value. A `-` that is a flag's value, a padded `' - '`, a dash before a non-letter, one before the subcommand and one after `--` stay. A terminal on standard input is refused before it is read (S2). [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/read/ ./cmd/mrw/ ./internal/guide/ -count=1 -v 2>&1) \
  && grep -qE '^--- PASS: TestReadingStandardInputLicensesNothing ' <<<"$out" \
  && test -z "$(gofmt -l .)" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a read of standard input serves it and licenses nothing' <<<"$c"
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestReadingStandardInputLicensesNothing` | `cmd/mrw/stdin161_test.go` | addresses, `--grep`, `-C`, `--max-lines`, `--max-cols` over stdin; nothing recorded; `no match:`; `./-` a file; the refusals; `write - --dry-run` dry-runs and writes nothing; `--files-from -` untouched | none | S2, S3, S5 |
| `TestALoneDashMovesLast` | `cmd/mrw/stdin161_test.go` | a lone `-` moves last after the subcommand and its flags; a flag's value `-`, one before the subcommand, after `--`, or already last stays | none | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the stdin source in `read.Run` |
| 2 — something selects it | the path `-` |
| 3 — the caller can discover it | `--help`, recipe 9 (T3) |
| 4 — it is used | agents narrowing output |

## Mutation Log
(empty until execute)
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S2: standard input is read and then a file named - is looked for instead · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S3: --quiet beside - is not refused · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/main.go` · S5: a lone - ends the parse again and write - --dry-run writes · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/main.go` · S4: - is read as a file named - · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S5: a second lone - drops the flags after it again (Codex review of T8) · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S5: an inherited root flag is not known to take a value, so its - is moved (Codex review of T8) · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S5: a path or a second - after a lone - is moved instead of refused, then dropped · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S5: - is moved to become the value of a flag left without one · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0
- 2026-10-16 · c1be2bc0* · mutant killed · exit 1 · `cmd/mrw/stdin.go` · S2: a terminal on standard input is waited on (Codex review of T8) · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0

## Invariants

- A read of a path is unchanged; nothing read from standard input licenses a line.

## Risks

- A caller pipes a file in to dodge the ledger; it licenses nothing, so a write still needs a read of the file.

## Stop Condition

Reading standard input would block a read that has no pipe; then `-` is refused on a terminal.

## Out of Scope

- `mrw_read` reading a request body as a file (permanent: boundary: the MCP surface has no standard input to give)

## Verification Log
(empty until execute)
- 2026-10-16 · c1be2bc0* · exit 1 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:48275 · test-lock-sha256:deeb36e69eee92091337010c87f199c96273cf4f30607b345e4a41e96cc4ff11 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIz
  ```
  ```
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:127140
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:128339
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:128522
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:128749
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:129204
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:133049
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:133482
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:133077
- 2026-10-16 · c1be2bc0* · exit 0 · `set -o pipefail …` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:134478
- 2026-10-16 · c1be2bc0* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:3a92f16fe9bf65433b48c423c8d4e88c4ec69a38da7fef924129c26221447cb0 · ms:0 · test-lock-sha256:daa709c31bed9fa2e21f64d0adc1aaed8290c03c7255640b6996ee1e2f23ee62 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9zdGRpbjE2MV90ZXN0LmdvCVRlc3RBTG9uZURhc2hNb3Zlc0xhc3QJODA1YTVjZmU2NDkxMGRhMzU4MmYxNmI3MDQ2NzljYWFmMTVlYzNjOWE3ZGViNGU0OWYyMmEyNTI4NDdkNGRhYQpib2R5CWNtZC9tcncvc3RkaW4xNjFfdGVzdC5nbwlUZXN0UmVhZGluZ1N0YW5kYXJkSW5wdXRMaWNlbnNlc05vdGhpbmcJNGNlZTEwMzdmMTY4YTc2MzIzZDVlMzYwZWViODIzZWNiNGFlNGE4NmQ2ZmU0MmU2ZDlkNTM5ZTU0NGJkOGEzYQ · test-lock-kind:replace
