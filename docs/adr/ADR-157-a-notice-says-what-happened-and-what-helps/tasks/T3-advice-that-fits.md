# Task ADR-157-T3: the own-state refusal, a doubled BOM, and JSON-path quoting

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.ErrOwnState` in `internal/rooted/rooted.go`; the counted BOM note in `internal/ingest/create.go`; the quoting paragraph in `AGENTS.md` and `README.md`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the own-state refusal advises no --root`, `an outside-root refusal keeps its advice`, `a create counts its byte order marks`, `the docs say how to quote a JSON path`, `contract §262 drives the built binary`

## Goal

Decisions 6–8 of the record, with tests that fail before them.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/rooted.go`, `internal/rooted/hardlink.go` | edit | the four own-state refusals (`rooted.go:185`, `rooted.go:528`, `hardlink.go:23`, `hardlink.go:30` at be76e27) wrap `ErrOwnState` |
| `internal/read/read.go` | edit | `read.go:527` returns the refusal without `read it with --root pointed where you mean` when it is `ErrOwnState` |
| `internal/apply/apply.go` | edit | `resolve` (`apply.go:2505`) returns it without `a plan may only change files under …` |
| `internal/check/check.go` | edit | `check.go:540` returns it without `check it with --root pointed where you mean` |
| `cmd/mrw/ownstate157_test.go` | add | read, write and check of a file in the state directory |
| `internal/ingest/create.go` | edit | `CreateContent` counts the leading byte order marks and names two or more with the count |
| `internal/ingest/bom157_test.go` | add | one, two and three marks |
| `AGENTS.md`, `README.md` | edit | one paragraph each beside the JSON-path address: how pwsh 7, Windows PowerShell 5.1, cmd and Git Bash pass `$["k"]`, and that an unquoted `[0]` is a glob |
| `scripts/contract.sh` | edit | §262 |

## Ordered Steps

1. [S1] Write the failing tests and see them red:
   - `TestTheOwnStateRefusalOffersNoRootAdvice` (cmd/mrw): with `XDG_STATE_HOME` inside the root, `mrw read <state>/seen`, a plan replacing its line 1, and `mrw check <state>` are each refused naming `mrw's own state directory`, and none holds `pointed where you mean` or `may only change files under`. The pair: a path outside the root through a symlink is still refused WITH `pointed where you mean`.
   - `TestACreateNamesEveryLeadingByteOrderMark` (internal/ingest): content beginning with one BOM gets today's note; with two, a note holding `2 UTF-8 byte order marks`; with three, `3 UTF-8 byte order marks`; every mark is kept as content.
