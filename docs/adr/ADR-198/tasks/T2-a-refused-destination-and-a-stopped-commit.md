# Task ADR-198-T2: A refused destination or source, and a stopped write's error, begin with the path as the row spells it

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** one leading-path function in `internal/apply`; its use at the four `rooted`-error sites; the same on a CLI `--json` refusal and on `mrw_write`'s error; `TestARefusedDestinationOrSourceBeginsAsTheRowDoes`, `TestAStoppedWritesErrorBeginsAsTheRowDoes`, `TestAStoppedWritesErrorBeginsAsTheRowDoesOnMCP`; a contract section
**Consumes:** `asPlan`, `reasonSep` (T1); `receiptSep` (ADR-132)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a refused destination begins with the path as the row spells it`, `a refused source begins with it after from=`, `text that does not begin with the path is kept whole`, `the CLI json error and its stderr line begin with it`, `the mrw_write error field and line begin with it`, `the built binary keeps a backslash in a refused destination's name`, `the packages the task does not own are unchanged`

## Goal

Decisions 6 and 7 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/lead198_test.go` | new | the red test for Decision 6 |
| `internal/apply/wirepath.go` | edit | one exported function that spells the start of a text with `/` when it begins with an exact path followed by a space or a colon, and does nothing else (not `SlashIn`, which also rewrites `mrw read` advice) |
| `internal/apply/pathop.go` | edit | `pathop.go:195` |
| `internal/apply/apply.go` | edit | `apply.go:743, 770`, the second resolve of a rename destination |
| `internal/apply/fromcopy.go` | edit | `fromcopy.go:169`: `from=P: ERR` converts `P` and the start of `ERR` |
| `cmd/mrw/error198_test.go` | new | the red test for Decision 7 on the CLI |
| `cmd/mrw/main.go` | edit | the `--json` refusal of a landing error is given the error's text converted against `land.Res`, before `shown` (`main.go:2305-2320`) |
| `internal/mcp/error198_test.go` | new | the red test for Decision 7 on `mrw_write` |
| `internal/mcp/tools.go` | edit | the landing error is converted against the result as `Apply` returned it where it is handed to the receipt (`tools.go:1061`), and only that error (a check's or the ledger's is not a stopped write's; the go-reviewer pass on #645); the separator `checkedReceipt` converts with becomes a package variable a test can set |
| `scripts/contract.sh` | edit | the next free section |
| `docs/adr/BACKLOG.md` | edit | the two entries "From ADR-198" are marked taken (the raw error) and closed as a boundary (the identity error) |

## Ordered Steps

1. [S1] Write the failing tests and see each RED.
   - `TestARefusedDestinationOrSourceBeginsAsTheRowDoes` (`internal/apply`), with the reason separator set to `\` where the platform's is `/`. A root with a link `l` to an outside directory whose own name holds a backslash (`o\x`); skipped where links cannot be made, and the test says so. A rename to `l/a\b.txt` gives a reason that begins `l/a/b.txt resolves to ` and holds the outside path byte for byte, backslash included. A `create` with `from=l/a\b.txt:1` gives `from=l/a/b.txt: l/a/b.txt resolves to `. Through the function itself: a text that holds the path only inside the sentence, one that begins with a longer name (`l/a\b.txt2 …`), and one that holds `` `mrw read P` `` are each returned unchanged. With the separator left as the platform's, the rename's reason begins `l/a\b.txt resolves to `.
   - `TestAStoppedWritesErrorBeginsAsTheRowDoes` (`cmd/mrw`), with `receiptSep` set to `\`, in process through the `write` command as the package's tests run it: the plan `replace q\n.txt`, where `q\n.txt` is a directory, stops at its load with its verdict on its hunk (#703); under `--json` the document's `error` and the stderr line both begin `q/n.txt: `, and without `--json` the exit-2 line is what it is today. Amended 2026-11-20: the fixture was the plan `create q\n.txt` beside `create q\n.txt/x`, which stopped at commit; #701 now refuses that pair before anything is staged, so the stop is a directory's. The commit-time forms (`appeared before commit`, `changed after mrw read it`) are covered end to end only at the apply level, by `SlashErr` in `internal/apply/lead198_test.go`; no surface test reaches them.
   - `TestAStoppedWritesErrorBeginsAsTheRowDoesOnMCP` (`internal/mcp`), with the new separator variable set to `\`, through `mrw_write` on the same plan: the structured `error` and the text's `error:` line each begin `q/n.txt: `, asserted separately (amended 2026-11-20, #701/#703, as above).
2. [S2] The conversions. Mutants, each killed:
   - `pathop.go:195` prints the error as it is, so the destination case goes red;
   - `fromcopy.go:169` converts `P` after `from=` and not the start of the error, so the source case goes red;
   - the function replaces the separator everywhere, so the outside path `o\x` case goes red;
   - the function matches a longer name, so the `l/a\b.txt2` case goes red;
   - the CLI converts against the shown result, so the cmd/mrw test goes red;
   - `mrw_write` converts after `Slashed`, so the mcp test goes red.
   [proof: mutation]
3. [S3] The contract section, through the built binary on Linux, a guard that a backslash in a name is kept on these surfaces: a rename to `l/a\b.txt`, `l` a link out of the root, exits 1 and its reason begins `l/a\b.txt resolves to`; the pair: a rename to `l/ok.txt` is refused the same way, and one to `d/ok.txt` applies. [proof: acceptance]
4. [S4] `go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ ./internal/writer/` pass: T1's test and ADR-132's hold unchanged. [proof: acceptance]

Before the merge, outside the fence: the built binary is probed on Windows with a rename to `d/nul/x` (the reason begins `d/nul/x leads to`) and with the stopped-write plan under `--json` and through `mrw_write`.

## Acceptance

```bash
set -o pipefail
a=$(go test ./internal/apply/ -count=1 -v -run '^(TestARefusedDestinationOrSourceBeginsAsTheRowDoes|TestAReasonSpellsAPlansPathAsItsRowDoes)$' 2>&1) \
  && grep -qE '^--- PASS: TestARefusedDestinationOrSourceBeginsAsTheRowDoes \(' <<<"$a" \
  && grep -qE '^--- PASS: TestAReasonSpellsAPlansPathAsItsRowDoes \(' <<<"$a" \
  && b=$(go test ./cmd/mrw/ -count=1 -v -run '^TestAStoppedWritesErrorBeginsAsTheRowDoes$' 2>&1) \
  && grep -qE '^--- PASS: TestAStoppedWritesErrorBeginsAsTheRowDoes \(' <<<"$b" \
  && m=$(go test ./internal/mcp/ -count=1 -v -run '^TestAStoppedWritesErrorBeginsAsTheRowDoesOnMCP$' 2>&1) \
  && grep -qE '^--- PASS: TestAStoppedWritesErrorBeginsAsTheRowDoesOnMCP \(' <<<"$m" \
  && go test ./internal/apply/ ./cmd/mrw/ ./internal/mcp/ ./internal/writer/ -count=1 >/dev/null 2>&1 \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  a refused rename destination keeps the backslash in its name' <<<"$c" \
  && grep -qF 'PASS  the pair: a destination through the link is refused, and one in the root applies' <<<"$c" \
  && f=$(gofmt -l internal cmd) && [ -z "$f" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state internal/seen internal/writer internal/rooted \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state internal/seen internal/writer internal/rooted)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARefusedDestinationOrSourceBeginsAsTheRowDoes` | `internal/apply/lead198_test.go` | a refusal `rooted` words for a destination or a source begins with the path spelled with `/`; anything else is kept | none | S1, S2 |
