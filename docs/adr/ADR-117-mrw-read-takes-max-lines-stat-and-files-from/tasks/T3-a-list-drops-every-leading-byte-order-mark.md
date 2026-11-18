# Task ADR-117-T3: A spec list drops every leading byte order mark and counts them

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `speclist.Parse`: every byte order mark at the head of the first line dropped, the note counting them when there are several; `TestAListDropsEveryLeadingByteOrderMarkAndCountsThem`; a contract section; the exception stated in `AGENTS.md`
**Consumes:** `speclist.Parse` (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every leading mark is dropped`, `the note counts the marks`, `marks past the first line are kept`

## Goal

A `--files-from` list that starts with more than one UTF-8 byte order mark serves its first spec.
Measured on main at 808d18eb, 2026-11-18, by the release review of v1.88.0 (Codex) and reproduced:
`printf '\357\273\277\357\273\277b.txt:1\n' | mrw read --files-from -` printed `dropped the byte order
mark at the start of the list`, then the path with the second mark in front of it as `UNREADABLE`,
exit 1. Windows PowerShell 5.1 pipes two marks in one measured configuration (`internal/ingest/marks.go`),
and the plan readers already drop every one and say how many (issue #537). T2 dropped one.

The same review found that `AGENTS.md` still said a `--files-from` list keeps the path as written, with
no exception for the mark T2 drops. This task states the exception and how to name a file whose name
really begins with U+FEFF.

Decided by Zy, 2026-11-18: *"Fix these"*, to the two findings.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/speclist/marks_test.go` | new | the red test |
| `internal/speclist/speclist.go` | edit | drop every mark at the head of the first line; count them in the note |
| `scripts/contract.sh` | edit | the next free section, through the built binary |
| `AGENTS.md` | edit | the exception to "keep the path as written", and the two spellings that reach a name beginning with U+FEFF |

## Ordered Steps

1. [S1] Write the failing test `TestAListDropsEveryLeadingByteOrderMarkAndCountsThem` and see it RED: two marks leave one in front of the first spec. Its cases: two marks before a first spec and three before a first comment are dropped and counted; one mark keeps T2's note byte for byte; marks at the start of a later line, and a mark after `./`, stay the spec's own with no note. [proof: acceptance]
2. [S2] `Parse` drops every mark at the head of the first line. One mark writes T2's note; several write `mrw: <label>: dropped N byte order marks at the start of the list`. Mutants, each killed: only the first mark is dropped; the count is wrong; marks are dropped on every line. [proof: mutation]
3. [S3] The contract section, through the built binary: a list led by two marks serves its file, exit 0, and standard error counts them; the pair: two marks at the start of the second line leave that spec UNREADABLE, exit 1, with no note; and with a file whose name begins with U+FEFF beside one that does not, a first entry written `./NAME` serves the one with the mark. [proof: acceptance]
4. [S4] `AGENTS.md` states the exception and the two spellings. [proof: acceptance]
5. [S5] `go test ./internal/speclist/ ./internal/mcp/` pass. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/speclist/ -count=1 -v -run '^TestAListDropsEveryLeadingByteOrderMarkAndCountsThem$' 2>&1) \
  && grep -qE '^--- PASS: TestAListDropsEveryLeadingByteOrderMarkAndCountsThem \(' <<<"$out" \
  && go test ./internal/speclist/ ./internal/mcp/ -count=1 >/dev/null 2>&1 \
  && t=$(mktemp -d) && mkdir "$t/r" && printf 'x\n' > "$t/r/a.txt" \
  && go build -o "$t/mrw" ./cmd/mrw \
  && printf '\357\273\277\357\273\277a.txt\n' | env XDG_STATE_HOME="$t/state" "$t/mrw" --root "$t/r" read --files-from - >"$t/out" 2>"$t/err" \
  && grep -q '^==> a.txt' "$t/out" \
  && grep -qxF 'mrw: --files-from -: dropped 2 byte order marks at the start of the list' "$t/err" \
  && rm -rf "$t" \
  && grep -q '^# 342\. ' scripts/contract.sh \
  && grep -qF 'ADR-117 T3' AGENTS.md \
  && f=$(gofmt -l internal cmd) && [ -z "$f" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/check internal/state internal/seen \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/apply internal/check internal/state internal/seen)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAListDropsEveryLeadingByteOrderMarkAndCountsThem` | `internal/speclist/marks_test.go` | two and three leading marks are dropped and counted; one mark keeps T2's note; marks past the first line and a mark after `./` are kept, unremarked | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `mrw read --files-from` and every `mrw_read` with `files_from` calls `Parse`; the fence and the contract section drive the built binary |
| 3 — the caller can discover it | the note on standard error names the list and the count; `AGENTS.md` states the exception |
| 4 — it is used | no telemetry (ADR-009); the evidence is the measurement in Goal |

## Mutation Log
- 2026-11-18 · 808d18eb* · mutant killed · exit 1 · `internal/speclist/speclist.go` · only the first mark is dropped, so the two-mark and three-mark cases must go red · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · covers:every leading mark is dropped
- 2026-11-18 · 808d18eb* · mutant killed · exit 1 · `internal/speclist/speclist.go` · the note counts one mark too many, so the counted notes must go red · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · covers:the note counts the marks
- 2026-11-18 · 808d18eb* · mutant killed · exit 1 · `internal/speclist/speclist.go` · marks are dropped on every line, so the marks at the start of a later line must go red · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · covers:marks past the first line are kept

## Verification Log
- 2026-11-18 · 808d18eb* · exit 1 · `set -o pipefail …` · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · ms:751 · test-lock-sha256:96f0448b85f70a0d5e819690b8f52815c21300923472c475a0b6fee54eb9074d · test-lock-file:docs/adr/locks/96f0448b85f70a0d5e819690b8f52815c21300923472c475a0b6fee54eb9074d.lock
  ```
  ```
- 2026-11-18 · 808d18eb* · exit 0 · `set -o pipefail …` · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · ms:39723
- 2026-11-18 · 808d18eb* · exit 0 · `set -o pipefail …` · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · ms:31065
- 2026-11-18 · 808d18eb* · exit 0 · `set -o pipefail …` · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · ms:32383
- 2026-11-18 · 808d18eb* · exit 0 · `set -o pipefail …` · acceptance-sha256:6304dc91bf7440ee741e504e01b4edb04ae2d6067fe928c2e5abcb04e9850b1f · ms:38519

## Invariants

- A list with no leading mark parses exactly as before, and a list with one mark gets T2's note byte for byte.
- The note goes to the writer the caller hands in and nowhere else; standard output is unchanged.

## Risks

- A list whose first spec is meant to begin with U+FEFF is renamed, now however many marks precede it. `AGENTS.md` names the two spellings that reach such a file.
- `mrw_read` drops several marks as silently as it drops one; the note on its answer stays deferred (T2).

## Stop Condition

The fence exits 0.

## Out of Scope

- A note on `mrw_read`'s answer for a `files_from` list with marks (deferred: docs/adr/BACKLOG.md)
- A list in UTF-16 (permanent: boundary: mrw reads a list as UTF-8 bytes and does not decode another encoding; the specs are reported as the paths they spell)