2. [S2] `ErrOwnState` wrapped at the four sites, and the three callers skip their advice. [proof: mutation]
3. [S3] `CreateContent` counts the marks. [proof: mutation]
4. [S4] AGENTS.md and README gain the quoting paragraph, opening `Quote a JSON path for the shell`, with the measured forms of the record's Context item 7. [proof: mutation]
5. [S5] Contract §262: the own-state read and plan refused without `--root` advice, beside an outside-root read that keeps it; a `--create` fed two byte order marks names `2 UTF-8 byte order marks`, beside one that names a single mark. [proof: human: the executor runs ./scripts/contract.sh once and reads the §262 rows PASS]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./cmd/mrw/ ./internal/ingest/ -count=1 -timeout 600s -run 'TestTheOwnStateRefusalOffersNoRootAdvice|TestACreateNamesEveryLeadingByteOrderMark|TestTheDocsQuoteAJSONPathPerShell' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestTheOwnStateRefusalOffersNoRootAdvice \(' "$out" \
  && grep -qE '^--- PASS: TestACreateNamesEveryLeadingByteOrderMark \(' "$out" \
  && grep -qE '^--- PASS: TestTheDocsQuoteAJSONPathPerShell \(' "$out" \
  && grep -q 'Quote a JSON path for the shell' AGENTS.md \
  && grep -q 'Quote a JSON path for the shell' README.md \
  && grep -q '^# 262\. ' scripts/contract.sh \
  && fmtout=$(gofmt -l internal cmd) && [ -z "$fmtout" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheOwnStateRefusalOffersNoRootAdvice` | `cmd/mrw/ownstate157_test.go` | read, plan and check of mrw's own state are refused without `--root` advice; an outside-root path keeps it | none | S1, S2 |
| `TestACreateNamesEveryLeadingByteOrderMark` | `internal/ingest/bom157_test.go` | one, two and three marks named by count, all kept | none | S1, S3 |
| `TestTheDocsQuoteAJSONPathPerShell` | `cmd/mrw/docs157_test.go` | AGENTS.md and README each pair every shell with its form for a JSON path (added on the review of #412) | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.ErrOwnState`; the counted note; the paragraph |
| 2 — something selects it | every `rooted.Resolve` on read, write and check; `CreateContent` on every `--create`; contract §262 drives the built binary |
| 3 — the caller can discover it | the refusal line; the `--create` note on stderr; AGENTS.md and README |
| 4 — it is used | nothing measures this; built because two Windows sessions followed the `--root` advice into the same refusal, one found the doubled BOM named as one, and four needed different quoting for `$["k"]` (2026-10-14) |

## Mutation Log
(empty until execute)
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/read/read.go` · a read of the own state is told --root pointed where you mean again · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/apply/apply.go` · a plan naming the own state is told it may only change files under the root again · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/check/check.go` · a check naming the own state is told --root pointed where you mean again · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `internal/ingest/create.go` · two byte order marks are named as one again · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md stops teaching how to quote a JSON path · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
- 2026-10-14 · be76e27f* · mutant survived · exit 0 · `README.md` · README stops giving the PowerShell 5.1 and cmd spellings · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-10-14 · be76e27f* · mutant killed · exit 1 · `README.md` · README stops teaching how to quote a JSON path · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `README.md` · README stops pairing PowerShell 5.1 with its form · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `README.md` · README stops pairing cmd with its form, the mutant that survived the first fence · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `AGENTS.md` · AGENTS.md stops pairing PowerShell 5.1 with its form · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `internal/read/read.go` · a read of the own state is told --root pointed where you mean again · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `internal/apply/apply.go` · a plan naming the own state is told it may only change files under the root again · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `internal/check/check.go` · a check naming the own state is told --root pointed where you mean again · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05
- 2026-10-14 · 85bfc8b5* · mutant killed · exit 1 · `internal/ingest/create.go` · two byte order marks are named as one again · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05

## Verification Log
(empty until execute)
- 2026-10-14 · be76e27f* · exit 1 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:556 · test-lock-sha256:36db19401eff6351150453e59454d54acf323a6250ee95fd65867531fc92737c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9vd25zdGF0ZTE1N190ZXN0LmdvCVRlc3RUaGVPd25TdGF0ZVJlZnVzYWxPZmZlcnNOb1Jvb3RBZHZpY2UJMTQ5MDljNGQ4YzU4YmM3NGQ2M2Q2N2ViM2MyZWYzNjQ2ZDc0YjFmOTFmOWYzY2I4MDZkZDM3MTA0ODBkZWJhZQpib2R5CWludGVybmFsL2luZ2VzdC9ib20xNTdfdGVzdC5nbwlUZXN0QUNyZWF0ZU5hbWVzRXZlcnlMZWFkaW5nQnl0ZU9yZGVyTWFyawlmMTA3ODA5M2E3NmU3ZTYxYjQ2NTNlNzcxZDc3YjUwMWUxYmE3MWJhYTU3NjJlMWU4NjdkYzE1NThiNjRjM2M0
  ```
  --- last 10 line(s) of stdout (of 20 after folding 20 raw)
  --- FAIL: TestTheOwnStateRefusalOffersNoRootAdvice (0.02s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/cmd/mrw	0.138s
  === RUN   TestACreateNamesEveryLeadingByteOrderMark
      bom157_test.go:26: 2 mark(s): notes ["the content begins with a UTF-8 byte order mark, kept as content (PowerShell adds one when $OutputEncoding is UTF-8 with a preamble, and Windows PowerShell 5.1 also when [Console]::InputEncoding is, as in a shell started under code page 65001)"], want one starting "the content begins with 2 UTF-8 byte order marks, kept as content"
      bom157_test.go:26: 3 mark(s): notes ["the content begins with a UTF-8 byte order mark, kept as content (PowerShell adds one when $OutputEncoding is UTF-8 with a preamble, and Windows PowerShell 5.1 also when [Console]::InputEncoding is, as in a shell started under code page 65001)"], want one starting "the content begins with 3 UTF-8 byte order marks, kept as content"
  --- FAIL: TestACreateNamesEveryLeadingByteOrderMark (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest	0.165s
  FAIL
  ```
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2295
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2085
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2589
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2416
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2093
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2043
- 2026-10-14 · be76e27f* · exit 0 · `set -o pipefail …` · acceptance-sha256:61c9f6794b4e80be63e45a840da408adc892e3e1ec290fab79d8cd665cf22546 · ms:2149
- 2026-10-14 · human-observed · observed 2026-10-14: ./scripts/contract.sh exit 0, last line contract holds; the 262 rows PASS, the own-state read and plan with no root advice, the outside-root read with it, two byte order marks named as 2 and one as one
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1283
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1425
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1290
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1428
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1344
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1339
- 2026-10-14 · 85bfc8b5* · exit 0 · `set -o pipefail …` · acceptance-sha256:157b1d7c0fde41c9ab943ff293d6cd28eb3700c3fc3f79d545779c2b50f28b05 · ms:1316

## Invariants

- ADR-077's refusal still refuses: only its trailing advice changes.
- A byte order mark is still kept as content (ADR-142).

## Risks

- None beyond the record's.

## Out of Scope

- Detecting what Windows PowerShell 5.1's ASCII `$OutputEncoding` turned into `?` before mrw saw it (permanent: fact: the bytes are already replaced when mrw reads them; citation: file `AGENTS.md:66`)

## Stop Condition

The fence exits 0 and `./scripts/contract.sh` exits 0.