| `TestAStoppedWritesErrorBeginsAsTheRowDoes` | `cmd/mrw/error198_test.go` | a CLI `--json` refusal's `error` and stderr line begin with the path spelled with `/`, through the command | none | S1, S2 |
| `TestAStoppedWritesErrorBeginsAsTheRowDoesOnMCP` | `internal/mcp/error198_test.go` | `mrw_write`'s `error` and `error:` line begin with it, through the tool | none | S1, S2 |
| `TestAReasonSpellsAPlansPathAsItsRowDoes` | `internal/apply/reason198_test.go` | T1's cases hold | none | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the tests |
| 2 — something selects it | a refused rename destination or source in `Apply`; every CLI `--json` landing refusal; every `mrw_write` receipt |
| 3 — the caller can discover it | it is the refusal itself |
| 4 — it is used | no telemetry (ADR-009); the evidence is the Windows probe of 0d49d6e8 (`d\nul\x leads to "nul"`) and, for the error line, a Windows 11 probe of bfeae770, an earlier head of this task: the error begins `q/n.txt appeared before commit` in the `--json` field, the stderr line, and `mrw_write`'s field and `error:` line. Later heads were not probed by hand |

## Mutation Log
- 2026-11-15 · bfeae770* · mutant killed · exit 1 · `internal/apply/pathop.go` · the destination site prints the rooted error as it is, so the refused destination case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:a refused destination begins with the path as the row spells it
- 2026-11-15 · bfeae770* · mutant killed · exit 1 · `internal/apply/fromcopy.go` · the source site converts P after from= and not the start of the error, so the refused source case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:a refused source begins with it after from=
- 2026-11-15 · bfeae770* · mutant killed · exit 1 · `internal/apply/wirepath.go` · SlashLead replaces the separator in the whole text, so the outside directory named with a backslash must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:text that does not begin with the path is kept whole
- 2026-11-15 · bfeae770* · mutant killed · exit 1 · `internal/apply/wirepath.go` · SlashLead matches a longer name that begins with the path, so the longer-name case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:text that does not begin with the path is kept whole
- 2026-11-15 · bfeae770* · mutant killed · exit 1 · `cmd/mrw/main.go` · the CLI converts against the shown result, whose paths are already slashed, so the json error case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:the CLI json error and its stderr line begin with it
- 2026-11-15 · bfeae770* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write converts against the slashed result, so the mcp error case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:the mrw_write error field and line begin with it
- 2026-11-15 · 4e02b7d9* · mutant killed · exit 1 · `internal/mcp/tools.go` · mrw_write hands the landing error to the receipt as it is, so the mcp error case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:the mrw_write error field and line begin with it
- 2026-11-15 · 4e02b7d9* · mutant killed · exit 1 · `internal/apply/wirepath.go` · SlashErr takes the first path that begins the text, so the longest-path case must go red · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · covers:text that does not begin with the path is kept whole

