# Task ADR-168-T1: `mrw skill` and `mrw --skill` print the skill

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** `guide.Skill(version)`, `mrw skill`, the `--skill` rewrite, contract §293 (print rows), the AGENTS.md `mrw skill` bullet
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the guide test run`, `the format test passed`, `the body test passed`, `the command test run`, `the command test passed`, `every subcommand reaches AGENTS.md`, `the contract run`, `the print rows pass`, `gofmt`, `one module requirement`, `the engine packages it does not own`

## Goal

Decision 1: `mrw skill`, and `mrw --skill`, print an Agent Skills `SKILL.md` whose frontmatter a host accepts and whose body is `guide.Core()` and `guide.Recipes()` verbatim, then the closing paragraph; no arguments; no state touched. AGENTS.md names `mrw skill`, because `TestEverySubcommandReachesTheAgentFacingGuide` (`cmd/mrw/agentsdoc_test.go:36-55`) refuses a subcommand it does not.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/guide/guide.go` | edit | `Skill(version string) string` |
| `internal/guide/skill168_test.go` | add | the format and body tests |
| `cmd/mrw/main.go` | edit | `skillCmd()` in the root command list — the line that makes it reachable; the `--skill` rewrite in `main` after `liftLoneDash` and `refusePaddedFlagValues`; `skill` in both verb lists of `Before` (`:169`, `:175`) |
| `cmd/mrw/skill168_test.go` | add | the command test |
| `AGENTS.md` | edit | §5: a `mrw skill` bullet |
| `scripts/contract.sh` | edit | §293 print rows |

## Ordered Steps

1. [S1] Write, and confirm RED:
   - `TestTheSkillIsAValidAgentSkill`: the text opens with a `---` frontmatter block; `name: mrw`, matching `^[a-z0-9]+(-[a-z0-9]+)*$` within 64 characters, equal to the folder name `mrw`; `description: >-` with every following line indented two spaces, the folded text 1–1024 bytes, naming `every file edit`, `mrw write`, `--grep` and `PATH`; `metadata:` holding exactly two string entries, `mrw-version` equal to the argument and `generated-by` equal to `mrw skill`, each double-quoted; stamps `v1.72.0 (18ebc2f)`, `dev` and `1.0` round-trip as strings; the body under 500 lines.
   - `TestTheSkillBodyIsTheCoreAndTheRecipes`: the body holds `Core()` then `Recipes()` byte for byte, and the closing paragraph names `` `mrw instructions` `` backticked, `mrw skill --dir` and the version.
   - `TestSkillCommandPrintsTheSkill`: `mrw skill`, `mrw --skill` and `mrw -C d --skill` print `guide.Skill(versionString())` and exit 0; `mrw skill x` exits 2; `mrw --root --skill instructions` runs `instructions` (the token is a value); `-skill`, `--skill=x` and `mrw read --skill` exit 2, the first two naming `mrw skill`; with a seeded legacy state tree and a ledger written by a newer mrw (the fixtures of `cmd/mrw/startup145_test.go`), `mrw skill` neither migrates nor reports, while `mrw read` on the same fixture does both.
2. [S2] `guide.Skill`, `skillCmd`, the rewrite, the verb lists, the AGENTS.md bullet. Mutants: the description grows past 1024 bytes; `name` becomes `mrw-skill`; the body drops `Recipes()`; `metadata` loses `generated-by`; the command is not registered; the rewrite also takes `--skill=x`; the rewrite takes a root flag's value; `skill` is left out of `Before`'s lists. [proof: mutation]
3. [S3] Contract §293 print rows, each paired with its failing case: `mrw skill and mrw --skill print the same skill` (and `mrw skill x` is exit 2). [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/guide/ -count=1 -timeout 300s -v -run 'TestTheSkillIsAValidAgentSkill$|TestTheSkillBodyIsTheCoreAndTheRecipes$' 2>&1) \
  && grep -qE '^--- PASS: TestTheSkillIsAValidAgentSkill \(' <<<"$out" \
  && grep -qE '^--- PASS: TestTheSkillBodyIsTheCoreAndTheRecipes \(' <<<"$out" \
  && out=$(go test ./cmd/mrw/ -count=1 -timeout 300s -v -run 'TestSkillCommandPrintsTheSkill$|TestEverySubcommandReachesTheAgentFacingGuide$' 2>&1) \
  && grep -qE '^--- PASS: TestSkillCommandPrintsTheSkill \(' <<<"$out" \
  && grep -qE '^--- PASS: TestEverySubcommandReachesTheAgentFacingGuide \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -qF 'PASS  mrw skill and mrw --skill print the same skill' <<<"$c" \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheSkillIsAValidAgentSkill` | `internal/guide/skill168_test.go` | the frontmatter keeps the Agent Skills rules, the folded description and the string metadata | none | S1, S2 |
| `TestTheSkillBodyIsTheCoreAndTheRecipes` | `internal/guide/skill168_test.go` | the body is Core and Recipes verbatim, then the closing paragraph | none | S1, S2 |
| `TestSkillCommandPrintsTheSkill` | `cmd/mrw/skill168_test.go` | both spellings, the argument matrix, no state read or written | none | S1, S2 |
| `TestEverySubcommandReachesTheAgentFacingGuide` | `cmd/mrw/agentsdoc_test.go` | AGENTS.md names `mrw skill` | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `guide.Skill` |
| 2 — something selects it | `skillCmd()` in the root command list; the rewrite in `main` |
| 3 — the caller can discover it | `mrw --help` lists the command; AGENTS.md; T3 adds the first help line |
| 4 — it is used | none yet; no telemetry (ADR-009) |

## Invariants

- `mrw instructions` and `mrw instructions --core` print what they printed.
- The MCP handshake document is unchanged.
- Every spelling of a subcommand-named flag other than a bare `--skill` keeps ADR-097's pointer.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if `Core()` plus `Recipes()` exceed 5,000 tokens or 500 lines, or if the rewrite cannot tell a root flag's value from a subcommand without restating the parser: the record would then need a choice it did not make.

## Out of Scope

- `--dir` (permanent: boundary: T2 of this record owns it)

## Mutation Log
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the description grows past 1024 bytes · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: the body drops Recipes() · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: the command is not registered · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: main never applies the --skill rewrite · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/main.go` · S2: skill left out of the ledger verb list in Before · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/skill168.go` · S2: the rewrite takes a root flag's value · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: name becomes mrw-skill, not the folder · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `internal/guide/guide.go` · S2: metadata loses generated-by · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7
- 2026-10-22 · 3a2e963e* · mutant killed · exit 1 · `cmd/mrw/skill168.go` · S2: the rewrite also takes --skill=x · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7