## Verification Log
- 2026-11-15 · bfeae770* · exit 1 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:3601 · test-lock-sha256:bfd1c0fa3f1fc8565d3132267dc5497261cda0caf19c1c9870782a35ab5a91e7 · test-lock-file:docs/adr/locks/bfd1c0fa3f1fc8565d3132267dc5497261cda0caf19c1c9870782a35ab5a91e7.lock
  ```
  ```
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:239412
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:199907
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:184558
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:178731
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:186802
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:192046
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:181993
- 2026-11-15 · bfeae770* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:0 · test-lock-sha256:50e1aff56c98c8ae1c1cc14502df9c9e2461602abd5b26c9a1c93f9955e25e74 · test-lock-file:docs/adr/locks/50e1aff56c98c8ae1c1cc14502df9c9e2461602abd5b26c9a1c93f9955e25e74.lock · test-lock-kind:replace
- 2026-11-15 · bfeae770* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:176409
- 2026-11-15 · 4e02b7d9* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:0 · test-lock-sha256:7842f78d15284fd32970a70c33cea5a3e629f830e22c6244a4ca165c14f951a9 · test-lock-file:docs/adr/locks/7842f78d15284fd32970a70c33cea5a3e629f830e22c6244a4ca165c14f951a9.lock · test-lock-kind:replace
- 2026-11-15 · 4e02b7d9* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:167345
- 2026-11-15 · 4e02b7d9* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:166559
- 2026-11-15 · 4e02b7d9* · exit 0 · `set -o pipefail …` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:168035
- 2026-11-15 · 58365dc0* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:1 · test-lock-sha256:81bae941edc9710a2366d509735201b84f7a8197c061c5f5984ac82b04cd106f · test-lock-file:docs/adr/locks/81bae941edc9710a2366d509735201b84f7a8197c061c5f5984ac82b04cd106f.lock · test-lock-kind:replace
- 2026-11-20 · 2ded135b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:0 · test-lock-sha256:0a467df9f299574c33676ed1bba17a6c9868b68f0a437a7de07f632905b4dd6e · test-lock-file:docs/adr/locks/0a467df9f299574c33676ed1bba17a6c9868b68f0a437a7de07f632905b4dd6e.lock · test-lock-kind:replace
- 2026-11-20 · 2ded135b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:0 · test-lock-sha256:f1b0b9f35298301330f74a9f82c34fb0c08e66e68c2631020051415607b1dce0 · test-lock-file:docs/adr/locks/f1b0b9f35298301330f74a9f82c34fb0c08e66e68c2631020051415607b1dce0.lock · test-lock-kind:replace
- 2026-11-20 · 2ded135b* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:0 · test-lock-sha256:f1b0b9f35298301330f74a9f82c34fb0c08e66e68c2631020051415607b1dce0 · test-lock-file:docs/adr/locks/f1b0b9f35298301330f74a9f82c34fb0c08e66e68c2631020051415607b1dce0.lock · test-lock-kind:replace
- 2026-11-20 · a4040443* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:0 · test-lock-sha256:b240cb25fe4b714f5525c8dc17ae97e62031bf2359137658fd63de89ec79f6b1 · test-lock-file:docs/adr/locks/b240cb25fe4b714f5525c8dc17ae97e62031bf2359137658fd63de89ec79f6b1.lock · test-lock-kind:replace
- 2026-11-26 · a92f3800* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6bf1f9f7692ec54d5d492c3d605878303575b1b5985e457ffb11b7b06f692b0f · ms:50 · test-lock-sha256:70ddbcab77fd7536b21305b8c8e7bb62975f98ab06fdef84e22adac0ee046b68 · test-lock-file:docs/adr/locks/70ddbcab77fd7536b21305b8c8e7bb62975f98ab06fdef84e22adac0ee046b68.lock · test-lock-kind:replace

## Invariants

- Where the separator is `/`, every reason and error is byte for byte what it is today.
- The error `Apply` returns and every path field keep the platform's spelling.

## Risks

- `apply.go:743, 770` are reached only when a destination's resolution changes between two resolves; no fixture is promised for them, and they share the function with `pathop.go:195`.
- The link cases skip where links need a privilege, so on such a machine the destination and source cases prove nothing; the parent test's PASS line does not show a skip.

## Stop Condition

The fence exits 0 and the Windows probe reads `d/nul/x leads to`. Stop and ask Zy if a test pins the platform's spelling at the start of `mrw_write`'s `error` or of a CLI `--json` `error`.

## Out of Scope

- `mrw cannot read the file identity of P` (permanent: boundary: the record keeps it)