## Verification Log
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:134 · test-lock-sha256:23cdcdf3ab66957d1225b6fa40e1ce5a76badf3b7440b6149676d10e6165ca54 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJYzM4MzI4OTM0MDM3ZGYxNTlhYWFkMjlmN2Q5NjIyNWYyNjc0Yjc3MzkzNDNiNTE0YjZmODJlNDYwMDRkOWU2Ywpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxDb21tYW5kUHJpbnRzVGhlU2tpbGwJOTQ2NjhlNjZhNjJjZGEzYzJmNzUzZjVhZDNmODcwNTI1YTExYWZkOTI4M2I1ZTg1Y2IxOTlhZTVjY2M1OGZmZQpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsQm9keUlzVGhlQ29yZUFuZFRoZVJlY2lwZXMJOTUzMjYwOWY4YTE5YzE5NDVhYTdjNTQyZDQxZDU2YmNiYTg2OGJlNGY3YzJiMTAyNTY1OWU5ZjUzMmY3YTI4Mgpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsSXNBVmFsaWRBZ2VudFNraWxsCWE3YTM2ZmM4MTY1Y2QxODMwODg1MTM5ZjA2YzExN2E1YjMzNTg2YmVjMjI4ODNlOWJlMWY3ZjJiOThiZjZiZmM
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:83135
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:137462
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:81540
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:91147
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:101900
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:91837
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:95578
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:87839
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:143065
- 2026-10-22 · 3a2e963e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:0 · test-lock-sha256:54ed4a9fe26d77066c856cc5bb0e795580376830aba242f65296659c0339b2f9 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJNmZjNzQ4MmZjMDE4ZTRkMjRiMWI0ZDFkNWRhNDhlOTZhYTJhNzZiOTI2ZDM2ODEzNmQ1NjA5MWFkYWQ5OTRkOQpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk0NjY4ZTY2YTYyY2RhM2MyZjc1M2Y1YWQzZjg3MDUyNWExMWFmZDkyODNiNWU4NWNiMTk5YWU1Y2NjNThmZmUKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JNWNjN2JjMDcwNTZkM2RkMTlmOGRlNGU3ZmI1MjIzN2YyMmJkMjBhNDQwMTIxNjNmNTA2OWUyZmFmYzFlNzM3NApib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlUmVwb3NpdG9yeVNraWxsSXNUaGVCaW5hcnlzU2tpbGwJMDhhYjNmODdiMGYwNzZkNGM2ZGNiNGE0NjVhMTIxYTVjMThmNTU5ZTY1YWIwMTFhYTYwOGVlZDQxNDhlMGJjNQpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsQm9keUlzVGhlQ29yZUFuZFRoZVJlY2lwZXMJOTUzMjYwOWY4YTE5YzE5NDVhYTdjNTQyZDQxZDU2YmNiYTg2OGJlNGY3YzJiMTAyNTY1OWU5ZjUzMmY3YTI4Mgpib2R5CWludGVybmFsL2d1aWRlL3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVNraWxsSXNBVmFsaWRBZ2VudFNraWxsCWE3YTM2ZmM4MTY1Y2QxODMwODg1MTM5ZjA2YzExN2E1YjMzNTg2YmVjMjI4ODNlOWJlMWY3ZjJiOThiZjZiZmM · test-lock-kind:replace
- 2026-10-22 · 3a2e963e* · exit 1 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:94247
  ```
  ```
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:79330
- 2026-10-22 · 3a2e963e* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:0 · test-lock-sha256:d03dfbb9afd4bf592316bdeafdfbbc36353ce1d00e6cfc24287355f4288af834 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0RXZlcnlTdWJjb21tYW5kUmVhY2hlc1RoZUFnZW50RmFjaW5nR3VpZGUJNmZjNzQ4MmZjMDE4ZTRkMjRiMWI0ZDFkNWRhNDhlOTZhYTJhNzZiOTI2ZDM2ODEzNmQ1NjA5MWFkYWQ5OTRkOQpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFJlYWRtZUFuZFNraWxsTmFtZUluc3RydWN0aW9ucwkwZTY0OTE2NjFlY2QwOTQ3N2YxMmNmMDE5ZjJkOWM2YTI3ZDY4MDlkZjBmNmQ4MDYwNGU1YWFjNWNkNzBlYTFjCmJvZHkJY21kL21ydy9hZ2VudHNkb2NfdGVzdC5nbwlUZXN0UmVhZG1lRG9lc05vdFRlYWNoQUxlZGdlclJhY2UJNGMwMDljMzgwOGQzNjY5MTYwMjk0YjkwMWVkZWY4ZGUyZDM0N2M0ZTNjODcwMTI0OWFjMTJmMDUxMTg3ZWVhYgpib2R5CWNtZC9tcncvYWdlbnRzZG9jX3Rlc3QuZ28JVGVzdFRoZUdhdGVEb2VzTm90QWNjZXB0QVByZWZpeE9mQW5vdGhlckNvbW1hbmQJNzgzZWViMDI4ZjkwZTdlYTQwNTQ2MWU2ZmVkOTBhOGNkZDI5ODY1NTg4NjQ1MWU4ZTdjMDY2OGJiODYxM2E2NQpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0Um9vdEhlbHBQb2ludHNBZ2VudHNBdFRoZVNraWxsCTNjY2ZkMDlmNWJlMjMwZDdmMTE1N2M1NGFmMDczNTMxMjhhODg1OTk4ZWNkYmMzMmVjYzEzNDVjY2JiOGI4ZTEKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsQ29tbWFuZFByaW50c1RoZVNraWxsCTk0NjY4ZTY2YTYyY2RhM2MyZjc1M2Y1YWQzZjg3MDUyNWExMWFmZDkyODNiNWU4NWNiMTk5YWU1Y2NjNThmZmUKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFNraWxsRGlySW5zdGFsbHNBbmRSZXBsYWNlc09ubHlJdHNPd24JYTYxMjAxOTBmMjVjNWZhMGVlOTg0NDFlODM2NGNhYTY5MzFkZTk1N2Y2MzFhMGMzYjQxNmYwZjUxNDk5OTE5Mwpib2R5CWNtZC9tcncvc2tpbGwxNjhfdGVzdC5nbwlUZXN0U2tpbGxEaXJSZWFkc09ubHlUaGVTaGFwZUl0UHJpbnRzCWZjMmJkZDZiYWNjZjY3OTRiMTA3Yjc1OGI1YzNlOTkyMmI1M2M0NDQ5MjQ2MmZjNTVmNDRiYWI2NjBmNDkxOTYKYm9keQljbWQvbXJ3L3NraWxsMTY4X3Rlc3QuZ28JVGVzdFRoZVJlcG9zaXRvcnlTa2lsbElzVGhlQmluYXJ5c1NraWxsCTA4YWIzZjg3YjBmMDc2ZDRjNmRjYjRhNDY1YTEyMWE1YzE4ZjU1OWU2NWFiMDExYWE2MDhlZWQ0MTQ4ZTBiYzUKYm9keQlpbnRlcm5hbC9ndWlkZS9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVTa2lsbEJvZHlJc1RoZUNvcmVBbmRUaGVSZWNpcGVzCTk1MzI2MDlmOGExOWMxOTQ1YWE3YzU0MmQ0MWQ1NmJjYmE4NjhiZTRmN2MyYjEwMjU2NTllOWY1MzJmN2EyODIKYm9keQlpbnRlcm5hbC9ndWlkZS9za2lsbDE2OF90ZXN0LmdvCVRlc3RUaGVTa2lsbElzQVZhbGlkQWdlbnRTa2lsbAlhN2EzNmZjODE2NWNkMTgzMDg4NTEzOWYwNmMxMTdhNWIzMzU4NmJlYzIyODgzZTliZTFmN2YyYjk4YmY2YmZjCmJvZHkJaW50ZXJuYWwvZ3VpZGUvc2tpbGwxNjhfdGVzdC5nbwlUZXN0VGhlU2tpbGxTYXlzSG9zdFBhdGhSdWxlc0RvTm90UmVhY2hNcncJNzU4YzA0NDNiYzI4ZTNkOGQyMzU4MjliMTYwNmY4NWFjZjQ4MjRjZmI0ZTYwZTkwMDZkYzZhZGM5ZDBjYWNlMQ · test-lock-kind:replace
- 2026-10-22 · 3a2e963e* · exit 0 · `set -o pipefail …` · acceptance-sha256:6a870f73c514de6c17a09637fad87612e6a68c682c99ea05d049bdf371caf9e7 · ms:95704
